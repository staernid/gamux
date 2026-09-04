<script>
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import TopBar from './components/header/TopBar.svelte';
  import Carousel3D from './components/stage/Carousel3D.svelte';
  import ModalHost from './components/common/ModalHost.svelte';
  import GameHubView from './components/views/GameHubView.svelte';
  import DownloadsView from './components/views/DownloadsView.svelte';
  import SettingsView from './components/views/SettingsView.svelte';
  import Toast from './components/common/Toast.svelte';

  import {
    libraryPath,
    scanLibrary,
    activeView,
    pathPopover,
    filteredGames,
    selectedGame,
    queueApplyGame,
    setGameRunning,
    refreshGame,
    renameGameInStore,
    updateGameInStore,
    sortBy,
    cycleSortOrder,
    getSortLabel
  } from './stores/library.js';
  import { tasksStore } from './stores/tasks.js';
  import { modalStore } from './stores/modal.js';
  import { notificationStore } from './stores/notifications.js';
  import { configStore } from './stores/config.js';
  import { multiseatStore } from './stores/multiseat.js';
  import { initRuntimeEvents, AppAPI } from './api/app.js';
  import { formatBytes } from './utils/format.js';
  import {
    getPlatformColors,
    getPlatformType,
    getGameSource,
    getMetadataInfo,
    isSamePath,
    formatPlaytime,
    formatLastPlayed,
    formatInstallDate
  } from './utils/game.js';
  import { playTick, playSelect, playBack, playNav } from './utils/audio.js';
  import { GamepadTracker } from './input/gamepad.js';
  import { gamuxDev } from './debug/debugBridge.js';

  let carouselRef;

  $: activeColors = $selectedGame ? getPlatformColors($selectedGame) : null;
  $: platformType = $selectedGame ? getPlatformType($selectedGame) : '';
  $: gameSource = $selectedGame ? getGameSource($selectedGame) : '';
  $: metaInfo = $selectedGame ? getMetadataInfo($selectedGame) : { provider: 'None', id: '' };

  const views = ['library', 'tasks', 'settings'];

  function navigateWithGuard(targetView, onProceed = null) {
    const curr = get(activeView) || 'library';
    const isDirty = get(configStore.isDirty);
    if (curr === 'settings' && isDirty && targetView !== 'settings') {
      modalStore.open('confirm', {
        title: 'Unsaved Changes',
        message: 'You have unsaved changes in Settings. Do you want to discard them or keep editing?',
        confirmLabel: 'Discard Changes',
        cancelLabel: 'Keep Editing',
        danger: true,
        onConfirm: () => {
          configStore.setDirty(false);
          if (typeof onProceed === 'function') {
            onProceed();
          } else {
            playNav();
            activeView.set(targetView);
          }
        }
      });
      return;
    }

    if (typeof onProceed === 'function') {
      onProceed();
    } else {
      playNav();
      activeView.set(targetView);
    }
  }

  function cycleView(dir) {
    const curr = get(activeView) || 'library';
    const currentBase = curr === 'game-hub' ? 'library' : curr;
    const idx = views.indexOf(currentBase);
    let next = idx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    navigateWithGuard(views[next]);
  }

  function openSelectedGameHub(targetGame = null) {
    const game = targetGame || $selectedGame;
    if (!game) return;
    if (targetGame) selectedGame.set(targetGame);
    playSelect();
    $activeView = 'game-hub';
  }

  function openCurrentDirectory() {
    if ($selectedGame?.gameDir) {
      AppAPI.openDirectory($selectedGame.gameDir);
    }
  }

  function openSearchGrid() {
    playNav();
    modalStore.open('search-grid');
  }

  function stepGameSelection(dir) {
    const list = get(filteredGames) || [];
    if (!list || list.length === 0) return;

    const current = get(selectedGame);
    const currentIndex = current ? list.findIndex(g => isSamePath(g.gameDir, current.gameDir)) : -1;

    let nextIndex = (currentIndex + dir + list.length) % list.length;
    playTick();
    selectedGame.set(list[nextIndex]);
  }

  onMount(() => {
    // 1. Initialize UI scale & config
    const savedScale = localStorage.getItem('gamux_ui_scale') || '18';
    configStore.setScale(savedScale);
    configStore.load();

    // 2. Setup runtime event bridges
    function handleEventWithStoreSync(ev) {
      tasksStore.handleTaskEvent(ev);
      if (ev?.type === 'task:completed' || ev?.type === 'transfer:completed') {
        const gameDir = ev.game_dir || ev.gameDir;
        if (ev.game) {
          if (ev.operation === 'normalize' && ev.metadata?.old_game_dir) {
            renameGameInStore(ev.metadata.old_game_dir, ev.game);
          } else {
            updateGameInStore(ev.game);
          }
        } else if (ev.operation === 'normalize' && ev.metadata?.new_game_dir && ev.metadata?.old_game_dir) {
          refreshGame(ev.metadata.new_game_dir).then((g) => {
            if (g) renameGameInStore(ev.metadata.old_game_dir, g);
          });
        } else if (gameDir) {
          refreshGame(gameDir);
        } else if (ev.operation === 'migrate') {
          scanLibrary();
        }
      }
    }

    initRuntimeEvents({
      onTaskEvent: handleEventWithStoreSync,
      onTransferEvent: handleEventWithStoreSync,
      onNotification: (data) => {
        notificationStore.items.add({
          type: 'info',
          title: data?.title || 'Gamux Alert',
          message: data?.message || ''
        });
      },
      onDownloadProgress: (data) => {
        tasksStore.updateDownloadProgress(data);
      },
      onStepProgress: (data) => {
        tasksStore.updateStepProgress(data);
      },
      onGameStatus: (data) => {
        if (data?.gameDir) {
          setGameRunning(data.gameDir, !!data.isRunning, data.playtimeSeconds, data.lastPlayed);
          if (!data.isRunning) {
            multiseatStore.setSessionInactive(data.gameDir);
          }
        }
      }
    });

    // 3. Check URL query parameters for view / drawer previewing
    const urlParams = new URLSearchParams(window.location.search);
    const viewParam = urlParams.get('view');
    if (viewParam && ['library', 'game-hub', 'tasks', 'settings'].includes(viewParam)) {
      activeView.set(viewParam);
    }
    if (urlParams.get('properties') === '1' || urlParams.get('properties') === 'true' || urlParams.get('drawer') === '1') {
      activeView.set('game-hub');
    }
    if (urlParams.get('search') === '1' || urlParams.get('search') === 'true') {
      modalStore.open('search-grid');
    }

    // 4. Scan library if path exists
    if ($libraryPath) {
      scanLibrary($libraryPath);
    }

    // 5. Global Handheld Keyboard Navigation
    function handleKeydown(e) {
      const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);

      if ((e.key === '/' || (e.ctrlKey && e.key === 'k')) && !isInput && !$modalStore) {
        e.preventDefault();
        openSearchGrid();
      } else if (e.key === 'Escape') {
        if ($modalStore) {
          playBack();
          modalStore.close();
        } else if ($pathPopover) {
          playBack();
          $pathPopover = false;
        } else if ($activeView !== 'library') {
          playBack();
          navigateWithGuard('library');
        }
      } else if (!isInput && (e.key === 'F5' || (e.ctrlKey && (e.key === 'r' || e.key === 'R')))) {
        e.preventDefault();
        scanLibrary();
      } else if (!isInput && !e.ctrlKey && !e.altKey && !e.metaKey && !$modalStore) {
        if (e.key === 'r' || e.key === 'R') {
          e.preventDefault();
          scanLibrary();
        } else if (e.key === 'q' || e.key === 'Q') {
          cycleView(-1);
        } else if (e.key === 'e' || e.key === 'E') {
          cycleView(1);
        } else if (e.key === '1') {
          navigateWithGuard('library');
        } else if (e.key === '2') {
          navigateWithGuard('tasks');
        } else if (e.key === '3') {
          navigateWithGuard('settings');
        } else if ((e.key === 'y' || e.key === 'Y') && $activeView === 'library') {
          playTick();
          cycleSortOrder();
        } else if ((e.key === 'f' || e.key === 'F' || e.key === 'o' || e.key === 'O') && $activeView === 'library') {
          openCurrentDirectory();
        } else if ((e.key === 'ArrowRight' || e.key === 'd' || e.key === 'D') && $activeView === 'library') {
          e.preventDefault();
          if (carouselRef) carouselRef.step(1);
          else stepGameSelection(1);
        } else if ((e.key === 'ArrowLeft' || e.key === 'a' || e.key === 'A') && $activeView === 'library') {
          e.preventDefault();
          if (carouselRef) carouselRef.step(-1);
          else stepGameSelection(-1);
        } else if ((e.key === 'Enter' || e.key === ' ') && $activeView === 'library' && $selectedGame) {
          e.preventDefault();
          openSelectedGameHub();
        }
      }
    }

    // 6. Gamepad API Polling Loop
    const gamepadTracker = new GamepadTracker();
    let padLoopId = null;

    function pollGamepad() {
      const pads = navigator.getGamepads ? navigator.getGamepads() : [];
      const pad = Array.from(pads).find(p => p && p.connected);

      const actions = gamepadTracker.process(pad);
      if (actions) {
        const currView = get(activeView) || 'library';
        const modalActive = !!get(modalStore);

        // 1. If a Modal is open, handle Back/Close
        if (modalActive) {
          if (actions.actionBack) {
            playBack();
            modalStore.close();
          }
          padLoopId = requestAnimationFrame(pollGamepad);
          return;
        }

        // 2. Main Shelf Navigation
        if (currView === 'library') {
          // Primary Action (A / Enter) -> Open Game Hub
          if (actions.actionPrimary) {
            openSelectedGameHub();
          }

          // Secondary (Y) -> Cycle Sort Order
          if (actions.actionFiles) {
            playTick();
            cycleSortOrder();
          }

          // Directional Navigation (D-Pad or Left Analog Stick)
          if (actions.navLeft) {
            if (carouselRef) carouselRef.step(-1);
            else stepGameSelection(-1);
          }
          if (actions.navRight) {
            if (carouselRef) carouselRef.step(1);
            else stepGameSelection(1);
          }

          // Rotary Wheel Scrubbing
          if (actions.rotaryDelta !== 0) {
            if (carouselRef) {
              carouselRef.addRotaryDelta(actions.rotaryDelta);
            }
          }
        } else if (currView === 'game-hub') {
          // Game Hub Navigation
          if (actions.actionBack) {
            playBack();
            activeView.set('library');
          }
          if (actions.actionFiles) {
            openCurrentDirectory();
          }
        } else if (currView === 'settings') {
          if (actions.actionBack) {
            playBack();
            navigateWithGuard('library');
          }
        }

        // Cycle Views (LB / RB)
        if (actions.prevView) cycleView(-1);
        if (actions.nextView) cycleView(1);
      }

      padLoopId = requestAnimationFrame(pollGamepad);
    }

    padLoopId = requestAnimationFrame(pollGamepad);
    window.addEventListener('keydown', handleKeydown);

    return () => {
      window.removeEventListener('keydown', handleKeydown);
      if (padLoopId) cancelAnimationFrame(padLoopId);
    };
  });
