<script>
  import { createEventDispatcher } from 'svelte';
  import { modalStore } from '../../../stores/modal.js';
  import { formatBytes } from '../../../utils/format.js';
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

  $: primaryButtonLabel = isTaskActive
    ? (activeGameTask?.status === 'queued'
        ? 'Queued...'
        : (activeGameTask?.type === 'download' || activeGameTask?.operation === 'download')
        ? (typeof activeGameTask?.percent === 'number' && activeGameTask.percent > 0
            ? `Downloading... ${activeGameTask.percent}%`
            : 'Downloading...')
        : 'Task in Progress...')
    : isGameRunning
    ? (isMultiseatActive ? 'Stop Session' : 'Stop Game')
    : !isInstalled
    ? 'Download'
    : isReady
    ? (isCoop ? 'Play Now (Co-Op)' : 'Play Now')
    : 'Setup & Play';

  function handleReturnToShelf() {
    dispatch('returnToShelf');
  }

  function handlePrimaryAction() {
    dispatch('primaryAction');
  }

  function handleOpenDirectory() {
    dispatch('openDirectory');
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
            <span class="truncate max-w-sm">{game.runner || globalCfg?.default_runner || 'UMU'}</span>
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
        {#if isMultiseatActive && multiseatSession}
          <div class="px-3 py-1.5 bg-neutral-900/90 border border-neutral-700 rounded-xl flex items-center space-x-3 text-2xs font-mono shadow-lg">
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            <span class="text-white font-bold">2-Player Active</span>
            <span class="text-neutral-400">P1: {multiseatSession.seat1?.mode || 'local'}</span>
            <span class="text-neutral-400">P2: {multiseatSession.seat2?.mode || 'local'}</span>
            {#if multiseatSession.seat2?.is_running}
              <button
                type="button"
                on:click={() => dispatch('stopSeat', 'p2')}
                class="px-2 py-0.5 rounded bg-surface-card hover:bg-surface-cardHover border border-surface-border text-rose-300 hover:text-white font-bold transition cursor-pointer"
                title="Stop Player 2 seat only"
              >
                Drop P2
              </button>
            {/if}
          </div>
        {/if}

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
        <div class="relative">
          <div class="flex items-center rounded-2xl shadow-2xl overflow-hidden {isTaskActive ? 'bg-surface-card border border-neutral-700' : isGameRunning ? 'bg-rose-600' : 'bg-white'}">
            <button
              type="button"
              on:click={handlePrimaryAction}
              disabled={isTaskActive}
              class="px-5 py-3 font-black text-sm transition-all flex items-center space-x-2 active:scale-95 focus:outline-none cursor-pointer disabled:cursor-not-allowed {isTaskActive ? 'text-neutral-300' : isGameRunning ? 'hover:bg-rose-500 text-white' : 'hover:bg-neutral-200 text-neutral-950'}"
            >
              {#if isTaskActive}
                <svg class="animate-spin w-4 h-4 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
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
              <span>{primaryButtonLabel}</span>
              {#if isCoop && isReady && !isGameRunning}
                <span class="ml-1 px-1.5 py-0.5 rounded text-3xs font-mono font-black bg-black text-white">2P</span>
              {/if}
            </button>

            {#if isInstalled && isReady && !isTaskActive && !isGameRunning}
              <!-- Split Menu Trigger Button -->
              <button
                type="button"
                on:click|stopPropagation={() => { showLaunchMenu = !showLaunchMenu; }}
                class="px-2.5 py-3.5 bg-neutral-100 hover:bg-neutral-200 text-neutral-800 border-l border-neutral-300 transition-colors cursor-pointer flex items-center justify-center"
                title="Launch Options (Single Player / 2-Player Co-op)"
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
              class="absolute right-0 bottom-full mb-2 w-72 bg-[#121215]/95 backdrop-blur-md border border-[#27272a] rounded-2xl shadow-2xl p-2 z-50 animate-fade-in space-y-1 text-left"
              on:click|stopPropagation
              on:keydown|stopPropagation
              role="menu"
              tabindex="-1"
            >
              <div class="px-2.5 py-1 text-3xs font-mono font-bold text-neutral-400 uppercase tracking-wider border-b border-surface-border mb-1">
                Launch Mode
              </div>

              <!-- Mode 1: Single Player -->
              <button
                type="button"
                role="menuitem"
                on:click={() => { if (isCoop) multiseatStore.toggleCoop(); showLaunchMenu = false; }}
                class="w-full px-3 py-2 rounded-xl flex items-center justify-between text-xs transition cursor-pointer {!isCoop ? 'bg-surface-cardHover text-white font-bold border border-surface-borderHighlight' : 'text-neutral-300 hover:bg-surface-input hover:text-white'}"
              >
                <div class="flex items-center space-x-2.5">
                  <svg class="w-4 h-4 text-neutral-400" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
                  </svg>
                  <div class="text-left">
                    <span class="block">Single Player</span>
                    <span class="text-3xs text-neutral-400 font-normal">Standard local launch</span>
                  </div>
                </div>
                {#if !isCoop}
                  <span class="w-1.5 h-1.5 rounded-full bg-white"></span>
                {/if}
              </button>

              <!-- Mode 2: 2-Player Co-op / Multiseat -->
              <button
                type="button"
                role="menuitem"
                on:click={() => { if (!isCoop) multiseatStore.toggleCoop(); showLaunchMenu = false; }}
                class="w-full px-3 py-2 rounded-xl flex items-center justify-between text-xs transition cursor-pointer {isCoop ? 'bg-surface-cardHover text-white font-bold border border-surface-borderHighlight' : 'text-neutral-300 hover:bg-surface-input hover:text-white'}"
              >
                <div class="flex items-center space-x-2.5">
                  <svg class="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                  <div class="text-left">
                    <div class="flex items-center space-x-1.5">
                      <span>2-Player Co-op</span>
                      <span class="px-1 py-0.2 rounded text-4xs font-mono font-bold bg-neutral-800 text-neutral-300 border border-neutral-700">Multiseat</span>
                    </div>
                    <span class="text-3xs text-neutral-400 font-normal">Dual instance & isolated seats</span>
                  </div>
                </div>
                {#if isCoop}
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                {/if}
              </button>
            </div>
          {/if}
        </div>

        {#if isInstalled && game.gameDir}
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
        {/if}
        </div>
      </div>
    </div>
  {/if}
</div>
