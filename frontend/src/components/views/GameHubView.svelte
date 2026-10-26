<script>
  import { onMount, onDestroy } from 'svelte';
  import {
    selectedGame,
    activeView,
    setGameRunning,
    runningGames,
    isGamePathRunning,
    libraryPath,
    resolvePlaceholderTitle
  } from '../../stores/library.js';
  import { tasksStore, isActiveStatus } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { drawerDraft, createAutosave } from '../../stores/drawerDraft.js';
  import { multiseatStore } from '../../stores/multiseat.js';
  import { AppAPI } from '../../api/app.js';
  import { countOverrides } from '../../utils/game.js';
  import { cycleHubTab, findActiveGameTask, findVerifyTask } from '../../utils/hub.js';
  import {
    executePrimaryAction,
    confirmTrashGame,
    addGameStubToLibrary,
    queueGameUpdate,
    applyMetaLink,
    unlinkMeta,
    normalizeDirectory
  } from '../../utils/hubActions.js';
  import GameHeroBanner from './hub/GameHeroBanner.svelte';
  import HubNavTabs from './hub/HubNavTabs.svelte';
  import HubTabContent from './hub/HubTabContent.svelte';

  let activeTab = $state('overview');
  let communitySubTab = $state('achievements');
  let communityNewsIndex = $state(0);
  let filesSubTab = $state('depots');

  let globalCfg = $state({});
  let userCfg = $state(null);
  let isSaving = $state(false);
  let saveSuccess = $state(false);
  let saveSuccessTimer = null;
  let autosave = null;
  let unsubSelected = null;

  // Integrity State
  let integrityReport = $state(null);
  let localIntegrityError = $state('');
  let isStartingVerify = $state(false);
  let isNormalizing = $state(false);

  const allTasks = tasksStore.items;

  let game = $derived($selectedGame);
  let isGameRunning = $derived(Boolean(game?.gameDir && isGamePathRunning($runningGames, game.gameDir)));

  let activeGameTask = $derived(findActiveGameTask($allTasks, game));
  let isTaskActive = $derived(Boolean(activeGameTask));

  let verifyTask = $derived(findVerifyTask($allTasks, game));
  let isCheckingIntegrity = $derived(Boolean(isStartingVerify || (verifyTask && isActiveStatus(verifyTask.status))));
  let integrityError = $derived(verifyTask?.status === 'failed' ? (verifyTask.error || 'Integrity check failed.') : localIntegrityError);

  let overrideCount = $derived(countOverrides($drawerDraft, game, globalCfg));

  onMount(async () => {
    autosave = createAutosave(drawerDraft, () => handleSaveSettings(true), 400);

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

    unsubSelected = selectedGame.subscribe(current => {
      if (current && $drawerDraft.gameDir !== current.gameDir) {
        drawerDraft.init(current, globalCfg);
        integrityReport = null;
        localIntegrityError = '';
      }
    });
  });

  onDestroy(() => {
    if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
    if (autosave) autosave.destroy();
    if (unsubSelected) unsubSelected();
  });

  // Resolve store title metadata for placeholder titles on game change
  $effect(() => {
    if (game) resolvePlaceholderTitle(game);
  });

  function returnToShelf() {
    $activeView = 'library';
  }

  function openDirectory() {
    if (game?.gameDir) {
      AppAPI.openDirectory(game.gameDir);
    }
  }

  async function handlePrimaryAction() {
    await executePrimaryAction({
      game,
      activeGameTask,
      isGameRunning,
      userCfg,
      globalCfg,
      drawerDraftExePath: $drawerDraft?.exePath,
      drawerDraftExeArgs: $drawerDraft?.exeArgs,
    });
  }

  async function handleSaveSettings(silent = false) {
    if (!game?.gameDir || isSaving) return;
    isSaving = true;
    try {
      const ok = await drawerDraft.save(game);
      if (ok) {
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

  async function runIntegrityCheck() {
    if (!game?.gameDir) return;
    isStartingVerify = true;
    localIntegrityError = '';
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
      localIntegrityError = err?.message || 'Failed to start integrity verification.';
    } finally {
      isStartingVerify = false;
    }
  }

  async function handleNormalize() {
    if (!game?.gameDir) return;
    isNormalizing = true;
    try {
      await normalizeDirectory(game);
    } finally {
      isNormalizing = false;
    }
  }

  function handleNavigateTab(e) {
    const { tab, subTab, newsIndex } = (e && typeof e === 'object' && 'detail' in e ? e.detail : e) || {};
    if (tab) activeTab = tab;
    if (subTab && tab === 'community') communitySubTab = subTab;
    if (subTab && tab === 'files') filesSubTab = subTab;
    if (newsIndex !== undefined) communityNewsIndex = newsIndex;
  }

  function handleKeydown(e) {
    const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);
    if (isInput || $modalStore) return;

    if (e.key === 'Escape' || e.key === 'b' || e.key === 'B') {
      e.preventDefault();
      returnToShelf();
    } else if (e.key === 'x' || e.key === 'X') {
      e.preventDefault();
      activeTab = 'config';
    } else if (e.key === 'y' || e.key === 'Y') {
      e.preventDefault();
      openDirectory();
    } else if (e.key === 'q' || e.key === 'Q' || e.key === 'PageUp') {
      e.preventDefault();
      activeTab = cycleHubTab(activeTab, -1);
    } else if (e.key === 'e' || e.key === 'E' || e.key === 'PageDown') {
      e.preventDefault();
      activeTab = cycleHubTab(activeTab, 1);
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
      onaddToLibrary={() => addGameStubToLibrary(game, $libraryPath)}
      onopenDirectory={openDirectory}
      ontrash={() => confirmTrashGame(game, isGameRunning)}
      onstopSeat={(e) => game?.gameDir && multiseatStore.stopSeat(game.gameDir, e.detail)}
      onopenBranchTab={() => { activeTab = 'files'; filesSubTab = 'branch'; }}
      onopenBetasTab={() => { activeTab = 'files'; filesSubTab = 'branch'; }}
    />

    <!-- Main Content & Consolidated Tabs Section -->
    <div class="max-w-6xl w-full mx-auto px-6 py-4 flex-1 flex flex-col space-y-4">
      <HubNavTabs
        {activeTab}
        {overrideCount}
        {isGameRunning}
        outdatedModsCount={game.outdatedModsCount || 0}
        onselectTab={(tab) => { activeTab = tab; }}
      />

      <HubTabContent
        bind:activeTab
        {game}
        {globalCfg}
        {overrideCount}
        {isSaving}
        {saveSuccess}
        {communitySubTab}
        {communityNewsIndex}
        {filesSubTab}
        {isGameRunning}
        {isTaskActive}
        {activeGameTask}
        {verifyTask}
        {isCheckingIntegrity}
        {isNormalizing}
        {integrityError}
        {integrityReport}
        onsaveSettings={() => handleSaveSettings(false)}
        onresetSettings={() => game && drawerDraft.resetDefaults(game, globalCfg)}
        onlinkMeta={(e) => {
          const { targetStore, targetId } = (e && typeof e === 'object' && 'detail' in e ? e.detail : e) || {};
          applyMetaLink(game, targetStore, targetId, globalCfg);
        }}
        onunlinkMeta={() => unlinkMeta(game, globalCfg)}
        onrunIntegrityCheck={runIntegrityCheck}
        onupdateGame={(e) => {
          const branch = (e && typeof e === 'object' && 'detail' in e ? e.detail?.branch : e?.branch) || game.branch || 'public';
          queueGameUpdate(game, branch);
        }}
        onnormalizeDirectory={handleNormalize}
        onnavigateTab={handleNavigateTab}
        onprimaryAction={handlePrimaryAction}
      />
    </div>
  </div>
{/if}
