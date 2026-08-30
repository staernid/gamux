<script>
  import { onMount } from 'svelte';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { games, activeView } from '../../stores/library.js';
  import { AppAPI } from '../../api/app.js';
  import SettingSection from '../common/SettingSection.svelte';
  import SeamlessSwitch from '../common/SeamlessSwitch.svelte';
  import InheritSelect from '../common/InheritSelect.svelte';
  import GhostInput from '../common/GhostInput.svelte';
  import StepperRow from '../common/StepperRow.svelte';

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
    steamgriddb_api_key: '',
    rawg_api_key: '',
    artwork_dir: '',
    backup_dir: '',
    wine_prefix: '',
    proton_path: '',
    dxvk_hud: '',
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

  // Logs state
  let logFilter = 'all'; // 'all' | 'error' | 'warning' | 'info' | 'success'
  const notifItems = notificationStore.items;

  $: filteredLogs = ($notifItems || []).filter(item => {
    if (logFilter === 'all') return true;
    return item.type === logFilter;
  });

  // Advanced toggles
  let showAdvancedStorage = false;
  let showAdvancedCompat = false;

  // Env variable input in global settings
  let newEnvKey = '';
  let newEnvVal = '';

  const runnersList = [
    { value: 'umu', label: 'UMU (Unified Linux Wine/Proton Runtime - Recommended)' },
    { value: 'proton', label: 'Proton (Direct Valve Proton Execution)' },
    { value: 'wine', label: 'Wine (System Wine Binary)' },
    { value: 'linux', label: 'Linux Native (Direct ELF Binary)' }
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
      saveSuccessTimer = setTimeout(() => { saveSuccess = false; }, 3000);
      notificationStore.items.add({
        type: 'success',
        title: 'Settings Saved',
        message: 'System settings and user persona updated successfully.'
      });
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

  async function handleAddLibraryPath() {
    try {
      const dir = await AppAPI.selectDirectory('Select Game Library Directory', '');
      if (dir && cfgDraft) {
        if (!cfgDraft.library_paths) cfgDraft.library_paths = [];
        if (!cfgDraft.library_paths.includes(dir)) {
          cfgDraft.library_paths = [...cfgDraft.library_paths, dir];
        }
      }
    } catch (err) {
      console.warn('Directory selection failed:', err);
    }
  }

  function handleRemoveLibraryPath(path) {
    if (cfgDraft && cfgDraft.library_paths) {
      cfgDraft.library_paths = cfgDraft.library_paths.filter(p => p !== path);
    }
  }

  function generateRandomSteamID() {
    const rand = Math.floor(100000000 + Math.random() * 900000000);
    userCfgDraft.account_steamid = `76561198${rand}`;
  }

  async function handleUpdateTool(key, displayName) {
    updatingTools = { ...updatingTools, [key]: true };
    try {
      await AppAPI.updateTool(key);
      await loadToolsStatus();
      notificationStore.items.add({
        type: 'info',
        title: 'Download Queued',
        message: `${displayName || key} update queued in background.`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Tool Update Failed',
        message: err?.message || String(err)
      });
    } finally {
      updatingTools = { ...updatingTools, [key]: false };
    }
  }

  async function handleBatchAchievements() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchLibraryOp('achievements');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Achievements Queued',
        message: 'Downloading missing achievement schemas and icons in background.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchArtwork() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchLibraryOp('artwork');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Artwork Queued',
        message: 'Resolving and caching missing posters, heroes, and banners in background.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchApply() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchLibraryOp('apply');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Setup Queued',
        message: 'Applying emulator and unpacker configuration across all unpatched titles.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchNormalize() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchLibraryOp('normalize');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Normalization Queued',
        message: 'Standardizing directory names across all library games.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchVerify() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchLibraryOp('verify');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Integrity Verification Queued',
        message: 'Verifying depot file integrity for all manifest-backed titles.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  function copyLogsToClipboard() {
    if (!$notifItems || $notifItems.length === 0) return;
    const text = $notifItems.map(item => {
      const time = new Date(item.timestamp).toISOString();
      const level = (item.type || 'info').toUpperCase().padEnd(7);
      const title = item.title ? `[${item.title}] ` : '';
      const game = item.gameName ? `(${item.gameName}) ` : '';
      const details = item.details ? ` - ${item.details}` : '';
      return `${time} ${level} ${title}${game}${item.message || ''}${details}`;
    }).join('\n');

    navigator.clipboard?.writeText(text);
    notificationStore.items.add({
      type: 'success',
      title: 'Logs Copied',
      message: `Copied ${$notifItems.length} log lines to clipboard.`
    });
  }

  function addGlobalEnv() {
    const k = newEnvKey.trim().toUpperCase();
    if (!k || !cfgDraft) return;
    if (!cfgDraft.global_env) cfgDraft.global_env = {};
    cfgDraft.global_env = { ...cfgDraft.global_env, [k]: newEnvVal.trim() };
    newEnvKey = '';
    newEnvVal = '';
  }

  function removeGlobalEnv(k) {
    if (!cfgDraft || !cfgDraft.global_env) return;
    const updated = { ...cfgDraft.global_env };
    delete updated[k];
    cfgDraft.global_env = updated;
  }
