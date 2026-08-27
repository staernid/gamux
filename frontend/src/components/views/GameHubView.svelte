<script>
  import { onMount, tick } from 'svelte';
  import {
    selectedGame,
    activeView,
    queueApplyGame,
    setGameRunning,
    refreshGame
  } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';
  import { getPlatformColors, getPlatformType, isSteamApp } from '../../utils/game.js';
  import PosterArt from '../common/PosterArt.svelte';
  import DrawerNewsTab from '../stage/drawer/DrawerNewsTab.svelte';
  import DrawerIntegrityTab from '../stage/drawer/DrawerIntegrityTab.svelte';

  let playButtonRef;
  let activeTab = 'overview'; // 'overview' | 'news' | 'integrity'
  let logoFailed = false;
  let heroFailed = false;

  // News State
  let newsItems = [];
  let isLoadingNews = false;
  let expandedNewsIndex = 0;
  let loadedNewsAppID = '';
  let newsError = '';

  // Integrity State
  let integrityReport = null;
  let integrityError = '';
  let isCheckingIntegrity = false;
  let isNormalizing = false;

  $: game = $selectedGame;
  $: if (game) { logoFailed = false; heroFailed = false; }
  $: pColors = game ? getPlatformColors(game) : { border: '#1a9fff', badgeBg: '#1a9fff', label: 'WIN64', glow: 'rgba(26,159,255,0.4)' };
  $: platformLabel = game ? getPlatformType(game) : 'Windows';
  $: isReady = game ? (game.isReady || game.isPatched || !!game.exePath) : false;
  $: isSteam = game ? isSteamApp(game.appId) : false;
  $: heroBackdropUrl = game ? (game.localHeroUrl || game.heroUrl || game.localBannerUrl || game.bannerUrl || (isSteam ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${game.appId}/library_hero.jpg` : '') || game.localCoverUrl || game.coverUrl || '') : '';

  onMount(async () => {
    await tick();
    if (playButtonRef) {
      playButtonRef.focus();
    }
  });

  function returnToShelf() {
    $activeView = 'library';
  }

  function handlePrimaryAction() {
    if (!game) return;

    if (game.isRunning) {
      AppAPI.stopGame(game.gameDir);
      setGameRunning(game.gameDir, false);
      notificationStore.info('Game Stopped', `Terminated process for ${game.name}`);
      return;
    }

    const launchPayload = {
      game_dir: game.gameDir,
      name: game.name,
      app_id: game.appId,
      platform: game.platform,
      exe_path: game.exePath,
      exe_args: game.exeArgs,
      operating_mode: game.operatingMode,
      runner: game.runner,
    };

    if (isReady) {
      AppAPI.launchGame(launchPayload);
    } else {
      AppAPI.quickPlay(launchPayload).catch(() => {
        queueApplyGame(game);
      });
    }
  }


  function openDirectory() {
    if (game?.gameDir) {
      AppAPI.openDirectory(game.gameDir);
    }
  }

  function openProperties() {
    if (game) {
      modalStore.open('game-properties', { game });
    }
  }

  function openLogs() {
    if (game) {
      modalStore.open('game-properties', { game, initialTab: 'logs' });
    }
  }

  function openSaves() {
    if (game) {
      modalStore.open('game-properties', { game, initialTab: 'saves' });
    }
  }


  async function loadNews(force = false) {
    if (!game?.appId || game.appId === '0') return;
    if (!force && loadedNewsAppID === game.appId && newsItems.length > 0) return;

    isLoadingNews = true;
    newsError = '';
    try {
      const items = await AppAPI.fetchNews(String(game.appId), 6);
      newsItems = items || [];
      loadedNewsAppID = game.appId;
      if (!items || items.length === 0) {
        newsError = 'No patch notes found on Steam for this title.';
      }
    } catch (err) {
      newsError = err?.message || String(err);
    } finally {
      isLoadingNews = false;
    }
  }

  function toggleNewsExpand(idx) {
    expandedNewsIndex = expandedNewsIndex === idx ? -1 : idx;
  }

  async function runIntegrityCheck() {
    if (!game?.gameDir) return;
    isCheckingIntegrity = true;
    integrityError = '';
    integrityReport = null;
    try {
      const rep = await AppAPI.verifyIntegrity(game.gameDir);
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

  async function handleNormalizeDirectory() {
    if (!game?.gameDir) return;
    const oldDir = game.gameDir;
    try {
      isNormalizing = true;
      await AppAPI.submitNormalize(oldDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Normalizing Directory',
        message: `Queued directory normalization for ${game.name || oldDir}`
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
    if (!game?.gameDir) return;
    const appIdNum = parseInt(game.appId, 10);
    if (appIdNum && !isNaN(appIdNum)) {
      try {
        await tasksStore.start({
          app_id: appIdNum,
          game_name: game.name,
          target_dir: game.gameDir,
          platform: game.platform,
          auto_apply: true,
        });
      } catch (err) {
        // Handled in store
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
        notificationStore.error('Update Failed', err);
      }
    }
  }

  function cycleTab(dir) {
    const tabs = ['overview'];
    if (isSteam) tabs.push('news');
    tabs.push('integrity');
    const idx = tabs.indexOf(activeTab);
    let next = idx + dir;
    if (next < 0) next = tabs.length - 1;
    if (next >= tabs.length) next = 0;
    activeTab = tabs[next];
  }

  $: if (activeTab === 'news' && loadedNewsAppID !== game?.appId && !isLoadingNews && game?.appId) {
    loadNews();
  }

  function handleKeydown(e) {
    const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);
    if (isInput || $modalStore) return;

    if (e.key === 'Escape' || e.key === 'b' || e.key === 'B') {
      e.preventDefault();
      returnToShelf();
    } else if (e.key === 'x' || e.key === 'X') {
      e.preventDefault();
      openProperties();
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
    <!-- Hero Banner Background & Vignette (Bleeds behind floating topbar) -->
    <div class="relative w-full h-96 sm:h-[420px] flex-shrink-0 overflow-hidden select-none">
      <!-- Background Image with dynamic blur -->
      <div class="absolute inset-0 bg-[#07070a]">
        {#if heroBackdropUrl && !heroFailed}
          <img
            src={heroBackdropUrl}
            alt=""
            class="w-full h-full object-cover object-top opacity-70 filter blur-[0.5px] scale-105 transition-opacity duration-700"
            on:error={() => { heroFailed = true; }}
          />
        {:else}
          <!-- Art-Poor Fallback: Dynamic Ambient Grid & Subtle Watermark -->
          <div class="absolute inset-0 bg-gradient-to-br from-[#0e0f18] via-[#09090e] to-[#040407]">
            <div class="absolute inset-0 opacity-[0.04]" style="background-image: radial-gradient(#ffffff 1px, transparent 1px); background-size: 24px 24px;"></div>
            <div class="absolute -right-10 -bottom-12 text-[160px] font-black text-white/[0.03] select-none pointer-events-none font-mono tracking-tighter">
              {game.name ? game.name.slice(0, 3).toUpperCase() : 'GMX'}
            </div>
          </div>
        {/if}
        <!-- Ambient Radial Color Glow (Top Left) -->
        <div
          class="absolute -top-20 -left-20 w-[600px] h-[350px] rounded-full opacity-25 filter blur-3xl pointer-events-none"
          style="background-color: {pColors.glow || 'rgba(26,159,255,0.4)'};"
        ></div>
        <!-- Ambient Secondary Glow (Bottom Right) -->
        <div
          class="absolute -bottom-10 right-10 w-[450px] h-[250px] rounded-full opacity-15 filter blur-3xl pointer-events-none"
          style="background-color: {pColors.border || '#1a9fff'};"
        ></div>
      </div>

      <!-- Ambient Vignette & Gradient Overlays -->
      <div class="absolute inset-0 bg-gradient-to-t from-[#09090c] via-[#09090c]/50 to-transparent"></div>
      <div class="absolute inset-0 bg-gradient-to-r from-[#09090c]/80 via-transparent to-[#09090c]/60"></div>
      <div class="absolute inset-x-0 top-0 h-24 bg-gradient-to-b from-black/40 to-transparent pointer-events-none"></div>

      <!-- Top Nav Breadcrumb -->
      <div class="absolute top-4 left-6 z-20 flex items-center space-x-3">
        <button
          type="button"
          on:click={returnToShelf}
          class="flex items-center space-x-2 bg-[#12121a]/80 hover:bg-[#1e1e2c] border border-[#27273a] text-slate-300 hover:text-white px-3.5 py-1.5 rounded-xl text-xs font-bold transition shadow-lg backdrop-blur-md group"
          title="Back to Shelf (Esc / B)"
        >
          <span class="text-slate-400 group-hover:text-cyan-400 transition-colors">←</span>
          <span>Shelf</span>
          <span class="text-[10px] text-slate-500 font-mono px-1 py-0.2 rounded bg-black/40 border border-[#2c2c3e]">B / Esc</span>
        </button>
      </div>

      <!-- Hero Content Container -->
      <div class="absolute bottom-6 left-6 right-6 z-20 flex flex-col sm:flex-row sm:items-end justify-between gap-6 max-w-6xl mx-auto">
        <!-- Left: Game Poster + Title/Logo + Badges -->
        <div class="flex items-end space-x-5 min-w-0">
          <!-- Poster Art (Cover 600x900) -->
          <div
            class="w-24 sm:w-32 aspect-[2/3] rounded-2xl overflow-hidden shadow-2xl border flex-shrink-0 hidden xs:block bg-black/40"
            style="border-color: {pColors.border}; box-shadow: 0 12px 30px -8px {pColors.glow};"
          >
            <PosterArt
              appId={game.appId}
              name={game.name}
              posterUrl={game.coverUrl}
              localCoverUrl={game.localCoverUrl}
              className="w-full h-full object-cover"
            />
          </div>

          <!-- Title/Logo & Meta Block -->
          <div class="space-y-2 min-w-0">
            <!-- Store / Running Badge -->
            <div class="flex items-center space-x-2">
              <span
                class="px-2.5 py-0.5 rounded-lg text-[10px] font-black uppercase tracking-wider text-white shadow-md"
                style="background-color: {pColors.badgeBg};"
              >
                {platformLabel}
              </span>
              {#if game.isRunning}
                <span class="px-2 py-0.5 rounded-lg text-[10px] font-bold bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 font-mono flex items-center space-x-1.5 animate-pulse">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                  <span>RUNNING</span>
                </span>
              {/if}
            </div>

            {#if (game.localLogoUrl || game.logoUrl) && !logoFailed}
              <div class="py-1">
                <img
                  src={game.localLogoUrl || game.logoUrl}
                  alt={game.name}
                  class="max-h-20 sm:max-h-24 md:max-h-28 max-w-[280px] sm:max-w-[360px] md:max-w-[420px] object-contain object-left drop-shadow-2xl"
                  on:error={() => { logoFailed = true; }}
                />
              </div>
            {:else}
              <h1 class="text-2xl sm:text-4xl font-black text-white tracking-tight drop-shadow-md truncate max-w-2xl">
                {game.name}
              </h1>
            {/if}

            <div class="flex items-center space-x-2.5 text-xs text-slate-400 font-mono">
              <span>{formatBytes(game.diskSizeBytes)}</span>
              <span class="text-slate-600">·</span>
              <span class="truncate max-w-sm">{game.runner || 'UMU'}</span>
              <span class="text-slate-600">·</span>
              <span class={game.isPatched ? 'text-emerald-400' : 'text-amber-400'}>
                {game.isPatched ? 'Goldberg Patched' : 'Setup Required'}
              </span>
              {#if game.appId && game.appId !== '0'}
                <span class="text-slate-600">·</span>
                <span class="text-slate-500">#{game.appId}</span>
              {/if}
            </div>
          </div>
        </div>

        <!-- Right: Primary CTA & Actions -->
        <div class="flex items-center space-x-3 flex-shrink-0">
          <!-- Primary Action Button (Play / Setup) -->
          <button
            bind:this={playButtonRef}
            type="button"
            on:click={handlePrimaryAction}
            class="px-6 py-3 rounded-2xl font-black text-sm transition-all flex items-center space-x-2.5 shadow-2xl active:scale-95 focus:outline-none focus:ring-2 focus:ring-white/40 {game.isRunning ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-600/30' : isReady ? 'bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 shadow-emerald-500/30' : 'bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-400 hover:to-orange-400 text-slate-950 shadow-amber-500/30'}"
          >
            {#if game.isRunning}
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
              </svg>
              <span>Stop Game</span>
            {:else if isReady}
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
              </svg>
              <span>Play Now</span>
            {:else}
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd" />
              </svg>
              <span>Setup & Play</span>
            {/if}
          </button>

          <!-- Files Button (Y) -->
          <button
            type="button"
            on:click={openDirectory}
            class="p-3 rounded-2xl bg-[#14141e]/90 hover:bg-[#1f1f2e] border border-[#2b2b3e] text-slate-300 hover:text-white transition shadow-xl flex items-center justify-center group cursor-pointer"
            title="Open Directory in File Manager (Y)"
          >
            <svg class="w-5 h-5 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </button>

          <!-- Properties Button (X) -->
          <button
            type="button"
            on:click={openProperties}
            class="p-3 rounded-2xl bg-[#14141e]/90 hover:bg-[#1f1f2e] border border-[#2b2b3e] text-slate-300 hover:text-white transition shadow-xl flex items-center justify-center group cursor-pointer"
            title="Configure Game Properties (X)"
          >
            <svg class="w-5 h-5 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Main Content & Tabs Section -->
    <div class="max-w-6xl w-full mx-auto px-6 py-4 flex-1 flex flex-col space-y-4">
      <!-- Tab Navigation Pill -->
      <div class="flex items-center space-x-2 border-b border-[#20202c] pb-2 text-xs font-bold select-none">
        <button
          type="button"
          on:click={() => { activeTab = 'overview'; }}
          class="px-3.5 py-2 rounded-xl transition-all flex items-center space-x-2 {activeTab === 'overview' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
          </svg>
          <span>Overview</span>
        </button>

        {#if isSteam}
          <button
            type="button"
            on:click={() => { activeTab = 'news'; }}
            class="px-3.5 py-2 rounded-xl transition-all flex items-center space-x-2 {activeTab === 'news' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
          >
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
            </svg>
            <span>News</span>
            {#if newsItems.length > 0}
              <span class="px-1.5 py-0.2 rounded-full bg-cyan-500/20 text-cyan-300 text-[10px] font-mono">
                {newsItems.length}
              </span>
            {/if}
          </button>
        {/if}

        <button
          type="button"
          on:click={() => { activeTab = 'integrity'; }}
          class="px-3.5 py-2 rounded-xl transition-all flex items-center space-x-2 {activeTab === 'integrity' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
          <span>Integrity</span>
        </button>
      </div>

      <!-- Tab View Content -->
      <div class="flex-1 pb-10">
        {#if activeTab === 'overview'}
          <div class="grid grid-cols-1 md:grid-cols-3 gap-5">
            <!-- Col 1 & 2: Technical Summary & Path Specs -->
            <div class="md:col-span-2 space-y-4">
              <!-- Technical Specs Card -->
              <div class="bg-[#121218] border border-[#222232] rounded-2xl p-5 shadow-xl space-y-3">
                <h3 class="font-bold text-white text-xs uppercase tracking-wider text-slate-400">
                  Execution & Environment
                </h3>

                <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  <div class="bg-[#0b0b10] border border-[#1b1b26] p-3 rounded-xl space-y-1">
                    <span class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Executable</span>
                    <p class="font-mono text-slate-200 truncate" title={game.exePath || 'Auto-detected'}>
                      {game.exePath || 'Auto-detected'}
                    </p>
                  </div>

                  <div class="bg-[#0b0b10] border border-[#1b1b26] p-3 rounded-xl space-y-1">
                    <span class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Runner</span>
                    <p class="font-mono text-slate-200 truncate">
                      {game.runner || 'UMU (umu-run)'}
                    </p>
                  </div>

                  <div class="bg-[#0b0b10] border border-[#1b1b26] p-3 rounded-xl space-y-1">
                    <span class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Emulator Mode</span>
                    <p class="font-mono text-slate-200 truncate">
                      {game.isPortable ? 'Portable (DLL)' : 'Loader (Non-destructive)'}
                    </p>
                  </div>

                  <div class="bg-[#0b0b10] border border-[#1b1b26] p-3 rounded-xl space-y-1">
                    <span class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Language</span>
                    <p class="font-mono text-slate-200 capitalize truncate">
                      {game.currentLanguage || 'Auto / Default'}
                    </p>
                  </div>
                </div>

                <!-- Game Directory Path Line -->
                <div class="bg-[#0b0b10] border border-[#1b1b26] p-3 rounded-xl space-y-1">
                  <span class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Installation Directory</span>
                  <p class="font-mono text-xs text-slate-300 select-all truncate" title={game.gameDir}>
                    {game.gameDir}
                  </p>
                </div>
              </div>

              <!-- Quick Launch Arguments -->
              {#if game.exeArgs}
                <div class="bg-[#121218] border border-[#222232] rounded-2xl p-4 shadow-xl space-y-1 text-xs">
                  <span class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Launch Flags</span>
                  <p class="font-mono text-slate-300 bg-[#0b0b10] p-2.5 rounded-xl border border-[#1c1c28]">
                    {game.exeArgs}
                  </p>
                </div>
              {/if}
            </div>

            <!-- Col 3: Sidebar Actions & Tools -->
            <div class="space-y-4">
              <div class="bg-[#121218] border border-[#222232] rounded-2xl p-5 shadow-xl space-y-3">
                <h3 class="font-bold text-white text-xs uppercase tracking-wider text-slate-400">
                  Quick Actions
                </h3>

                <div class="space-y-2 text-xs">
                  <button
                    type="button"
                    on:click={openProperties}
                    class="w-full bg-[#181824] hover:bg-[#222232] border border-[#2b2b3e] text-slate-200 p-3 rounded-xl font-bold transition flex items-center justify-between group cursor-pointer"
                  >
                    <div class="flex items-center space-x-2.5">
                      <svg class="w-4 h-4 text-slate-400 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
                      </svg>
                      <span>Properties</span>
                    </div>
                    <span class="text-slate-500 group-hover:text-white">→</span>
                  </button>

                  <button
                    type="button"
                    on:click={openSaves}
                    class="w-full bg-[#181824] hover:bg-[#222232] border border-[#2b2b3e] text-slate-200 p-3 rounded-xl font-bold transition flex items-center justify-between group cursor-pointer"
                  >
                    <div class="flex items-center space-x-2.5">
                      <svg class="w-4 h-4 text-slate-400 group-hover:text-emerald-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
                      </svg>
                      <span>Saves & Backups</span>
                    </div>
                    <span class="text-slate-500 group-hover:text-emerald-400">→</span>
                  </button>

                  <button
                    type="button"
                    on:click={() => modalStore.open('artwork-picker', { game })}
                    class="w-full bg-[#181824] hover:bg-[#222232] border border-[#2b2b3e] text-slate-200 p-3 rounded-xl font-bold transition flex items-center justify-between group cursor-pointer"
                  >
                    <div class="flex items-center space-x-2.5">
                      <svg class="w-4 h-4 text-slate-400 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                      <span>Grid Artwork</span>
                    </div>
                    <span class="text-slate-500 group-hover:text-white">→</span>
                  </button>

                  <button
                    type="button"
                    on:click={openLogs}
                    class="w-full bg-[#181824] hover:bg-[#222232] border border-[#2b2b3e] text-slate-200 p-3 rounded-xl font-bold transition flex items-center justify-between group cursor-pointer"
                  >
                    <div class="flex items-center space-x-2.5">
                      <svg class="w-4 h-4 text-slate-400 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                      <span>Launch Logs</span>
                    </div>
                    <span class="text-slate-500 group-hover:text-white">→</span>
                  </button>

                  <button
                    type="button"
                    on:click={() => modalStore.open('metadata-link', { game })}
                    class="w-full bg-[#181824] hover:bg-[#222232] border border-[#2b2b3e] text-slate-200 p-3 rounded-xl font-bold transition flex items-center justify-between group cursor-pointer"
                  >
                    <div class="flex items-center space-x-2.5">
                      <svg class="w-4 h-4 text-slate-400 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                      </svg>
                      <span>Store Metadata</span>
                    </div>
                    <span class="text-slate-500 group-hover:text-white">→</span>
                  </button>

                </div>
              </div>
            </div>
          </div>
        {:else if activeTab === 'news'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl">
            <DrawerNewsTab
              {isLoadingNews}
              {newsError}
              {newsItems}
              {expandedNewsIndex}
              onRefreshNews={() => loadNews(true)}
              onToggleExpand={toggleNewsExpand}
            />
          </div>
        {:else if activeTab === 'integrity'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl">
            <DrawerIntegrityTab
              {isCheckingIntegrity}
              {isNormalizing}
              {integrityError}
              {integrityReport}
              onRunIntegrityCheck={runIntegrityCheck}
              onUpdateGame={handleUpdateGame}
              onNormalizeDirectory={handleNormalizeDirectory}
            />
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
