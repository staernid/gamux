<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { tasksStore } from '../../../../stores/tasks.js';
  import { notificationStore } from '../../../../stores/notifications.js';
  import { configStore } from '../../../../stores/config.js';
  import { modalStore } from '../../../../stores/modal.js';
  import { updateGameInStore } from '../../../../stores/library.js';
  import { formatBytes } from '../../../../utils/format.js';
  import { isSamePath, normalizeDlc } from '../../../../utils/game.js';
  import {
    getContentCategory,
    getDepotLanguage,
    computeDefaultDepotSelections,
    filterDepotsByCategory,
    isDefaultLanguageDepot,
    groupDepotsByCategory,
    updateSelectedLanguageDepots,
  } from '../../../../utils/depot.js';
  import HubDLCConfiguredList from './HubDLCConfiguredList.svelte';
  import HubDepotSummaryBanner from './HubDepotSummaryBanner.svelte';
  import HubDepotFilters from './HubDepotFilters.svelte';
  import HubDepotRow from './HubDepotRow.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../../../types/domain').Game} [game]
   * @property {(updated: import('../../../../types/domain').Game) => void} [onupdateGame]
   */

  /** @type {Props} */
  let { game = null, onupdateGame = undefined } = $props();

  let depots = $state([]);
  let dlcs = $state([]);
  let isLoading = $state(true);
  let isLoadingDLCs = false;
  let isSyncingDLCs = $state(false);
  let isActionRunning = $state(false);
  let selectedCategoryFilter = $state('all');
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
      dlcs = Array.isArray(res) ? res.map(normalizeDlc).filter(Boolean) : [];
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
        selectedDepotIds = computeDefaultDepotSelections(depots, configuredLanguage);
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
      selectedDepotIds = updateSelectedLanguageDepots(depots, selectedDepotIds, newLang);
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

  let grouped = $derived(groupDepotsByCategory(depots));
  let availableLanguages = $derived(Array.from(new Set(
    grouped.language.map(d => getDepotLanguage(d)).filter(Boolean)
  )).sort());

  let filteredDepots = $derived(filterDepotsByCategory(depots, selectedCategoryFilter));

  let selectedDepotsSize = $derived(depots
    .filter(d => selectedDepotIds.has(Number(d.depot_id)))
    .reduce((acc, d) => acc + (d.size || 0), 0));
</script>

<div class="space-y-4">
  <!-- Configured DLCs Section -->
  <HubDLCConfiguredList
    {dlcs}
    appId={appID}
    isSyncing={isSyncingDLCs}
    onsync={handleSyncDLCs}
  />

  <!-- Summary Banner -->
  <HubDepotSummaryBanner
    {isInstalled}
    {installedCount}
    totalDepotsCount={depots.length}
    {totalInstalledSize}
    selectedCount={selectedDepotIds.size}
    {selectedDepotsSize}
    appId={appID}
    {isActionRunning}
    {isLoading}
    {activeTransfer}
    ondownloadSelected={handleDownloadSelected}
    onrefresh={loadDepots}
  />

  <!-- Category Filter Pills & Language Preference -->
  {#if depots.length > 0}
    <HubDepotFilters
      totalDepotsCount={depots.length}
      languageDepotsCount={grouped.language.length}
      optionalDepotsCount={grouped.optional.length}
      soundtrackDepotsCount={grouped.soundtrack.length}
      dlcDepotsCount={grouped.dlc.length}
      coreDepotsCount={grouped.core.length}
      {selectedCategoryFilter}
      {availableLanguages}
      {configuredLanguage}
      onselectFilter={(filter) => { selectedCategoryFilter = filter; }}
      onchangeLanguage={handleLanguageChange}
    />
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
        <HubDepotRow
          {depot}
          {isInstalled}
          isSelected={selectedDepotIds.has(Number(depot.depot_id))}
          {isActionRunning}
          canUninstall={depots.length > 1 || getContentCategory(depot) !== 'core' || depot.category !== 'common' || Boolean(depot.osarch)}
          isDefaultLocale={isDefaultLanguageDepot(depot, configuredLanguage)}
          ontoggle={() => toggleDepotSelection(depot.depot_id)}
          ondownloadSoundtrack={() => handleDownloadSoundtrack(depot)}
          ondownload={() => handleDownloadDepot(depot)}
          onuninstall={() => handleUninstallDepot(depot)}
        />
      {/each}
    </div>
  {/if}
</div>
