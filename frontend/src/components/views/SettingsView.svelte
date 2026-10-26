<script>
  import { onMount, onDestroy } from 'svelte';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import GeneralTab from './settings/GeneralTab.svelte';
  import ProfilesTab from './settings/ProfilesTab.svelte';
  import LibraryStorageTab from './settings/LibraryStorageTab.svelte';
  import CompatibilityTab from './settings/CompatibilityTab.svelte';
  import SettingsHeader from './settings/SettingsHeader.svelte';
  import SettingsNav from './settings/SettingsNav.svelte';
  import SettingsAdvancedTab from './settings/SettingsAdvancedTab.svelte';
  import { RUNNER_OPTIONS, LANGUAGE_OPTIONS, COUNTRY_OPTIONS } from '../../constants/options.js';
  import { normalizeTool } from '../../utils/tools.js';
  import {
    SETTINGS_TABS,
    DEFAULT_CFG_DRAFT,
    DEFAULT_USER_CFG_DRAFT,
    parseTabParam,
    getAdjacentTab
  } from '../../utils/settings.js';

  let activeTab = $state('general'); // 'general' | 'storage' | 'compatibility' | 'profiles' | 'advanced'
  let advancedSubtab = $state('coop'); // 'coop' | 'maintenance' | 'logs' | 'schema'

  let cfgDraft = $state(JSON.parse(JSON.stringify(DEFAULT_CFG_DRAFT)));
  let userCfgDraft = $state(JSON.parse(JSON.stringify(DEFAULT_USER_CFG_DRAFT)));

  let initialCfgJSON = $state('');
  let initialUserCfgJSON = $state('');

  let isSaving = $state(false);
  let saveSuccess = $state(false);
  let saveSuccessTimer = null;
  let autoSaveTimeout = null;

  let isDirty = $derived(Boolean(
    initialCfgJSON &&
    initialUserCfgJSON &&
    (JSON.stringify(cfgDraft) !== initialCfgJSON || JSON.stringify(userCfgDraft) !== initialUserCfgJSON)
  ));

  // Synchronize dirty draft state to global configStore
  $effect(() => {
    configStore.setDirty(isDirty);
  });

  // Trigger debounced autosave when settings draft becomes dirty
  $effect(() => {
    if (isDirty) {
      triggerAutoSave();
    }
  });

  onDestroy(() => {
    configStore.setDirty(false);
    if (autoSaveTimeout) clearTimeout(autoSaveTimeout);
    if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
  });

  function handleTabKeyDown(e, currentIndex) {
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      activeTab = getAdjacentTab(SETTINGS_TABS[currentIndex], 'next');
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      activeTab = getAdjacentTab(SETTINGS_TABS[currentIndex], 'prev');
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
        if (SETTINGS_TABS[idx]) {
          e.preventDefault();
          activeTab = SETTINGS_TABS[idx];
        }
      }
    }
  }

  // Toolchain state
  let toolStatuses = $state([]);
  let isCheckingTools = $state(false);
  let settingsLoadError = $state(null);

  async function loadSettings() {
    settingsLoadError = null;
    try {
      const urlParams = new URLSearchParams(window.location.search);
      const parsed = parseTabParam(urlParams.get('tab'));
      if (parsed) {
        activeTab = parsed.activeTab;
        if (parsed.advancedSubtab) {
          advancedSubtab = parsed.advancedSubtab;
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
      settingsLoadError = err?.message || 'Failed to load initial settings';
    }
  }

  onMount(() => {
    loadSettings();
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

  function triggerAutoSave() {
    if (!initialCfgJSON || !initialUserCfgJSON) return;
    if (!isDirty || isSaving) return;
    if (autoSaveTimeout) clearTimeout(autoSaveTimeout);
    autoSaveTimeout = setTimeout(() => {
      handleSave(true);
    }, 350);
  }

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
</script>

<svelte:window onkeydown={handleWindowKeyDown} />

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-20 max-w-4xl mx-auto w-full space-y-4 text-xs sm:text-sm animate-fade-in custom-scrollbar">
  <SettingsHeader
    {isDirty}
    {isSaving}
    {saveSuccess}
    {settingsLoadError}
    onretry={loadSettings}
  />

  <SettingsNav
    {activeTab}
    onselectTab={(t) => { activeTab = t; }}
    ontabKeyDown={handleTabKeyDown}
  />

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
        runnersList={RUNNER_OPTIONS}
      />
    {:else if activeTab === 'profiles' || activeTab === 'persona'}
      <ProfilesTab
        languagesList={LANGUAGE_OPTIONS}
        countriesList={COUNTRY_OPTIONS}
      />
    {:else if activeTab === 'advanced'}
      <SettingsAdvancedTab
        bind:cfgDraft
        bind:advancedSubtab
        {toolStatuses}
        {isCheckingTools}
        onloadToolsStatus={loadToolsStatus}
      />
    {/if}
  {/if}
</div>
