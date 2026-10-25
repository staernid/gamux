<script>
  import { onMount, onDestroy } from 'svelte';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { games } from '../../stores/library.js';
  import { AppAPI } from '../../api/app.js';
  import GeneralTab from './settings/GeneralTab.svelte';
  import ProfilesTab from './settings/ProfilesTab.svelte';
  import CoopTab from './settings/CoopTab.svelte';
  import LibraryStorageTab from './settings/LibraryStorageTab.svelte';
  import CompatibilityTab from './settings/CompatibilityTab.svelte';
  import ToolsMaintenanceTab from './settings/ToolsMaintenanceTab.svelte';
  import LogsTab from './settings/LogsTab.svelte';
  import PropertySheet from '../schema/PropertySheet.svelte';
  import { RUNNER_OPTIONS, LANGUAGE_OPTIONS, COUNTRY_OPTIONS } from '../../constants/options.js';
  import { normalizeTool } from '../../utils/tools.js';

  let activeTab = $state('general'); // 'general' | 'storage' | 'compatibility' | 'profiles' | 'advanced'
  let advancedSubtab = $state('coop'); // 'coop' | 'maintenance' | 'logs' | 'schema'

  // Backward compatibility alias mapper
  $effect(() => {
    if (activeTab === 'coop') {
      activeTab = 'advanced';
      advancedSubtab = 'coop';
    } else if (activeTab === 'tools' || activeTab === 'maintenance') {
      activeTab = 'advanced';
      advancedSubtab = 'maintenance';
    } else if (activeTab === 'logs') {
      activeTab = 'advanced';
      advancedSubtab = 'logs';
    } else if (activeTab === 'schema') {
      activeTab = 'advanced';
      advancedSubtab = 'schema';
    }
  });
  let cfgDraft = $state({
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
      player2_profile_id: 'p2',
      default_seat1_mode: 'local',
      default_seat2_mode: 'local',
      default_layout: 'dual_monitor',
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
  });
  let userCfgDraft = $state({
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  });

  let initialCfgJSON = $state('');
  let initialUserCfgJSON = $state('');

  let isSaving = $state(false);
  let saveSuccess = $state(false);
  let saveSuccessTimer = null;

  let isDirty = $derived(Boolean(
    initialCfgJSON &&
    initialUserCfgJSON &&
    (JSON.stringify(cfgDraft) !== initialCfgJSON || JSON.stringify(userCfgDraft) !== initialUserCfgJSON)
  ));

  $effect(() => {
    configStore.setDirty(isDirty);
  });

  onDestroy(() => {
    configStore.setDirty(false);
  });

  const tabOrder = ['general', 'storage', 'compatibility', 'profiles', 'advanced'];

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
    const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      if (isDirty && !isSaving) {
        e.preventDefault();
        handleSave();
      }
    } else if (!isInput && !e.ctrlKey && !e.altKey && !e.metaKey) {
      if (['1', '2', '3', '4', '5'].includes(e.key)) {
        const idx = parseInt(e.key, 10) - 1;
        if (tabOrder[idx]) {
          e.preventDefault();
          activeTab = tabOrder[idx];
        }
      }
    }
  }

  // Toolchain state
  let toolStatuses = $state([]);
  let isCheckingTools = $state(false);
  let updatingTools = $state({});
  let isSubmittingBatch = $state(false);

  const runnersList = RUNNER_OPTIONS;
  const languagesList = LANGUAGE_OPTIONS;
  const countriesList = COUNTRY_OPTIONS;

  onMount(async () => {
    try {
      const urlParams = new URLSearchParams(window.location.search);
      const tabParam = urlParams.get('tab');
      if (tabParam) {
        if (['general', 'display', 'ui', 'network', 'bandwidth'].includes(tabParam)) activeTab = 'general';
        else if (['storage', 'library', 'paths', 'credentials', 'media', 'api', 'apis', 'key', 'keys'].includes(tabParam)) activeTab = 'storage';
        else if (['compatibility', 'compat', 'runner'].includes(tabParam)) activeTab = 'compatibility';
        else if (['profiles', 'profile', 'steam', 'persona', 'player', 'user', 'emu', 'drm'].includes(tabParam)) activeTab = 'profiles';
        else if (['advanced'].includes(tabParam)) activeTab = 'advanced';
        else if (['coop', 'co-op', 'multiseat', 'split'].includes(tabParam)) {
          activeTab = 'advanced';
          advancedSubtab = 'coop';
        } else if (['maintenance', 'tools', 'components'].includes(tabParam)) {
          activeTab = 'advanced';
          advancedSubtab = 'maintenance';
        } else if (['logs', 'log', 'activity'].includes(tabParam)) {
          activeTab = 'advanced';
          advancedSubtab = 'logs';
        } else if (['schema', 'all', 'properties'].includes(tabParam)) {
          activeTab = 'advanced';
          advancedSubtab = 'schema';
        }
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
      toolStatuses = Array.isArray(statuses) ? statuses.map(normalizeTool).filter(Boolean) : [];
    } catch {
      toolStatuses = [];
    } finally {
      isCheckingTools = false;
    }
  }

  let autoSaveTimeout = null;

  function triggerAutoSave() {
    if (!initialCfgJSON || !initialUserCfgJSON) return;
    if (!isDirty || isSaving) return;
    if (autoSaveTimeout) clearTimeout(autoSaveTimeout);
    autoSaveTimeout = setTimeout(() => {
      handleSave(true);
    }, 350);
  }

  $effect(() => {
    if (isDirty) {
      triggerAutoSave();
    }
  });

  async function handleSave(silent = false) {
    if (!cfgDraft || !isDirty) return;
    isSaving = true;
    try {
      await AppAPI.saveConfig(cfgDraft);
      await AppAPI.saveGSEUserConfig(userCfgDraft);
      await configStore.load();

      initialCfgJSON = JSON.stringify(cfgDraft);
      initialUserCfgJSON = JSON.stringify(userCfgDraft);

      saveSuccess = true;
      if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
      saveSuccessTimer = setTimeout(() => {
        saveSuccess = false;
      }, 2500);

      if (!silent) {
        notificationStore.items.add({
          type: 'success',
          title: 'Settings Saved',
          message: 'System configuration and persona preferences updated.'
        });
      }
    } catch (err) {
      if (!silent) {
        notificationStore.items.add({
          type: 'error',
          title: 'Save Failed',
          message: err?.message || String(err)
        });
      }
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

<svelte:window onkeydown={handleWindowKeyDown} />

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-20 max-w-4xl mx-auto w-full space-y-4 text-xs sm:text-sm animate-fade-in custom-scrollbar">
  <!-- Clean Title & Live Status Bar -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-surface-border">
    <div class="flex items-center space-x-3">
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2.5">
        System Settings
      </h2>
      {#if isDirty && !isSaving}
        <span class="text-xs font-mono text-amber-400 bg-amber-500/10 border border-amber-500/30 px-2.5 py-0.5 rounded-full flex items-center space-x-1.5 animate-fade-in">
          <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
          <span>Unsaved Changes (Ctrl+S)</span>
        </span>
      {:else if isSaving}
        <span class="text-xs font-mono text-neutral-300 bg-surface-header border border-surface-border px-2.5 py-0.5 rounded-full flex items-center space-x-1.5 animate-pulse">
          <svg class="animate-spin w-3 h-3 text-neutral-400" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Saving...</span>
        </span>
      {:else if saveSuccess}
        <span class="text-xs font-mono font-semibold text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 px-2.5 py-0.5 rounded-full flex items-center space-x-1.5 animate-fade-in">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <span>Saved</span>
        </span>
      {/if}
    </div>
  </div>

  <!-- Segmented Top Tabs (5 Consolidated Macro-Surfaces) -->
  <div class="flex items-center space-x-1.5 overflow-x-auto pb-1 text-xs" role="tablist">
    <!-- 0. General Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'general'}
      onclick={() => { activeTab = 'general'; }}
      onkeydown={(e) => handleTabKeyDown(e, 0)}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'general' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">1</span>
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
      </svg>
      <span>General</span>
    </button>

    <!-- 1. Storage Tab (Library & Backups & Credentials) -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'storage' || activeTab === 'library'}
      onclick={() => { activeTab = 'storage'; }}
      onkeydown={(e) => handleTabKeyDown(e, 1)}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'storage' || activeTab === 'library' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">2</span>
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      <span>Storage &amp; Library</span>
    </button>

    <!-- 2. Compatibility Tab (System Baseline) -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'compatibility'}
      onclick={() => { activeTab = 'compatibility'; }}
      onkeydown={(e) => handleTabKeyDown(e, 2)}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'compatibility' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">3</span>
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Compatibility</span>
    </button>

    <!-- 3. Profiles Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'profiles' || activeTab === 'persona'}
      onclick={() => { activeTab = 'profiles'; }}
      onkeydown={(e) => handleTabKeyDown(e, 3)}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'profiles' || activeTab === 'persona' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">4</span>
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <span>Profiles</span>
    </button>

    <!-- 4. Advanced Tab -->
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === 'advanced'}
      onclick={() => { activeTab = 'advanced'; }}
      onkeydown={(e) => handleTabKeyDown(e, 4)}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {activeTab === 'advanced' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">5</span>
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" />
      </svg>
      <span>Advanced</span>
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
    {:else if activeTab === 'profiles' || activeTab === 'persona'}
      <ProfilesTab
        {languagesList}
        {countriesList}
      />
    {:else if activeTab === 'advanced'}
      <div class="space-y-4">
        <!-- Advanced Segmented Sub-Navigation -->
        <div class="flex items-center space-x-1 p-1 bg-surface-header/80 border border-surface-border rounded-xl text-xs overflow-x-auto" role="tablist">
          <!-- Co-Op Subtab -->
          <button
            type="button"
            role="tab"
            aria-selected={advancedSubtab === 'coop'}
            onclick={() => { advancedSubtab = 'coop'; }}
            class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'coop' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
          >
            <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
            </svg>
            <span>Co-Op Multiseat</span>
          </button>

          <!-- Maintenance Subtab -->
          <button
            type="button"
            role="tab"
            aria-selected={advancedSubtab === 'maintenance'}
            onclick={() => { advancedSubtab = 'maintenance'; }}
            class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'maintenance' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
          >
            <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z" />
            </svg>
            <span>Maintenance</span>
          </button>

          <!-- Logs Subtab -->
          <button
            type="button"
            role="tab"
            aria-selected={advancedSubtab === 'logs'}
            onclick={() => { advancedSubtab = 'logs'; }}
            class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'logs' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
          >
            <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>Logs</span>
          </button>

          <!-- Schema Subtab -->
          <button
            type="button"
            role="tab"
            aria-selected={advancedSubtab === 'schema'}
            onclick={() => { advancedSubtab = 'schema'; }}
            class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'schema' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
            title="Dynamic declarative settings schema"
          >
            <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            <span>All Options</span>
          </button>
        </div>

        <!-- Advanced Subtab Content -->
        {#if advancedSubtab === 'coop'}
          <CoopTab bind:cfgDraft />
        {:else if advancedSubtab === 'maintenance'}
          <ToolsMaintenanceTab
            {toolStatuses}
            {isCheckingTools}
            {updatingTools}
            {isSubmittingBatch}
            gamesCount={($games || []).length}
            unpatchedGamesCount={($games || []).filter(g => !g.is_patched && !g.isPatched).length}
            oncheckAll={loadToolsStatus}
            onupdateTool={handleUpdateTool}
            onbatchAchievements={handleBatchAchievements}
            onbatchArtwork={handleBatchArtwork}
            onbatchApply={handleBatchApply}
            onbatchNormalize={handleBatchNormalize}
            onbatchVerify={handleBatchVerify}
          />
        {:else if advancedSubtab === 'logs'}
          <LogsTab />
        {:else if advancedSubtab === 'schema'}
          <div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-xl space-y-4">
            <div class="border-b border-surface-border pb-3">
              <h3 class="text-sm font-bold text-white">Declarative Configuration Engine</h3>
              <p class="text-xs text-neutral-400 mt-0.5 leading-relaxed">All settings defined in the Go backend schema are automatically discovered and rendered here.</p>
            </div>
            <PropertySheet
              bind:values={cfgDraft}
              scope="global"
              groupByCategory={true}
            />
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>