</script>

<div class="flex-1 flex flex-col max-w-5xl mx-auto w-full px-5 py-5 space-y-4 animate-fade-in pb-16">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#1c1c28] pb-3.5">
    <div class="space-y-0.5">
      <h1 class="text-lg font-bold text-white tracking-tight flex items-center space-x-2">
        <span>System Settings</span>
      </h1>
      <p class="text-[11px] text-slate-500 font-mono">
        ~/.config/gamux/config.json
      </p>
    </div>

    <!-- Actions -->
    <div class="flex items-center space-x-3">
      {#if saveSuccess}
        <span class="text-xs font-mono font-bold text-emerald-400 flex items-center space-x-1 animate-fade-in">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <span>Saved</span>
        </span>
      {/if}

      <button
        type="button"
        on:click={handleSave}
        disabled={isSaving || !cfgDraft}
        class="px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs shadow-md transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isSaving}
          <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Saving...</span>
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
          </svg>
          <span>Save Changes</span>
        {/if}
      </button>
    </div>
  </div>

  <!-- Primary Navigation Pills -->
  <div class="flex items-center space-x-1 bg-[#0c0c12] p-1 rounded-xl border border-[#1e1e2c] overflow-x-auto">
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
      <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
      <span>Compatibility</span>
    </button>

    <button
      type="button"
      on:click={() => { activeTab = 'media'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'media' ? 'bg-[#1e1e2c] text-white shadow-sm border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
    <!-- ================= 1. PLAYER PERSONA ================= -->
    {#if activeTab === 'persona'}
      <div class="space-y-3">
        <SettingSection title="Virtual Steam Persona">
          <div class="space-y-2.5">
            <GhostInput
              label="Display Name"
              description="Persona nickname used in local emulated multiplayer and offline save profiles"
              bind:value={userCfgDraft.account_name}
              placeholder="GamuxPlayer"
            />

            <InheritSelect
              label="Emulation Language"
              description="Default language injected into Steam emulators and Goldberg Steamworks wrappers"
              bind:value={userCfgDraft.language}
              options={languagesList}
            />

            <GhostInput
              label="Steam ID (64-Bit)"
              description="Unique 64-bit numerical identifier for local emulator save isolation and DLC entitlement"
              bind:value={userCfgDraft.account_steamid}
              placeholder="76561197960287930"
              mono={true}
            >
              <button
                slot="action"
                type="button"
                on:click={generateRandomSteamID}
                class="px-3 py-1.5 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
                title="Generate a new randomized Steam ID"
              >
                <svg class="w-3.5 h-3.5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                <span>Random</span>
              </button>
            </GhostInput>

            <InheritSelect
              label="Region Code (IP Country)"
              description="Two-letter ISO country code reported by Steamworks emulation (e.g. US, DE, JP)"
              bind:value={userCfgDraft.ip_country}
              options={countriesList}
            />
          </div>
        </SettingSection>
      </div>

    <!-- ================= 2. LIBRARY & STORAGE ================= -->
    {:else if activeTab === 'library'}
      <div class="space-y-3">
        <!-- Library Folders -->
        <SettingSection
          title="Library Folders"
        >
          <svelte:fragment slot="action">
            <button
              type="button"
              on:click={handleAddLibraryPath}
              class="px-2.5 py-1 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-bold transition-colors flex items-center space-x-1 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              <span>Add Folder</span>
            </button>
          </svelte:fragment>

          {#if (cfgDraft.library_paths || []).length === 0}
            <div class="p-3 bg-[#0d0d14] border border-[#1e1e2c] rounded-xl text-center">
              <p class="text-xs text-slate-400">No library directories configured. Add a directory to start scanning.</p>
            </div>
          {:else}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
              {#each cfgDraft.library_paths as p}
                <div class="flex items-center justify-between p-2.5 bg-[#0d0d14] border border-[#1e1e2c] rounded-lg">
                  <div class="flex items-center space-x-2 min-w-0">
                    <svg class="w-3.5 h-3.5 text-emerald-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                    </svg>
                    <span class="font-mono text-xs text-slate-200 truncate">{p}</span>
                  </div>
                  <button
                    type="button"
                    on:click={() => handleRemoveLibraryPath(p)}
                    class="p-1 text-slate-400 hover:text-rose-400 transition-colors cursor-pointer flex-shrink-0 ml-2"
                    title="Remove library directory"
                    aria-label={`Remove library path ${p}`}
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        </SettingSection>

        <!-- Automation & Save Backups -->
        <SettingSection
          title="Automation & Backups"
        >
          <div class="space-y-2.5">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
              <SeamlessSwitch
                label="Normalize Folders"
                description="Standardize folder names to official Steam game titles"
                bind:value={cfgDraft.normalize}
              />

              <SeamlessSwitch
                label="Auto-Backup Saves"
                description="Create a rolling save backup before launching any game"
                bind:value={cfgDraft.auto_backup_saves}
              />
            </div>

            {#if cfgDraft.auto_backup_saves}
              <div class="pt-1">
                <StepperRow
                  label="Backup Retention Count"
                  min={1}
                  max={30}
                  bind:value={cfgDraft.backup_retention_count}
                />
              </div>
            {/if}
          </div>

          <!-- Advanced Storage Accordion -->
          <div class="pt-3 mt-3 border-t border-[#1a1a26]">
            <button
              type="button"
              on:click={() => { showAdvancedStorage = !showAdvancedStorage; }}
              class="text-xs font-semibold text-slate-400 hover:text-slate-200 flex items-center space-x-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 transform transition-transform {showAdvancedStorage ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
              <span>Advanced Storage Locations</span>
            </button>

            {#if showAdvancedStorage}
              <div class="pt-3 grid grid-cols-1 md:grid-cols-2 gap-3 animate-fade-in">
                <GhostInput
                  label="Artwork Directory"
                  bind:value={cfgDraft.artwork_dir}
                  placeholder="~/.local/share/gamux/art"
                  mono={true}
                />

                <GhostInput
                  label="Backup Directory"
                  bind:value={cfgDraft.backup_dir}
                  placeholder="~/.local/share/gamux/backups"
                  mono={true}
                />
              </div>
            {/if}
          </div>
        </SettingSection>
      </div>

    <!-- ================= 3. COMPATIBILITY ================= -->
    {:else if activeTab === 'compatibility'}
      <div class="space-y-3">
        <SettingSection
          title="Runtime & Compatibility"
        >
          <div class="space-y-2.5">
            <InheritSelect
              label="Default Runner"
              bind:value={cfgDraft.default_runner}
              options={runnersList}
            />

            <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
              <SeamlessSwitch
                label="MangoHud Overlay"
                description="FPS and performance telemetry overlay"
                bind:value={cfgDraft.mangohud}
              />

              <SeamlessSwitch
                label="DRM Unpacking"
                description="Automatically run Steamless unpacker during setup"
                bind:value={cfgDraft.steamless}
              />
            </div>
          </div>

          <!-- Advanced Compatibility Options -->
          <div class="pt-3 mt-3 border-t border-[#1a1a26]">
            <button
              type="button"
              on:click={() => { showAdvancedCompat = !showAdvancedCompat; }}
              class="text-xs font-semibold text-slate-400 hover:text-slate-200 flex items-center space-x-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 transform transition-transform {showAdvancedCompat ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
              <span>Advanced Compatibility & Environment</span>
            </button>

            {#if showAdvancedCompat}
              <div class="pt-3 space-y-3 animate-fade-in">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <GhostInput
                    label="Default Wine Prefix"
                    bind:value={cfgDraft.wine_prefix}
                    placeholder="~/.local/share/wineprefixes/default"
                    mono={true}
                  />

                  <GhostInput
                    label="Proton Distribution Path"
                    bind:value={cfgDraft.proton_path}
                    placeholder="~/.local/share/Steam/compatibilitytools.d/GE-Proton"
                    mono={true}
                  />
                </div>

                <!-- Global Environment Variables -->
                <div class="pt-2 border-t border-[#1a1a26] space-y-2">
                  <span class="text-xs font-bold text-slate-200 uppercase tracking-wider block">
                    Global Environment Variables
                  </span>

                  {#if Object.keys(cfgDraft.global_env || {}).length > 0}
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-1.5">
                      {#each Object.keys(cfgDraft.global_env) as ek}
                        <div class="flex items-center justify-between p-2 bg-[#0e0e14] border border-[#1e1e2c] rounded-lg">
                          <div class="flex items-center space-x-2 font-mono text-xs min-w-0 truncate">
                            <span class="text-white font-bold truncate">{ek}</span>
                            <span class="text-slate-500">=</span>
                            <span class="text-sky-300 truncate">{cfgDraft.global_env[ek]}</span>
                          </div>
                          <button
                            type="button"
                            on:click={() => removeGlobalEnv(ek)}
                            class="p-1 text-slate-400 hover:text-rose-400 transition-colors cursor-pointer flex-shrink-0"
                            aria-label={`Remove global environment variable ${ek}`}
                          >
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                    </div>
                  {/if}

                  <div class="flex items-center gap-2 pt-1">
                    <input
                      type="text"
                      placeholder="VAR_NAME"
                      bind:value={newEnvKey}
                      class="flex-1 bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono uppercase outline-none transition-colors"
                    />
                    <input
                      type="text"
                      placeholder="value"
                      bind:value={newEnvVal}
                      class="flex-1 bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono outline-none transition-colors"
                    />
                    <button
                      type="button"
                      on:click={addGlobalEnv}
                      class="px-3 py-1.5 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex-shrink-0 cursor-pointer"
                    >
                      Add
                    </button>
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </SettingSection>
      </div>

    <!-- ================= 4. MEDIA & APIS ================= -->
    {:else if activeTab === 'media'}
      <div class="space-y-3">
        <SettingSection
          title="Metadata & Artwork APIs"
          tooltip="Configure third-party API keys to enrich your library with game metadata, official news, and high-resolution artwork."
        >
          <div class="space-y-2.5">
            <GhostInput
              label="Steam Web API Key"
              description="Used to fetch official Steam depot manifests, DLC catalogs, community news, and store metadata"
              type="password"
              bind:value={cfgDraft.steam_web_api_key}
              placeholder="32-char hex key (e.g. 7F8A1B2C...)"
              mono={true}
              status={cfgDraft.steam_web_api_key ? 'SET' : ''}
            />

            <GhostInput
              label="SteamGridDB API Key"
              description="Automated downloads for high-resolution vertical covers, banners, hero artwork, and clear logos"
              type="password"
              bind:value={cfgDraft.steamgriddb_api_key}
              placeholder="SteamGridDB API token"
              mono={true}
              status={cfgDraft.steamgriddb_api_key ? 'SET' : ''}
            />

            <GhostInput
              label="RAWG API Key"
              description="Provides fallback descriptions, release dates, genres, and metadata discovery for standalone titles"
              type="password"
              bind:value={cfgDraft.rawg_api_key}
              placeholder="RAWG API key"
              mono={true}
              status={cfgDraft.rawg_api_key ? 'SET' : ''}
            />

            <GhostInput
              label="Artwork Storage Directory"
              description="Local filesystem cache directory where downloaded covers, heroes, and logos are persisted"
              bind:value={cfgDraft.artwork_dir}
              placeholder="~/.local/share/gamux/art"
              mono={true}
            >
              <button
                slot="action"
                type="button"
                on:click={async () => {
                  const selected = await AppAPI.selectDirectory('Select Artwork Directory', cfgDraft.artwork_dir);
                  if (selected) cfgDraft.artwork_dir = selected;
                }}
                class="px-3 py-1.5 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
              >
                Browse
              </button>
            </GhostInput>
          </div>
        </SettingSection>
      </div>

    <!-- ================= 5. TOOLS & MAINTENANCE ================= -->
    {:else if activeTab === 'tools'}
      <div class="space-y-3">
        <!-- Toolchain Management -->
        <SettingSection
          title="Helper Tools"
          tooltip="External tools for Steam emulation, DRM removal, and Proton execution."
        >
          <svelte:fragment slot="action">
            <button
              type="button"
              on:click={loadToolsStatus}
              disabled={isCheckingTools}
              class="px-2.5 py-1 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
            >
              {#if isCheckingTools}
                <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Checking...</span>
              {:else}
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                <span>Check All</span>
              {/if}
            </button>
          </svelte:fragment>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5">
            {#each toolStatuses as tool}
              <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3 rounded-xl flex flex-col justify-between gap-2.5">
                <div class="space-y-1 min-w-0">
                  <div class="flex items-center justify-between gap-1.5">
                    <span class="font-semibold text-xs text-white truncate">{tool.name || tool.display_name || tool.key}</span>
                    {#if tool.has_update}
                      <span class="px-1.5 py-0.2 rounded text-[9px] font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 flex-shrink-0">
                        Update
                      </span>
                    {/if}
                  </div>

                  <div class="text-[11px] font-mono text-slate-400 truncate">
                    <span>{tool.installed_tag || tool.installed_version || 'Installed'}</span>
                    {#if tool.latest_tag || tool.latest_version}
                      <span class="text-slate-600">·</span>
                      <span class="text-emerald-400">{tool.latest_tag || tool.latest_version}</span>
                    {/if}
                  </div>
                </div>

                <div class="flex items-center space-x-1.5 pt-1 border-t border-[#181824]">
                  {#if tool.release_url}
                    <button
                      type="button"
                      on:click={() => AppAPI.openURL(tool.release_url)}
                      class="text-slate-400 hover:text-cyan-300 text-xs px-2 py-1 rounded-lg border border-[#262638] bg-[#14141e] transition-colors cursor-pointer"
                      title="Open GitHub Release"
                    >
                      ↗
                    </button>
                  {/if}
                  <button
                    type="button"
                    on:click={() => handleUpdateTool(tool.key || tool.tool_name, tool.name || tool.display_name)}
                    disabled={updatingTools[tool.key || tool.tool_name]}
                    class="flex-1 py-1 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs transition-colors flex items-center justify-center space-x-1 cursor-pointer disabled:opacity-50"
                  >
                    {#if updatingTools[tool.key || tool.tool_name]}
                      <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                      </svg>
                      <span>Updating...</span>
                    {:else}
                      <span>{tool.has_update ? 'Update' : 'Reinstall'}</span>
                    {/if}
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </SettingSection>

        <!-- Batch Maintenance Operations -->
        <SettingSection
          title="Batch Maintenance"
          tooltip="Automated multi-game library operations executed in the background."
        >
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
            <!-- 1. Batch Achievements -->
            <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
              <div class="space-y-1">
                <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
                  <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                  </svg>
                  <span>Achievements</span>
                </span>
                <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
                  Download missing achievement schemas and icons across library.
                </p>
              </div>
              <button
                type="button"
                on:click={handleBatchAchievements}
                disabled={isSubmittingBatch || ($games || []).length === 0}
                class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-sky-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-sky-900/40 hover:border-sky-700/60 transition-all cursor-pointer"
              >
                Download Missing
              </button>
            </div>

            <!-- 2. Batch Artwork -->
            <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
              <div class="space-y-1">
                <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
                  <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  <span>Artwork</span>
                </span>
                <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
                  Resolve and cache missing posters, hero banners, and logos.
                </p>
              </div>
              <button
                type="button"
                on:click={handleBatchArtwork}
                disabled={isSubmittingBatch || ($games || []).length === 0}
                class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-purple-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-purple-900/40 hover:border-purple-700/60 transition-all cursor-pointer"
              >
                Refresh Artwork
              </button>
            </div>

            <!-- 3. Batch Apply DRM & Emulator -->
            <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
              <div class="space-y-1">
                <div class="flex items-center justify-between gap-1.5">
                  <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5 truncate">
                    <svg class="w-3.5 h-3.5 text-amber-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                    </svg>
                    <span class="truncate">Apply Setup</span>
                  </span>
                  <span class="text-[9px] font-mono bg-[#161622] text-amber-300 border border-amber-500/30 px-1.5 py-0.2 rounded">
                    {($games || []).filter(g => !g.is_patched && !g.isPatched).length}
                  </span>
                </div>
                <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
                  Apply Goldberg emulator and unpacker to unpatched games.
                </p>
              </div>
              <button
                type="button"
                on:click={handleBatchApply}
                disabled={isSubmittingBatch || ($games || []).length === 0}
                class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-amber-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-amber-900/40 hover:border-amber-700/60 transition-all cursor-pointer"
              >
                Apply All
              </button>
            </div>

            <!-- 4. Batch Normalize -->
            <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
              <div class="space-y-1">
                <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
                  <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  <span>Normalize Folders</span>
                </span>
                <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
                  Standardize directory names to match official Steam titles.
                </p>
              </div>
              <button
                type="button"
                on:click={handleBatchNormalize}
                disabled={isSubmittingBatch || ($games || []).length === 0}
                class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-emerald-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-emerald-900/40 hover:border-emerald-700/60 transition-all cursor-pointer"
              >
                Normalize
              </button>
            </div>

            <!-- 5. Batch Verify Manifests -->
            <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors sm:col-span-2 lg:col-span-2">
              <div class="space-y-1">
                <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
                  <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                  <span>Verify Manifests</span>
                </span>
                <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
                  Verify SHA-1 file integrity against official depot manifests.
                </p>
              </div>
              <button
                type="button"
                on:click={handleBatchVerify}
                disabled={isSubmittingBatch || ($games || []).length === 0}
                class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-cyan-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-cyan-900/40 hover:border-cyan-700/60 transition-all cursor-pointer"
              >
                Verify All Manifests
              </button>
            </div>
          </div>
        </SettingSection>
      </div>

    <!-- ================= 6. LOGS & ACTIVITY ================= -->
    {:else if activeTab === 'logs'}
      <div class="space-y-3">
        <SettingSection
          title="System Logs & Activity Stream"
          tooltip="Real-time telemetry, notifications, and event activity log stream."
        >
          <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2.5 pb-2 border-b border-[#1c1c28]">
            <div class="flex items-center space-x-2">
              <span class="text-[10px] font-mono bg-[#0c0c10] text-slate-400 px-2 py-0.5 rounded-full border border-[#20202c]">
                {($notifItems || []).length} entries
              </span>
            </div>

            <div class="flex items-center space-x-2 flex-wrap gap-y-1">
              <!-- Filter Chips -->
              <div class="flex items-center space-x-1 bg-[#0c0c10] p-0.5 rounded-lg border border-[#242434] text-[11px] font-mono">
                {#each ['all', 'error', 'warning', 'info', 'success'] as f}
                  <button
                    type="button"
                    on:click={() => { logFilter = f; }}
                    class="px-2 py-0.5 rounded capitalize transition-colors {logFilter === f ? 'bg-[#222232] text-white font-bold' : 'text-slate-500 hover:text-slate-300'}"
                  >
                    {f}
                  </button>
                {/each}
              </div>

              {#if ($notifItems || []).length > 0}
                <button
                  type="button"
                  on:click={copyLogsToClipboard}
                  class="text-[11px] font-bold bg-[#181824] hover:bg-[#222232] text-slate-300 px-2.5 py-1 rounded-lg border border-[#2c2c3e] transition-colors flex items-center space-x-1.5 cursor-pointer"
                  title="Copy log to clipboard"
                >
                  <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  <span>Copy</span>
                </button>
                <button
                  type="button"
                  on:click={() => notificationStore.items.clearAll()}
                  class="text-[11px] font-bold bg-[#181824] hover:bg-rose-950/40 text-slate-400 hover:text-rose-300 px-2.5 py-1 rounded-lg border border-[#2c2c3e] hover:border-rose-900/50 transition-colors cursor-pointer"
                >
                  Clear
                </button>
              {/if}
            </div>
          </div>

          <!-- Monospace Terminal Log Stream -->
          <div class="bg-[#08080c] border border-[#1e1e2c] rounded-xl p-3 sm:p-3.5 max-h-[440px] overflow-y-auto space-y-1.5 font-mono text-xs custom-scrollbar">
            {#if filteredLogs.length === 0}
              <div class="py-10 text-center text-slate-500 text-xs">
                <span>No log entries recorded.</span>
              </div>
            {:else}
              {#each filteredLogs as item (item.id || item.timestamp)}
                <div class="flex items-start space-x-2 py-1 border-b border-[#14141d] last:border-0 hover:bg-[#0e0e14] px-1.5 rounded transition-colors">
                  <span class="text-slate-600 text-[10px] select-none pt-0.5 whitespace-nowrap">
                    {new Date(item.timestamp).toLocaleTimeString()}
                  </span>

                  <!-- Level Badge -->
                  <span class="px-1.5 py-0.2 rounded text-[9px] font-black uppercase tracking-wider flex-shrink-0 {item.type === 'error' ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30' : item.type === 'warning' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30' : item.type === 'success' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' : 'bg-sky-500/20 text-sky-300 border border-sky-500/30'}">
                    {item.type || 'INFO'}
                  </span>

                  <!-- Content -->
                  <div class="min-w-0 flex-1 space-y-0.5">
                    <div class="flex items-center space-x-1.5 flex-wrap">
                      <strong class="text-slate-200 font-bold text-xs">{item.title}</strong>
                      {#if item.gameName}
                        <span class="text-[10px] text-cyan-400 bg-cyan-950/40 px-1.5 py-0.2 rounded border border-cyan-900/50">
                          {item.gameName}
                        </span>
                      {/if}
                    </div>
                    {#if item.message}
                      <p class="text-slate-400 text-[11px] break-words">{item.message}</p>
                    {/if}
                    {#if item.details}
                      <p class="text-slate-500 text-[10px] break-words">{item.details}</p>
                    {/if}
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        </SettingSection>
      </div>
    {/if}
  {/if}
</div>
