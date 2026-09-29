<script>
  import { createEventDispatcher } from 'svelte';
  import { modalStore } from '../../../stores/modal.js';
  import { formatBytes, formatETA } from '../../../utils/format.js';
  import {
    getPlatformColors,
    getPlatformType,
    getMetadataInfo,
    isSteamApp,
    formatPlaytime,
    formatLastPlayed,
    formatInstallDate
  } from '../../../utils/game.js';
  import PosterArt from '../../common/PosterArt.svelte';
  import { multiseatStore, getActiveSession } from '../../../stores/multiseat.js';
  import { tasksStore, isDownloadTask } from '../../../stores/tasks.js';
  import { AppAPI } from '../../../api/app.js';
  import GameHeroBackdrop from '../../blocks/GameHeroBackdrop.svelte';
  import GameLogo from '../../blocks/GameLogo.svelte';
  import LiveCommunityBadge from '../../blocks/LiveCommunityBadge.svelte';
  import MultiseatStatusPill from '../../blocks/MultiseatStatusPill.svelte';

  export let game = null;
  export let isGameRunning = false;
  export let isTaskActive = false;
  export let activeGameTask = null;
  export let globalCfg = {};

  const dispatch = createEventDispatcher();

  let logoFailed = false;
  let heroFailed = false;
  let livePlayerCount = 0;
  let loadedPlayerAppId = '';
  let versionStatus = null;
  let loadedVersionGameDir = '';
  let showLaunchMenu = false;

  $: if (game) {
    logoFailed = false;
    heroFailed = false;
  }

  $: if (game?.appId && isSteamApp(game.appId)) {
    const aid = String(game.appId).trim();
    if (aid !== loadedPlayerAppId) {
      loadedPlayerAppId = aid;
      livePlayerCount = 0;
      fetchLivePlayers(aid);
    }
  } else if (!isSteamApp(game?.appId)) {
    livePlayerCount = 0;
    loadedPlayerAppId = '';
  }

  async function fetchLivePlayers(appId) {
    try {
      const count = await AppAPI.getGameCurrentPlayers(appId);
      if (typeof count === 'number' && count > 0) {
        livePlayerCount = count;
      }
    } catch {
      livePlayerCount = 0;
    }
  }

  $: if (game?.gameDir && isSteam) {
    const gdir = game.gameDir;
    if (gdir !== loadedVersionGameDir) {
      loadedVersionGameDir = gdir;
      versionStatus = game.versionStatus || null;
      checkGameUpdates(gdir);
    }
  } else if (!game?.gameDir || !isSteam) {
    versionStatus = null;
    loadedVersionGameDir = '';
  }

  async function checkGameUpdates(gameDir) {
    try {
      const res = await AppAPI.checkGameUpdates(gameDir);
      if (res && res.checked) {
        versionStatus = res;
      }
    } catch {
      // Quiet fallback
    }
  }

  $: isCoop = $multiseatStore.isCoopEnabled;
  $: multiseatSession = game?.gameDir ? getActiveSession($multiseatStore.activeSessions, game.gameDir) : null;
  $: isMultiseatActive = Boolean(multiseatSession?.is_running);

  $: pColors = game ? getPlatformColors(game) : { border: '#1a9fff', badgeBg: '#1a9fff', label: 'WIN64', glow: 'rgba(26,159,255,0.4)' };
  $: platformLabel = game ? getPlatformType(game) : 'Windows';
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };
  $: isInstalled = game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false;
  $: isStub = game ? (game.isStub ?? (!isInstalled && Boolean(game.gameDir))) : false;
  $: isVirtual = game ? (game.isVirtual ?? (!isInstalled && !isStub && !game.gameDir)) : false;
  $: isReady = game ? (game.isReady || game.isPatched || !!game.exePath) : false;
  $: isSteam = game ? isSteamApp(game.appId) : false;
  $: heroBackdropUrl = game ? (game.localHeroUrl || game.heroUrl || game.localBannerUrl || game.bannerUrl || (isSteam ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${game.appId}/library_hero.jpg` : '') || game.localCoverUrl || game.coverUrl || '') : '';

  $: seatCount = ($multiseatStore.staging?.seats?.length) || 2;
  $: isTaskPaused = activeGameTask?.status === 'paused';
  $: isTaskQueued = activeGameTask?.status === 'queued' || activeGameTask?.status === 'pending';
  $: isPausableTask = Boolean(activeGameTask && (isDownloadTask(activeGameTask) || ((activeGameTask.capabilities || 0) & 2) !== 0 || isTaskPaused));
  $: taskPercent = typeof activeGameTask?.percent === 'number'
    ? Math.min(100, Math.max(0, Math.round(activeGameTask.percent)))
    : 0;

  $: primaryButtonLabel = isTaskActive
    ? (isTaskPaused
        ? (typeof activeGameTask?.percent === 'number' && activeGameTask.percent > 0
            ? `Resume (${Math.round(activeGameTask.percent)}%)`
            : 'Resume')
        : isTaskQueued
        ? (isPausableTask ? 'Queued (Pause)' : 'Queued...')
        : isPausableTask
        ? (typeof activeGameTask?.percent === 'number' && activeGameTask.percent > 0
            ? `Pause (${Math.round(activeGameTask.percent)}%)`
            : 'Pause')
        : 'Task in Progress...')
    : isGameRunning
    ? (isMultiseatActive ? 'Stop Session' : 'Stop Game')
    : !isInstalled
    ? 'Download'
    : isReady
    ? (isCoop ? `Play (${seatCount}P)` : 'Play Now')
    : (isCoop ? `Setup & Play (${seatCount}P)` : 'Setup & Play');

  function handleReturnToShelf() {
    dispatch('returnToShelf');
  }

  function handlePrimaryAction() {
    dispatch('primaryAction');
  }

  function handleOpenDirectory() {
    dispatch('openDirectory');
  }

  function handleTrash() {
    dispatch('trash');
  }
</script>

<svelte:window on:click={() => { showLaunchMenu = false; }} on:keydown={(e) => { if (e.key === 'Escape') showLaunchMenu = false; }} />

<!-- Hero Banner Background & Vignette -->
<div class="relative w-full h-96 sm:h-[420px] flex-shrink-0 overflow-hidden select-none">
  <!-- Hero Backdrop Component -->
  <div class="absolute inset-0">
    <GameHeroBackdrop
      heroUrl={heroBackdropUrl}
      localHeroUrl={game?.localHeroUrl}
      glowColor={pColors.glow || 'rgba(255,255,255,0.05)'}
      accentColor={pColors.border || 'rgba(255,255,255,0.03)'}
    />
  </div>

  <!-- Top Nav Breadcrumb -->
  <div class="absolute top-4 left-6 z-20 flex items-center space-x-3">
    <button
      type="button"
      on:click={handleReturnToShelf}
      class="flex items-center space-x-2 bg-surface-header/90 hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white px-3.5 py-1.5 rounded-xl text-xs font-bold transition shadow-lg backdrop-blur-md group cursor-pointer"
      title="Back to Shelf (Esc / B)"
    >
      <span class="text-neutral-400 group-hover:text-white transition-colors">←</span>
      <span>Shelf</span>
      <span class="text-3xs text-neutral-400 font-mono px-1.5 py-0.5 rounded bg-black/40 border border-surface-border">B / Esc</span>
    </button>
  </div>

  <!-- Hero Content Container -->
  {#if game}
    <div class="absolute bottom-6 left-6 right-6 z-20 flex flex-col sm:flex-row sm:items-end justify-between gap-6 max-w-6xl mx-auto">
      <!-- Left: Game Poster + Title/Logo + Badges -->
      <div class="flex items-end space-x-5 min-w-0">
        <!-- Poster Art (Cover 600x900) -->
        <div
          class="w-24 sm:w-32 aspect-[2/3] rounded-2xl overflow-hidden shadow-2xl border flex-shrink-0 hidden xs:block bg-black/40 group/poster relative cursor-pointer"
          style="border-color: {pColors.border}; box-shadow: 0 12px 30px -8px {pColors.glow};"
          on:click={() => modalStore.open('artwork-picker', { game, initialTab: 'cover' })}
          title="Change Cover Artwork"
          role="button"
          tabindex="0"
          on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); modalStore.open('artwork-picker', { game, initialTab: 'cover' }); } }}
        >
          <PosterArt
            appId={game.appId}
            name={game.name}
            posterUrl={game.coverUrl}
            localCoverUrl={game.localCoverUrl}
            className="w-full h-full object-cover group-hover/poster:scale-105 transition-transform duration-300"
          />
          <div class="absolute inset-0 bg-black/60 opacity-0 group-hover/poster:opacity-100 transition-opacity flex flex-col items-center justify-center gap-1 text-white p-1 text-center backdrop-blur-[2px]">
            <svg class="w-4 h-4 text-neutral-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span class="text-3xs font-bold">Change Art</span>
          </div>
        </div>

        <!-- Title/Logo & Meta Block -->
        <div class="space-y-2 min-w-0">
          <!-- Store / Running Badge -->
          <div class="flex items-center space-x-2 flex-wrap gap-y-1">
            <span
              class="px-2.5 py-0.5 rounded-lg text-3xs font-black uppercase tracking-wider text-neutral-200 bg-surface-card border border-surface-border shadow-md"
            >
              {platformLabel} Source
            </span>
            {#if metaInfo.provider && metaInfo.provider !== 'None'}
              <span class="px-2.5 py-0.5 rounded-lg text-3xs font-mono font-bold bg-surface-header/90 border border-surface-border text-neutral-300 shadow-md">
                Meta: {metaInfo.provider}
              </span>
            {/if}
            {#if isGameRunning}
              <span class="px-2 py-0.5 rounded-lg text-3xs font-bold bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 font-mono flex items-center space-x-1.5 animate-pulse">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                <span>RUNNING</span>
              </span>
            {:else if isTaskPaused}
              <span class="px-2 py-0.5 rounded-lg text-3xs font-bold bg-amber-500/20 border border-amber-500/40 text-amber-300 font-mono flex items-center space-x-1.5">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                <span>DOWNLOAD PAUSED</span>
              </span>
            {:else if isTaskActive}
              <span class="px-2 py-0.5 rounded-lg text-3xs font-bold bg-sky-500/20 border border-sky-500/40 text-sky-300 font-mono flex items-center space-x-1.5 animate-pulse">
                <span class="w-1.5 h-1.5 rounded-full bg-sky-400"></span>
                <span>{isDownloadTask(activeGameTask) ? 'DOWNLOADING' : 'TASK ACTIVE'}</span>
              </span>
            {/if}
          </div>

          <!-- Title & Logo Component -->
          <GameLogo
            logoUrl={game.localLogoUrl || game.logoUrl}
            title={game.name}
          />

          <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-neutral-400 font-mono">
            {#if isInstalled && game.diskSizeBytes > 0}
              <span>{formatBytes(game.diskSizeBytes)}</span>
              <span class="text-neutral-500">·</span>
            {:else if isStub}
              <span class="text-neutral-300 font-bold">In Library (Uninstalled)</span>
              <span class="text-neutral-500">·</span>
            {:else if isVirtual}
              <span class="text-neutral-300 font-bold">Online Store Title</span>
              <span class="text-neutral-500">·</span>
            {/if}
            <span class="truncate max-w-sm">{game.platform === 'linux' ? 'Linux Native' : (game.runner || globalCfg?.default_runner || 'UMU')}</span>
            {#if isInstalled && !game.isReady}
              <span class="text-neutral-500">·</span>
              <span class="text-amber-400">Setup Required</span>
            {/if}
            {#if game.playtimeSeconds > 0}
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-200 font-bold flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{formatPlaytime(game.playtimeSeconds)}</span>
              </span>
            {/if}
            {#if game.lastPlayed > 0}
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-400">Played {formatLastPlayed(game.lastPlayed)}</span>
            {/if}
            {#if game.installedAt > 0}
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-400">Installed {formatInstallDate(game.installedAt)}</span>
            {/if}
            {#if game.appId && game.appId !== '0'}
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-500">#{game.appId}</span>
            {/if}
            {#if game.appId && isSteamApp(game.appId)}
              <span class="text-neutral-500">·</span>
              <button
                type="button"
                on:click={() => { dispatch('openBranchTab'); dispatch('openBetasTab'); }}
                class="hover:text-neutral-200 transition-colors flex items-center gap-1 cursor-pointer"
                title="Manage Steam Beta branch"
              >
                <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v10M8 7a3 3 0 100-6 3 3 0 000 6zm0 10a3 3 0 100 6 3 3 0 000-6zm8-6a3 3 0 100-6 3 3 0 000 6zm-8 4h3a5 5 0 005-5V7" />
                </svg>
                <span>Branch: {game.branch || 'public'}</span>
              </button>
            {/if}
            {#if livePlayerCount > 0}
              <span class="text-neutral-500">·</span>
              <LiveCommunityBadge playerCount={livePlayerCount} label="playing" />
            {/if}
            {#if versionStatus && !versionStatus.is_up_to_date}
              <span class="text-neutral-500">·</span>
              <span
                class="px-2 py-0.5 rounded-lg text-3xs font-bold bg-amber-500/15 border border-amber-500/30 text-amber-300 font-mono flex items-center space-x-1.5"
                title={versionStatus.message || versionStatus.latest_patch_title || 'Update available'}
              >
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                <span>UPDATE AVAILABLE</span>
              </span>
            {/if}
          </div>
        </div>
      </div>

      <!-- Right: Primary CTA & Actions -->
      <div class="flex flex-col items-start sm:items-end space-y-2.5 flex-shrink-0">
        <div class="flex items-center space-x-3">
        {#if isVirtual}
          <button
            type="button"
            on:click={() => dispatch('addToLibrary')}
            class="px-4 py-3 rounded-2xl font-bold text-xs bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white transition flex items-center space-x-2 shadow-xl cursor-pointer"
            title="Add skeleton metadata stub to your library without downloading files"
          >
            <svg class="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            <span>Add to Library</span>
          </button>
        {/if}

        <!-- Split Launch Action Container -->
        <div class="relative flex flex-col items-start sm:items-end">
          {#if isTaskActive}
            <!-- Floating Micro-Telemetry above Button (zero layout displacement) -->
            <div class="absolute bottom-full mb-2 left-0 sm:left-auto sm:right-0 flex items-center space-x-2 text-3xs font-mono text-neutral-300 bg-surface-header/90 backdrop-blur-md px-2.5 py-1 rounded-xl border border-surface-border shadow-lg pointer-events-none select-none whitespace-nowrap animate-fade-in z-20">
              {#if isTaskPaused}
                <span class="w-1.5 h-1.5 rounded-full bg-neutral-400"></span>
                <span class="text-neutral-300 font-medium">Download paused</span>
              {:else}
                <span class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-pulse"></span>
                {#if activeGameTask?.speedBps > 0}
                  <span class="text-white font-bold">{formatBytes(activeGameTask.speedBps)}/s</span>
                {/if}
                {#if activeGameTask?.etaSeconds > 0}
                  <span class="text-neutral-400">· ETA: {formatETA(activeGameTask.etaSeconds)}</span>
                {/if}
              {/if}
              {#if activeGameTask?.totalBytes > 0}
                <span class="text-neutral-400 font-medium">({formatBytes(activeGameTask.currentBytes || 0)} / {formatBytes(activeGameTask.totalBytes)})</span>
              {/if}
            </div>
          {/if}

          <div class="flex items-center rounded-2xl shadow-2xl overflow-hidden {isTaskActive ? 'bg-surface-card border border-neutral-700 min-w-[170px] sm:min-w-[200px]' : isGameRunning ? 'bg-rose-600' : 'bg-white'}">
            <button
              type="button"
              on:click={handlePrimaryAction}
              disabled={isTaskActive && !isPausableTask}
              class="relative overflow-hidden px-5 py-3 font-black text-sm transition-all flex items-center justify-center space-x-2 active:scale-95 focus:outline-none cursor-pointer disabled:cursor-not-allowed flex-1 {isTaskActive ? 'text-white' : isGameRunning ? 'hover:bg-rose-500 text-white' : 'hover:bg-neutral-200 text-neutral-950'}"
            >
              {#if isTaskActive}
                <!-- Animated / Frozen Progress Fill Bar Background -->
                <div
                  class="absolute inset-y-0 left-0 pointer-events-none transition-[width] duration-300 ease-out {isTaskPaused ? 'bg-neutral-700' : 'bg-sky-600'}"
                  style="width: {taskPercent}%"
                  data-testid="hero-progress-bar"
                >
                  {#if !isTaskPaused}
                    <!-- Moving diagonal stripes overlay -->
                    <div class="absolute inset-0 opacity-25 bg-[linear-gradient(45deg,rgba(255,255,255,0.35)_25%,transparent_25%,transparent_50%,rgba(255,255,255,0.35)_50%,rgba(255,255,255,0.35)_75%,transparent_75%,transparent)] bg-[length:20px_20px] animate-progress-stripes"></div>
                  {/if}
                </div>
              {/if}

              <!-- Foreground Icon & Text -->
              <div class="relative z-10 flex items-center space-x-2">
                {#if isTaskPaused}
                  <svg class="w-4 h-4 fill-current text-neutral-300 drop-shadow-sm" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
                  </svg>
                {:else if isTaskActive}
                  {#if isPausableTask}
                    <svg class="w-4 h-4 fill-current text-white drop-shadow-sm" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                    </svg>
                  {:else}
                    <svg class="animate-spin w-4 h-4 text-white drop-shadow-sm" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                    </svg>
                  {/if}
                {:else if isGameRunning}
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
                  </svg>
                {:else if !isInstalled}
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                  </svg>
                {:else if isReady}
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
                  </svg>
                {:else}
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd" />
                  </svg>
                {/if}
                <span class="drop-shadow-sm">{primaryButtonLabel}</span>
              </div>
            </button>

            {#if isTaskActive && isPausableTask && activeGameTask?.id}
              <!-- Cancel Transfer Button -->
              <button
                type="button"
                on:click|stopPropagation={() => tasksStore.cancel(activeGameTask.id)}
                class="relative z-10 px-3 py-3.5 bg-neutral-900/80 hover:bg-rose-950 text-neutral-400 hover:text-rose-200 border-l border-neutral-700/80 transition-colors cursor-pointer flex items-center justify-center self-stretch"
                title="Cancel Task"
                aria-label="Cancel Task"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            {/if}

            {#if isInstalled && isReady && !isTaskActive && !isGameRunning}
              <!-- Split Menu Trigger Button -->
              <button
                type="button"
                on:click|stopPropagation={() => { showLaunchMenu = !showLaunchMenu; }}
                class="px-2.5 py-3.5 bg-neutral-100 hover:bg-neutral-200 text-neutral-800 border-l border-neutral-300 transition-colors cursor-pointer flex items-center justify-center"
                title="Launch Options (Single Player / Co-Op)"
                aria-label="Launch Options"
              >
                <svg class="w-3.5 h-3.5 transition-transform duration-200 {showLaunchMenu ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M19 9l-7 7-7-7" />
                </svg>
              </button>
            {/if}
          </div>

          <!-- Launch Options Dropdown Popover -->
          {#if showLaunchMenu}
            <div
              class="absolute right-0 bottom-full mb-2 w-52 bg-[#121215]/95 backdrop-blur-md border border-[#27272a] rounded-xl shadow-2xl p-1.5 z-50 animate-fade-in space-y-1 text-left"
              on:click|stopPropagation
              on:keydown|stopPropagation
              role="menu"
              tabindex="-1"
            >
              <div class="px-2.5 py-1 text-4xs font-mono font-bold text-neutral-400 uppercase tracking-wider border-b border-surface-border mb-1">
                Launch Mode
              </div>

              <!-- Mode 1: Single Player -->
              <button
                type="button"
                role="menuitem"
                on:click={() => { if (isCoop) multiseatStore.toggleCoop(); showLaunchMenu = false; }}
                class="w-full px-2.5 py-1.5 rounded-lg flex items-center justify-between text-xs transition cursor-pointer {!isCoop ? 'bg-surface-cardHover text-white font-bold' : 'text-neutral-300 hover:bg-surface-input hover:text-white'}"
              >
                <span>Single Player</span>
                {#if !isCoop}
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                {/if}
              </button>

              <!-- Mode 2: Co-op -->
              <button
                type="button"
                role="menuitem"
                on:click={() => { if (!isCoop) multiseatStore.toggleCoop(); showLaunchMenu = false; }}
                class="w-full px-2.5 py-1.5 rounded-lg flex items-center justify-between text-xs transition cursor-pointer {isCoop ? 'bg-surface-cardHover text-white font-bold' : 'text-neutral-300 hover:bg-surface-input hover:text-white'}"
              >
                <span>Co-Op ({seatCount}P)</span>
                {#if isCoop}
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                {/if}
              </button>
            </div>
          {/if}
        </div>

        {#if !isVirtual && game.gameDir}
          <!-- Files Button (Y) -->
          <button
            type="button"
            on:click={handleOpenDirectory}
            class="px-4 py-3 rounded-2xl bg-surface-header/90 hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white transition shadow-xl flex items-center space-x-2 group cursor-pointer"
            title="Open Directory in File Manager (Y)"
          >
            <svg class="w-4 h-4 text-neutral-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span class="text-xs font-bold">Files</span>
          </button>

          <!-- Trash / Delete Game Button -->
          <button
            type="button"
            on:click={handleTrash}
            disabled={isGameRunning}
            class="p-3 rounded-2xl bg-surface-header/90 hover:bg-surface-card border border-surface-border hover:border-rose-900/50 text-neutral-300 hover:text-rose-400 transition shadow-xl flex items-center justify-center group cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            title="Move Game to Trash"
            aria-label="Move Game to Trash"
          >
            <svg class="w-4 h-4 text-neutral-400 group-hover:text-rose-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  @keyframes progressStripes {
    0% {
      background-position: 0 0;
    }
    100% {
      background-position: 20px 0;
    }
  }
  :global(.animate-progress-stripes) {
    animation: progressStripes 1s linear infinite;
  }
</style>
