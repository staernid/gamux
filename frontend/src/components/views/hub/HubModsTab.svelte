<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../api/app.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import { modalStore } from '../../../stores/modal.js';
  import { formatBytes, formatTimeAgo } from '../../../utils/format.js';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null } = $props();

  let modStatus = $state(null);
  let isLoading = $state(true);
  let isCheckingUpdates = $state(false);
  let isInstalling = $state(false);
  let workshopInput = $state('');
  let searchQuery = $state('');

  let gameDir = $derived(game?.gameDir || '');
  let topology = $derived(modStatus?.topology || game?.modTopology || null);
  let mods = $derived(modStatus?.mods || game?.mods || []);
  let outdatedCount = $derived(modStatus?.outdatedCount ?? game?.outdatedModsCount ?? 0);

  let filteredMods = $derived.by(() => {
    if (!searchQuery.trim()) return mods;
    const q = searchQuery.toLowerCase().trim();
    return mods.filter(m =>
      (m.name || '').toLowerCase().includes(q) ||
      (m.id || '').toLowerCase().includes(q) ||
      (m.description || '').toLowerCase().includes(q) ||
      String(m.workshopId || '').includes(q)
    );
  });

  function normalizeModStatus(status) {
    if (!status) return null;
    const topo = status.topology || status.Topology || {};
    const rawMods = status.mods || status.Mods || [];
    return {
      topology: {
        engine: topo.engine || topo.Engine || 'generic',
        modDir: topo.mod_dir || topo.modDir || topo.ModDir || 'mods',
        absoluteModDir: topo.absolute_mod_dir || topo.absoluteModDir || topo.AbsoluteModDir || '',
        detected: Boolean(topo.detected ?? topo.Detected ?? false),
        supportedFormats: topo.supported_formats || topo.supportedFormats || topo.SupportedFormats || [],
      },
      mods: rawMods.map(m => ({
        id: String(m.id || m.ID || ''),
        name: m.name || m.Name || '',
        workshopId: Number(m.workshop_id ?? m.workshopId ?? m.WorkshopID ?? 0),
        enabled: Boolean(m.enabled ?? m.Enabled ?? true),
        path: m.path || m.Path || '',
        relativePath: m.relative_path || m.relativePath || m.RelativePath || '',
        timeUpdated: Number(m.time_updated ?? m.timeUpdated ?? m.TimeUpdated ?? 0),
        latestTimeUpdated: Number(m.latest_time_updated ?? m.latestTimeUpdated ?? m.LatestTimeUpdated ?? 0),
        hasUpdate: Boolean(m.has_update ?? m.hasUpdate ?? m.HasUpdate ?? false),
        description: m.description || m.Description || '',
        previewUrl: m.preview_url || m.previewUrl || m.PreviewURL || '',
        size: Number(m.size ?? m.Size ?? 0),
        files: m.files || m.Files || [],
      })),
      outdatedCount: Number(status.outdated_count ?? status.outdatedCount ?? status.OutdatedCount ?? 0)
    };
  }

  onMount(async () => {
    await loadMods();
  });

  async function loadMods() {
    if (!gameDir) return;
    isLoading = true;
    try {
      const res = await AppAPI.getGameMods(gameDir);
      if (res) {
        modStatus = normalizeModStatus(res);
      }
    } catch (err) {
      console.warn('Failed to load mods:', err);
    } finally {
      isLoading = false;
    }
  }

  async function handleToggle(mod, enable) {
    if (!gameDir || !mod?.id) return;
    try {
      const updated = await AppAPI.toggleGameMod(gameDir, mod.id, enable);
      if (updated && modStatus) {
        modStatus.mods = modStatus.mods.map(m => m.id === mod.id ? { ...m, enabled: enable, path: updated.path, relativePath: updated.relativePath } : m);
      }
      notificationStore.items.add({
        type: 'success',
        title: enable ? 'Mod Enabled' : 'Mod Disabled',
        message: `${mod.name} is now ${enable ? 'enabled' : 'disabled'}.`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Toggle Failed',
        message: err?.message || String(err)
      });
    }
  }

  function confirmDeleteMod(mod) {
    if (!gameDir || !mod?.id) return;
    modalStore.open('confirm', {
      title: `Delete Mod "${mod.name}"?`,
      message: `Are you sure you want to delete this mod? All mod files at "${mod.relativePath || mod.id}" will be permanently removed.`,
      confirmLabel: 'Delete Mod',
      cancelLabel: 'Keep Mod',
      danger: true,
      onConfirm: async () => {
        try {
          await AppAPI.deleteGameMod(gameDir, mod.id);
          notificationStore.items.add({
            type: 'success',
            title: 'Mod Deleted',
            message: `Removed ${mod.name} from game.`
          });
          await loadMods();
        } catch (err) {
          notificationStore.items.add({
            type: 'error',
            title: 'Deletion Failed',
            message: err?.message || String(err)
          });
        }
      }
    });
  }

  async function handleCheckUpdates() {
    if (!gameDir) return;
    isCheckingUpdates = true;
    try {
      const res = await AppAPI.checkGameModUpdates(gameDir);
      if (res) {
        modStatus = normalizeModStatus(res);
      }
      if (res?.outdatedCount > 0) {
        notificationStore.items.add({
          type: 'warning',
          title: 'Mod Updates Found',
          message: `${res.outdatedCount} workshop mod(s) have newer versions available on Steam.`
        });
      } else {
        notificationStore.items.add({
          type: 'success',
          title: 'Mods Up to Date',
          message: 'All installed workshop mods match the latest Steam Community versions.'
        });
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Check Updates Failed',
        message: err?.message || String(err)
      });
    } finally {
      isCheckingUpdates = false;
    }
  }

  async function handleInstallWorkshop() {
    const input = workshopInput.trim();
    if (!input || !gameDir) return;

    isInstalling = true;
    try {
      const taskId = await AppAPI.installWorkshopMod(gameDir, input);
      workshopInput = '';
      notificationStore.items.add({
        type: 'success',
        title: 'Workshop Mod Download Started',
        message: `Downloading workshop item (${input}) in background task ${taskId || ''}`
      });
      // Give worker a moment, then refresh
      setTimeout(loadMods, 2000);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Installation Failed',
        message: err?.message || String(err)
      });
    } finally {
      isInstalling = false;
    }
  }

  function openModDirectory() {
    const target = topology?.absoluteModDir || gameDir;
    if (target) {
      AppAPI.openDirectory(target);
    }
  }

  function openWorkshopURL(workshopId) {
    if (!workshopId) return;
    const url = `https://steamcommunity.com/sharedfiles/filedetails/?id=${workshopId}`;
    if (AppAPI.openURL) {
      AppAPI.openURL(url);
    } else {
      window.open(url, '_blank');
    }
  }
