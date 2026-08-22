<script>
  import { onMount } from 'svelte';
  import TopBar from './components/header/TopBar.svelte';
  import HeroStage from './components/stage/HeroStage.svelte';
  import CartridgeShelf from './components/deck/CartridgeShelf.svelte';
  import GameDrawer from './components/stage/GameDrawer.svelte';
  import DownloaderView from './components/views/DownloaderView.svelte';
  import ActivityView from './components/views/ActivityView.svelte';
  import SettingsView from './components/views/SettingsView.svelte';
  import Toast from './components/common/Toast.svelte';

  import {
    libraryPath,
    scanLibrary,
    activeView,
    showInspector,
    pathPopover,
    searchQuery,
    filteredGames,
    selectedGame,
    queueApplyGame
  } from './stores/library.js';
  import { taskQueue } from './stores/queue.js';
  import { notificationStore } from './stores/notifications.js';
  import { configStore } from './stores/config.js';
  import { initRuntimeEvents, AppAPI } from './api/app.js';

  onMount(() => {
    // 1. Initialize UI scale
    const savedScale = localStorage.getItem('gamux_ui_scale') || '16';
    configStore.setScale(savedScale);

    // 2. Setup runtime event bridges
    initRuntimeEvents({
      onNotification: (data) => {
        notificationStore.items.add({
          type: 'info',
          title: data?.title || 'Gamux Alert',
          message: data?.message || ''
        });
      },
      onDownloadProgress: (data) => {
        taskQueue.updateDownloadProgress(data);
      },
      onStepProgress: (data) => {
        taskQueue.updateStepProgress(data);
      }
    });

    // 3. Scan library if path exists
    if ($libraryPath) {
      scanLibrary($libraryPath);
    }

    // 4. Global Handheld Keyboard Navigation
    function handleKeydown(e) {
      const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);

      if ((e.key === '/' || (e.ctrlKey && e.key === 'k')) && !isInput) {
        e.preventDefault();
        const searchInput = document.querySelector('input[type="text"]');
        if (searchInput) searchInput.focus();
      } else if (e.key === 'Escape') {
        if ($showInspector) {
          $showInspector = false;
        } else if ($pathPopover) {
          $pathPopover = false;
        } else if ($searchQuery) {
          $searchQuery = '';
        } else if ($activeView !== 'library') {
          $activeView = 'library';
        }
      } else if (!isInput && !e.ctrlKey && !e.altKey && !e.metaKey) {
        if (e.key === '1') {
          $activeView = 'library';
        } else if (e.key === '2') {
          $activeView = 'downloader';
        } else if (e.key === '3') {
          $activeView = 'activity';
        } else if (e.key === '4') {
          $activeView = 'settings';
        } else if ((e.key === 'x' || e.key === 'X') && $activeView === 'library' && $selectedGame) {
          $showInspector = !$showInspector;
        } else if ((e.key === 'ArrowRight' || e.key === 'ArrowDown') && $activeView === 'library' && !$showInspector) {
          e.preventDefault();
          stepGameSelection(1);
        } else if ((e.key === 'ArrowLeft' || e.key === 'ArrowUp') && $activeView === 'library' && !$showInspector) {
          e.preventDefault();
          stepGameSelection(-1);
        } else if (e.key === 'Enter' && $activeView === 'library' && !$showInspector && $selectedGame) {
          e.preventDefault();
          if ($selectedGame.lutris_registered) {
            const slug = $selectedGame.AppID || $selectedGame.Name.toLowerCase().replace(/[^\w-]/g, '-');
            AppAPI.launchLutrisGame(slug);
          } else {
            queueApplyGame($selectedGame);
          }
        }
      }
    }

    function stepGameSelection(dir) {
      let list = [];
      filteredGames.subscribe(v => { list = v; })();
      if (!list || list.length === 0) return;

      let current = null;
      selectedGame.subscribe(v => { current = v; })();
      const currentIndex = current ? list.findIndex(g => g.GameDir === current.GameDir) : -1;

      let nextIndex = currentIndex + dir;
      if (nextIndex < 0) nextIndex = list.length - 1;
      if (nextIndex >= list.length) nextIndex = 0;

      selectedGame.set(list[nextIndex]);
    }

    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  });
</script>

<div class="bg-[#09090c] text-[#f4f4f6] flex flex-col h-screen overflow-hidden antialiased select-none">
  <!-- Slim System Top Bar (40px) -->
  <TopBar />

  <!-- Main View Area -->
  {#if $activeView === 'library'}
    <!-- Handheld Dual-Tier Stage (Hero on top, Cartridge shelf on bottom) -->
    <div class="flex-1 flex flex-col min-h-0 relative">
      <HeroStage />
      <CartridgeShelf />
      <GameDrawer />
    </div>
  {:else if $activeView === 'downloader'}
    <DownloaderView />
  {:else if $activeView === 'activity'}
    <ActivityView />
  {:else if $activeView === 'settings'}
    <SettingsView />
  {/if}

  <!-- Global Toasts -->
  <Toast />
</div>
