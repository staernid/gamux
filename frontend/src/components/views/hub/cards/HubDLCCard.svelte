<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { tasksStore } from '../../../../stores/tasks.js';
  import { notificationStore } from '../../../../stores/notifications.js';
  import { formatBytes } from '../../../../utils/format.js';

  export let game = null;

  const activeDownloads = tasksStore.activeDownloads;

  let depots = [];
  let dlcs = [];
  let isLoading = true;
  let isLoadingDLCs = false;
  let isSyncingDLCs = false;
  let isActionRunning = false;

  $: gameDir = game?.gameDir || '';
  $: appID = game?.appId || '';
  $: activeTransfer = ($activeDownloads || []).find(
    (t) => String(t.appId) === String(appID) || (t.gameDir && isSamePath(t.gameDir, gameDir))
  );

  onMount(async () => {
    await Promise.all([loadDepots(), loadDLCs()]);
  });

  async function loadDLCs() {
    if (!gameDir) return;
    isLoadingDLCs = true;
    try {
      const res = await AppAPI.getGameDLCs(gameDir);
      dlcs = Array.isArray(res) ? res : [];
    } catch {
      dlcs = [];
    } finally {
      isLoadingDLCs = false;
    }
  }

  async function handleSyncDLCs() {
    if (!gameDir || isSyncingDLCs) return;
    isSyncingDLCs = true;
    try {
      await AppAPI.fetchDLCs(gameDir);
      await loadDLCs();
      notificationStore.items.add({
        type: 'success',
        title: 'DLCs Synchronized',
        message: `Found and configured ${dlcs.length} DLC items`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Sync Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSyncingDLCs = false;
    }
  }

  async function loadDepots() {
    if (!gameDir) return;
    isLoading = true;
    try {
      const res = await AppAPI.getGameDepots(gameDir);
      depots = Array.isArray(res) ? res : (game?.depots || []);
    } catch (err) {
      depots = game?.depots || [];
    } finally {
      isLoading = false;
    }
  }

  function getCategoryLabel(cat) {
    switch (cat) {
      case 'common': return 'Base Content';
      case 'windows': return 'Windows Binaries';
      case 'linux': return 'Linux Native';
      case 'macos': return 'macOS Bundle';
      case 'soundtrack': return 'Official Soundtrack';
      default: return 'Modular Content';
    }
  }

  function getCategoryBadgeClass(cat) {
    return 'bg-surface-header text-neutral-300 border-surface-border';
  }

  async function handleDownloadSoundtrack(depot) {
    if (isActionRunning || !appID) return;
    isActionRunning = true;
    try {
      await tasksStore.start({
        app_id: Number(appID),
        platform: 'soundtrack',
        game_name: `${game?.name || 'Game'} Soundtrack`,
        auto_apply: false
      });
      notificationStore.items.add({
        type: 'success',
        title: 'Soundtrack Download Queued',
        message: `Queued download for ${depot.name || 'Soundtrack'} to Music folder`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Download Failed',
        message: err?.message || String(err)
      });
    } finally {
      isActionRunning = false;
    }
  }

  async function handleDownloadDepot(depot) {
    if (isActionRunning || !appID) return;
    isActionRunning = true;
    try {
      await tasksStore.start({
        app_id: Number(appID),
        platform: depot.category || 'all',
        target_dir: gameDir,
        game_name: `${game?.name || 'Game'} (${depot.name || depot.depot_id})`,
        auto_apply: true
      });
      notificationStore.items.add({
        type: 'success',
        title: 'Depot Download Queued',
        message: `Queued download for ${depot.name}`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Download Failed',
        message: err?.message || String(err)
      });
    } finally {
      isActionRunning = false;
    }
  }

  async function handleUninstallDepot(depot) {
    if (isActionRunning || !gameDir) return;
    isActionRunning = true;
    try {
      await AppAPI.uninstallDepot(gameDir, Number(depot.depot_id));
      notificationStore.items.add({
        type: 'success',
        title: 'Depot Uninstalled',
        message: `Removed ${depot.name} (${formatBytes(depot.size || 0)} freed)`
      });
      await loadDepots();
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Uninstall Failed',
        message: err?.message || String(err)
      });
    } finally {
      isActionRunning = false;
    }
  }

  $: totalInstalledSize = depots.filter(d => d.installed).reduce((acc, d) => acc + (d.size || 0), 0);
  $: installedCount = depots.filter(d => d.installed).length;
</script>

