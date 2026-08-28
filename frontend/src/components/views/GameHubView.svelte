<script>
  import { onMount, tick } from 'svelte';
  import {
    selectedGame,
    activeView,
    queueApplyGame,
    setGameRunning,
    runningGames,
    refreshGame
  } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { drawerDraft } from '../../stores/drawerDraft.js';
  import { configStore } from '../../stores/config.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';
  import {
    getPlatformColors,
    getPlatformType,
    getMetadataInfo,
    isSteamApp,
    hasMetadataMatch,
    formatPlaytime,
    formatLastPlayed
  } from '../../utils/game.js';
  import PosterArt from '../common/PosterArt.svelte';
  import DrawerAboutTab from '../stage/drawer/DrawerAboutTab.svelte';
  import DrawerResourcesTab from '../stage/drawer/DrawerResourcesTab.svelte';
  import DrawerAchievementsTab from '../stage/drawer/DrawerAchievementsTab.svelte';
  import DrawerNewsTab from '../stage/drawer/DrawerNewsTab.svelte';
  import DrawerGeneralTab from '../stage/drawer/DrawerGeneralTab.svelte';
  import DrawerIntegrationsTab from '../stage/drawer/DrawerIntegrationsTab.svelte';
  import DrawerEmuTab from '../stage/drawer/DrawerEmuTab.svelte';
  import DrawerEnvTab from '../stage/drawer/DrawerEnvTab.svelte';
  import DrawerSavesTab from '../stage/drawer/DrawerSavesTab.svelte';
  import DrawerDLCTab from '../stage/drawer/DrawerDLCTab.svelte';
  import DrawerIntegrityTab from '../stage/drawer/DrawerIntegrityTab.svelte';
  import DrawerLogsTab from '../stage/drawer/DrawerLogsTab.svelte';

  let playButtonRef;

  // 5 Primary Consolidated Tabs: 'overview' | 'community' | 'settings' | 'saves' | 'files'
  let activeTab = 'overview';

  // Sub-Navigation Tabs
  let activeCommunitySubTab = 'achievements'; // 'achievements' | 'news'
  let activeSettingsSubTab = 'general'; // 'general' | 'runner' | 'emulation' | 'env'
  let activeFilesSubTab = 'depots'; // 'depots' | 'integrity' | 'logs'

  let logoFailed = false;
  let heroFailed = false;
  let globalCfg = {};
  let isSaving = false;

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
  $: isGameRunning = Boolean(game?.gameDir && $runningGames.has(game.gameDir));
  $: pColors = game ? getPlatformColors(game) : { border: '#1a9fff', badgeBg: '#1a9fff', label: 'WIN64', glow: 'rgba(26,159,255,0.4)' };
  $: platformLabel = game ? getPlatformType(game) : 'Windows';
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };
  $: isReady = game ? (game.isReady || game.isPatched || !!game.exePath) : false;
  $: isSteam = game ? isSteamApp(game.appId) : false;
  $: hasMeta = game ? hasMetadataMatch(game) : false;
  $: heroBackdropUrl = game ? (game.localHeroUrl || game.heroUrl || game.localBannerUrl || game.bannerUrl || (isSteam ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${game.appId}/library_hero.jpg` : '') || game.localCoverUrl || game.coverUrl || '') : '';

  $: candidates = (game?.launchCandidates || [])
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

  $: languageOptions = (game?.supportedLanguages || [
    'english', 'french', 'italian', 'german', 'spanish', 'japanese', 'schinese', 'russian', 'brazilian', 'koreana', 'polish', 'turkish'
  ]);

  $: if (game && globalCfg) {
    drawerDraft.init(game, globalCfg);
  }

  onMount(async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const tabParam = urlParams.get('tab');
    const isDrawerOpen = urlParams.get('drawer') === '1' || urlParams.get('properties') === '1';

    if (isDrawerOpen) {
      activeTab = 'settings';
      if (tabParam === 'emu' || tabParam === 'emulation') activeSettingsSubTab = 'emulation';
      else if (tabParam === 'runner' || tabParam === 'compat') activeSettingsSubTab = 'runner';
      else if (tabParam === 'env') activeSettingsSubTab = 'env';
      else activeSettingsSubTab = 'general';
    } else if (tabParam) {
      if (tabParam === 'overview' || tabParam === 'about' || tabParam === 'resources') {
        activeTab = 'overview';
      } else if (tabParam === 'community' || tabParam === 'achievements') {
        activeTab = 'community';
        activeCommunitySubTab = 'achievements';
      } else if (tabParam === 'news') {
        activeTab = 'community';
        activeCommunitySubTab = 'news';
      } else if (tabParam === 'config' || tabParam === 'settings') {
        activeTab = 'settings';
        activeSettingsSubTab = 'general';
      } else if (tabParam === 'emu' || tabParam === 'emulation') {
        activeTab = 'settings';
        activeSettingsSubTab = 'emulation';
      } else if (tabParam === 'runner' || tabParam === 'compat') {
        activeTab = 'settings';
        activeSettingsSubTab = 'runner';
      } else if (tabParam === 'env') {
        activeTab = 'settings';
        activeSettingsSubTab = 'env';
      } else if (tabParam === 'saves') {
        activeTab = 'saves';
      } else if (tabParam === 'files' || tabParam === 'dlc' || tabParam === 'depots') {
        activeTab = 'files';
        activeFilesSubTab = 'depots';
      } else if (tabParam === 'integrity') {
        activeTab = 'files';
        activeFilesSubTab = 'integrity';
      } else if (tabParam === 'logs') {
        activeTab = 'files';
        activeFilesSubTab = 'logs';
      }
    } else {
      activeTab = 'overview';
    }

    const cfg = await configStore.load();
    if (cfg) globalCfg = cfg;
    if (game) {
      drawerDraft.init(game, globalCfg);
    }

    await tick();
    if (playButtonRef) {
      playButtonRef.focus();
    }
  });

  async function handleSave() {
    if (!game || isSaving) return;
    isSaving = true;
    try {
      const success = await drawerDraft.save(game);
      if (success) {
        await refreshGame(game.gameDir);
      }
    } finally {
      isSaving = false;
    }
  }

  async function handleResetDefaults() {
    if (!game) return;
    const ok = await drawerDraft.resetDefaults(game);
    if (ok) {
      await refreshGame(game.gameDir);
    }
  }

  function handleRollback() {
    if (!game) return;
    drawerDraft.rollback(game);
    refreshGame(game.gameDir);
  }

  function returnToShelf() {
    $activeView = 'library';
  }

  function handlePrimaryAction() {
    if (!game) return;

    if (isGameRunning) {
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

  function openSettingsTab(subTab = 'general') {
    activeTab = 'settings';
    activeSettingsSubTab = subTab;
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
      const total = rep?.official_count ?? rep?.OfficialCount ?? 0;
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

  const primaryTabs = ['overview', 'community', 'settings', 'saves', 'files'];

  function cycleTab(dir) {
    const idx = primaryTabs.indexOf(activeTab);
    let next = idx + dir;
    if (next < 0) next = primaryTabs.length - 1;
    if (next >= primaryTabs.length) next = 0;
    activeTab = primaryTabs[next];
  }

  $: if (activeTab === 'community' && activeCommunitySubTab === 'news' && loadedNewsAppID !== game?.appId && !isLoadingNews && game?.appId) {
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
          class="flex items-center space-x-2 bg-[#12121a]/80 hover:bg-[#1e1e2c] border border-[#27273a] text-slate-300 hover:text-white px-3.5 py-1.5 rounded-xl text-xs font-bold transition shadow-lg backdrop-blur-md group cursor-pointer"
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
            <div class="flex items-center space-x-2 flex-wrap gap-y-1">
              <span
                class="px-2.5 py-0.5 rounded-lg text-[10px] font-black uppercase tracking-wider text-white shadow-md"
                style="background-color: {pColors.badgeBg};"
              >
                {platformLabel} Source
              </span>
              {#if metaInfo.provider && metaInfo.provider !== 'None'}
                <span class="px-2.5 py-0.5 rounded-lg text-[10px] font-mono font-bold bg-[#141420] border border-[#2c2c42] text-sky-300 shadow-md">
                  Meta: {metaInfo.provider}{metaInfo.id ? ` #${metaInfo.id}` : ''}
                </span>
              {/if}
              {#if isGameRunning}
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

            <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-slate-400 font-mono">
              <span>{formatBytes(game.diskSizeBytes)}</span>
              <span class="text-slate-600">·</span>
              <span class="truncate max-w-sm">{game.runner || 'UMU'}</span>
              {#if !game.isReady}
                <span class="text-slate-600">·</span>
                <span class="text-amber-400">Setup Required</span>
              {/if}
              {#if game.playtimeSeconds > 0}
                <span class="text-slate-600">·</span>
                <span class="text-sky-300 font-bold flex items-center gap-1.5">
                  <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>{formatPlaytime(game.playtimeSeconds)}</span>
                </span>
              {/if}
              {#if game.lastPlayed > 0}
                <span class="text-slate-600">·</span>
                <span class="text-slate-400">Played {formatLastPlayed(game.lastPlayed)}</span>
              {/if}
              {#if game.appId && game.appId !== '0'}
                <span class="text-slate-600">·</span>
                <span class="text-slate-500">#{game.appId}</span>
              {/if}
            </div>
          </div>
        </div>

        <!-- Right: Primary CTA & Actions -->
        <div class="flex items-center space-x-3 flex-shrink-0">
          <!-- Primary Action Button (Play / Stop / Setup) -->
          <button
            bind:this={playButtonRef}
            type="button"
            on:click={handlePrimaryAction}
            class="px-6 py-3 rounded-2xl font-black text-sm transition-all flex items-center space-x-2.5 shadow-2xl active:scale-95 focus:outline-none focus:ring-2 focus:ring-white/40 cursor-pointer {isGameRunning ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-600/30' : isReady ? 'bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 shadow-emerald-500/30' : 'bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-400 hover:to-orange-400 text-slate-950 shadow-amber-500/30'}"
          >
            {#if isGameRunning}
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
        </div>
      </div>
    </div>

    <!-- Main Content & 5-Tab Section -->
    <div class="max-w-6xl w-full mx-auto px-6 py-4 flex-1 flex flex-col space-y-4">
      <!-- 5 Primary Navigation Tabs -->
      <div class="flex items-center space-x-2 border-b border-[#20202c] pb-2 text-xs font-bold select-none overflow-x-auto custom-scrollbar">
        <!-- 1. Overview Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'overview'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'overview' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Overview</span>
        </button>

        <!-- 2. Community Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'community'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'community' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
          </svg>
          <span>Community</span>
        </button>

        <!-- 3. Settings Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'settings'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'settings' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          <span>Settings</span>
        </button>

        <!-- 4. Saves Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'saves'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'saves' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
          </svg>
          <span>Saves</span>
        </button>

        <!-- 5. Files & Diagnostics Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'files'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'files' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
        <!-- ================= 1. OVERVIEW TAB ================= -->
        {#if activeTab === 'overview'}
          <div class="space-y-6">
            <!-- Game Summary Synopsis & Facts -->
            <DrawerAboutTab {game} />

            <!-- Categorized Technical & Community Resources Grid -->
            <DrawerResourcesTab {game} />
          </div>

        <!-- ================= 2. COMMUNITY TAB ================= -->
        {:else if activeTab === 'community'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl space-y-6">
            <!-- Community Sub-Pills -->
            <div class="flex items-center space-x-2 bg-[#0c0c10] p-1.5 rounded-xl border border-[#20202e] w-fit">
              <button
                type="button"
                on:click={() => { activeCommunitySubTab = 'achievements'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeCommunitySubTab === 'achievements' ? 'bg-[#202030] text-amber-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
                <span>Achievements</span>
              </button>

              {#if isSteam}
                <button
                  type="button"
                  on:click={() => { activeCommunitySubTab = 'news'; }}
                  class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeCommunitySubTab === 'news' ? 'bg-[#202030] text-sky-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
                  </svg>
                  <span>News & Updates</span>
                  {#if newsItems.length > 0}
                    <span class="px-1.5 py-0.2 rounded-full bg-cyan-500/20 text-cyan-300 text-[10px] font-mono">
                      {newsItems.length}
                    </span>
                  {/if}
                </button>
              {/if}
            </div>

            {#if activeCommunitySubTab === 'achievements'}
              <DrawerAchievementsTab gameDir={game.gameDir} />
            {:else if activeCommunitySubTab === 'news'}
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

        <!-- ================= 3. SETTINGS TAB ================= -->
        {:else if activeTab === 'settings'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl space-y-6">
            <!-- Settings Sub-Pills -->
            <div class="flex items-center space-x-2 bg-[#0c0c10] p-1.5 rounded-xl border border-[#20202e] flex-wrap gap-y-1">
              <button
                type="button"
                on:click={() => { activeSettingsSubTab = 'general'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeSettingsSubTab === 'general' ? 'bg-[#202030] text-sky-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
                </svg>
                <span>Launch & Identity</span>
              </button>

              <button
                type="button"
                on:click={() => { activeSettingsSubTab = 'runner'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeSettingsSubTab === 'runner' ? 'bg-[#202030] text-indigo-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                </svg>
                <span>Runner & Compatibility</span>
              </button>

              <button
                type="button"
                on:click={() => { activeSettingsSubTab = 'emulation'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeSettingsSubTab === 'emulation' ? 'bg-[#202030] text-amber-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span>Emulation & DRM</span>
              </button>

              <button
                type="button"
                on:click={() => { activeSettingsSubTab = 'env'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeSettingsSubTab === 'env' ? 'bg-[#202030] text-emerald-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span>Environment</span>
              </button>
            </div>

            <!-- Settings Sub-View Content -->
            <div class="space-y-4">
              {#if activeSettingsSubTab === 'general'}
                <DrawerGeneralTab
                  {game}
                  {candidates}
                  onUpdateGame={handleUpdateGame}
                />
              {:else if activeSettingsSubTab === 'runner'}
                <DrawerIntegrationsTab
                  {game}
                  {globalCfg}
                />
              {:else if activeSettingsSubTab === 'emulation'}
                <DrawerEmuTab
                  {game}
                  {globalCfg}
                  {languageOptions}
                />
              {:else if activeSettingsSubTab === 'env'}
                <DrawerEnvTab
                  {globalCfg}
                />
              {/if}
            </div>

            <!-- Unified Save / Reset Actions Bar -->
            <div class="pt-4 border-t border-[#20202c] flex items-center justify-between gap-3 flex-wrap">
              <div class="flex items-center space-x-2">
                <button
                  type="button"
                  on:click={handleResetDefaults}
                  class="bg-[#161622] hover:bg-[#202030] text-slate-300 px-4 py-2 rounded-xl text-xs font-bold border border-[#28283c] transition-colors cursor-pointer"
                >
                  Reset Defaults
                </button>
                <button
                  type="button"
                  on:click={handleRollback}
                  class="bg-[#1a1418] hover:bg-[#281c22] text-rose-300 px-4 py-2 rounded-xl text-xs font-bold border border-rose-900/40 transition-colors cursor-pointer"
                >
                  Rollback Setup
                </button>
              </div>

              <button
                type="button"
                on:click={handleSave}
                disabled={isSaving}
                class="bg-gradient-to-r from-sky-500 to-blue-600 hover:from-sky-400 hover:to-blue-500 text-white px-6 py-2 rounded-xl text-xs font-bold shadow-lg shadow-sky-950/40 transition-all flex items-center space-x-2 disabled:opacity-50 cursor-pointer"
              >
                {#if isSaving}
                  <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span>Saving...</span>
                {:else}
                  <span>Save Configuration</span>
                {/if}
              </button>
            </div>
          </div>

        <!-- ================= 4. SAVES TAB ================= -->
        {:else if activeTab === 'saves'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl">
            <DrawerSavesTab {game} />
          </div>

        <!-- ================= 5. FILES & DIAGNOSTICS TAB ================= -->
        {:else if activeTab === 'files'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl space-y-6">
            <!-- Files Sub-Pills -->
            <div class="flex items-center space-x-2 bg-[#0c0c10] p-1.5 rounded-xl border border-[#20202e] flex-wrap gap-y-1">
              <button
                type="button"
                on:click={() => { activeFilesSubTab = 'depots'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'depots' ? 'bg-[#202030] text-purple-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                </svg>
                <span>DLC & Depots</span>
              </button>

              <button
                type="button"
                on:click={() => { activeFilesSubTab = 'integrity'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'integrity' ? 'bg-[#202030] text-cyan-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
                <span>Integrity & Repair</span>
              </button>

              <button
                type="button"
                on:click={() => { activeFilesSubTab = 'logs'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'logs' ? 'bg-[#202030] text-slate-200 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span>Execution Logs</span>
                {#if isGameRunning}
                  <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Game currently executing"></span>
                {/if}
              </button>
            </div>

            <!-- Files Sub-View Content -->
            {#if activeFilesSubTab === 'depots'}
              <DrawerDLCTab {game} />
            {:else if activeFilesSubTab === 'integrity'}
              <DrawerIntegrityTab
                {isCheckingIntegrity}
                {isNormalizing}
                {integrityError}
                {integrityReport}
                onRunIntegrityCheck={runIntegrityCheck}
                onUpdateGame={handleUpdateGame}
                onNormalizeDirectory={handleNormalizeDirectory}
              />
            {:else if activeFilesSubTab === 'logs'}
              <DrawerLogsTab {game} />
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