</script>

<div class="bg-surface-base text-[#f4f4f6] flex flex-col h-screen overflow-hidden antialiased select-none relative">
  <!-- Minimal Floating Navigation Pill at Top -->
  <TopBar />

  <!-- Main View Area -->
  {#if $activeView === 'library'}
    <div class="flex-1 flex flex-col min-h-0 relative items-center justify-between">
      <!-- Center 3D 5-Item Carousel -->
      <Carousel3D bind:this={carouselRef} on:launch={(e) => openSelectedGameHub(e.detail)} />

      <!-- Bottom HUD: Left Command Glossary & Right Game Details -->
      <div class="w-full px-8 pb-6 flex items-end justify-between z-20 flex-shrink-0">
        <!-- Bottom Left: Command Glossary (Clickable & Controller-Mapped) -->
        <div class="flex flex-col space-y-1.5 text-xs text-neutral-400 font-mono select-none">
          <!-- Library Shelf Mode Glossary -->
          <div class="flex items-center space-x-3">
            <!-- Details / Hub -->
            <button
              type="button"
              on:click={() => openSelectedGameHub()}
              class="flex items-center space-x-1.5 hover:text-neutral-200 transition-colors group cursor-pointer"
              title="Open Game Hub (A or Enter)"
            >
              <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-300 group-hover:text-white text-3xs font-bold">
                A / ↵
              </span>
              <span class="group-hover:text-neutral-200">Game Hub</span>
            </button>

            <span class="text-neutral-700">·</span>

            <!-- Sort Cycling via (Y) -->
            <button
              type="button"
              on:click={() => cycleSortOrder()}
              class="flex items-center space-x-1.5 hover:text-neutral-200 transition-colors group cursor-pointer"
              title="Cycle library sort order (Y)"
            >
              <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-300 group-hover:text-white text-3xs font-bold">
                Y
              </span>
              <span class="group-hover:text-neutral-200">Sort: {getSortLabel($sortBy)}</span>
            </button>
          </div>

          <!-- Line 2: Search (Single Newline) -->
          <div class="flex items-center">
            <button
              type="button"
              on:click={openSearchGrid}
              class="flex items-center space-x-1.5 hover:text-neutral-200 transition-colors group cursor-pointer"
              title="Open library search grid (/)"
            >
              <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-300 group-hover:text-white text-3xs font-bold">
                /
              </span>
              <span class="group-hover:text-neutral-200">Search</span>
            </button>
          </div>
        </div>

        <!-- Bottom Right: Game Details & Telemetry -->
        {#if $selectedGame}
          <div class="flex flex-col items-end space-y-0.5 text-2xs text-neutral-400 font-mono text-right max-w-sm truncate">
            <div>
              <span class="text-neutral-500">Source:</span>
              <span class="text-neutral-200 font-semibold">{gameSource}</span>
              {#if metaInfo.provider && metaInfo.provider !== 'None'}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-500">Meta:</span>
                <span class="text-neutral-300 font-semibold">{metaInfo.provider}{metaInfo.id ? ` #${metaInfo.id}` : ''}</span>
              {/if}
            </div>
            <div>
              <span class="text-neutral-500">Platform:</span>
              <span class="text-neutral-300 font-semibold">{$selectedGame.platform || 'Linux'}</span>
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-300">{$selectedGame.runner || $configStore?.default_runner || 'UMU'}</span>
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-300 font-semibold">{formatBytes($selectedGame.diskSizeBytes)}</span>
              {#if $selectedGame.playtimeSeconds > 0}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-200 font-medium">{formatPlaytime($selectedGame.playtimeSeconds)}</span>
              {/if}
              {#if $selectedGame.lastPlayed > 0}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-400">Played {formatLastPlayed($selectedGame.lastPlayed)}</span>
              {/if}
              {#if $selectedGame.installedAt > 0}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-400">Installed {formatInstallDate($selectedGame.installedAt)}</span>
              {/if}
            </div>
            <div class="truncate max-w-full text-neutral-500" title={$selectedGame.gameDir}>
              <span class="text-neutral-500">Path:</span>
              <span class="text-neutral-500">{$selectedGame.gameDir}</span>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {:else if $activeView === 'game-hub'}
    <GameHubView />
  {:else if $activeView === 'tasks'}
    <DownloadsView />
  {:else if $activeView === 'settings'}
    <SettingsView />
  {/if}

  <!-- Centralized Modal Host -->
  <ModalHost />

  <!-- Global Toasts -->
  <Toast />
</div>
