<script>
  import { onMount } from 'svelte';
  import { selectedGame, showInspector, games, saveGameProfile, resetGameProfile, rollbackGame, refreshGame, renameGameInStore, scanLibrary } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { isSteamApp } from '../../utils/game.js';
  import { AppAPI } from '../../api/app.js';

  import DrawerHeader from './drawer/DrawerHeader.svelte';
  import DrawerTabs from './drawer/DrawerTabs.svelte';
  import DrawerGeneralTab from './drawer/DrawerGeneralTab.svelte';
  import DrawerIntegrationsTab from './drawer/DrawerIntegrationsTab.svelte';
  import DrawerEmuTab from './drawer/DrawerEmuTab.svelte';
  import DrawerIntegrityTab from './drawer/DrawerIntegrityTab.svelte';
  import DrawerNewsTab from './drawer/DrawerNewsTab.svelte';
  import DrawerFooter from './drawer/DrawerFooter.svelte';

  let activeTab = 'general'; // 'general' | 'integrations' | 'emu' | 'integrity' | 'news'
  let integrityReport = null;
  let integrityError = '';
  let isCheckingIntegrity = false;
  let newsItems = [];
  let isLoadingNews = false;
  let expandedNewsIndex = 0;
  let loadedNewsAppID = '';
  let newsError = '';
  let globalCfg = {};

  // Local form state
  let customExe = '';
  let customArgs = '';
  let customAppID = '';
  let customMode = '';
  let customLanguage = '';
  let customWinePrefix = '';
  let customOperatingMode = '';
  let customRunner = '';
  let customMangoHud = null; // null = inherit global, true = force on, false = force off
  let customFSR = null;
  let customNVAPI = null;
  let customDXVKHud = '';
  let customEnvMap = {};
  let optNormalize = true;
  let optSteamless = true;
  let optAchievements = true;

  onMount(async () => {
    const cfg = await configStore.load();
    if (cfg) globalCfg = cfg;
  });

  $: candidates = ($selectedGame?.launchCandidates || [])
    .filter(c => {
      const exe = (c.executable || '').toLowerCase();
      return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
    })
    .map(c => ({
      name: c.name || c.executable || '',
      executable: c.executable || '',
      arguments: c.arguments || '',
      description: c.description || '',
    }));

  $: languageOptions = ($selectedGame?.supportedLanguages || [
    'english', 'french', 'italian', 'german', 'spanish', 'japanese', 'schinese', 'russian', 'brazilian', 'koreana', 'polish', 'turkish'
  ]);

  $: if ($selectedGame) {
    const lp = $selectedGame.launchProfile || null;
    let baseExe = lp?.exe_path || $selectedGame.exePath || (candidates[0]?.executable || '');
    if (baseExe.toLowerCase().includes('steamclient_loader') || baseExe.toLowerCase().includes('coldclientloader')) {
      baseExe = candidates[0]?.executable || '';
    }
    customExe = baseExe;
    customAppID = $selectedGame.appId || '';
    customMode = $selectedGame.isPortable ? 'portable' : ($selectedGame.isPatched ? 'loader' : '');
    customLanguage = $selectedGame.currentLanguage || '';
    customArgs = lp?.exe_args || $selectedGame.exeArgs || candidates[0]?.arguments || '';
    
    customOperatingMode = lp?.operating_mode ?? '';
    customRunner = lp?.runner ?? '';
    customWinePrefix = lp?.wine_prefix || $selectedGame.winePrefix || '';
    customMangoHud = (lp?.mangohud !== undefined && lp?.mangohud !== null) ? lp.mangohud : null;
    customFSR = (lp?.wine_fullscreen_fsr !== undefined && lp?.wine_fullscreen_fsr !== null) ? lp.wine_fullscreen_fsr : null;
    customNVAPI = (lp?.proton_nvapi !== undefined && lp?.proton_nvapi !== null) ? lp.proton_nvapi : null;
    customDXVKHud = lp?.dxvk_hud || '';
    customEnvMap = { ...(lp?.custom_env || {}) };

    optNormalize = globalCfg.normalize ?? true;
    optSteamless = globalCfg.steamless ?? true;
    optAchievements = globalCfg.achievements ?? true;
    integrityReport = null;
    newsItems = [];
    expandedNewsIndex = 0;
  }

  let envEntries = [];

  $: {
    const globalEnvs = globalCfg.global_env || {};
    const entries = [];
    const seen = new Set();

    // Process global envs
    for (const [k, v] of Object.entries(globalEnvs)) {
      seen.add(k);
      if (k in customEnvMap) {
        entries.push({
          key: k,
          value: customEnvMap[k],
          globalValue: v,
          isInherited: false,
          isOverriding: true
        });
      } else {
        entries.push({
          key: k,
          value: v,
          globalValue: v,
          isInherited: true,
          isOverriding: false
        });
      }
    }

    // Process custom game envs
    for (const [k, v] of Object.entries(customEnvMap)) {
      if (!seen.has(k)) {
        entries.push({
          key: k,
          value: v,
          globalValue: '',
          isInherited: false,
          isOverriding: false
        });
      }
    }

    envEntries = entries;
  }

  function handleOverrideEnv(key, globalVal) {
    customEnvMap = { ...customEnvMap, [key]: globalVal };
  }

  function handleRevertEnv(key) {
    const next = { ...customEnvMap };
    delete next[key];
    customEnvMap = next;
  }

  function handleUpdateCustomEnv(key, newVal) {
    customEnvMap = { ...customEnvMap, [key]: newVal };
  }

  function handleDeleteCustomEnv(key) {
    const next = { ...customEnvMap };
    delete next[key];
    customEnvMap = next;
  }

  function handleAddCustomEnv() {
    let newKey = 'CUSTOM_VAR';
    let counter = 1;
    while (newKey in customEnvMap || (globalCfg.global_env && newKey in globalCfg.global_env)) {
      newKey = `CUSTOM_VAR_${counter++}`;
    }
    customEnvMap = { ...customEnvMap, [newKey]: '' };
  }

  function handleSelectCandidate(e) {
    const val = e.target.value;
    if (val === '__custom__') return;
    const found = candidates.find(c => c.executable === val);
    if (found) {
      customExe = found.executable;
      if (found.arguments) customArgs = found.arguments;
    } else if (val) {
      customExe = val;
    }
  }

  async function browseExecutable() {
    if (!$selectedGame?.gameDir) return;
    try {
      const selected = await AppAPI.selectFile('Select Game Executable', 'Executables', '*');
      if (selected) {
        let rel = selected;
        if (selected.startsWith($selectedGame.gameDir)) {
          rel = selected.slice($selectedGame.gameDir.length).replace(/^[/\\]+/, '');
        }
        customExe = rel;
      }
    } catch (err) {
      notificationStore.error('File Selection Failed', err);
    }
  }

  async function handleResetDefaults() {
    if (!$selectedGame?.gameDir) return;
    try {
      await resetGameProfile($selectedGame.gameDir);
      notificationStore.success('Defaults Restored', `Restored global default settings for ${$selectedGame.name}`);
      $showInspector = false;
    } catch (err) {
      notificationStore.error('Reset Failed', err);
    }
  }

  async function handleSave() {
    if (!$selectedGame) return;
    const gameDir = $selectedGame.gameDir;
    if (!gameDir) return;

    const launchProfile = {
      game_dir: gameDir,
      exe_path: customExe.trim(),
      exe_args: customArgs.trim(),
      app_id: customAppID.trim(),
      name: $selectedGame.name || '',
      platform: $selectedGame.platform || '',
      operating_mode: customOperatingMode,
      runner: customRunner,
      wine_prefix: customWinePrefix.trim(),
      mangohud: customMangoHud,
      wine_fullscreen_fsr: customFSR,
      proton_nvapi: customNVAPI,
      dxvk_hud: customDXVKHud.trim(),
      custom_env: customEnvMap,
    };

    try {
      await saveGameProfile(gameDir, launchProfile);

      // If the game is already patched or in Lutris mode, keep on-disk integrations synced
      const isLutris = customOperatingMode === 'lutris' || (!customOperatingMode && globalCfg.operating_mode === 'lutris');
      if ($selectedGame.isPatched || $selectedGame.lutrisRegistered || isLutris) {
        const opts = {
          path: $selectedGame.gameDir,
          exe_path: customExe.trim(),
          exe_args: customArgs.trim(),
          app_id: customAppID.trim(),
          portable: customMode === 'portable',
          language: customLanguage.trim(),
          apply_gbe: true,
          add_lutris: isLutris,
          normalize_dir: optNormalize,
          no_steamless: !optSteamless,
          fetch_achievements: optAchievements,
          wine_prefix: customWinePrefix.trim(),
        };
        try {
          await AppAPI.submitApply(opts);
        } catch (err) {
          console.warn('Sync setup notice:', err);
        }
      }

      await refreshGame($selectedGame.gameDir);

      notificationStore.success('Settings Saved', `Saved configuration for ${$selectedGame.name}`);
      $showInspector = false;
    } catch (err) {
      notificationStore.error('Save Failed', err);
    }
  }

  async function runIntegrityCheck() {
    if (!$selectedGame?.gameDir) return;
    isCheckingIntegrity = true;
    integrityError = '';
    integrityReport = null;
    try {
      const rep = await AppAPI.verifyIntegrity($selectedGame.gameDir);
      const total = rep?.official_count ?? 0;
      if (!rep || total === 0) {
        integrityError = 'No official depot manifest entries found for this title.';
      } else {
        integrityReport = rep;
      }
    } catch (err) {
      integrityError = err?.message || String(err);
      notificationStore.error('Integrity Check Notice', integrityError);
    } finally {
      isCheckingIntegrity = false;
    }
  }

  let isNormalizing = false;

  async function handleNormalizeDirectory() {
    if (!$selectedGame?.gameDir) return;
    const oldDir = $selectedGame.gameDir;
    try {
      isNormalizing = true;
      await AppAPI.submitNormalize(oldDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Normalizing Directory',
        message: `Queued directory normalization for ${$selectedGame.name || oldDir}`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Normalization Failed',
        message: err?.message || String(err)
      });
    } finally {
      isNormalizing = false;
    }
  }

  async function handleUpdateGame() {
    if (!$selectedGame?.gameDir) return;
    const appIdNum = parseInt($selectedGame.appId, 10);
    if (appIdNum && !isNaN(appIdNum)) {
      try {
        await tasksStore.start({
          app_id: appIdNum,
          game_name: $selectedGame.name,
          target_dir: $selectedGame.gameDir,
          platform: $selectedGame.platform,
          auto_apply: true,
        });
      } catch (err) {
        // Handled in store
      }
    } else {
      try {
        await AppAPI.submitUpdate($selectedGame.gameDir);
        notificationStore.items.add({
          type: 'info',
          title: 'Update Enqueued',
          message: `Update task started for ${$selectedGame.name}`,
          gameName: $selectedGame.name
        });
      } catch (err) {
        notificationStore.error('Update Failed', err);
      }
    }
    $showInspector = false;
  }

  async function loadNews(force = false) {
    if (!$selectedGame?.appId || $selectedGame.appId === '0') return;
    if (!force && loadedNewsAppID === $selectedGame.appId && newsItems.length > 0) return;

    isLoadingNews = true;
    newsError = '';
    try {
      const items = await AppAPI.fetchNews(String($selectedGame.appId), 6);
      newsItems = items || [];
      loadedNewsAppID = $selectedGame.appId;
      if (!items || items.length === 0) {
        newsError = 'No patch notes found on Steam for this title.';
      }
    } catch (err) {
      newsError = err?.message || String(err);
      notificationStore.error('Failed to Fetch Patch Notes', newsError);
    } finally {
      isLoadingNews = false;
    }
  }

  function toggleNewsExpand(idx) {
    expandedNewsIndex = expandedNewsIndex === idx ? -1 : idx;
  }

  function handleRollback() {
    if (!$selectedGame?.gameDir) return;
    if (confirm(`Rollback emulator modifications for ${$selectedGame.name}?`)) {
      rollbackGame($selectedGame.gameDir, false);
      $showInspector = false;
    }
  }

  $: if ($showInspector && activeTab === 'news' && loadedNewsAppID !== $selectedGame?.appId && !isLoadingNews && $selectedGame?.appId) {
    loadNews();
  }
