<script>
  import { onMount } from 'svelte';
  import ModalShell from '../common/ModalShell.svelte';
  import { libraryPaths, libraryPath } from '../../stores/library/paths.js';
  import { configStore } from '../../stores/config.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';
  import { updateGameInStore, selectedGame } from '../../stores/library.js';

  /**
   * @typedef {Object} Props
   * @property {any} game
   * @property {() => void} [onclose]
   */

  /** @type {Props} */
  let { game = null, onclose = null } = $props();

  let depots = $state([]);
  let isLoading = $state(true);
  let isStarting = $state(false);
  let dontAskAgain = $state(false);
  let showDlcList = $state(false);

  // Available libraries
  let configuredLibraries = $derived($libraryPaths && $libraryPaths.length > 0 ? $libraryPaths : [$libraryPath || '/home/staernid/Games'].filter(Boolean));
  let selectedLibrary = $state('');

  // Sanitized game folder name
  let cleanGameName = $derived(
    game?.name
      ? game.name.replace(/[™®©]/g, '').replace(/[:/\\?*<>|"]/g, '').replace(/\s+/g, ' ').trim()
      : 'Game'
  );

  let targetDir = $derived(
    game?.gameDir && !game.isVirtual
      ? game.gameDir
      : `${selectedLibrary || configuredLibraries[0] || '/home/staernid/Games'}/${cleanGameName}`
  );

  // Preferred language
  // svelte-ignore state_referenced_locally
  let selectedLanguage = $state(
    (game?.currentLanguage || configStore.current?.language || 'english').toLowerCase()
  );

  let selectedDepotIds = $state(new Set());

  let targetPlatform = $derived((game?.platform || 'win64').toLowerCase());

  function matchesPlatform(depot, plat) {
    const osList = (depot.oslist || depot.category || '').toLowerCase();
    if (!osList || osList === 'common' || osList === 'all') {
      return true;
    }
    if (plat.startsWith('win')) {
      if (osList === 'linux' || osList === 'macos' || osList === 'osx') return false;
      return true;
    }
    if (plat.startsWith('linux')) {
      if (osList === 'windows' || osList === 'win' || osList === 'macos' || osList === 'osx') return false;
      return true;
    }
    if (plat.startsWith('mac') || plat.startsWith('osx')) {
      if (osList === 'windows' || osList === 'win' || osList === 'linux') return false;
      return true;
    }
    return true;
  }

  // Filter depots for target platform & prefer 64-bit binaries on 64-bit systems
  let platformDepots = $derived.by(() => {
    const plat = targetPlatform;
    const is64 = plat.includes('64');
    const matchingOS = depots.filter(d => matchesPlatform(d, plat));

    if (is64) {
      return matchingOS.filter(d => {
        if (d.osarch === '32') {
          const has64 = matchingOS.some(other =>
            other.depot_id !== d.depot_id &&
            other.osarch === '64' &&
            Number(other.dlc_app_id || 0) === Number(d.dlc_app_id || 0)
          );
          if (has64) return false;
        }
        return true;
      });
    }
    return matchingOS;
  });

  onMount(async () => {
    selectedLibrary = configuredLibraries[0] || '/home/staernid/Games';
    await fetchDepots();
  });

  function getContentCategory(depot) {
    if (depot.content_category) return depot.content_category;
    if (depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0)) return 'dlc';
    if (depot.category === 'soundtrack') return 'soundtrack';
    return 'core';
  }

  function getDepotLanguage(depot) {
    if (depot.language) return depot.language;
    const cat = getContentCategory(depot);
    if (cat.startsWith('locale:')) return cat.slice(7);
    return '';
  }

  function computeDefaultSelections(depotList, lang) {
    const selected = new Set();
    const targetLang = (lang || 'english').toLowerCase();

    for (const d of depotList) {
      const cat = getContentCategory(d);
      // Core, common, and DLC depots are enabled by default
      if (cat === 'core' || cat === 'common' || cat === 'dlc' || d.is_dlc || (d.dlc_app_id && Number(d.dlc_app_id) > 0)) {
        selected.add(Number(d.depot_id));
      } else if (cat.startsWith('locale:')) {
        const dLang = getDepotLanguage(d).toLowerCase();
        if (dLang === targetLang || targetLang.startsWith(dLang) || dLang.startsWith(targetLang)) {
          selected.add(Number(d.depot_id));
        }
      }
    }

    // Fallback to English if no matching language depot found
    const hasLangDepot = Array.from(selected).some(id => {
      const d = depotList.find(x => Number(x.depot_id) === id);
      return d && getContentCategory(d).startsWith('locale:');
    });
    if (!hasLangDepot) {
      const engDepot = depotList.find(d => getDepotLanguage(d).toLowerCase() === 'english');
      if (engDepot) selected.add(Number(engDepot.depot_id));
    }

    return selected;
  }

  async function fetchDepots() {
    const rawAppId = game?.appId || game?.app_id || game?.AppID;
    const strAppId = rawAppId ? String(rawAppId).trim() : '';
    if (!strAppId && !game?.gameDir) {
      isLoading = false;
      return;
    }

    isLoading = true;
    try {
      let res;
      // If we have an AppID, getAppDepots yields the authoritative catalog with DLCs
      if (strAppId && strAppId !== '0') {
        res = await AppAPI.getAppDepots(strAppId);
      } else if (game?.gameDir) {
        res = await AppAPI.getGameDepots(game.gameDir);
      }
      depots = Array.isArray(res) ? res : (game?.depots || []);
      selectedDepotIds = computeDefaultSelections(platformDepots, selectedLanguage);
    } catch (err) {
      console.warn('Failed to load depots for install dialog:', err);
      depots = game?.depots || [];
      selectedDepotIds = computeDefaultSelections(platformDepots, selectedLanguage);
    } finally {
      isLoading = false;
    }
  }

  function handleLanguageChange(newLang) {
    selectedLanguage = newLang;
    selectedDepotIds = computeDefaultSelections(platformDepots, newLang);
  }

  // DLC Groups: aggregate depots by DLC AppID so users see clean DLC packages
  let dlcGroups = $derived.by(() => {
    const groups = new Map();
    for (const d of platformDepots) {
      if (getContentCategory(d) === 'dlc' || d.is_dlc || (d.dlc_app_id && Number(d.dlc_app_id) > 0)) {
        const key = d.dlc_app_id ? Number(d.dlc_app_id) : Number(d.depot_id);
        if (!groups.has(key)) {
          let title = d.dlc_name || d.name || `DLC ${key}`;
          if (d.dlc_name) {
            title = d.dlc_name;
          }
          groups.set(key, {
            id: key,
            name: title,
            size: 0,
            depotIds: [],
          });
        }
        const g = groups.get(key);
        g.size += (d.size || 0);
        g.depotIds.push(Number(d.depot_id));
        if (d.dlc_name) {
          g.name = d.dlc_name;
        }
      }
    }
    return Array.from(groups.values());
  });

  function isDlcGroupSelected(group) {
    return group.depotIds.length > 0 && group.depotIds.every(id => selectedDepotIds.has(id));
  }

  function toggleDlcGroup(group) {
    const next = new Set(selectedDepotIds);
    const allSelected = isDlcGroupSelected(group);
    if (allSelected) {
      for (const id of group.depotIds) {
        next.delete(id);
      }
    } else {
      for (const id of group.depotIds) {
        next.add(id);
      }
    }
    selectedDepotIds = next;
  }

  let selectedDlcGroupCount = $derived(dlcGroups.filter(g => isDlcGroupSelected(g)).length);

  // Derived language depots for platform
  let languageDepots = $derived(platformDepots.filter(d => getContentCategory(d).startsWith('locale:')));
  let availableLanguages = $derived(Array.from(new Set(
    languageDepots.map(d => getDepotLanguage(d)).filter(Boolean)
  )).sort());

  let totalDownloadSize = $derived(
    platformDepots
      .filter(d => selectedDepotIds.has(Number(d.depot_id)))
      .reduce((acc, d) => acc + (d.download_size || d.downloadSize || d.size || 0), 0)
  );

  let totalDiskSize = $derived(
    platformDepots
      .filter(d => selectedDepotIds.has(Number(d.depot_id)))
      .reduce((acc, d) => acc + (d.size || 0), 0)
  );

  async function handleStartInstall() {
    if (isStarting) return;
    const rawAppId = game?.appId || game?.app_id || game?.AppID;
    const appID = parseInt(rawAppId, 10);
    if (!appID || isNaN(appID)) {
      notificationStore.error('Install Failed', 'Invalid Steam AppID');
      return;
    }

    isStarting = true;

    if (dontAskAgain) {
      configStore.setSkipInstallConfirm(true);
    }

    let finalDir = targetDir;
    const libRoot = selectedLibrary || configuredLibraries[0] || '/home/staernid/Games';

    // Ensure game stub is registered if virtual or lacking directory
    if (!game?.gameDir || game.isVirtual) {
      try {
        const stub = await AppAPI.createGameStub(
          libRoot,
          String(appID),
          game.name,
          game.store || 'Steam',
          game.branch || 'public'
        );
        if (stub) {
          const normalized = updateGameInStore(stub);
          if (normalized?.gameDir) {
            selectedGame.set(normalized);
            finalDir = normalized.gameDir;
          } else if (stub.game_dir || stub.gameDir) {
            finalDir = stub.game_dir || stub.gameDir;
          }
        }
      } catch (err) {
        console.warn('Could not initialize library stub prior to download:', err);
      }
    }

    try {
      const selected = Array.from(selectedDepotIds);
      await tasksStore.start({
        app_id: appID,
        game_name: game.name,
        target_dir: finalDir,
        platform: game.platform || 'win64',
        included_depots: selected.length > 0 ? selected : undefined,
        language: selectedLanguage,
        auto_apply: true,
        store_type: (game.store || 'steam').toLowerCase(),
        branch: game.branch || 'public',
      });
      onclose?.();
    } catch (err) {
      notificationStore.error('Download Failed', err?.message || String(err));
    } finally {
      isStarting = false;
    }
  }
</script>

<ModalShell
  title="Install {game?.name || 'Game'}"
  badge={game?.store || 'Steam'}
  maxWidth="max-w-xl"
  padding="p-4 sm:p-6"
  {onclose}
>
  <div class="w-full max-w-xl mx-auto bg-surface-header border border-surface-border rounded-2xl p-5 sm:p-6 shadow-2xl space-y-5 text-xs text-neutral-300">
    <!-- Target Library Directory Selection -->
    <div class="space-y-1.5">
      <label for="install-library-select" class="block font-bold text-neutral-200 text-2xs uppercase tracking-wider">
        Install Location
      </label>
      {#if configuredLibraries.length > 1}
        <select
          id="install-library-select"
          bind:value={selectedLibrary}
          class="w-full bg-surface-input border border-surface-border text-neutral-100 rounded-xl px-3 py-2 text-xs outline-none focus:border-neutral-400 transition-colors cursor-pointer"
        >
          {#each configuredLibraries as lib}
            <option value={lib}>{lib}</option>
          {/each}
        </select>
      {:else}
        <div class="p-2.5 bg-surface-input border border-surface-border rounded-xl font-mono text-2xs text-neutral-300 truncate">
          {selectedLibrary || configuredLibraries[0] || '/home/staernid/Games'}
        </div>
      {/if}
      <div class="text-3xs text-neutral-400 font-mono truncate px-1">
        Target: <span class="text-neutral-200">{targetDir}</span>
      </div>
    </div>

    <!-- Language Selector if multiple languages available -->
    {#if availableLanguages.length > 1}
      <div class="space-y-1.5">
        <label for="install-language-select" class="block font-bold text-neutral-200 text-2xs uppercase tracking-wider">
          Language / Audio Pack
        </label>
        <select
          id="install-language-select"
          value={selectedLanguage}
          onchange={(e) => handleLanguageChange(e.target.value)}
          class="w-full bg-surface-input border border-surface-border text-neutral-100 rounded-xl px-3 py-2 text-xs outline-none focus:border-neutral-400 transition-colors cursor-pointer capitalize"
        >
          {#each availableLanguages as lang}
            <option value={lang}>{lang}</option>
          {/each}
        </select>
      </div>
    {/if}

    <!-- DLC Inclusion Summary & Toggle -->
    {#if dlcGroups.length > 0}
      <div class="p-3.5 bg-surface-card border border-surface-border rounded-xl space-y-2.5">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <span class="px-2 py-0.5 rounded text-3xs font-bold bg-emerald-950/60 text-emerald-400 border border-emerald-800/40">
              DLC
            </span>
            <span class="font-bold text-neutral-200 text-xs">
              {selectedDlcGroupCount} of {dlcGroups.length} DLCs Included
            </span>
          </div>
          <button
            type="button"
            onclick={() => { showDlcList = !showDlcList; }}
            class="text-3xs font-bold text-neutral-400 hover:text-white transition-colors cursor-pointer"
          >
            {showDlcList ? 'Hide List ▲' : 'Customize DLCs ▼'}
          </button>
        </div>

        {#if showDlcList}
          <div class="space-y-1.5 max-h-48 overflow-y-auto pr-1 pt-1 border-t border-surface-border/60">
            {#each dlcGroups as group (group.id)}
              <label class="flex items-center justify-between p-2 rounded-lg bg-surface-input hover:bg-surface-cardHover transition-colors cursor-pointer select-none">
                <div class="flex items-center space-x-2.5 min-w-0">
                  <input
                    type="checkbox"
                    checked={isDlcGroupSelected(group)}
                    onchange={() => toggleDlcGroup(group)}
                    class="w-3.5 h-3.5 rounded bg-surface-input border border-surface-border text-white cursor-pointer accent-white"
                  />
                  <span class="text-2xs font-medium text-neutral-200 truncate">
                    {group.name}
                  </span>
                </div>
                {#if group.size > 0}
                  <span class="text-3xs font-mono text-neutral-400 flex-shrink-0 ml-2">
                    {formatBytes(group.size)}
                  </span>
                {/if}
              </label>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Storage & Download Metrics -->
    <div class="flex items-center justify-between p-3 bg-surface-card border border-surface-border rounded-xl text-2xs font-medium">
      <div class="space-y-0.5">
        <span class="text-neutral-400 block text-3xs uppercase font-mono">Download Size</span>
        <span class="text-neutral-100 font-bold font-mono text-xs">
          {isLoading ? 'Calculating...' : (totalDownloadSize > 0 ? formatBytes(totalDownloadSize) : 'Standard Content')}
        </span>
      </div>
      <div class="text-right space-y-0.5">
        <span class="text-neutral-400 block text-3xs uppercase font-mono">Disk Space Required</span>
        <span class="text-neutral-100 font-bold font-mono text-xs">
          {isLoading ? 'Calculating...' : (totalDiskSize > 0 ? formatBytes(totalDiskSize) : 'Auto')}
        </span>
      </div>
    </div>

    <!-- Skippable Preference Checkbox -->
    <label class="flex items-center space-x-2.5 pt-1 cursor-pointer select-none">
      <input
        type="checkbox"
        bind:checked={dontAskAgain}
        class="w-3.5 h-3.5 rounded bg-surface-input border border-surface-border text-white cursor-pointer accent-white"
      />
      <span class="text-2xs text-neutral-400 hover:text-neutral-300 transition-colors">
        Don't ask again (download directly with sane defaults in the future)
      </span>
    </label>

    <!-- Modal Actions -->
    <div class="flex items-center justify-end space-x-2.5 pt-3 border-t border-surface-border">
      <button
        type="button"
        onclick={onclose}
        disabled={isStarting}
        class="px-4 py-2 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-xs font-bold text-neutral-300 hover:text-white transition-colors cursor-pointer"
      >
        Cancel
      </button>

      <button
        type="button"
        onclick={handleStartInstall}
        disabled={isStarting || isLoading}
        class="px-5 py-2 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 text-xs font-black transition-all shadow-lg flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isStarting}
          <svg class="w-3.5 h-3.5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Starting...</span>
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>Install Game</span>
        {/if}
      </button>
    </div>
  </div>
</ModalShell>
