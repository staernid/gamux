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
    updateGameInStore,
    trashGame
  } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { drawerDraft } from '../../stores/drawerDraft.js';
  import { multiseatStore, getActiveSession } from '../../stores/multiseat.js';
  import { profilesStore } from '../../stores/profiles.js';
  import { AppAPI } from '../../api/app.js';
  import { isSamePath, countOverrides, normalizeGame, isSteamApp, isPlaceholderTitle } from '../../utils/game.js';
  import HubSavesCard from './hub/cards/HubSavesCard.svelte';
  import GameHeroBanner from './hub/GameHeroBanner.svelte';
  import HubOverviewTab from './hub/HubOverviewTab.svelte';
  import HubCommunityTab from './hub/HubCommunityTab.svelte';
  import HubConfigTab from './hub/HubConfigTab.svelte';
  import HubFilesTab from './hub/HubFilesTab.svelte';

  // 4 Primary Consolidated Surfaces: 'overview' | 'config' | 'files' | 'saves'
  let activeTab = $state('overview');
  let communitySubTab = $state('achievements');
  let communityNewsIndex = $state(0);
  let filesSubTab = $state('depots');

  let globalCfg = $state({});
  let userCfg = $state(null);
  let isSaving = $state(false);
  let saveSuccess = $state(false);
  let saveSuccessTimer = null;
  let autoSaveTimeout = null;
  let initialDraftJSON = $state('');

  // Integrity State
  let integrityReport = $state(null);
  let integrityError = $state('');
  let isCheckingIntegrity = $state(false);
  let isNormalizing = $state(false);

  const allTasks = tasksStore.items;

  let game = $derived($selectedGame);
  let isGameRunning = $derived(Boolean(game?.gameDir && isGamePathRunning($runningGames, game.gameDir)));

  let activeGameTask = $derived.by(() => {
    const tasks = $allTasks || [];
    return tasks.find(t => {
      if (t.status !== 'running' && t.status !== 'downloading' && t.status !== 'queued' && t.status !== 'pending' && t.status !== 'paused') return false;
      if (game?.gameDir && isSamePath(t.gameDir, game.gameDir)) return true;
      if (game?.appId && game.appId !== '0' && (String(t.appId) === String(game.appId))) return true;
      return false;
    });
  });
  let isTaskActive = $derived(Boolean(activeGameTask));

  $effect(() => {
    const tasks = $allTasks || [];
    const vTask = tasks.find(t => {
      if (t.type !== 'verify' && t.operation !== 'verify') return false;
      if (game?.gameDir && isSamePath(t.gameDir, game.gameDir)) return true;
      if (game?.appId && game.appId !== '0' && (String(t.appId) === String(game.appId))) return true;
      return false;
    });
    if (vTask) {
      if (vTask.status === 'completed') {
        if (isCheckingIntegrity) isCheckingIntegrity = false;
        if (vTask.metadata?.integrity_report && !integrityReport) {
          try {
            integrityReport = JSON.parse(vTask.metadata.integrity_report);
          } catch (_) {}
        }
      } else if (vTask.status === 'failed') {
        if (isCheckingIntegrity) {
          isCheckingIntegrity = false;
          integrityError = vTask.error || 'Integrity check failed.';
        }
      }
    }
  });

  let overrideCount = $derived(countOverrides($drawerDraft, game, globalCfg));

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
      initialDraftJSON = JSON.stringify($drawerDraft);
    }
  });

  onDestroy(() => {
    if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
    if (autoSaveTimeout) clearTimeout(autoSaveTimeout);
  });

  function triggerAutoSave() {
    if (!game?.gameDir || !initialDraftJSON) return;
    const currentJSON = JSON.stringify($drawerDraft);
    if (currentJSON === initialDraftJSON || isSaving) return;
    if (autoSaveTimeout) clearTimeout(autoSaveTimeout);
    autoSaveTimeout = setTimeout(() => {
      handleSaveSettings(true);
    }, 400);
  }

  $effect(() => {
    if ($drawerDraft && initialDraftJSON) {
      triggerAutoSave();
    }
  });

  $effect(() => {
    if (game && $drawerDraft) {
      if ($drawerDraft.gameDir !== game.gameDir) {
        drawerDraft.init(game, globalCfg);
        initialDraftJSON = JSON.stringify($drawerDraft);
        integrityReport = null;
        integrityError = '';
      } else if (!initialDraftJSON) {
        initialDraftJSON = JSON.stringify($drawerDraft);
      }
    }
  });

  $effect(() => {
    if (game && isPlaceholderTitle(game.name, game.appId)) {
      const prov = (game.metadataProvider || game.store || 'steam').toLowerCase();
      const id = game.metadataId || game.appId;
      if (id && id !== '0') {
        AppAPI.getGameMetadata(prov, id).then(meta => {
          if (meta && meta.name && !isPlaceholderTitle(meta.name, id)) {
            const updated = { ...game, name: meta.name };
            if (meta.header_image && (!game.coverUrl || game.coverUrl.includes('header.jpg'))) {
              updated.coverUrl = meta.header_image;
            }
            selectedGame.set(normalizeGame(updated));
            updateGameInStore(updated);
          }
        }).catch(() => {});
      }
    }
  });

  function returnToShelf() {
    $activeView = 'library';
  }

  function openSettingsTab(tabName = 'config') {
    activeTab = tabName;
  }

  function openDirectory() {
    if (game?.gameDir) {
      AppAPI.openDirectory(game.gameDir);
    }
  }

  function handleTrashGame() {
    if (!game?.gameDir || isGameRunning) return;
    const gameName = game.name || 'this game';
    const gameDir = game.gameDir;

    modalStore.open('confirm', {
      title: `Move "${gameName}" to Trash?`,
      message: `This will move the game files (${gameDir}) to your desktop Trash. Saves and emulation configuration will remain intact.`,
      confirmLabel: 'Move to Trash',
      cancelLabel: 'Keep Game',
      danger: true,
      onConfirm: async () => {
        try {
          await trashGame(gameDir);
          activeView.set('library');
          notificationStore.items.add({
            type: 'success',
            title: 'Moved to Trash',
            message: `${gameName} was moved to Trash.`
          });
        } catch (err) {
          notificationStore.error('Failed to Move Game to Trash', err);
        }
      }
    });
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

    if (activeGameTask) {
      if (activeGameTask.status === 'paused') {
        await tasksStore.resume(activeGameTask.id);
        return;
      }
      if (activeGameTask.status === 'running' || activeGameTask.status === 'downloading' || activeGameTask.status === 'queued' || activeGameTask.status === 'pending') {
        await tasksStore.pause(activeGameTask.id);
        return;
      }
    }

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

    const targetExe = game.exePath || $drawerDraft?.exePath || '';
    if (!targetExe) {
      notificationStore.warn('No Target Executable', 'Please configure or browse for a target executable in the General tab before launching.');
      return;
    }

    try {
      if ($multiseatStore.isCoopEnabled) {
        await multiseatStore.launchSession(game, userCfg || {});
      } else {
        const isNative = (game.platform || '').toLowerCase().includes('linux') || game.runner === 'linux';
        const profId = profilesStore.activeProfile?.id || 'main';
        await AppAPI.launchGame({
          name: game.name,
          game_dir: game.gameDir,
          app_id: game.appId,
          platform: game.platform || 'win64',
          runner: isNative ? 'linux' : (game.runner || globalCfg?.default_runner || 'umu'),
          exe_path: targetExe,
          exe_args: game.exeArgs || $drawerDraft?.exeArgs || '',
          profile_id: profId,
        });
        setGameRunning(game.gameDir, true, 0, 0, {
          profileId: profId,
          gameName: game.name,
          appId: game.appId,
        });
      }
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


  async function handleSaveSettings(silent = false) {
    if (!game?.gameDir) return;
    isSaving = true;
    try {
      const ok = await drawerDraft.save(game);
      if (ok) {
        initialDraftJSON = JSON.stringify($drawerDraft);
        saveSuccess = true;
        if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
        saveSuccessTimer = setTimeout(() => {
          saveSuccess = false;
        }, 2500);
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

  async function handleResetSettings() {
    if (game) {
      await drawerDraft.resetDefaults(game, globalCfg);
      initialDraftJSON = JSON.stringify($drawerDraft);
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
    integrityReport = null;
    try {
      await AppAPI.startVerifyGameTask(game.gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Integrity Verification Started',
        message: `Scanning files for ${game.name}...`,
        gameName: game.name
      });
    } catch (err) {
      integrityError = err?.message || 'Failed to start integrity verification.';
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

  let isSteam = $derived(Boolean((game?.appId && isSteamApp(game.appId)) || ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId))));
  const primaryTabs = ['overview', 'config', 'files', 'saves'];

  function handleNavigateTab(e) {
    const { tab, subTab, newsIndex } = (e && typeof e === 'object' && 'detail' in e ? e.detail : e) || {};
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
      openSettingsTab('config');
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

<svelte:window onkeydown={handleKeydown} />

{#if game}
  <div class="flex-1 overflow-y-auto w-full relative flex flex-col animate-fade-in custom-scrollbar">
    <!-- Hero Banner -->
    <GameHeroBanner
      {game}
      {isGameRunning}
      {isTaskActive}
      {activeGameTask}
      {globalCfg}
      onreturnToShelf={returnToShelf}
      onprimaryAction={handlePrimaryAction}
      onaddToLibrary={handleAddToLibrary}
      onopenDirectory={openDirectory}
      ontrash={handleTrashGame}
      onstopSeat={handleStopSeat}
      onopenBranchTab={() => { activeTab = 'files'; filesSubTab = 'branch'; }}
      onopenBetasTab={() => { activeTab = 'files'; filesSubTab = 'branch'; }}
    />

    <!-- Main Content & Consolidated Tabs Section -->
    <div class="max-w-6xl w-full mx-auto px-6 py-4 flex-1 flex flex-col space-y-4">
      <!-- Primary Navigation Tabs (4 Surfaces) -->
      <div class="flex items-center space-x-2 border-b border-surface-border pb-2 text-xs font-bold select-none overflow-x-auto custom-scrollbar">
        <!-- 1. Overview Tab -->
        <button
          type="button"
          onclick={() => { activeTab = 'overview'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'overview' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Overview</span>
        </button>

        <!-- 2. Configuration & Overrides Tab (Diff Lens) -->
        <button
          type="button"
          onclick={() => { activeTab = 'config'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'config' || activeTab === 'general' || activeTab === 'environment' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          <span>Configuration</span>
          {#if overrideCount > 0}
            <span class="px-1.5 py-0.2 rounded-full bg-neutral-800 text-neutral-200 text-3xs font-mono border border-neutral-600">
              {overrideCount}
            </span>
          {/if}
        </button>

        <!-- 3. Files & Depots Tab -->
        <button
          type="button"
          onclick={() => { activeTab = 'files'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'files' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
          <span>Files & Depots</span>
          {#if isGameRunning}
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Execution log active"></span>
          {/if}
        </button>

        <!-- 4. Saves Tab -->
        <button
          type="button"
          onclick={() => { activeTab = 'saves'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'saves' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        >
          <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
          </svg>
          <span>Saves</span>
        </button>
      </div>

      <!-- Tab Content Area -->
      <div class="flex-1 pb-12">
        {#if activeTab === 'overview'}
          <HubOverviewTab {game} onnavigateTab={handleNavigateTab} />
        {:else if activeTab === 'community'}
          <div class="space-y-4">
            <button
              type="button"
              onclick={() => { activeTab = 'overview'; }}
              class="text-xs font-bold text-neutral-400 hover:text-white flex items-center space-x-1.5 cursor-pointer pb-1"
            >
              <span>← Back to Overview</span>
            </button>
            <HubCommunityTab
              {game}
              initialSubTab={communitySubTab}
              initialNewsIndex={communityNewsIndex}
            />
          </div>
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
            onrunIntegrityCheck={runIntegrityCheck}
            onupdateGame={handleUpdateGame}
            onnormalizeDirectory={handleNormalizeDirectory}
          />
        {:else if activeTab === 'config' || activeTab === 'general' || activeTab === 'environment'}
          <HubConfigTab
            {game}
            {globalCfg}
            {overrideCount}
            {isSaving}
            {saveSuccess}
            onsave={() => handleSaveSettings(false)}
            onreset={handleResetSettings}
            onlinkMeta={handleApplyMetaLink}
            onunlinkMeta={handleUnlinkMeta}
          />
        {:else if activeTab === 'saves'}
          <div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-xl">
            <HubSavesCard {game} />
          </div>
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
              <p class="text-xs text-neutral-400 leading-relaxed">Download and install {game.name} to inspect files and run diagnostic verification.</p>
            </div>
            <button
              type="button"
              onclick={handlePrimaryAction}
              class="px-5 py-2.5 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 font-black text-xs shadow-lg cursor-pointer"
            >
              Download Game Now
            </button>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

