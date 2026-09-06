<script>
  import { onMount, onDestroy } from 'svelte';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { games } from '../../stores/library.js';
  import { AppAPI } from '../../api/app.js';
  import GeneralTab from './settings/GeneralTab.svelte';
  import PersonaTab from './settings/PersonaTab.svelte';
  import CoopTab from './settings/CoopTab.svelte';
  import LibraryStorageTab from './settings/LibraryStorageTab.svelte';
  import CompatibilityTab from './settings/CompatibilityTab.svelte';
  import CredentialsTab from './settings/CredentialsTab.svelte';
  import ToolsMaintenanceTab from './settings/ToolsMaintenanceTab.svelte';
  import LogsTab from './settings/LogsTab.svelte';
  import { RUNNER_OPTIONS, LANGUAGE_OPTIONS, COUNTRY_OPTIONS } from '../../constants/options.js';

  let activeTab = 'general'; // 'general' | 'storage' | 'compatibility' | 'persona' | 'coop' | 'credentials' | 'tools' | 'logs'
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
    enable_launch_notify: true,
    launch_window_behavior: 'keep',
    close_to_tray: false,
    disable_overlay: false,
    offline_mode: false,
    achievements: true,
    transfer_speed_limit: 0,
    save_manifest_url: '',
    global_env: {},
    multiseat: {
      player2_account_name: 'PlayerTwo',
      player2_account_steamid: '76561198998877665',
      player2_language: 'english',
      player2_ip_country: 'US',
      player2_wine_prefix: '',
      default_seat1_mode: 'local',
      default_seat2_mode: 'local',
      default_layout: 'dual_monitor',
      broadcast_relay_enabled: true,
      seat1_gamepad: 'auto',
      seat2_gamepad: 'auto',
      seat1_keyboard: 'auto',
      seat2_keyboard: 'auto',
      seat1_mouse: 'auto',
      seat2_mouse: 'auto',
      seat2_force_grab_cursor: false,
      seat2_grab_keyboard: false,
      seat2_show_cursor: true,
      seat2_compositor: 'auto',
      seat1_display_index: 0,
      seat2_display_index: 1,
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    }
  };
  let userCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  let initialCfgJSON = '';
  let initialUserCfgJSON = '';

  let isSaving = false;
  let saveSuccess = false;
  let saveSuccessTimer = null;

  $: isDirty = Boolean(
    initialCfgJSON &&
    initialUserCfgJSON &&
    (JSON.stringify(cfgDraft) !== initialCfgJSON || JSON.stringify(userCfgDraft) !== initialUserCfgJSON)
  );

  $: configStore.setDirty(isDirty);

  onDestroy(() => {
    configStore.setDirty(false);
  });

  const tabOrder = ['general', 'storage', 'compatibility', 'persona', 'coop', 'credentials', 'tools', 'logs'];

  function handleTabKeyDown(e, currentIndex) {
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      const nextIndex = (currentIndex + 1) % tabOrder.length;
      activeTab = tabOrder[nextIndex];
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      const prevIndex = (currentIndex - 1 + tabOrder.length) % tabOrder.length;
      activeTab = tabOrder[prevIndex];
    }
  }

  function handleWindowKeyDown(e) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      if (isDirty && !isSaving) {
        e.preventDefault();
        handleSave();
      }
    }
  }

  // Toolchain state
  let toolStatuses = [];
  let isCheckingTools = false;
  let updatingTools = {};
  let isSubmittingBatch = false;

  const runnersList = RUNNER_OPTIONS;
  const languagesList = LANGUAGE_OPTIONS;
  const countriesList = COUNTRY_OPTIONS;

  onMount(async () => {
    try {
      const urlParams = new URLSearchParams(window.location.search);
      const tabParam = urlParams.get('tab');
      if (tabParam) {
        if (['general', 'display', 'ui', 'network', 'bandwidth'].includes(tabParam)) activeTab = 'general';
        else if (['storage', 'library', 'paths'].includes(tabParam)) activeTab = 'storage';
        else if (['compatibility', 'compat', 'runner'].includes(tabParam)) activeTab = 'compatibility';
        else if (['steam', 'persona', 'player', 'user', 'emu', 'drm'].includes(tabParam)) activeTab = 'persona';
        else if (['coop', 'co-op', 'multiseat', 'split'].includes(tabParam)) activeTab = 'coop';
        else if (['credentials', 'media', 'api', 'apis', 'key', 'keys'].includes(tabParam)) activeTab = 'credentials';
        else if (['maintenance', 'tools', 'components'].includes(tabParam)) activeTab = 'tools';
        else if (['logs', 'log', 'activity'].includes(tabParam)) activeTab = 'logs';
      }

      const cfg = await AppAPI.getConfig();
      if (cfg) {
        cfgDraft = {
          ...cfgDraft,
          ...JSON.parse(JSON.stringify(cfg)),
          multiseat: {
            ...cfgDraft.multiseat,
            ...(cfg.multiseat ? JSON.parse(JSON.stringify(cfg.multiseat)) : {})
          }
        };
      }
      const userCfg = await AppAPI.getGSEUserConfig();
      if (userCfg) {
        userCfgDraft = { ...userCfgDraft, ...JSON.parse(JSON.stringify(userCfg)) };
      }

      initialCfgJSON = JSON.stringify(cfgDraft);
      initialUserCfgJSON = JSON.stringify(userCfgDraft);

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
    if (!cfgDraft || !isDirty) return;
    isSaving = true;
    try {
      await AppAPI.saveConfig(cfgDraft);
      await AppAPI.saveGSEUserConfig(userCfgDraft);

      initialCfgJSON = JSON.stringify(cfgDraft);
      initialUserCfgJSON = JSON.stringify(userCfgDraft);

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
    if (!isDirty) return;
    const cfg = await AppAPI.getConfig();
    if (cfg) {
      cfgDraft = {
        ...cfgDraft,
        ...JSON.parse(JSON.stringify(cfg)),
        multiseat: {
          ...cfgDraft.multiseat,
          ...(cfg.multiseat ? JSON.parse(JSON.stringify(cfg.multiseat)) : {})
        }
      };
    }
    const userCfg = await AppAPI.getGSEUserConfig();
    if (userCfg) userCfgDraft = { ...userCfgDraft, ...JSON.parse(JSON.stringify(userCfg)) };

    initialCfgJSON = JSON.stringify(cfgDraft);
    initialUserCfgJSON = JSON.stringify(userCfgDraft);

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
      const taskId = await AppAPI.updateTool(key);
      notificationStore.items.add({
        type: 'success',
        title: 'Tool Update Started',
        message: `${name || key} installation/update started in background.`,
        actionLabel: 'View in Tasks',
        actionView: 'tasks',
        actionTaskId: taskId
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
      const taskId = await AppAPI.submitBatchLibraryOp('achievements');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Achievements Started',
        message: 'Downloading achievement schemas and assets for your library.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
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
      const taskId = await AppAPI.submitBatchLibraryOp('artwork');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Artwork Started',
        message: 'Fetching missing game artwork and covers from providers.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
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
      const taskId = await AppAPI.submitBatchLibraryOp('apply');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Setup Started',
        message: 'Applying Goldberg emulators and Steamless unpackers to unpatched games.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
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
      const taskId = await AppAPI.submitBatchLibraryOp('normalize');
      notificationStore.items.add({
        type: 'info',
        title: 'Normalization Started',
        message: 'Standardizing game directory folder names to official Steam titles.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
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
      const taskId = await AppAPI.submitBatchLibraryOp('verify');
      notificationStore.items.add({
        type: 'info',
        title: 'Verification Started',
        message: 'Verifying SHA-1 file integrity against depot manifests.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
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

<svelte:window on:keydown={handleWindowKeyDown} />

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-20 max-w-4xl mx-auto w-full space-y-4 text-xs sm:text-sm animate-fade-in custom-scrollbar">
  <!-- Minimalist Title & Save Bar -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-surface-border">
    <div class="flex items-center space-x-2.5">
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2.5">
        System Settings
      </h2>
      {#if isDirty}
        <span class="px-2 py-0.5 rounded-full text-3xs font-mono font-bold bg-neutral-800 text-neutral-200 border border-neutral-700 animate-fade-in">
          Unsaved Changes
        </span>
      {/if}
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
        disabled={isSaving || !isDirty}
        class="px-3.5 py-1.5 rounded-xl border font-bold text-xs transition cursor-pointer {isDirty ? 'bg-surface-card hover:bg-surface-cardHover border-surface-border text-neutral-300 hover:text-white' : 'bg-transparent border-surface-border text-neutral-600 opacity-40 cursor-not-allowed'}"
      >
        Discard
      </button>

      <button
        type="button"
        on:click={handleSave}
        disabled={isSaving || !isDirty}
        class="font-black px-5 py-1.5 rounded-xl transition text-xs shadow flex items-center space-x-1.5 cursor-pointer {isDirty ? 'bg-white hover:bg-neutral-200 text-black shadow-md' : 'bg-surface-card text-neutral-500 border border-surface-border opacity-50 cursor-not-allowed'}"
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
  <div class="flex items-center space-x-1.5 overflow-x-auto pb-1 text-xs" role="tablist">
    <!-- 0. General Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'general'}
      on:click={() => { activeTab = 'general'; }}
      on:keydown={(e) => handleTabKeyDown(e, 0)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'general' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
      </svg>
      <span>General</span>
    </button>

    <!-- 1. Storage Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'storage' || activeTab === 'library'}
      on:click={() => { activeTab = 'storage'; }}
      on:keydown={(e) => handleTabKeyDown(e, 1)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'storage' || activeTab === 'library' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      <span>Storage</span>
    </button>

    <!-- 2. Compatibility Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'compatibility'}
      on:click={() => { activeTab = 'compatibility'; }}
      on:keydown={(e) => handleTabKeyDown(e, 2)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'compatibility' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Compatibility</span>
    </button>

    <!-- 3. Steam Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'persona'}
      on:click={() => { activeTab = 'persona'; }}
      on:keydown={(e) => handleTabKeyDown(e, 3)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'persona' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <span>Steam</span>
    </button>

    <!-- 4. Co-Op Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'coop'}
      on:click={() => { activeTab = 'coop'; }}
      on:keydown={(e) => handleTabKeyDown(e, 4)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'coop' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
      </svg>
      <span>Co-Op</span>
    </button>

    <!-- 5. Credentials Tab (API Keys) -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'credentials'}
      on:click={() => { activeTab = 'credentials'; }}
      on:keydown={(e) => handleTabKeyDown(e, 5)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'credentials' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
      <span>Credentials</span>
    </button>

    <!-- 6. Maintenance Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'tools'}
      on:click={() => { activeTab = 'tools'; }}
      on:keydown={(e) => handleTabKeyDown(e, 6)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'tools' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z" />
      </svg>
      <span>Maintenance</span>
    </button>

    <!-- 7. Logs Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'logs'}
      on:click={() => { activeTab = 'logs'; }}
      on:keydown={(e) => handleTabKeyDown(e, 7)}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'logs' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <span>Logs</span>
    </button>
  </div>

  <!-- Tab Contents -->
  {#if cfgDraft}
    {#if activeTab === 'general'}
      <GeneralTab
        bind:cfgDraft
      />
    {:else if activeTab === 'storage' || activeTab === 'library'}
      <LibraryStorageTab
        bind:cfgDraft
      />
    {:else if activeTab === 'compatibility'}
      <CompatibilityTab
        bind:cfgDraft
        {runnersList}
      />
    {:else if activeTab === 'persona'}
      <PersonaTab
        bind:userCfgDraft
        {languagesList}
        {countriesList}
      />
    {:else if activeTab === 'coop'}
      <CoopTab
        bind:cfgDraft
        bind:userCfgDraft
        {languagesList}
        {countriesList}
      />
    {:else if activeTab === 'credentials'}
      <CredentialsTab
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

<!-- Floating / Sticky Save Bar when dirty -->
{#if isDirty}
  <div class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 animate-fade-in pointer-events-auto max-w-sm sm:max-w-md w-full px-4">
    <div class="flex items-center justify-between gap-3 bg-[#12121a]/95 border border-sky-500/70 px-4 py-2.5 rounded-2xl shadow-2xl shadow-sky-950/30">
      <div class="flex items-center space-x-2 min-w-0">
        <span class="w-2 h-2 rounded-full bg-sky-400 animate-pulse flex-shrink-0"></span>
        <span class="text-xs font-bold text-white truncate">Unsaved Changes</span>
      </div>
      <div class="flex items-center space-x-2 flex-shrink-0">
        <button
          type="button"
          on:click={handleDiscard}
          disabled={isSaving}
          class="px-3 py-1 rounded-xl border border-[#242436] bg-[#14141e] hover:bg-[#1e1e2c] font-bold text-xs text-slate-300 hover:text-white transition cursor-pointer"
        >
          Discard
        </button>
        <button
          type="button"
          on:click={handleSave}
          disabled={isSaving}
          class="font-black px-4 py-1 rounded-xl transition text-xs bg-white hover:bg-neutral-200 text-black shadow flex items-center space-x-1.5 cursor-pointer"
        >
          {#if isSaving}
            <svg class="animate-spin w-3.5 h-3.5 text-black" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Saving...</span>
          {:else}
            <span>Save</span>
            <span class="text-3xs opacity-60 font-mono hidden sm:inline">(Ctrl+S)</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}
