<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { formatBytes } from '../../../../utils/format.js';
  import {
    formatPlaytime,
    formatLastPlayed,
    formatInstallDate,
    getPlatformType,
    isSteamApp
  } from '../../../../utils/game.js';
  import LiveCommunityBadge from '../../../blocks/LiveCommunityBadge.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null } = $props();

  let livePlayerCount = $state(0);
  let loadedPlayerAppId = $state('');
  let copiedPath = $state(false);
  let copyTimer = null;

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

  $effect(() => {
    if (game?.appId && isSteamApp(game.appId)) {
      const aid = String(game.appId).trim();
      if (aid !== loadedPlayerAppId) {
        loadedPlayerAppId = aid;
        livePlayerCount = 0;
        fetchLivePlayers(aid);
      }
    } else {
      livePlayerCount = 0;
      loadedPlayerAppId = '';
    }
  });

  async function copyGameDir() {
    if (!game?.gameDir) return;
    try {
      await navigator.clipboard.writeText(game.gameDir);
      copiedPath = true;
      if (copyTimer) clearTimeout(copyTimer);
      copyTimer = setTimeout(() => {
        copiedPath = false;
      }, 2000);
    } catch {
      // Ignore clipboard fallback
    }
  }

  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);
  let isStub = $derived(game ? (game.isStub ?? (!isInstalled && Boolean(game.gameDir))) : false);
  let isVirtual = $derived(game ? (game.isVirtual ?? (!isInstalled && !isStub && !game.gameDir)) : false);
  let platformLabel = $derived(game ? getPlatformType(game) : 'Windows');
  let effectiveRunner = $derived((game?.platform || '').toLowerCase().includes('linux') || game?.runner === 'linux'
    ? 'Linux Native'
    : (game?.runner || 'UMU-Proton'));
</script>

<div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-sm space-y-4" data-testid="extended-info-card">
  <!-- Header -->
  <div class="flex items-center justify-between pb-1">
    <div class="flex items-center space-x-2 text-xs font-mono font-bold uppercase tracking-wider text-neutral-400">
      <svg class="w-4 h-4 text-neutral-400 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span>Extended Info</span>
    </div>
    {#if livePlayerCount > 0}
      <LiveCommunityBadge playerCount={livePlayerCount} label="playing now" />
    {/if}
  </div>

  <!-- Key Metrics & Statistics Grid -->
  <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3 text-xs">
    <!-- Playtime -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Playtime</span>
      <p class="font-mono text-xs font-semibold text-neutral-200">
        {game?.playtimeSeconds > 0 ? formatPlaytime(game.playtimeSeconds) : 'Never played'}
      </p>
    </div>

    <!-- Last Played -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Last Played</span>
      <p class="font-mono text-xs font-semibold text-neutral-200 truncate">
        {game?.lastPlayed > 0 ? formatLastPlayed(game.lastPlayed) : 'Never'}
      </p>
    </div>

    <!-- Size on Disk -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Disk Size</span>
      <p class="font-mono text-xs font-semibold text-neutral-200">
        {isInstalled && (game?.diskSizeBytes || 0) > 0 ? formatBytes(game.diskSizeBytes) : isStub ? 'In Library (Uninstalled)' : isVirtual ? 'Online Store Title' : '—'}
      </p>
    </div>

    <!-- Install Date -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Installed</span>
      <p class="font-mono text-xs font-semibold text-neutral-200 truncate">
        {game?.installedAt > 0 ? formatInstallDate(game.installedAt) : isInstalled ? 'Installed' : 'Not installed'}
      </p>
    </div>

    <!-- App / Store ID -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">App ID</span>
      <p class="font-mono text-xs font-semibold text-neutral-200">
        {game?.appId && game.appId !== '0' ? `#${game.appId}` : game?.gogId ? `GOG #${game.gogId}` : 'Standalone'}
      </p>
    </div>

    <!-- Platform / Architecture -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Platform</span>
      <p class="font-mono text-xs font-semibold text-neutral-200">
        {platformLabel} ({game?.platform || 'win64'})
      </p>
    </div>

    <!-- Runner -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Runner</span>
      <p class="font-mono text-xs font-semibold text-neutral-200 truncate" title={effectiveRunner}>
        {effectiveRunner}
      </p>
    </div>

    <!-- Build / Version Status -->
    <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-1">
      <span class="text-3xs text-neutral-500 uppercase font-mono font-bold tracking-wider block">Build</span>
      <p class="font-mono text-xs font-semibold text-neutral-200 truncate">
        {game?.buildId ? `#${game.buildId}` : game?.branch ? `Branch: ${game.branch}` : 'Standard'}
      </p>
    </div>
  </div>

  <!-- Install Location Row -->
  {#if game?.gameDir}
    <div class="pt-2 border-t border-surface-border flex items-center justify-between gap-3 text-2xs">
      <div class="flex items-center space-x-2 min-w-0 font-mono text-neutral-400">
        <span class="text-neutral-500 uppercase font-bold text-3xs flex-shrink-0">Directory:</span>
        <span class="truncate select-all text-neutral-300">{game.gameDir}</span>
      </div>
      <button
        type="button"
        onclick={copyGameDir}
        class="px-2.5 py-1 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-3xs font-mono"
        title="Copy path to clipboard"
      >
        {#if copiedPath}
          <span class="text-emerald-400">✓ Copied</span>
        {:else}
          <svg class="w-3 h-3 opacity-70" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
          <span>Copy Path</span>
        {/if}
      </button>
    </div>
  {/if}
</div>
