<script>
  import { onMount } from 'svelte';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { games, activeView } from '../../stores/library.js';
  import { AppAPI } from '../../api/app.js';
  import PersonaTab from './settings/PersonaTab.svelte';
  import LibraryStorageTab from './settings/LibraryStorageTab.svelte';
  import CompatibilityTab from './settings/CompatibilityTab.svelte';
  import MediaApisTab from './settings/MediaApisTab.svelte';
  import ToolsMaintenanceTab from './settings/ToolsMaintenanceTab.svelte';
  import LogsTab from './settings/LogsTab.svelte';

  let activeTab = 'persona'; // 'persona' | 'library' | 'compatibility' | 'media' | 'tools' | 'logs'
  let cfgDraft = {
    library_paths: [],
    default_runner: 'umu',
    operating_mode: 'loader',
    language: 'auto',
    mangohud: false,
    steamless: true,
    normalize: true,
    auto_backup_saves: false,
    backup_retention_count: 5,
    steam_web_api_key: '',
    hubcap_api_key: '',
    steamgriddb_api_key: '',
    rawg_api_key: '',
    artwork_dir: '',
    backup_dir: '',
    wine_prefix: '',
    proton_path: '',
    dxvk_hud: '',
    enable_launch_notify: true,
    achievements: true,
    transfer_speed_limit: 0,
    save_manifest_url: '',
    global_env: {}
  };
  let userCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  let isSaving = false;
  let saveSuccess = false;
  let saveSuccessTimer = null;

  // Toolchain state
  let toolStatuses = [];
  let isCheckingTools = false;
  let updatingTools = {};
  let isSubmittingBatch = false;

  const runnersList = [
    { value: 'umu', label: 'UMU (Unified Linux Wine/Proton Runtime - Recommended)' },
    { value: 'wine', label: 'Wine (System Wine Binary)' }
  ];

  const languagesList = [
    { value: 'auto', label: 'Auto (System Locale)' },
    { value: 'english', label: 'English' },
    { value: 'french', label: 'French' },
    { value: 'german', label: 'German' },
    { value: 'italian', label: 'Italian' },
    { value: 'spanish', label: 'Spanish' },
    { value: 'japanese', label: 'Japanese' },
    { value: 'schinese', label: 'Simplified Chinese' },
    { value: 'russian', label: 'Russian' },
    { value: 'brazilian', label: 'Brazilian Portuguese' },
    { value: 'koreana', label: 'Korean' },
    { value: 'polish', label: 'Polish' }
  ];

  const countriesList = [
    { value: 'US', label: 'United States (US)' },
    { value: 'DE', label: 'Germany (DE)' },
    { value: 'FR', label: 'France (FR)' },
    { value: 'GB', label: 'United Kingdom (GB)' },
    { value: 'CA', label: 'Canada (CA)' },
    { value: 'JP', label: 'Japan (JP)' },
    { value: 'KR', label: 'South Korea (KR)' },
    { value: 'CN', label: 'China (CN)' },
    { value: 'RU', label: 'Russia (RU)' },
    { value: 'BR', label: 'Brazil (BR)' },
    { value: 'AU', label: 'Australia (AU)' }
  ];

  onMount(async () => {
    try {
      const urlParams = new URLSearchParams(window.location.search);
      const tabParam = urlParams.get('tab');
      if (tabParam) {
        if (['persona', 'player', 'user'].includes(tabParam)) activeTab = 'persona';
        else if (['library', 'storage', 'paths'].includes(tabParam)) activeTab = 'library';
        else if (['compatibility', 'compat', 'runner'].includes(tabParam)) activeTab = 'compatibility';
        else if (['media', 'api', 'apis'].includes(tabParam)) activeTab = 'media';
        else if (['tools', 'maintenance', 'components'].includes(tabParam)) activeTab = 'tools';
        else if (['logs', 'log', 'activity'].includes(tabParam)) activeTab = 'logs';
        else if (['emu', 'drm'].includes(tabParam)) activeTab = 'persona';
      }

      const cfg = await AppAPI.getConfig();
      if (cfg) {
        cfgDraft = JSON.parse(JSON.stringify(cfg));
      }
      const userCfg = await AppAPI.getGSEUserConfig();
      if (userCfg) {
        userCfgDraft = { ...userCfgDraft, ...userCfg };
      }
      await loadToolsStatus();
    } catch (err) {
      console.warn('Failed to load initial settings:', err);
    }
  });

  async function loadToolsStatus() {
    isCheckingTools = true;
    try {
      const statuses = await AppAPI.getToolsStatus();
      toolStatuses = Array.isArray(statuses) ? statuses : Object.entries(statuses || {}).map(([k, v]) => ({ key: k, ...v }));
    } catch {
      toolStatuses = [];
    } finally {
      isCheckingTools = false;
    }
  }

  async function handleSave() {
    if (!cfgDraft) return;
    isSaving = true;
    try {
      await AppAPI.saveConfig(cfgDraft);
      await AppAPI.saveGSEUserConfig(userCfgDraft);
      saveSuccess = true;
      if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
      saveSuccessTimer = setTimeout(() => {
        saveSuccess = false;
      }, 3000);

      notificationStore.items.add({
        type: 'success',
        title: 'Settings Saved',
        message: 'System and player configuration updated successfully.'
      });
      configStore.reload();
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Save Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSaving = false;
    }
  }

  async function handleDiscard() {
    const cfg = await AppAPI.getConfig();
    if (cfg) cfgDraft = JSON.parse(JSON.stringify(cfg));
    const userCfg = await AppAPI.getGSEUserConfig();
    if (userCfg) userCfgDraft = { ...userCfgDraft, ...userCfg };

    notificationStore.items.add({
      type: 'info',
      title: 'Changes Discarded',
      message: 'Restored saved configuration settings.'
    });
  }

  async function handleUpdateTool(e) {
    const { key, name } = e.detail;
    if (!key) return;
    updatingTools[key] = true;
    updatingTools = { ...updatingTools };

    try {
      await AppAPI.updateTool(key);
      notificationStore.items.add({
        type: 'success',
        title: 'Tool Updated',
        message: `${name || key} is now at the latest version.`
      });
      await loadToolsStatus();
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Update Failed',
        message: `Failed to update ${name || key}: ${err?.message || String(err)}`
      });
    } finally {
      updatingTools[key] = false;
      updatingTools = { ...updatingTools };
    }
  }

  async function handleBatchAchievements() {
    isSubmittingBatch = true;
    try {
      await AppAPI.batchFetchAchievements();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Achievements Started',
        message: 'Downloading achievement schemas and assets for your library.'
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchArtwork() {
    isSubmittingBatch = true;
    try {
      await AppAPI.batchFetchArtwork();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Artwork Started',
        message: 'Fetching missing game artwork and covers from providers.'
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchApply() {
    isSubmittingBatch = true;
    try {
      await AppAPI.batchApplySetup();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Setup Started',
        message: 'Applying Goldberg emulators and Steamless unpackers to unpatched games.'
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchNormalize() {
    isSubmittingBatch = true;
    try {
      await AppAPI.batchNormalizeDirectories();
      notificationStore.items.add({
        type: 'info',
        title: 'Normalization Started',
        message: 'Standardizing game directory folder names to official Steam titles.'
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchVerify() {
    isSubmittingBatch = true;
    try {
      await AppAPI.batchVerifyManifests();
      notificationStore.items.add({
        type: 'info',
        title: 'Verification Started',
        message: 'Verifying SHA-1 file integrity against depot manifests.'
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }
</script>

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-12 max-w-4xl mx-auto w-full space-y-4 text-xs sm:text-sm animate-fade-in custom-scrollbar select-none">
  <!-- Minimalist Title & Save Bar -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-[#1c1c28]">
    <div>
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2.5">
        System Settings
      </h2>
    </div>

    <!-- Header Actions -->
    <div class="flex items-center space-x-2 w-full sm:w-auto justify-end">
      {#if saveSuccess}
        <span class="text-xs font-mono font-bold text-emerald-400 flex items-center space-x-1 animate-fade-in pr-2">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <span>Saved</span>
        </span>
      {/if}

      <button
        type="button"
        on:click={handleDiscard}
        class="px-3.5 py-1.5 rounded-xl bg-[#14141e] hover:bg-[#1e1e2c] border border-[#242436] text-slate-400 hover:text-white font-bold text-xs transition cursor-pointer"
      >
        Discard
      </button>

      <button
        type="button"
        on:click={handleSave}
        disabled={isSaving}
        class="bg-white hover:bg-neutral-200 disabled:opacity-50 text-black font-black px-5 py-1.5 rounded-xl transition text-xs shadow flex items-center space-x-1.5 cursor-pointer"
      >
        {#if isSaving}
          <svg class="animate-spin w-3.5 h-3.5 text-black" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Saving...</span>
        {:else}
          <span>Save Changes</span>
        {/if}
      </button>
    </div>
  </div>

  <!-- Segmented Top Tabs -->
  <div class="flex items-center space-x-1.5 overflow-x-auto pb-1 text-xs">
    <button
      type="button"
      on:click={() => { activeTab = 'persona'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'persona' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <span>Player Persona</span>
    </button>

    <button
      type="button"
      on:click={() => { activeTab = 'library'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'library' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      <span>Library & Storage</span>
    </button>

    <button
      type="button"
      on:click={() => { activeTab = 'compatibility'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'compatibility' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Compatibility</span>
    </button>

    <button
      type="button"
      on:click={() => { activeTab = 'media'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'media' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-pink-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <span>Media & APIs</span>
    </button>

    <button
      type="button"
      on:click={() => { activeTab = 'tools'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'tools' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Tools & Maintenance</span>
    </button>

    <button
      type="button"
      on:click={() => { activeTab = 'logs'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'logs' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <span>Logs & Activity</span>
    </button>
  </div>

  <!-- Tab Contents -->
  {#if cfgDraft}
    {#if activeTab === 'persona'}
      <PersonaTab
        bind:userCfgDraft
        {languagesList}
        {countriesList}
      />
    {:else if activeTab === 'library'}
      <LibraryStorageTab
        bind:cfgDraft
      />
    {:else if activeTab === 'compatibility'}
      <CompatibilityTab
        bind:cfgDraft
        {runnersList}
      />
    {:else if activeTab === 'media'}
      <MediaApisTab
        bind:cfgDraft
      />
    {:else if activeTab === 'tools'}
      <ToolsMaintenanceTab
        {toolStatuses}
        {isCheckingTools}
        {updatingTools}
        {isSubmittingBatch}
        gamesCount={($games || []).length}
        unpatchedGamesCount={($games || []).filter(g => !g.is_patched && !g.isPatched).length}
        on:checkAll={loadToolsStatus}
        on:updateTool={handleUpdateTool}
        on:batchAchievements={handleBatchAchievements}
        on:batchArtwork={handleBatchArtwork}
        on:batchApply={handleBatchApply}
        on:batchNormalize={handleBatchNormalize}
        on:batchVerify={handleBatchVerify}
      />
    {:else if activeTab === 'logs'}
      <LogsTab />
    {/if}
  {/if}
</div>