</script>

<div class="space-y-6 animate-fade-in text-neutral-200">
  <!-- Topology & Status Header -->
  <div class="bg-surface-card rounded-2xl p-5 border border-surface-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
    <div class="flex items-start md:items-center space-x-3.5">
      <div class="w-10 h-10 rounded-xl bg-surface-input border border-surface-borderHighlight flex items-center justify-center flex-shrink-0 text-neutral-300">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
        </svg>
      </div>
      <div>
        <div class="flex items-center space-x-2">
          <h2 class="text-sm font-bold text-white tracking-wide">Mod Layout & Engine Topology</h2>
          {#if topology}
            <span class="px-2 py-0.5 rounded-full text-3xs font-mono font-semibold uppercase tracking-wider bg-neutral-800 text-neutral-200 border border-surface-borderHighlight">
              {topology.engine}
            </span>
          {/if}
        </div>
        <p class="text-2xs text-neutral-400 mt-0.5 font-mono">
          Target directory: <span class="text-neutral-300">{topology?.modDir || 'mods'}</span>
        </p>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center space-x-2 self-start md:self-auto">
      <button
        type="button"
        onclick={openModDirectory}
        class="px-3 py-1.5 rounded-xl bg-surface-input hover:bg-surface-cardHover text-xs font-medium text-neutral-300 hover:text-white border border-surface-border flex items-center space-x-1.5 transition-colors cursor-pointer"
        title="Open mods directory in file manager"
      >
        <svg class="w-3.5 h-3.5 opacity-70" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <span>Open Folder</span>
      </button>

      <button
        type="button"
        onclick={handleCheckUpdates}
        disabled={isCheckingUpdates}
        class="px-3 py-1.5 rounded-xl bg-surface-input hover:bg-surface-cardHover text-xs font-medium text-neutral-300 hover:text-white border border-surface-border flex items-center space-x-1.5 transition-colors cursor-pointer disabled:opacity-50"
      >
        <svg class="w-3.5 h-3.5 opacity-70 {isCheckingUpdates ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>{isCheckingUpdates ? 'Checking...' : 'Check Updates'}</span>
      </button>
    </div>
  </div>

  <!-- Outdated Mod Alert Banner -->
  {#if outdatedCount > 0}
    <div class="bg-amber-950/30 border border-amber-600/40 rounded-2xl p-4 flex items-center justify-between text-xs text-amber-200 animate-fade-in">
      <div class="flex items-center space-x-3">
        <svg class="w-5 h-5 text-amber-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div>
          <span class="font-bold">{outdatedCount} Workshop Item{outdatedCount > 1 ? 's have' : ' has'} updates</span>
          <p class="text-2xs text-amber-300/80 mt-0.5">Steam Community has published newer revisions for installed workshop mods.</p>
        </div>
      </div>
      <button
        type="button"
        onclick={handleCheckUpdates}
        class="px-3 py-1.5 rounded-xl bg-amber-500/20 hover:bg-amber-500/30 text-amber-200 text-xs font-semibold border border-amber-500/30 cursor-pointer transition-colors"
      >
        Refresh
      </button>
    </div>
  {/if}

  <!-- Install Workshop Item Card -->
  <div class="bg-surface-card rounded-2xl p-5 border border-surface-border shadow-sm space-y-3">
    <div>
      <h3 class="text-xs font-bold text-white tracking-wide flex items-center space-x-2">
        <svg class="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        <span>Install Steam Workshop Mod</span>
      </h3>
      <p class="text-2xs text-neutral-400 mt-1">
        Enter a Workshop URL or published item ID (e.g. <span class="font-mono text-neutral-300">315783921</span> or <span class="font-mono text-neutral-300">https://steamcommunity.com/sharedfiles/filedetails/?id=...</span>).
      </p>
    </div>

    <form onsubmit={(e) => { e.preventDefault(); handleInstallWorkshop(); }} class="flex items-center gap-2">
      <input
        type="text"
        bind:value={workshopInput}
        placeholder="Enter Steam Workshop URL or ID..."
        class="flex-1 bg-surface-input border border-surface-borderHighlight rounded-xl px-3.5 py-2 text-xs text-white placeholder-neutral-500 focus:outline-none focus:border-neutral-400 font-mono"
      />
      <button
        type="submit"
        disabled={!workshopInput.trim() || isInstalling}
        class="px-4 py-2 rounded-xl bg-white hover:bg-neutral-200 text-black text-xs font-bold transition-all disabled:opacity-40 disabled:hover:bg-white cursor-pointer flex-shrink-0"
      >
        {isInstalling ? 'Installing...' : 'Download & Install'}
      </button>
    </form>
  </div>

  <!-- Mods Inventory Section -->
  <div class="space-y-3">
    <div class="flex items-center justify-between gap-4">
      <div class="flex items-center space-x-2">
        <h3 class="text-xs font-bold text-white uppercase tracking-wider">Installed Mods</h3>
        <span class="px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-300 text-3xs font-mono border border-surface-borderHighlight">
          {mods.length}
        </span>
      </div>

      {#if mods.length > 3}
        <div class="w-64">
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="Search mods..."
            class="w-full bg-surface-input border border-surface-borderHighlight rounded-xl px-3 py-1.5 text-xs text-white placeholder-neutral-500 focus:outline-none focus:border-neutral-400"
          />
        </div>
      {/if}
    </div>

    {#if isLoading}
      <div class="p-8 text-center text-xs text-neutral-400 bg-surface-card rounded-2xl border border-surface-border">
        Loading installed mods...
      </div>
    {:else if filteredMods.length === 0}
      <div class="p-10 text-center bg-surface-card rounded-2xl border border-surface-border flex flex-col items-center justify-center space-y-3">
        <div class="w-12 h-12 rounded-2xl bg-surface-input flex items-center justify-center text-neutral-500">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
        </div>
        <div>
          <p class="text-xs font-bold text-neutral-300">No mods installed</p>
          <p class="text-2xs text-neutral-500 mt-1 max-w-sm">
            Enter a Steam Workshop URL or published file ID above to install mods automatically into <span class="font-mono text-neutral-400">{topology?.modDir || 'mods'}</span>.
          </p>
        </div>
      </div>
    {:else}
      <div class="space-y-2">
        {#each filteredMods as mod (mod.id)}
          <div class="bg-surface-card hover:bg-surface-cardHover transition-colors rounded-2xl p-4 border border-surface-border flex items-center justify-between gap-4">
            <div class="flex items-center space-x-3.5 min-w-0 flex-1">
              <!-- Non-destructive Toggle Switch -->
              <div class="flex-shrink-0">
                <SeamlessSwitch
                  compact={true}
                  checked={mod.enabled}
                  onchange={(val) => handleToggle(mod, val)}
                />
              </div>

              <!-- Mod Details -->
              <div class="min-w-0 flex-1">
                <div class="flex items-center space-x-2">
                  <h4 class="text-xs font-bold text-white truncate {mod.enabled ? '' : 'line-through opacity-60'}">
                    {mod.name}
                  </h4>

                  {#if mod.hasUpdate}
                    <span class="px-1.5 py-0.2 rounded text-3xs font-bold uppercase bg-amber-500/20 text-amber-300 border border-amber-500/30">
                      Update Available
                    </span>
                  {/if}

                  {#if mod.workshopId}
                    <span class="px-1.5 py-0.2 rounded text-3xs font-mono bg-neutral-800 text-neutral-300 border border-neutral-700">
                      Workshop #{mod.workshopId}
                    </span>
                  {:else}
                    <span class="px-1.5 py-0.2 rounded text-3xs font-mono bg-neutral-800 text-neutral-400 border border-neutral-700">
                      Local
                    </span>
                  {/if}
                </div>

                {#if mod.description}
                  <p class="text-2xs text-neutral-400 mt-0.5 line-clamp-1 leading-relaxed">
                    {mod.description}
                  </p>
                {/if}

                <div class="flex items-center space-x-3 text-3xs text-neutral-500 font-mono mt-1">
                  <span>{mod.relativePath || mod.path}</span>
                  {#if mod.size > 0}
                    <span>• {formatBytes(mod.size)}</span>
                  {/if}
                  {#if mod.timeUpdated > 0}
                    <span>• Updated {formatTimeAgo(mod.timeUpdated)}</span>
                  {/if}
                </div>
              </div>
            </div>

            <!-- Mod Actions -->
            <div class="flex items-center space-x-1.5 flex-shrink-0">
              {#if mod.workshopId}
                <button
                  type="button"
                  onclick={() => openWorkshopURL(mod.workshopId)}
                  class="p-2 rounded-xl text-neutral-400 hover:text-white hover:bg-surface-input transition-colors cursor-pointer"
                  title="View on Steam Workshop"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </button>
              {/if}

              <button
                type="button"
                onclick={() => confirmDeleteMod(mod)}
                class="p-2 rounded-xl text-neutral-500 hover:text-rose-400 hover:bg-rose-950/20 transition-colors cursor-pointer"
                title="Delete Mod"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