<div class="space-y-4">
  <!-- Configured DLCs Section -->
  <div class="p-3.5 bg-surface-header border border-surface-border rounded-xl text-xs space-y-3">
    <div class="flex items-center justify-between">
      <div class="space-y-0.5">
        <strong class="text-neutral-200 block font-bold">Downloadable Content (DLC)</strong>
        <span class="text-neutral-400 text-2xs">
          {dlcs.length} DLCs configured for this game
        </span>
      </div>
      {#if appID && appID !== '0'}
        <button
          type="button"
          on:click={handleSyncDLCs}
          disabled={isSyncingDLCs}
          class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border transition-colors font-bold text-2xs flex items-center space-x-1.5 cursor-pointer"
        >
          <svg class="w-3 h-3 {isSyncingDLCs ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Sync Steam DLCs</span>
        </button>
      {/if}
    </div>

    {#if dlcs.length > 0}
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
        {#each dlcs as dlc (dlc.app_id)}
          <div class="p-2.5 bg-surface-card border border-surface-border rounded-lg flex items-center justify-between gap-2">
            <div class="min-w-0">
              <span class="font-bold text-neutral-200 text-xs block truncate">{dlc.name || `DLC ${dlc.app_id}`}</span>
              <span class="text-3xs font-mono text-neutral-400">AppID #{dlc.app_id}</span>
            </div>
            <span class="px-2 py-0.5 rounded text-3xs font-bold bg-emerald-950/50 text-emerald-400 border border-emerald-800/40 flex-shrink-0">
              Active
            </span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Summary Banner -->
  <div class="flex items-center justify-between p-3.5 bg-surface-header border border-surface-border rounded-xl text-xs">
    <div class="space-y-0.5">
      <strong class="text-neutral-200 block font-bold">DLC & Content Depots</strong>
      <span class="text-neutral-400 text-2xs">
        {installedCount} of {depots.length} depots installed ({formatBytes(totalInstalledSize)})
      </span>
    </div>
    <button
      type="button"
      on:click={loadDepots}
      disabled={isLoading}
      class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border transition-colors font-bold text-2xs flex items-center space-x-1.5 cursor-pointer"
    >
      <svg class="w-3 h-3 {isLoading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
      <span>Refresh</span>
    </button>
  </div>

  {#if activeTransfer}
    <!-- Active Transfer Telemetry Banner -->
    <div class="p-3 bg-surface-header border border-surface-borderHighlight rounded-xl space-y-2">
      <div class="flex items-center justify-between text-xs">
        <div class="flex items-center space-x-2">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span class="font-bold text-neutral-100">{activeTransfer.stage || 'Downloading Depots'}</span>
        </div>
        <span class="font-mono text-neutral-200 font-bold">{activeTransfer.percent || 0}%</span>
      </div>
      <div class="w-full bg-black/40 rounded-full h-1.5 overflow-hidden">
        <div class="bg-white h-full transition-all duration-300" style="width: {activeTransfer.percent || 0}%"></div>
      </div>
      <div class="flex items-center justify-between text-2xs text-neutral-400 font-mono">
        <span>{formatBytes(activeTransfer.currentBytes || 0)} / {formatBytes(activeTransfer.totalBytes || 0)}</span>
        <span>{formatBytes(activeTransfer.speedBps || 0)}/s</span>
      </div>
    </div>
  {/if}

  {#if isLoading}
    <div class="py-12 text-center text-neutral-400 text-xs">
      Loading content manifests...
    </div>
  {:else if depots.length === 0}
    <div class="py-12 text-center text-neutral-400 text-xs bg-surface-header border border-surface-border rounded-xl space-y-1">
      <p class="font-bold text-neutral-200">No Manifest Depots Discovered</p>
      <p class="text-2xs text-neutral-400">Local <code class="font-mono text-neutral-300">[Manifests]/</code> folder is empty or not yet synchronized.</p>
    </div>
  {:else}
    <!-- Depots List -->
    <div class="space-y-2">
      {#each depots as depot (depot.depot_id)}
        <div class="p-3 bg-surface-card border border-surface-border hover:border-surface-borderHighlight rounded-xl flex items-center justify-between gap-3 transition-colors">
          <div class="flex items-center space-x-3 min-w-0">
            <!-- Installed Status Indicator -->
            <div class="w-4 h-4 rounded-md flex items-center justify-center flex-shrink-0 {depot.installed ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-700/60' : 'bg-surface-header text-neutral-600 border border-surface-border'}">
              {#if depot.installed}
                <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                </svg>
              {/if}
            </div>

            <div class="space-y-1 min-w-0">
              <div class="flex items-center space-x-2 flex-wrap gap-y-1">
                <span class="font-bold text-neutral-100 text-xs truncate">{depot.name || `Depot ${depot.depot_id}`}</span>
                <span class="px-2 py-0.5 rounded-full text-3xs font-bold border {getCategoryBadgeClass(depot.category)}">
                  {getCategoryLabel(depot.category)}
                </span>
              </div>
              <p class="text-2xs text-neutral-400 font-mono">
                {depot.file_count || 0} files • {formatBytes(depot.size || 0)}
              </p>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center space-x-2 flex-shrink-0">
            {#if depot.category === 'soundtrack'}
              <button
                type="button"
                on:click={() => handleDownloadSoundtrack(depot)}
                disabled={isActionRunning}
                class="px-2.5 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border font-bold text-2xs transition-colors flex items-center space-x-1 cursor-pointer"
              >
                <span>🎵 Save to Music</span>
              </button>
            {:else if !depot.installed}
              <button
                type="button"
                on:click={() => handleDownloadDepot(depot)}
                disabled={isActionRunning}
                class="px-2.5 py-1.5 rounded-lg bg-white hover:bg-neutral-200 text-neutral-950 font-bold text-2xs transition-colors flex items-center space-x-1 cursor-pointer"
              >
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                <span>Install</span>
              </button>
            {:else}
              <div class="flex items-center space-x-1.5">
                <span class="text-emerald-400 font-bold text-2xs px-2 py-1 bg-emerald-950/40 rounded-lg border border-emerald-800/40">
                  Installed
                </span>
                {#if depot.category !== 'common'}
                  <button
                    type="button"
                    on:click={() => handleUninstallDepot(depot)}
                    disabled={isActionRunning}
                    title="Remove depot files to free space"
                    class="px-2 py-1 rounded-lg bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 border border-rose-800/50 font-bold text-3xs transition-colors"
                  >
                    Uninstall
                  </button>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

