<script>
  import { onMount, onDestroy } from 'svelte';
  import {
    selectedGame,
    activeView,
    queueApplyGame,
    setGameRunning,
    runningGames,
    isGamePathRunning,
    refreshGame,
    libraryPath,
    updateGameInStore
  } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { drawerDraft } from '../../stores/drawerDraft.js';
  import { multiseatStore, getActiveSession } from '../../stores/multiseat.js';
  import { AppAPI } from '../../api/app.js';
  import { isSamePath, countOverrides, normalizeGame, isSteamApp } from '../../utils/game.js';
  import HubSavesCard from './hub/cards/HubSavesCard.svelte';
  import GameHeroBanner from './hub/GameHeroBanner.svelte';
  import GameTaskHud from './hub/GameTaskHud.svelte';
  import HubOverviewTab from './hub/HubOverviewTab.svelte';
  import HubCommunityTab from './hub/HubCommunityTab.svelte';
  import HubGeneralTab from './hub/HubGeneralTab.svelte';
  import HubEnvironmentTab from './hub/HubEnvironmentTab.svelte';
  import HubFilesTab from './hub/HubFilesTab.svelte';
  import HubResourcesTab from './hub/HubResourcesTab.svelte';
  import { RUNNER_OPTIONS, OPERATING_MODE_OPTIONS, LANGUAGE_OPTIONS } from '../../constants/options.js';

  // 7 Primary Consolidated Tabs: 'overview' | 'community' | 'resources' | 'general' | 'environment' | 'saves' | 'files'
  let activeTab = 'overview';
  let communitySubTab = 'achievements';
  let communityNewsIndex = 0;
  let filesSubTab = 'depots';

  let globalCfg = {};
  let userCfg = null;
  let isSaving = false;
  let saveSuccess = false;
  let saveSuccessTimer = null;

  // Integrity State
  let integrityReport = null;
  let integrityError = '';
  let isCheckingIntegrity = false;
  let isNormalizing = false;

  const allTasks = tasksStore.items;

  $: game = $selectedGame;
  $: isGameRunning = Boolean(game?.gameDir && isGamePathRunning($runningGames, game.gameDir));

  $: activeGameTask = ($allTasks || []).find(t => {
    if (t.status !== 'running' && t.status !== 'downloading' && t.status !== 'queued' && t.status !== 'pending') return false;
    if (game?.gameDir && isSamePath(t.gameDir, game.gameDir)) return true;
    if (game?.appId && game.appId !== '0' && (String(t.appId) === String(game.appId))) return true;
    return false;
  });
  $: isTaskActive = Boolean(activeGameTask);

  $: overrideCount = countOverrides($drawerDraft, game, globalCfg);

  const runnerOptions = RUNNER_OPTIONS;
  const operatingModeOptions = OPERATING_MODE_OPTIONS;
  const gameLanguageOptions = LANGUAGE_OPTIONS;

  onMount(async () => {
    try {
      globalCfg = await AppAPI.getConfig();
    } catch {
      // Ignored
    }
    try {
      userCfg = await AppAPI.getGSEUserConfig();
    } catch {
      // Ignored
    }

    if (game?.gameDir) {
      try {
        const running = await AppAPI.isGameRunning(game.gameDir);
        setGameRunning(game.gameDir, running);
      } catch {
        // Ignored
      }
    }

    if (game) {
      drawerDraft.init(game, globalCfg);
    }
  });

  onDestroy(() => {
    if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
  });

  $: if (game && $drawerDraft) {
    if ($drawerDraft.gameDir !== game.gameDir) {
      drawerDraft.init(game, globalCfg);
      integrityReport = null;
      integrityError = '';
    }
  }

  function returnToShelf() {
    $activeView = 'library';
  }

  function openSettingsTab(tabName = 'general') {
    activeTab = tabName;
  }

  function openDirectory() {
    if (game?.gameDir) {
      AppAPI.openDirectory(game.gameDir);
    }
  }

  async function handleAddToLibrary() {
    if (!game) return;
    const lib = $libraryPath || '/home/staernid/Games';
    try {
      const stub = await AppAPI.createGameStub(lib, game.appId, game.name, game.store || 'Steam', game.branch || 'public');
      if (stub) {
        const normalized = updateGameInStore(stub);
        if (normalized) {
          selectedGame.set(normalized);
        }
        notificationStore.items.add({
          type: 'success',
          title: 'Added to Library',
          message: `Saved skeleton metadata stub for ${game.name}`
        });
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Add to Library',
        message: err?.message || String(err)
      });
    }
  }

  async function handlePrimaryAction() {
    if (!game) return;

    if (isGameRunning) {
      try {
        const activeSession = getActiveSession($multiseatStore.activeSessions, game.gameDir);
        if (activeSession?.is_running) {
          await multiseatStore.stopSession(game.gameDir);
        } else {
          await AppAPI.stopGame(game.gameDir);
        }
        setGameRunning(game.gameDir, false);
      } catch (err) {
        notificationStore.items.add({
          type: 'error',
          title: 'Failed to Stop Game',
          message: err?.message || String(err)
        });
      }
      return;
    }

    const isInstalled = game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0));
    if (!isInstalled) {
      const appID = parseInt(game.appId, 10);
      if (!appID || isNaN(appID)) {
        notificationStore.items.add({
          type: 'error',
          title: 'Identifier Required',
          message: 'This title does not have a valid Steam AppID for depot downloading.'
        });
        return;
      }

      let dest = game.gameDir;
      const lib = $libraryPath || '/home/staernid/Games';

      // If it's a virtual store title or lacks a directory, register the stub in the library first
      if (!dest || game.isVirtual) {
        try {
          const stub = await AppAPI.createGameStub(lib, String(appID), game.name, game.store || 'Steam', game.branch || 'public');
          if (stub) {
            const normalized = updateGameInStore(stub);
            if (normalized) {
              selectedGame.set(normalized);
              dest = normalized.gameDir;
            } else if (stub.game_dir || stub.gameDir) {
              dest = stub.game_dir || stub.gameDir;
            }
          }
        } catch (err) {
          console.warn('Could not initialize library stub prior to download:', err);
          const cleanName = game.name.replace(/[™®©]/g, '').replace(/[:/\\?*<>|"]/g, '').replace(/\s+/g, ' ').trim() || game.name;
          dest = `${lib}/${cleanName}`;
        }
      }

      try {
        await tasksStore.start({
          app_id: appID,
          game_name: game.name,
          target_dir: dest,
          platform: game.platform || 'win64',
          auto_apply: true,
          store_type: (game.store || 'steam').toLowerCase(),
          branch: game.branch || 'public',
        });
      } catch (err) {
        notificationStore.items.add({
          type: 'error',
          title: 'Download Failed',
          message: err?.message || String(err)
        });
      }
      return;
    }

    const isReady = game.isReady || game.isPatched || !!game.exePath;
    if (!isReady) {
      await queueApplyGame(game);
      return;
    }

    try {
      if ($multiseatStore.isCoopEnabled) {
        await multiseatStore.launchSession(game, userCfg || {});
      } else {
        await AppAPI.launchGame({
          name: game.name,
          game_dir: game.gameDir,
          app_id: game.appId,
          runner: game.runner || globalCfg?.default_runner || 'umu',
          exe_path: game.exePath || $drawerDraft?.exePath || '',
          exe_args: game.exeArgs || $drawerDraft?.exeArgs || '',
        });
      }
      setGameRunning(game.gameDir, true);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Launch Failed',
        message: err?.message || String(err)
      });
    }
  }

  async function handleStopSeat(e) {
    const seatId = e.detail;
    if (game?.gameDir) {
      await multiseatStore.stopSeat(game.gameDir, seatId);
    }
  }


  async function handleSaveSettings() {
    if (!game?.gameDir) return;
    isSaving = true;
    try {
      const ok = await drawerDraft.save(game);
      if (ok) {
        saveSuccess = true;
        if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
        saveSuccessTimer = setTimeout(() => {
          saveSuccess = false;
        }, 3000);
      }
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

  async function handleResetSettings() {
    if (game) {
      await drawerDraft.resetDefaults(game, globalCfg);
    }
  }

  async function handleApplyMetaLink(e) {
    if (!game?.gameDir) return;
    const { targetStore, targetId } = e.detail;
    try {
      const updated = await AppAPI.linkGameMetadata(game.gameDir, targetStore, targetId);
      notificationStore.items.add({
        type: 'success',
        title: 'Metadata Linked',
        message: targetStore === 'none' || targetStore === 'standalone' || !targetId || targetId === '0'
          ? `Unlinked metadata for ${game.name}`
          : `Linked ${game.name} to ${targetStore.toUpperCase()} (#${targetId})`
      });
      if (updated) {
        selectedGame.update(curr => (curr && isSamePath(curr.gameDir, game.gameDir) ? { ...curr, ...updated } : curr));
        drawerDraft.init(updated, globalCfg);
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Linking Failed',
        message: err?.message || String(err)
      });
    }
  }

  async function handleUnlinkMeta() {
    if (!game?.gameDir) return;
    try {
      const updated = await AppAPI.unlinkGameMetadata(game.gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Metadata Reset',
        message: `Restored auto-detected identity for ${game.name}.`
      });
      if (updated) {
        selectedGame.update(curr => (curr && isSamePath(curr.gameDir, game.gameDir) ? { ...curr, ...updated } : curr));
        drawerDraft.init(updated, globalCfg);
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Reset Failed',
        message: err?.message || String(err)
      });
    }
  }

  async function runIntegrityCheck() {
    if (!game?.gameDir) return;
    isCheckingIntegrity = true;
    integrityError = '';
    try {
      integrityReport = await AppAPI.verifyGameIntegrity(game.gameDir);
    } catch (err) {
      integrityError = err?.message || 'Integrity check failed.';
    } finally {
      isCheckingIntegrity = false;
    }
  }

  async function handleNormalizeDirectory() {
    if (!game?.gameDir) return;
    isNormalizing = true;
    try {
      const newDir = await AppAPI.normalizeGameDirectory(game.gameDir);
      if (newDir && newDir !== game.gameDir) {
        await refreshGame(newDir);
      }
      notificationStore.items.add({
        type: 'success',
        title: 'Folder Normalized',
        message: `Normalized directory structure for ${game.name}`
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

  async function handleUpdateGame(e) {
    if (!game?.gameDir) return;
    const branchToUpdate = e?.detail?.branch || game.branch || 'public';
    if (game.appId && game.appId !== '0') {
      try {
        await tasksStore.start({
          app_id: parseInt(game.appId, 10),
          game_name: game.name,
          target_dir: game.gameDir,
          platform: 'win64',
          auto_apply: true,
          store_type: game.store || 'steam',
          branch: branchToUpdate,
        });
        notificationStore.items.add({
          type: 'info',
          title: 'Update Enqueued',
          message: `Update task started for ${game.name}${branchToUpdate !== 'public' ? ` (${branchToUpdate})` : ''}`,
          gameName: game.name
        });
      } catch (err) {
        notificationStore.items.add({
          type: 'error',
          title: 'Update Failed',
          message: err?.message || String(err)
        });
      }
    } else {
      try {
        await AppAPI.submitUpdate(game.gameDir);
        notificationStore.items.add({
          type: 'info',
          title: 'Update Enqueued',
          message: `Update task started for ${game.name}`,
          gameName: game.name
        });
      } catch (err) {
        notificationStore.items.add({
          type: 'error',
          title: 'Update Failed',
          message: err?.message || String(err)
        });
      }
    }
  }

  $: isSteam = Boolean((game?.appId && isSteamApp(game.appId)) || ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId)));
  const primaryTabs = ['overview', 'community', 'resources', 'general', 'environment', 'saves', 'files'];

  function handleNavigateTab(e) {
    const { tab, subTab, newsIndex } = e.detail || {};
    if (tab) activeTab = tab;
    if (subTab && tab === 'community') communitySubTab = subTab;
    if (subTab && tab === 'files') filesSubTab = subTab;
    if (newsIndex !== undefined) communityNewsIndex = newsIndex;
  }

  function cycleTab(dir) {
    const idx = primaryTabs.indexOf(activeTab);
    let next = idx + dir;
    if (next < 0) next = primaryTabs.length - 1;
    if (next >= primaryTabs.length) next = 0;
    activeTab = primaryTabs[next];
  }

  function handleKeydown(e) {
    const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);
    if (isInput || $modalStore) return;

    if (e.key === 'Escape' || e.key === 'b' || e.key === 'B') {
      e.preventDefault();
      returnToShelf();
    } else if (e.key === 'x' || e.key === 'X') {
      e.preventDefault();
      openSettingsTab('general');
    } else if (e.key === 'y' || e.key === 'Y') {
      e.preventDefault();
      openDirectory();
    } else if (e.key === 'q' || e.key === 'Q' || e.key === 'PageUp') {
      e.preventDefault();
      cycleTab(-1);
    } else if (e.key === 'e' || e.key === 'E' || e.key === 'PageDown') {
      e.preventDefault();
      cycleTab(1);
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if game}
  <div class="flex-1 overflow-y-auto w-full relative flex flex-col animate-fade-in custom-scrollbar">
    <!-- Hero Banner -->
    <GameHeroBanner
      {game}
      {isGameRunning}
      {isTaskActive}
      {activeGameTask}
      {globalCfg}
      on:returnToShelf={returnToShelf}
      on:primaryAction={handlePrimaryAction}
      on:addToLibrary={handleAddToLibrary}
      on:openDirectory={openDirectory}
      on:stopSeat={handleStopSeat}
      on:openBranchTab={() => { activeTab = 'files'; filesSubTab = 'branch'; }}
      on:openBetasTab={() => { activeTab = 'files'; filesSubTab = 'branch'; }}
    />

    <!-- Main Content & Consolidated Tabs Section -->
    <div class="max-w-6xl w-full mx-auto px-6 py-4 flex-1 flex flex-col space-y-4">
      <!-- Live Task Progress HUD -->
      {#if activeGameTask}
        <GameTaskHud task={activeGameTask} />
      {/if}

      <!-- Primary Navigation Tabs -->
      <div class="flex items-center space-x-2 border-b border-surface-border pb-2 text-xs font-bold select-none overflow-x-auto custom-scrollbar">
        <!-- 1. Overview Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'overview'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'overview' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Overview</span>
        </button>

        <!-- 2. Community Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'community'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'community' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
          </svg>
          <span>Community</span>
        </button>

        <!-- 3. Resources Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'resources'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'resources' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <span>Resources</span>
        </button>

        <!-- 4. General Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'general'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'general' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          <span>General</span>
        </button>

        <!-- 5. Environment Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'environment'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'environment' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <span>Environment</span>
        </button>

        <!-- 6. Saves Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'saves'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'saves' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
          </svg>
          <span>Saves</span>
        </button>

        <!-- 7. Files & Diagnostics Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'files'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'files' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
          <span>Files & Diagnostics</span>
          {#if isGameRunning}
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Execution log active"></span>
          {/if}
        </button>
      </div>

      <!-- Tab Content Area -->
      <div class="flex-1 pb-12">
        {#if activeTab === 'overview'}
          <HubOverviewTab {game} on:navigateTab={handleNavigateTab} />
        {:else if activeTab === 'community'}
          <HubCommunityTab
            {game}
            initialSubTab={communitySubTab}
            initialNewsIndex={communityNewsIndex}
          />
        {:else if activeTab === 'resources'}
          <HubResourcesTab {game} />
        {:else if activeTab === 'files'}
          <HubFilesTab
            {game}
            activeSubTab={filesSubTab}
            {isGameRunning}
            {isTaskActive}
            {activeGameTask}
            {isCheckingIntegrity}
            {isNormalizing}
            {integrityError}
            {integrityReport}
            on:runIntegrityCheck={runIntegrityCheck}
            on:updateGame={handleUpdateGame}
            on:normalizeDirectory={handleNormalizeDirectory}
          />
        {:else if !(game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0)))}
          <!-- Uninstalled State Placeholder for Technical Tabs -->
          <div class="bg-surface-card border border-surface-border rounded-2xl p-10 text-center space-y-4 max-w-lg mx-auto mt-8">
            <div class="w-12 h-12 rounded-2xl bg-surface-cardHover border border-surface-borderHighlight text-neutral-300 flex items-center justify-center mx-auto">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
            </div>
            <div class="space-y-1">
              <h3 class="font-bold text-white text-base">Game Files Not Installed</h3>
              <p class="text-xs text-neutral-400 leading-relaxed">Download and install {game.name} to configure launch profiles, Wine prefixes, and inspect disk files.</p>
            </div>
            <button
              type="button"
              on:click={handlePrimaryAction}
              class="px-5 py-2.5 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 font-black text-xs shadow-lg cursor-pointer"
            >
              Download Game Now
            </button>
          </div>
        {:else if activeTab === 'general'}
          <HubGeneralTab
            {game}
            {overrideCount}
            {isSaving}
            {saveSuccess}
            on:save={handleSaveSettings}
            on:reset={handleResetSettings}
            on:linkMeta={handleApplyMetaLink}
            on:unlinkMeta={handleUnlinkMeta}
          />
        {:else if activeTab === 'environment'}
          <HubEnvironmentTab
            {game}
            {globalCfg}
            {overrideCount}
            {isSaving}
            {saveSuccess}
            {runnerOptions}
            {operatingModeOptions}
            {gameLanguageOptions}
            on:save={handleSaveSettings}
            on:reset={handleResetSettings}
          />
        {:else if activeTab === 'saves'}
          <div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-xl">
            <HubSavesCard {game} />
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

