<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { tasksStore } from '../../../../stores/tasks.js';
  import { notificationStore } from '../../../../stores/notifications.js';
  import { configStore } from '../../../../stores/config.js';
  import { modalStore } from '../../../../stores/modal.js';
  import { updateGameInStore } from '../../../../stores/library.js';
  import { formatBytes } from '../../../../utils/format.js';
  import { isSamePath } from '../../../../utils/game.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   * @property {(updated: any) => void} [onupdateGame]
   */

  /** @type {Props} */
  let { game = null, onupdateGame = undefined } = $props();

  let depots = $state([]);
  let dlcs = $state([]);
  let isLoading = $state(true);
  let isLoadingDLCs = false;
  let isSyncingDLCs = $state(false);
  let isActionRunning = $state(false);
  let selectedCategoryFilter = $state('all'); // 'all' | 'languages' | 'optional' | 'soundtrack' | 'core'
  let selectedDepotIds = $state(new Set());
  let userPreferredLanguage = $state('');

  let gameDir = $derived(game?.gameDir || '');
  let appID = $derived(game?.appId || '');
  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);

  let activeTransfer = $derived((tasksStore.activeDownloads || []).find(
    (t) => String(t.appId) === String(appID) || (t.gameDir && isSamePath(t.gameDir, gameDir))
  ));

  let configuredLanguage = $derived((
    userPreferredLanguage ||
    game?.currentLanguage ||
    configStore.current?.language ||
    'english'
  ).toLowerCase());

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

  function getContentCategory(depot) {
    if (depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0) || depot.content_category === 'dlc') return 'dlc';
    if (depot.content_category) return depot.content_category;
    if (depot.category === 'soundtrack') return 'soundtrack';
    return 'core';
  }

  function getDepotLanguage(depot) {
    if (depot.language) return depot.language;
    const cat = getContentCategory(depot);
    if (cat.startsWith('locale:')) {
      return cat.slice(7);
    }
    return '';
  }

  function getCategoryLabel(depot) {
    const cat = getContentCategory(depot);
    if (cat.startsWith('locale:')) {
      const lang = getDepotLanguage(depot);
      const capLang = lang ? lang.charAt(0).toUpperCase() + lang.slice(1) : 'Locale';
      return `${capLang} Voice / Audio`;
    }
    if (cat === 'optional_hd') {
      return 'High-Res / Optional Assets';
    }
    if (cat === 'soundtrack') {
      return 'Official Soundtrack';
    }
    if (cat === 'dlc' || depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0)) {
      if (depot.dlc_app_id) {
        return `DLC #${depot.dlc_app_id}`;
      }
      return 'DLC Content';
    }
    switch (depot.category) {
      case 'common': return 'Base Content';
      case 'windows': return 'Windows Binaries';
      case 'linux': return 'Linux Native';
      case 'macos': return 'macOS Bundle';
      default: return 'Core Content';
    }
  }

  function getCategoryBadgeClass(depot) {
    const cat = getContentCategory(depot);
    if (cat.startsWith('locale:')) {
      return 'bg-blue-950/60 text-blue-300 border-blue-800/50';
    }
    if (cat === 'optional_hd') {
      return 'bg-amber-950/60 text-amber-300 border-amber-800/50';
    }
    if (cat === 'soundtrack') {
      return 'bg-purple-950/60 text-purple-300 border-purple-800/50';
    }
    if (cat === 'dlc' || depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0)) {
      return 'bg-emerald-950/60 text-emerald-300 border-emerald-800/50';
    }
    return 'bg-surface-header text-neutral-300 border-surface-border';
  }

  function isDefaultLanguageDepot(depot) {
    const lang = getDepotLanguage(depot).toLowerCase();
    if (!lang) return false;
    return lang === configuredLanguage || configuredLanguage.startsWith(lang) || lang.startsWith(configuredLanguage);
  }

  function initDefaultSelections(depotList) {
    const selected = new Set();
    const confLang = configuredLanguage;
    for (const d of depotList) {
      const cat = getContentCategory(d);
      if (cat === 'core' || cat === 'common' || cat === 'dlc' || d.is_dlc) {
        selected.add(Number(d.depot_id));
      } else if (cat.startsWith('locale:')) {
        const lang = getDepotLanguage(d).toLowerCase();
        if (lang === confLang || confLang.startsWith(lang) || lang.startsWith(confLang)) {
          selected.add(Number(d.depot_id));
        }
      }
    }
    // If no language matched, include english if available
    const hasLang = Array.from(selected).some(id => {
      const d = depotList.find(x => Number(x.depot_id) === id);
      return d && getContentCategory(d).startsWith('locale:');
    });
    if (!hasLang) {
      const engDepot = depotList.find(d => getDepotLanguage(d).toLowerCase() === 'english');
      if (engDepot) selected.add(Number(engDepot.depot_id));
    }
    selectedDepotIds = selected;
  }

  async function loadDepots() {
    if (!gameDir && !appID) return;
    isLoading = true;
    try {
      let res;
      if (gameDir) {
        res = await AppAPI.getGameDepots(gameDir);
      } else if (appID) {
        res = await AppAPI.getAppDepots(appID);
      }
      depots = Array.isArray(res) ? res : (game?.depots || []);
      if (!isInstalled && selectedDepotIds.size === 0) {
        initDefaultSelections(depots);
      }
    } catch (err) {
      depots = game?.depots || [];
    } finally {
      isLoading = false;
    }
  }

  function toggleDepotSelection(depotId) {
    const id = Number(depotId);
    const next = new Set(selectedDepotIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    selectedDepotIds = next;
  }

  function handleLanguageChange(newLang) {
    userPreferredLanguage = newLang;
    if (!isInstalled) {
      // Recompute language selection
      const next = new Set(selectedDepotIds);
      // Remove all locale depots first
      for (const d of depots) {
        if (getContentCategory(d).startsWith('locale:')) {
          next.delete(Number(d.depot_id));
        }
      }
      // Add the chosen language depot
      for (const d of depots) {
        if (getContentCategory(d).startsWith('locale:')) {
          const l = getDepotLanguage(d).toLowerCase();
          if (l === newLang.toLowerCase() || newLang.toLowerCase().startsWith(l)) {
            next.add(Number(d.depot_id));
          }
        }
      }
      selectedDepotIds = next;
    }
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
        platform: game?.platform || (depot.category === 'linux' ? 'linux' : 'win64'),
        target_dir: gameDir,
        game_name: `${game?.name || 'Game'} (${depot.name || depot.depot_id})`,
        included_depots: [Number(depot.depot_id)],
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

  async function handleDownloadSelected() {
    if (isActionRunning || !appID) return;
    isActionRunning = true;
    try {
      const selected = Array.from(selectedDepotIds);
      await tasksStore.start({
        app_id: Number(appID),
        platform: game?.platform || 'win64',
        target_dir: gameDir,
        game_name: game?.name || 'Game',
        included_depots: selected.length > 0 ? selected : undefined,
        language: configuredLanguage,
        auto_apply: true
      });
      notificationStore.items.add({
        type: 'success',
        title: 'Download Queued',
        message: `Download started with ${selected.length} selected depots`
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

  function handleUninstallDepot(depot) {
    if (isActionRunning || !gameDir) return;
    const depotName = depot.name || `Depot ${depot.depot_id}`;
    modalStore.open('confirm', {
      title: `Uninstall ${depotName}?`,
      message: `This will remove local files for this depot from ${gameDir}. Shared game assets and manifests will remain intact.`,
      confirmLabel: 'Uninstall Depot',
      cancelLabel: 'Cancel',
      danger: true,
      onConfirm: async () => {
        isActionRunning = true;
        try {
          const res = await AppAPI.uninstallDepot(gameDir, Number(depot.depot_id));
          if (res) {
            const normalized = updateGameInStore(res);
            onupdateGame?.(normalized);
          }
          notificationStore.items.add({
            type: 'success',
            title: 'Depot Uninstalled',
            message: `Removed ${depotName} (${formatBytes(depot.size || 0)} freed)`
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
    });
  }

  let totalInstalledSize = $derived(depots.filter(d => d.installed).reduce((acc, d) => acc + (d.size || 0), 0));
  let installedCount = $derived(depots.filter(d => d.installed).length);

  let languageDepots = $derived(depots.filter(d => getContentCategory(d).startsWith('locale:')));
  let optionalDepots = $derived(depots.filter(d => getContentCategory(d) === 'optional_hd'));
  let soundtrackDepots = $derived(depots.filter(d => getContentCategory(d) === 'soundtrack'));
  let dlcDepots = $derived(depots.filter(d => getContentCategory(d) === 'dlc'));
  let coreDepots = $derived(depots.filter(d => {
    const cat = getContentCategory(d);
    return cat === 'core' || (!cat.startsWith('locale:') && cat !== 'optional_hd' && cat !== 'soundtrack' && cat !== 'dlc');
  }));

  let availableLanguages = $derived(Array.from(new Set(
    languageDepots.map(d => getDepotLanguage(d)).filter(Boolean)
  )).sort());

  let filteredDepots = $derived(depots.filter(depot => {
    if (selectedCategoryFilter === 'all') return true;
    const cat = getContentCategory(depot);
    if (selectedCategoryFilter === 'languages') return cat.startsWith('locale:');
    if (selectedCategoryFilter === 'optional') return cat === 'optional_hd';
    if (selectedCategoryFilter === 'soundtrack') return cat === 'soundtrack';
    if (selectedCategoryFilter === 'dlc') return cat === 'dlc';
    if (selectedCategoryFilter === 'core') return cat === 'core' || (!cat.startsWith('locale:') && cat !== 'optional_hd' && cat !== 'soundtrack' && cat !== 'dlc');
    return true;
  }));

  let selectedDepotsSize = $derived(depots
    .filter(d => selectedDepotIds.has(Number(d.depot_id)))
    .reduce((acc, d) => acc + (d.size || 0), 0));
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
          onclick={handleSyncDLCs}
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
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3.5 bg-surface-header border border-surface-border rounded-xl text-xs">
    <div class="space-y-0.5">
      <strong class="text-neutral-200 block font-bold">DLC & Content Depots</strong>
      <span class="text-neutral-400 text-2xs">
        {#if isInstalled}
          {installedCount} of {depots.length} depots installed ({formatBytes(totalInstalledSize)})
        {:else}
          {depots.length} total depots discovered • {selectedDepotIds.size} selected for download ({formatBytes(selectedDepotsSize)})
        {/if}
      </span>
    </div>
    <div class="flex items-center space-x-2">
      {#if !isInstalled && appID && depots.length > 0}
        <button
          type="button"
          onclick={handleDownloadSelected}
          disabled={isActionRunning || selectedDepotIds.size === 0}
          class="px-3 py-1.5 rounded-lg bg-white hover:bg-neutral-200 text-neutral-950 font-bold text-2xs transition-colors flex items-center space-x-1.5 cursor-pointer shadow-sm"
        >
          <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>Download Selected ({selectedDepotIds.size})</span>
        </button>
      {/if}
      <button
        type="button"
        onclick={loadDepots}
        disabled={isLoading}
        class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border transition-colors font-bold text-2xs flex items-center space-x-1.5 cursor-pointer"
      >
        <svg class="w-3 h-3 {isLoading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>Refresh</span>
      </button>
    </div>
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

  <!-- Category Filter Pills & Language Preference -->
  {#if depots.length > 0}
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 pb-1">
      <div class="flex items-center space-x-1.5 overflow-x-auto py-0.5">
        <button
          type="button"
          onclick={() => { selectedCategoryFilter = 'all'; }}
          class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all cursor-pointer {selectedCategoryFilter === 'all' ? 'bg-neutral-200 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
        >
          All ({depots.length})
        </button>

        {#if languageDepots.length > 0}
          <button
            type="button"
            onclick={() => { selectedCategoryFilter = 'languages'; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'languages' ? 'bg-blue-500 text-white font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
          >
            <span>🌐 Languages</span>
            <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{languageDepots.length}</span>
          </button>
        {/if}

        {#if optionalDepots.length > 0}
          <button
            type="button"
            onclick={() => { selectedCategoryFilter = 'optional'; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'optional' ? 'bg-amber-500 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
          >
            <span>🎨 High-Res / Optional</span>
            <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{optionalDepots.length}</span>
          </button>
        {/if}

        {#if soundtrackDepots.length > 0}
          <button
            type="button"
            onclick={() => { selectedCategoryFilter = 'soundtrack'; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'soundtrack' ? 'bg-purple-500 text-white font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
          >
            <span>🎵 Soundtracks</span>
            <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{soundtrackDepots.length}</span>
          </button>
        {/if}

        {#if dlcDepots.length > 0}
          <button
            type="button"
            onclick={() => { selectedCategoryFilter = 'dlc'; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'dlc' ? 'bg-emerald-500 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
          >
            <span>📦 DLCs</span>
            <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{dlcDepots.length}</span>
          </button>
        {/if}

        {#if coreDepots.length > 0}
          <button
            type="button"
            onclick={() => { selectedCategoryFilter = 'core'; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'core' ? 'bg-neutral-200 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
          >
            <span>Core Content</span>
            <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{coreDepots.length}</span>
          </button>
        {/if}
      </div>

      {#if availableLanguages.length > 0}
        <div class="flex items-center space-x-2 text-2xs text-neutral-400 flex-shrink-0">
          <span class="font-bold">Voice Pack:</span>
          <select
            value={configuredLanguage}
            onchange={(e) => handleLanguageChange(e.target.value)}
            class="bg-surface-card border border-surface-border text-neutral-200 rounded-lg px-2.5 py-1 text-2xs outline-none cursor-pointer hover:border-neutral-500 transition-colors"
          >
            {#each availableLanguages as lang}
              <option value={lang}>{lang.charAt(0).toUpperCase() + lang.slice(1)}</option>
            {/each}
          </select>
        </div>
      {/if}
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
      {#each filteredDepots as depot (depot.depot_id)}
        <div class="p-3 bg-surface-card border border-surface-border hover:border-surface-borderHighlight rounded-xl flex items-center justify-between gap-3 transition-colors">
          <div class="flex items-center space-x-3 min-w-0">
            {#if !isInstalled}
              <!-- Pre-download selection checkbox -->
              <label class="flex items-center cursor-pointer flex-shrink-0" title="Select depot for download">
                <input
                  type="checkbox"
                  checked={selectedDepotIds.has(Number(depot.depot_id))}
                  onchange={() => toggleDepotSelection(depot.depot_id)}
                  class="w-4 h-4 rounded bg-surface-input border border-surface-border text-white cursor-pointer accent-white"
                />
              </label>
            {:else}
              <!-- Installed Status Indicator -->
              <div class="w-4 h-4 rounded-md flex items-center justify-center flex-shrink-0 {depot.installed ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-700/60' : 'bg-surface-header text-neutral-600 border border-surface-border'}">
                {#if depot.installed}
                  <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                  </svg>
                {/if}
              </div>
            {/if}

            <div class="space-y-1 min-w-0">
              <div class="flex items-center space-x-2 flex-wrap gap-y-1">
                <span class="font-bold text-neutral-100 text-xs">{depot.name || `Depot ${depot.depot_id}`}</span>
                <span class="px-2 py-0.5 rounded-full text-3xs font-bold border {getCategoryBadgeClass(depot)}">
                  {getCategoryLabel(depot)}
                </span>
                {#if depot.osarch}
                  <span class="px-1.5 py-0.5 rounded text-3xs font-bold bg-surface-header text-neutral-400 border border-surface-border">
                    {depot.osarch}-bit
                  </span>
                {/if}
                {#if isDefaultLanguageDepot(depot)}
                  <span class="px-1.5 py-0.5 rounded text-3xs font-bold bg-blue-950/40 text-blue-300 border border-blue-800/40">
                    Active Locale
                  </span>
                {/if}
              </div>
              <p class="text-2xs text-neutral-400 font-mono">
                {#if depot.file_count > 0}
                  {depot.file_count} files • {formatBytes(depot.size || 0)}
                {:else if depot.size > 0}
                  Estimated {formatBytes(depot.size)}
                  {#if depot.download_size && depot.download_size !== depot.size}
                    ({formatBytes(depot.download_size)} download)
                  {/if}
                {:else}
                  0 files • 0 B
                {/if}
              </p>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center space-x-2 flex-shrink-0">
            {#if depot.category === 'soundtrack'}
              <button
                type="button"
                onclick={() => handleDownloadSoundtrack(depot)}
                disabled={isActionRunning}
                class="px-2.5 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border font-bold text-2xs transition-colors flex items-center space-x-1 cursor-pointer"
              >
                <span>🎵 Save to Music</span>
              </button>
            {:else if !depot.installed}
              <button
                type="button"
                onclick={() => handleDownloadDepot(depot)}
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
                {#if depots.length > 1 || getContentCategory(depot) !== 'core' || depot.category !== 'common' || depot.osarch}
                  <button
                    type="button"
                    onclick={() => handleUninstallDepot(depot)}
                    disabled={isActionRunning}
                    title="Remove depot files to free space"
                    class="px-2 py-1 rounded-lg bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 border border-rose-800/50 font-bold text-3xs transition-colors cursor-pointer"
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
