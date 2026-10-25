<script>
  import { onMount } from 'svelte';
  import TopBar from './components/header/TopBar.svelte';
  import LibraryView from './components/views/LibraryView.svelte';
  import ModalHost from './components/common/ModalHost.svelte';
  import GameHubView from './components/views/GameHubView.svelte';
  import DownloadsView from './components/views/DownloadsView.svelte';
  import SettingsView from './components/views/SettingsView.svelte';
  import Toast from './components/common/Toast.svelte';

  import {
    libraryPath,
    scanLibrary,
    games,
    isScanning,
    activeView,
    pathPopover,
    filteredGames,
    selectedGame,
    queueApplyGame,
    setGameRunning,
    setRunningInstances,
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
  import { themeStore } from './stores/theme.js';
  import { multiseatStore } from './stores/multiseat.js';
  import { profilesStore } from './stores/profiles.js';
  import { initRuntimeEvents, AppAPI } from './api/app.js';
  import { formatBytes } from './utils/format.js';
  import {
    getPlatformColors,
    getPlatformType,
    getGameSource,
    getMetadataInfo,
    isSamePath,
    normalizeGame,
    normalizeGameMetadata,
    formatPlaytime,
    formatLastPlayed,
    formatInstallDate
  } from './utils/game.js';
  import { playTick, playSelect, playBack, playNav } from './utils/audio.js';
  import { GamepadTracker } from './input/gamepad.js';
  import { gamuxDev } from './debug/debugBridge.js';

  let libraryViewRef = $state();

  let activeColors = $derived($selectedGame ? getPlatformColors($selectedGame) : null);
  let platformType = $derived($selectedGame ? getPlatformType($selectedGame) : '');
  let gameSource = $derived($selectedGame ? getGameSource($selectedGame) : '');
  let metaInfo = $derived($selectedGame ? getMetadataInfo($selectedGame) : { provider: 'None', id: '' });

  const views = ['library', 'tasks', 'game-hub', 'settings', 'catalog'];

  function navigateWithGuard(targetView, onProceed = null) {
    if (typeof onProceed === 'function') {
      onProceed();
    } else {
      playNav();
      activeView.set(targetView);
    }
  }

  function cycleView(dir) {
    const curr = $activeView || 'library';
    const idx = views.indexOf(curr);
    const validIdx = idx >= 0 ? idx : 0;
    let next = validIdx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    navigateWithGuard(views[next]);
  }

  function openSelectedGameHub(targetGame = null) {
    const game = targetGame || $selectedGame;
    if (!game) {
      const all = $filteredGames || [];
      if (all.length > 0) {
        selectedGame.set(all[0]);
        playSelect();
        $activeView = 'game-hub';
      }
      return;
    }
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
    activeView.set('catalog');
  }

  function stepGameSelection(dir) {
    const list = $filteredGames || [];
    if (!list || list.length === 0) return;

    const current = $selectedGame;
    const currentIndex = current ? list.findIndex(g => isSamePath(g.gameDir, current.gameDir)) : -1;

    let nextIndex = (currentIndex + dir + list.length) % list.length;
    playTick();
    selectedGame.set(list[nextIndex]);
  }

  let pendingTargetAppId = $state(null);

  function parseAppIdFromTarget(target) {
    if (!target) return '';
    const clean = String(target).trim();
    // 1. gamux://app/123 or gamux://game/123 or gamux://123
    const gamuxMatch = clean.match(/^gamux:\/\/(?:app\/|game\/)?(\d+)/i);
    if (gamuxMatch) return gamuxMatch[1];
    // 2. steam://store/123 or steam://rungameid/123
    const steamMatch = clean.match(/^steam:\/\/(?:store\/|rungameid\/)?(\d+)/i);
    if (steamMatch) return steamMatch[1];
    // 3. https://store.steampowered.com/app/123
    const webMatch = clean.match(/\/app\/(\d+)/i);
    if (webMatch) return webMatch[1];
    // 4. Raw numeric AppID
    if (/^\d+$/.test(clean)) return clean;
    return '';
  }

  async function handleOpenTarget(target) {
    if (!target) return;
    const appId = parseAppIdFromTarget(target);
    if (!appId) {
      notificationStore.items.add({
        type: 'warning',
        title: 'Unknown Target',
        message: `Unable to extract Steam AppID from "${target}"`
      });
      return;
    }

    // 1. Check local library first
    const currentGames = $games || [];
    const found = currentGames.find(g => g && String(g.appId) === String(appId));

    if (found) {
      openSelectedGameHub(found);
      notificationStore.items.add({
        type: 'info',
        title: 'Game Opened',
        message: `Navigated to ${found.name} in Game Hub`
      });
      pendingTargetAppId = null;
      return;
    }

    if ($isScanning) {
      // Library scan is actively in progress; remember target to promote if found when scan completes
      pendingTargetAppId = appId;
    }

    // 2. Open directly as Virtual Store Title in GameHubView
    let meta = null;
    try {
      meta = normalizeGameMetadata(await AppAPI.getGameMetadata('steam', appId));
    } catch {
      // Offline or metadata fetch fallback
    }

    const virtualGame = normalizeGame({
      name: meta?.name || `Steam App ${appId}`,
      app_id: String(appId),
      store: 'Steam',
      source: 'Steam',
      cover_url: meta?.headerImage || `https://cdn.cloudflare.steamstatic.com/steam/apps/${appId}/header.jpg`,
      release_date: meta?.releaseDate || '',
      is_installed: false,
      is_stub: false,
      is_virtual: true,
      is_ready: false,
      file_count: 0,
      disk_size_bytes: 0,
      platform: 'win64'
    });

    selectedGame.set(virtualGame);
    $activeView = 'game-hub';
    playSelect();

    notificationStore.items.add({
      type: 'info',
      title: 'Store Game Opened',
      message: `Navigated to ${virtualGame.name} in Game Hub`
    });
  }

  $effect(() => {
    if (pendingTargetAppId && $games?.length > 0) {
      const found = $games.find(g => g && String(g.appId) === String(pendingTargetAppId));
      if (found) {
        openSelectedGameHub(found);
        pendingTargetAppId = null;
      }
    }
  });

  onMount(() => {
    // 1. Initialize UI scale, theme & config
    themeStore.init();
    const savedScale = localStorage.getItem('gamux_ui_scale') || '18';
    configStore.setScale(savedScale);
    configStore.load();
    profilesStore.load();

    // 2. Setup runtime event bridges
    function handleEventWithStoreSync(ev) {
      tasksStore.handleTaskEvent(ev);
      if (ev?.type === 'task:completed' || ev?.type === 'transfer:completed') {
        const gameDir = ev.game_dir || ev.gameDir;
        if (ev.game) {
          if (ev.metadata?.old_game_dir) {
            renameGameInStore(ev.metadata.old_game_dir, ev.game);
          } else {
            updateGameInStore(ev.game);
          }
        } else if (ev.metadata?.new_game_dir && ev.metadata?.old_game_dir) {
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
          setGameRunning(data.gameDir, !!data.isRunning, data.playtimeSeconds, data.lastPlayed, {
            profileId: data.profileId,
            seatId: data.seatId,
            instanceId: data.instanceId,
            gameName: data.gameName,
            appId: data.appId,
            pid: data.pid
          });
          if (!data.isRunning && !data.seatId) {
            multiseatStore.setSessionInactive(data.gameDir);
          }
        }
      },
      onOpenTarget: (target) => {
        handleOpenTarget(target);
      },
      onSystemThemeChanged: (data) => {
        if (typeof data?.isDark === 'boolean') {
          themeStore.handleSystemThemeChange(data.isDark);
        }
      }
    });

    // Reconcile active running instances on application startup
    AppAPI.getRunningInstances().then((instances) => {
      if (Array.isArray(instances) && instances.length > 0) {
        setRunningInstances(instances);
      } else {
        AppAPI.getRunningGames().then((running) => {
          if (Array.isArray(running)) {
            for (const dir of running) {
              if (dir) setGameRunning(dir, true);
            }
          }
        }).catch(() => {});
      }
    }).catch(() => {});

    // Check for cold-start initial target forwarded via CLI/protocol
    AppAPI.getInitialTarget().then((target) => {
      if (target) {
        handleOpenTarget(target);
      }
    }).catch(() => {});

    // 3. Check URL query parameters for view / drawer previewing
    const urlParams = new URLSearchParams(window.location.search);
    const viewParam = urlParams.get('view');
    if (viewParam && ['library', 'game-hub', 'tasks', 'settings', 'catalog'].includes(viewParam)) {
      activeView.set(viewParam);
    }
    if (urlParams.get('properties') === '1' || urlParams.get('properties') === 'true' || urlParams.get('drawer') === '1') {
      activeView.set('game-hub');
    }
    if (urlParams.get('search') === '1' || urlParams.get('search') === 'true') {
      activeView.set('catalog');
    }

    // 4. Scan library if path exists
    if ($libraryPath) {
      scanLibrary($libraryPath);
    }

    // 5. Global Handheld Keyboard Navigation
    function handleKeydown(e) {
      const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        if ($activeView === 'catalog' && libraryViewRef?.focusSearch) {
          libraryViewRef.focusSearch();
        } else {
          openSearchGrid();
        }
      } else if (e.key === '/' && !isInput && !$modalStore) {
        e.preventDefault();
        if ($activeView === 'library' && libraryViewRef?.focusSearch) {
          libraryViewRef.focusSearch();
        } else {
          openSearchGrid();
        }
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
          openSelectedGameHub();
        } else if (e.key === '4') {
          navigateWithGuard('settings');
        } else if (e.key === '5') {
          navigateWithGuard('catalog');
        } else if ((e.key === 'y' || e.key === 'Y') && $activeView === 'library') {
          playTick();
          cycleSortOrder();
        } else if ((e.key === 'f' || e.key === 'F' || e.key === 'o' || e.key === 'O') && $activeView === 'library') {
          openCurrentDirectory();
        } else if ((e.key === 'ArrowRight' || e.key === 'd' || e.key === 'D') && $activeView === 'library') {
          e.preventDefault();
          if (libraryViewRef) libraryViewRef.step(1);
          else stepGameSelection(1);
        } else if ((e.key === 'ArrowLeft' || e.key === 'a' || e.key === 'A') && $activeView === 'library') {
          e.preventDefault();
          if (libraryViewRef) libraryViewRef.step(-1);
          else stepGameSelection(-1);
        } else if ((e.key === 'Enter' || e.key === ' ') && $activeView === 'library' && $selectedGame) {
          e.preventDefault();
          openSelectedGameHub();
        }
      }
    }

    // 6. Gamepad API Polling Loop (connection-aware & idle-safe)
    const gamepadTracker = new GamepadTracker();
    let padLoopId = null;
    let isGamepadConnected = false;

    function hasConnectedGamepad() {
      if (typeof navigator === 'undefined' || !navigator.getGamepads) return false;
      const pads = navigator.getGamepads();
      return Array.from(pads).some(p => p && p.connected);
    }

    function pollGamepad() {
      // If hidden in background/tray, yield until visible
      if (typeof document !== 'undefined' && document.hidden) {
        padLoopId = requestAnimationFrame(pollGamepad);
        return;
      }

      const pads = navigator.getGamepads ? navigator.getGamepads() : [];
      const pad = Array.from(pads).find(p => p && p.connected);

      if (!pad) {
        // Sleep if no controller is present
        isGamepadConnected = false;
        padLoopId = null;
        return;
      }

      const actions = gamepadTracker.process(pad);
      if (actions) {
        const currView = $activeView || 'library';
        const modalActive = !!$modalStore;

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
            if (libraryViewRef) libraryViewRef.step(-1);
            else stepGameSelection(-1);
          }
          if (actions.navRight) {
            if (libraryViewRef) libraryViewRef.step(1);
            else stepGameSelection(1);
          }

          // Rotary Wheel Scrubbing
          if (actions.rotaryDelta !== 0) {
            if (libraryViewRef) {
              libraryViewRef.addRotaryDelta(actions.rotaryDelta);
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

    function wakeGamepadLoop() {
      if (!padLoopId && hasConnectedGamepad()) {
        isGamepadConnected = true;
        padLoopId = requestAnimationFrame(pollGamepad);
      }
    }

    function onGamepadConnected() {
      wakeGamepadLoop();
    }

    function onGamepadDisconnected() {
      if (!hasConnectedGamepad()) {
        isGamepadConnected = false;
        if (padLoopId) {
          cancelAnimationFrame(padLoopId);
          padLoopId = null;
        }
      }
    }

    // Initialize loop only if a gamepad is currently connected
    wakeGamepadLoop();

    window.addEventListener('gamepadconnected', onGamepadConnected);
    window.addEventListener('gamepaddisconnected', onGamepadDisconnected);
    window.addEventListener('keydown', handleKeydown);

    return () => {
      window.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('gamepadconnected', onGamepadConnected);
      window.removeEventListener('gamepaddisconnected', onGamepadDisconnected);
      if (padLoopId) cancelAnimationFrame(padLoopId);
    };
  });
</script>

<div class="bg-surface-base text-neutral-900 dark:text-[#f4f4f6] flex flex-col h-screen overflow-hidden antialiased select-none relative">
  <!-- Minimal Floating Navigation Pill at Top -->
  <TopBar />

  <!-- Main View Area -->
  {#if $activeView === 'library'}
    <LibraryView bind:this={libraryViewRef} mode="local" />
  {:else if $activeView === 'catalog'}
    <LibraryView bind:this={libraryViewRef} mode="catalog" />
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