</script>

{#if $showInspector && $selectedGame}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 bg-black/75 backdrop-blur-sm z-40 flex items-center justify-center p-4 sm:p-6 animate-fade-in"
    on:click|self={() => { $showInspector = false; }}
    on:keydown={(e) => { if (e.key === 'Escape') $showInspector = false; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <!-- Modal Card -->
    <div class="bg-[#121218] border border-[#2c2c3e] w-full max-w-2xl rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh] animate-slide-up">
      <!-- Modal Header -->
      <DrawerHeader
        game={$selectedGame}
        onClose={() => { $showInspector = false; }}
      />

      <!-- Navigation Tabs -->
      <DrawerTabs
        {activeTab}
        showNewsTab={isSteamApp($selectedGame.appId)}
        onSelectTab={(tab) => { activeTab = tab; }}
      />

      <!-- Modal Body (Scrollable) -->
      <div class="p-5 overflow-y-auto space-y-4 text-xs">
        {#if activeTab === 'general'}
          <DrawerGeneralTab
            game={$selectedGame}
            {candidates}
            bind:customExe
            bind:customArgs
            {envEntries}
            onSelectCandidate={handleSelectCandidate}
            onBrowseExecutable={browseExecutable}
            onUpdateGame={handleUpdateGame}
            envActions={{
              add: handleAddCustomEnv,
              update: handleUpdateCustomEnv,
              override: handleOverrideEnv,
              revert: handleRevertEnv,
              delete: handleDeleteCustomEnv
            }}
          />
        {:else if activeTab === 'integrations'}
          <DrawerIntegrationsTab
            {globalCfg}
            bind:customOperatingMode
            bind:customRunner
            bind:customWinePrefix
            bind:customMangoHud
            bind:customFSR
            bind:customNVAPI
            bind:customDXVKHud
          />
        {:else if activeTab === 'emu'}
          <DrawerEmuTab
            game={$selectedGame}
            {globalCfg}
            bind:customMode
            bind:customLanguage
            {languageOptions}
            bind:customAppID
            bind:optSteamless
            bind:optAchievements
            bind:optNormalize
          />
        {:else if activeTab === 'integrity'}
          <DrawerIntegrityTab
            {isCheckingIntegrity}
            {isNormalizing}
            {integrityError}
            {integrityReport}
            onRunIntegrityCheck={runIntegrityCheck}
            onUpdateGame={handleUpdateGame}
            onNormalizeDirectory={handleNormalizeDirectory}
          />
        {:else if activeTab === 'news'}
          <DrawerNewsTab
            {isLoadingNews}
            {newsError}
            {newsItems}
            {expandedNewsIndex}
            onRefreshNews={() => loadNews(true)}
            onToggleExpand={toggleNewsExpand}
          />
        {/if}
      </div>

      <!-- Modal Footer -->
      <DrawerFooter
        isPatched={$selectedGame.isPatched}
        onRollback={handleRollback}
        onResetDefaults={handleResetDefaults}
        onCancel={() => { $showInspector = false; }}
        onSave={handleSave}
      />
    </div>
  </div>
{/if}
