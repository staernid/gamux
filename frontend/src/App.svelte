<script>
  import { onMount } from 'svelte';
  import TopBar from './components/header/TopBar.svelte';
  import Carousel3D from './components/stage/Carousel3D.svelte';
  import ModalHost from './components/common/ModalHost.svelte';
  import GameDrawer from './components/stage/GameDrawer.svelte';
  import DownloadsView from './components/views/DownloadsView.svelte';
  import ActivityView from './components/views/ActivityView.svelte';
  import SettingsView from './components/views/SettingsView.svelte';
  import Toast from './components/common/Toast.svelte';

  import {
    libraryPath,
    scanLibrary,
    activeView,
    showInspector,
    pathPopover,
    filteredGames,
    selectedGame,
    queueApplyGame,
    setGameRunning,
    refreshGame,
    renameGameInStore
  } from './stores/library.js';
  import { tasksStore } from './stores/tasks.js';
  import { modalStore } from './stores/modal.js';
  import { notificationStore } from './stores/notifications.js';
  import { configStore } from './stores/config.js';
  import { initRuntimeEvents, AppAPI } from './api/app.js';
  import { formatBytes } from './utils/format.js';
  import { getPlatformColors, getPlatformType } from './utils/game.js';
  import { gamuxDev } from './debug/debugBridge.js';

  let carouselRef;

  $: activeColors = $selectedGame ? getPlatformColors($selectedGame) : null;
  $: platformType = $selectedGame ? getPlatformType($selectedGame) : '';

  const views = ['library', 'downloads', 'activity', 'settings'];

  function cycleView(dir) {
    let curr = 'library';
    activeView.subscribe(v => { curr = v; })();
    const idx = views.indexOf(curr);
    let next = idx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    activeView.set(views[next]);
  }

  function executePrimaryAction(targetGame) {
    const game = targetGame || $selectedGame;
    if (!game) return;

    const isReady = game.isReady || (game.isPatched && (game.lutrisRegistered || game.operatingMode === 'standalone'));
    if (isReady) {
      AppAPI.launchGame({
        game_dir: game.gameDir,
        name: game.name,
        app_id: game.appId,
        platform: game.platform,
        exe_path: game.exePath,
        exe_args: game.exeArgs,
        operating_mode: game.operatingMode,
        runner: game.runner,
      });
    } else {
      queueApplyGame(game);
    }
  }

  function openCurrentDirectory() {
    if ($selectedGame?.gameDir) {
      AppAPI.openDirectory($selectedGame.gameDir);
    }
  }

  function toggleInspector() {
    if ($selectedGame) {
      $showInspector = !$showInspector;
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
    const savedScale = localStorage.getItem('gamux_ui_scale') || '16';
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
    if (viewParam && ['library', 'downloads', 'activity', 'settings'].includes(viewParam)) {
      activeView.set(viewParam);
    }
    if (urlParams.get('drawer') === '1' || urlParams.get('drawer') === 'true') {
      showInspector.set(true);
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
        } else if ($showInspector) {
          $showInspector = false;
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
          $activeView = 'downloads';
        } else if (e.key === '3') {
          $activeView = 'activity';
        } else if (e.key === '4') {
          $activeView = 'settings';
        } else if ((e.key === 'x' || e.key === 'X') && $activeView === 'library' && $selectedGame) {
          toggleInspector();
        } else if ((e.key === 'y' || e.key === 'Y' || e.key === 'f' || e.key === 'F' || e.key === 'o' || e.key === 'O') && $activeView === 'library') {
          openCurrentDirectory();
        } else if ((e.key === 'ArrowRight' || e.key === 'd' || e.key === 'D') && $activeView === 'library' && !$showInspector) {
          e.preventDefault();
          if (carouselRef) carouselRef.step(1);
          else stepGameSelection(1);
        } else if ((e.key === 'ArrowLeft' || e.key === 'a' || e.key === 'A') && $activeView === 'library' && !$showInspector) {
          e.preventDefault();
          if (carouselRef) carouselRef.step(-1);
          else stepGameSelection(-1);
        } else if ((e.key === 'Enter' || e.key === ' ') && $activeView === 'library' && !$showInspector && $selectedGame) {
          e.preventDefault();
          executePrimaryAction();
        }
      }
    }

    // 6. Gamepad API Polling Loop
    let lastPadStates = {};
    let padLoopId = null;

    function pollGamepad() {
      const pads = navigator.getGamepads ? navigator.getGamepads() : [];
      const pad = Array.from(pads).find(p => p && p.connected);

      if (pad) {
        const isPressed = (idx) => pad.buttons[idx]?.pressed;
        const wasPressed = (idx) => lastPadStates[idx];
        const justPressed = (idx) => isPressed(idx) && !wasPressed(idx);

        let inspectorOpen = false;
        showInspector.subscribe(v => { inspectorOpen = v; })();

        let currView = 'library';
        activeView.subscribe(v => { currView = v; })();
        let modalActive = false;
        modalStore.subscribe(v => { modalActive = !!v; })();

        // Button A (0): Launch / Prepare or Select
        if (justPressed(0) && currView === 'library' && !inspectorOpen && !modalActive) {
          executePrimaryAction();
        }

        // Button B (1): Close Modal / Inspector / Escape
        if (justPressed(1)) {
          if (modalActive) {
            modalStore.close();
          } else if (inspectorOpen) {
            showInspector.set(false);
          } else if (currView !== 'library') {
            activeView.set('library');
          }
        }

        // Button X (2): Toggle Inspector
        if (justPressed(2) && currView === 'library' && !modalActive) {
          showInspector.set(!inspectorOpen);
        }

        // Button Y (3): Open Folder
        if (justPressed(3) && currView === 'library' && !modalActive) {
          openCurrentDirectory();
        }

        // LB (4) / RB (5): Cycle Views
        if (justPressed(4)) cycleView(-1);
        if (justPressed(5)) cycleView(1);

        // D-Pad Left (14) / Right (15) & Left Analog Stick
        const stickX = pad.axes[0] || 0;
        const wasStickLeft = lastPadStates['stick_left'] || false;
        const wasStickRight = lastPadStates['stick_right'] || false;
        const isStickLeft = stickX < -0.55;
        const isStickRight = stickX > 0.55;

        if (currView === 'library' && !inspectorOpen && !modalActive) {
          if (justPressed(14) || (isStickLeft && !wasStickLeft)) {
            if (carouselRef) carouselRef.step(-1);
            else stepGameSelection(-1);
          }
          if (justPressed(15) || (isStickRight && !wasStickRight)) {
            if (carouselRef) carouselRef.step(1);
            else stepGameSelection(1);
          }
        }

        lastPadStates['stick_left'] = isStickLeft;
        lastPadStates['stick_right'] = isStickRight;

        // Right Analog Stick Rotary Wheel (iPod Click Wheel on Right Stick only)
        const rx = pad.axes[2] || 0;
        const ry = pad.axes[3] || 0;
        const rMag = Math.sqrt(rx * rx + ry * ry);
        if (currView === 'library' && !inspectorOpen && !modalActive) {
          if (rMag >= 0.30) {
            const angle = Math.atan2(ry, rx);
            if (lastPadStates['stick_angle_r'] !== undefined && lastPadStates['stick_angle_r'] !== null) {
              let d = angle - lastPadStates['stick_angle_r'];
              while (d > Math.PI) d -= 2 * Math.PI;
              while (d < -Math.PI) d += 2 * Math.PI;

              if (Math.abs(d) <= Math.PI * 0.7 && Math.abs(d) > 0.001) {
                // 1 full rotation (2*PI) scrolls 7 cards
                const scrollStep = d * (7.0 / (2 * Math.PI));
                if (carouselRef) {
                  carouselRef.addRotaryDelta(scrollStep);
                }
              }
            }
            lastPadStates['stick_angle_r'] = angle;
          } else {
            lastPadStates['stick_angle_r'] = null;
          }
        }

        // Record button states
        for (let i = 0; i < pad.buttons.length; i++) {
          lastPadStates[i] = pad.buttons[i]?.pressed;
        }
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
      <Carousel3D bind:this={carouselRef} on:launch={(e) => executePrimaryAction(e.detail)} />

      <!-- Bottom HUD: Left Command Glossary & Right Game Details (Subtle & Barely Noticeable) -->
      <div class="w-full px-8 pb-6 flex items-end justify-between z-20 flex-shrink-0">
        <!-- Bottom Left: Command Glossary (Clickable & Controller-Mapped) -->
        <div class="flex flex-col space-y-1.5 text-xs text-slate-500 font-mono select-none">
          <!-- Line 1: Primary Actions -->
          <div class="flex items-center space-x-3">
            <!-- Launch -->
            <button
              type="button"
              on:click={() => executePrimaryAction()}
              class="flex items-center space-x-1.5 hover:text-slate-200 transition-colors group cursor-pointer"
              title="Launch selected game (A or Enter)"
            >
              <span class="px-1.5 py-0.5 rounded bg-[#181824] border border-[#27273a] text-slate-400 group-hover:text-white text-[10px] font-bold">
                A / ↵
              </span>
              <span class="group-hover:text-slate-300">Launch</span>
            </button>

            <span class="text-slate-700">·</span>

            <!-- Options -->
            <button
              type="button"
              on:click={toggleInspector}
              class="flex items-center space-x-1.5 hover:text-slate-200 transition-colors group cursor-pointer"
              title="Open per-game settings drawer (X)"
            >
              <span class="px-1.5 py-0.5 rounded bg-[#181824] border border-[#27273a] text-slate-400 group-hover:text-white text-[10px] font-bold">
                X
              </span>
              <span class="group-hover:text-slate-300">Options</span>
            </button>

            <span class="text-slate-700">·</span>

            <!-- Files -->
            <button
              type="button"
              on:click={openCurrentDirectory}
              class="flex items-center space-x-1.5 hover:text-slate-200 transition-colors group cursor-pointer"
              title="Open game directory in file manager (Y)"
            >
              <span class="px-1.5 py-0.5 rounded bg-[#181824] border border-[#27273a] text-slate-400 group-hover:text-white text-[10px] font-bold">
                Y
              </span>
              <span class="group-hover:text-slate-300">Files</span>
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

        <!-- Bottom Right: Game Details (Subtle & Barely Noticeable) -->
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

      <!-- Per-game Drawer Inspector -->
      <GameDrawer />
    </div>
  {:else if $activeView === 'downloads' || $activeView === 'downloader' || $activeView === 'transfers'}
    <DownloadsView />
  {:else if $activeView === 'activity'}
    <ActivityView />
  {:else if $activeView === 'settings'}
    <SettingsView />
  {/if}

  <!-- Centralized Modal Host -->
  <ModalHost />

  <!-- Global Toasts -->
  <Toast />
</div>
