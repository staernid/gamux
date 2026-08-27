<script>
  import { onMount } from 'svelte';
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
    sortBy,
    cycleSortOrder,
    getSortLabel
  } from './stores/library.js';
  import { tasksStore } from './stores/tasks.js';
  import { modalStore } from './stores/modal.js';
  import { notificationStore } from './stores/notifications.js';
  import { configStore } from './stores/config.js';
  import { initRuntimeEvents, AppAPI } from './api/app.js';
  import { formatBytes } from './utils/format.js';
  import { getPlatformColors, getPlatformType, isSamePath } from './utils/game.js';
  import { GamepadTracker } from './input/gamepad.js';
  import { gamuxDev } from './debug/debugBridge.js';

  let carouselRef;

  $: activeColors = $selectedGame ? getPlatformColors($selectedGame) : null;
  $: platformType = $selectedGame ? getPlatformType($selectedGame) : '';

  const views = ['library', 'tasks', 'settings'];

  function cycleView(dir) {
    let curr = 'library';
    activeView.subscribe(v => { curr = v; })();
    const currentBase = curr === 'game-hub' ? 'library' : (curr === 'downloads' || curr === 'transfers' ? 'tasks' : curr);
    const idx = views.indexOf(currentBase);
    let next = idx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    activeView.set(views[next]);
  }

  function openSelectedGameHub(targetGame = null) {
    const game = targetGame || $selectedGame;
    if (!game) return;
    if (targetGame) selectedGame.set(targetGame);
    $activeView = 'game-hub';
  }

  function openCurrentDirectory() {
    if ($selectedGame?.gameDir) {
      AppAPI.openDirectory($selectedGame.gameDir);
    }
  }

  function openSearchGrid() {
    modalStore.open('search-grid');
  }

  function stepGameSelection(dir) {
    let list = [];
    filteredGames.subscribe(v => { list = v; })();
    if (!list || list.length === 0) return;

    let current = null;
    selectedGame.subscribe(v => { current = v; })();
    const currentIndex = current ? list.findIndex(g => isSamePath(g.gameDir, current.gameDir)) : -1;

    let nextIndex = (currentIndex + dir + list.length) % list.length;
    selectedGame.set(list[nextIndex]);
  }

  onMount(() => {
    // 1. Initialize UI scale
    const savedScale = localStorage.getItem('gamux_ui_scale') || '18';
    configStore.setScale(savedScale);

    // 2. Setup runtime event bridges
    function handleEventWithStoreSync(ev) {
      tasksStore.handleTaskEvent(ev);
      if (ev?.type === 'task:completed' || ev?.type === 'transfer:completed') {
        const gameDir = ev.game_dir || ev.gameDir;
        if (ev.operation === 'normalize' && ev.metadata?.new_game_dir && ev.metadata?.old_game_dir) {
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
          setGameRunning(data.gameDir, !!data.isRunning);
        }
      }
    });

    // 3. Check URL query parameters for view / drawer previewing
    const urlParams = new URLSearchParams(window.location.search);
    const viewParam = urlParams.get('view');
    if (viewParam && ['library', 'game-hub', 'tasks', 'downloads', 'transfers', 'settings'].includes(viewParam)) {
      activeView.set(viewParam === 'downloads' || viewParam === 'transfers' ? 'tasks' : viewParam);
    } else if (viewParam === 'activity') {
      activeView.set('settings');
    }
    if (urlParams.get('properties') === '1' || urlParams.get('properties') === 'true' || urlParams.get('drawer') === '1') {
      const unsub = selectedGame.subscribe(g => {
        if (g) {
          modalStore.open('game-properties', { game: g });
          setTimeout(() => unsub(), 0);
        }
      });
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
          modalStore.close();
        } else if ($pathPopover) {
          $pathPopover = false;
        } else if ($activeView !== 'library') {
          $activeView = 'library';
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
          $activeView = 'library';
        } else if (e.key === '2') {
          $activeView = 'tasks';
        } else if (e.key === '3') {
          $activeView = 'settings';
        } else if ((e.key === 'y' || e.key === 'Y') && $activeView === 'library') {
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
        let currView = 'library';
        activeView.subscribe(v => { currView = v; })();
        let modalActive = false;
        modalStore.subscribe(v => { modalActive = !!v; })();

        // 1. If a Modal is open, handle Back/Close
        if (modalActive) {
          if (actions.actionBack) {
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
            activeView.set('library');
          }
          if (actions.actionFiles) {
            openCurrentDirectory();
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

<div class="bg-[#09090c] text-[#f4f4f6] flex flex-col h-screen overflow-hidden antialiased select-none relative">
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
        <div class="flex flex-col space-y-1.5 text-xs text-slate-500 font-mono select-none">
          <!-- Library Shelf Mode Glossary -->
          <div class="flex items-center space-x-3">
            <!-- Details / Hub -->
            <button
              type="button"
              on:click={() => openSelectedGameHub()}
              class="flex items-center space-x-1.5 hover:text-slate-200 transition-colors group cursor-pointer"
              title="Open Game Hub (A or Enter)"
            >
              <span class="px-1.5 py-0.5 rounded bg-[#181824] border border-[#27273a] text-slate-400 group-hover:text-white text-[10px] font-bold">
                A / ↵
              </span>
              <span class="group-hover:text-slate-300">Game Hub</span>
            </button>

            <span class="text-slate-700">·</span>

            <!-- Sort Cycling via (Y) -->
            <button
              type="button"
              on:click={() => cycleSortOrder()}
              class="flex items-center space-x-1.5 hover:text-slate-200 transition-colors group cursor-pointer"
              title="Cycle library sort order (Y)"
            >
              <span class="px-1.5 py-0.5 rounded bg-[#181824] border border-[#27273a] text-slate-400 group-hover:text-white text-[10px] font-bold">
                Y
              </span>
              <span class="group-hover:text-slate-300">Sort: {getSortLabel($sortBy)}</span>
            </button>
          </div>

          <!-- Line 2: Search (Single Newline) -->
          <div class="flex items-center">
            <button
              type="button"
              on:click={openSearchGrid}
              class="flex items-center space-x-1.5 hover:text-slate-200 transition-colors group cursor-pointer"
              title="Open library search grid (/)"
            >
              <span class="px-1.5 py-0.5 rounded bg-[#181824] border border-[#27273a] text-slate-400 group-hover:text-white text-[10px] font-bold">
                /
              </span>
              <span class="group-hover:text-slate-300">Search</span>
            </button>
          </div>
        </div>

        <!-- Bottom Right: Game Details -->
        {#if $selectedGame}
          <div class="flex flex-col items-end space-y-0.5 text-[11px] text-slate-500 font-mono text-right max-w-xs sm:max-w-sm truncate">
            <div>
              <span class="text-slate-600">Size:</span>
              <span class="text-slate-400 font-semibold">{formatBytes($selectedGame.diskSizeBytes)}</span>
            </div>
            <div>
              <span class="text-slate-600">Platform:</span>
              <span class="text-slate-400 font-semibold">{platformType}</span>
              <span class="text-slate-600">·</span>
              <span class="text-slate-400">{$selectedGame.runner || 'UMU'}</span>
            </div>
            <div class="truncate max-w-full text-slate-600" title={$selectedGame.gameDir}>
              <span class="text-slate-600">Path:</span>
              <span class="text-slate-500">{$selectedGame.gameDir}</span>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {:else if $activeView === 'game-hub'}
    <GameHubView />
  {:else if $activeView === 'tasks' || $activeView === 'downloads' || $activeView === 'downloader' || $activeView === 'transfers' || $activeView === 'operations'}
    <DownloadsView />
  {:else if $activeView === 'settings' || $activeView === 'activity'}
    <SettingsView />
  {/if}

  <!-- Centralized Modal Host -->
  <ModalHost />

  <!-- Global Toasts -->
  <Toast />
</div>
