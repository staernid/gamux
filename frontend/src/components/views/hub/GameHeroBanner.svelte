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

  export let game = null;
  export let isGameRunning = false;
  export let isTaskActive = false;
  export let activeGameTask = null;
  export let globalCfg = {};

  const dispatch = createEventDispatcher();

  let logoFailed = false;
  let heroFailed = false;

  $: if (game) {
    logoFailed = false;
    heroFailed = false;
  }

  $: pColors = game ? getPlatformColors(game) : { border: '#1a9fff', badgeBg: '#1a9fff', label: 'WIN64', glow: 'rgba(26,159,255,0.4)' };
  $: platformLabel = game ? getPlatformType(game) : 'Windows';
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };
  $: isReady = game ? (game.isReady || game.isPatched || !!game.exePath) : false;
  $: isSteam = game ? isSteamApp(game.appId) : false;
  $: heroBackdropUrl = game ? (game.localHeroUrl || game.heroUrl || game.localBannerUrl || game.bannerUrl || (isSteam ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${game.appId}/library_hero.jpg` : '') || game.localCoverUrl || game.coverUrl || '') : '';

  $: primaryButtonLabel = isTaskActive
    ? (activeGameTask?.status === 'queued' ? 'Queued...' : 'Task in Progress...')
    : isGameRunning
    ? 'Stop Game'
    : isReady
    ? 'Play Now'
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

<!-- Hero Banner Background & Vignette -->
<div class="relative w-full h-96 sm:h-[420px] flex-shrink-0 overflow-hidden select-none">
  <!-- Background Image with dynamic blur -->
  <div class="absolute inset-0 bg-[#07070a]">
    {#if game?.localCoverUrl || game?.coverUrl}
      <img
        src={game.localCoverUrl || game.coverUrl}
        alt=""
        class="absolute inset-0 w-full h-full object-cover opacity-25 filter blur-xl scale-125 pointer-events-none"
      />
    {/if}

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
          {game?.name ? game.name.slice(0, 3).toUpperCase() : 'GMX'}
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
      on:click={handleReturnToShelf}
      class="flex items-center space-x-2 bg-[#12121a]/80 hover:bg-[#1e1e2c] border border-[#27273a] text-slate-300 hover:text-white px-3.5 py-1.5 rounded-xl text-xs font-bold transition shadow-lg backdrop-blur-md group cursor-pointer"
      title="Back to Shelf (Esc / B)"
    >
      <span class="text-slate-400 group-hover:text-cyan-400 transition-colors">←</span>
      <span>Shelf</span>
      <span class="text-[10px] text-slate-500 font-mono px-1 py-0.2 rounded bg-black/40 border border-[#2c2c3e]">B / Esc</span>
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
            <svg class="w-4 h-4 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span class="text-[10px] font-bold">Change Art</span>
          </div>
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
            <span class="truncate max-w-sm">{game.runner || globalCfg?.default_runner || 'UMU'}</span>
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
            {#if game.installedAt > 0}
              <span class="text-slate-600">·</span>
              <span class="text-slate-400">Installed {formatInstallDate(game.installedAt)}</span>
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
        <button
          type="button"
          on:click={handlePrimaryAction}
          disabled={isTaskActive}
          class="px-6 py-3 rounded-2xl font-black text-sm transition-all flex items-center space-x-2.5 shadow-2xl active:scale-95 focus:outline-none focus:ring-2 focus:ring-white/40 cursor-pointer disabled:cursor-not-allowed {isTaskActive ? 'bg-[#1e1e2c] border border-cyan-500/50 text-cyan-300 shadow-cyan-950/40' : isGameRunning ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-600/30' : isReady ? 'bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 shadow-emerald-500/30' : 'bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-400 hover:to-orange-400 text-slate-950 shadow-amber-500/30'}"
        >
          {#if isTaskActive}
            <svg class="animate-spin w-4 h-4 text-cyan-400" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
          {:else if isGameRunning}
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
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
        </button>

        <!-- Files Button (Y) -->
        <button
          type="button"
          on:click={handleOpenDirectory}
          class="p-3 rounded-2xl bg-[#14141e]/90 hover:bg-[#1f1f2e] border border-[#2b2b3e] text-slate-300 hover:text-white transition shadow-xl flex items-center justify-center group cursor-pointer"
          title="Open Directory in File Manager (Y)"
        >
          <svg class="w-5 h-5 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
        </button>
      </div>
    </div>
  {/if}
</div>
