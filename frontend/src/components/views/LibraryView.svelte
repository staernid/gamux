<script>
  import { get } from 'svelte/store';
  import { onDestroy } from 'svelte';
  import CoverCard from '../blocks/CoverCard.svelte';
  import SearchFilterBar from '../blocks/SearchFilterBar.svelte';
  import Icon from '../common/Icon.svelte';
  import CatalogCard from '../stage/catalog/CatalogCard.svelte';
  import CatalogEmptyState from '../stage/catalog/CatalogEmptyState.svelte';
  import {
    games,
    filteredGames,
    selectedGame,
    activeView,
    sortBy,
    activeFilter,
    searchQuery,
    stats
  } from '../../stores/library.js';
  import {
    getPlatformColors,
    isSamePath,
    normalizeGame
  } from '../../utils/game.js';
  import { playTick, playSelect } from '../../utils/audio.js';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';

  /**
   * @typedef {Object} Props
   * @property {'local' | 'catalog'} [mode]
   */

  /** @type {Props} */
  let { mode = 'local' } = $props();

  let effectiveMode = $derived(mode || ($activeView === 'catalog' ? 'catalog' : 'local'));

  // Local mode states
  let searchInputEl = $state();
  let focusedIndex = 0;
  let isCheckingUpdates = $state(false);

  async function handleCheckLibraryUpdates() {
    if (isCheckingUpdates) return;
    isCheckingUpdates = true;
    try {
      const rep = await AppAPI.checkLibraryUpdates();
      const count = rep?.updates_found ?? 0;
      if (count > 0) {
        notificationStore.info('Updates Found', `${count} game update${count === 1 ? '' : 's'} available.`);
      } else {
        notificationStore.info('Up to Date', 'All games in library are up to date.');
      }
    } catch (err) {
      notificationStore.error('Update Check Failed', err?.message || String(err));
    } finally {
      isCheckingUpdates = false;
    }
  }

  const sortOptions = [
    { id: 'name_asc', label: 'Name (A-Z)' },
    { id: 'name_desc', label: 'Name (Z-A)' },
    { id: 'last_played_desc', label: 'Last Played' },
    { id: 'playtime_desc', label: 'Playtime' },
    { id: 'install_desc', label: 'Installed' },
    { id: 'size_desc', label: 'Size' }
  ];

  // Online Catalog mode states
  let catalogInputEl = $state();
  let catalogQuery = $state('');
  let focusedCatalogIndex = $state(0);
  let storeResults = $state([]);
  let isSearchingStore = $state(false);
  let storeSearchTimeout = $state(null);
  let lastStoreQuery = $state('');
  let activeStoreFilter = $state('all'); // 'all' | 'steam' | 'gog'

  let allGames = $derived($games || []);

  // Filtered store results by provider
  let filteredStoreResults = $derived(storeResults.filter(item => {
    if (activeStoreFilter === 'all') return true;
    const store = (item.store || 'steam').toLowerCase();
    return store === activeStoreFilter;
  }));

  let steamCount = $derived(storeResults.filter(item => (item.store || 'steam').toLowerCase() === 'steam').length);
  let gogCount = $derived(storeResults.filter(item => (item.store || '').toLowerCase() === 'gog').length);

  // Debounced store catalog search
  $effect(() => {
    if (effectiveMode !== 'catalog') return;
    const q = (catalogQuery || '').trim();
    if (q !== lastStoreQuery) {
      lastStoreQuery = q;
      if (storeSearchTimeout) clearTimeout(storeSearchTimeout);
      if (!q) {
        storeResults = [];
        isSearchingStore = false;
      } else {
        isSearchingStore = true;
        storeSearchTimeout = setTimeout(async () => {
          try {
            const res = await AppAPI.searchStoreGames(q);
            if (lastStoreQuery === q) {
              storeResults = res || [];
            }
          } catch (err) {
            console.warn('Store search failed:', err);
            if (lastStoreQuery === q) storeResults = [];
          } finally {
            if (lastStoreQuery === q) isSearchingStore = false;
          }
        }, 250);
      }
    }
  });

  onDestroy(() => {
    if (storeSearchTimeout) clearTimeout(storeSearchTimeout);
  });

  function getMatchingLocalGame(res) {
    if (!res) return null;
    const resId = String(res.id);
    const resStore = (res.store || 'steam').toLowerCase();
    return allGames.find(g => {
      if (resStore === 'steam' && String(g.appId) === resId) return true;
      if (resStore === 'gog' && String(g.gogId) === resId) return true;
      if (g.name && res.name && g.name.toLowerCase() === res.name.toLowerCase()) return true;
      return false;
    });
  }

  function handleSelectCatalogItem(item) {
    if (!item) return;
    const localMatch = getMatchingLocalGame(item);
    if (localMatch) {
      selectedGame.set(localMatch);
    } else {
      const storeName = item.store ? (item.store.charAt(0).toUpperCase() + item.store.slice(1)) : 'Steam';
      const virtualGame = normalizeGame({
        name: item.name,
        app_id: String(item.id),
        store: storeName,
        source: storeName,
        cover_url: item.poster_url || '',
        release_date: item.release_date || '',
        is_installed: false,
        is_stub: false,
        is_virtual: true,
        is_ready: false,
        file_count: 0,
        disk_size_bytes: 0,
        platform: 'win64'
      });
      selectedGame.set(virtualGame);
    }
    playSelect();
    activeView.set('game-hub');
  }

  function handleCatalogSuggest(suggestedQuery) {
    catalogQuery = suggestedQuery;
    if (catalogInputEl) {
      catalogInputEl.focus();
    }
  }

  function openGameHub(game) {
    if (!game) return;
    selectedGame.set(game);
    playSelect();
    activeView.set('game-hub');
  }

  export function step(dir) {
    if (effectiveMode === 'catalog') {
      const list = filteredStoreResults || [];
      if (list.length === 0) return;
      focusedCatalogIndex = (focusedCatalogIndex + dir + list.length) % list.length;
      playTick();
      return;
    }
    const list = $filteredGames || [];
    if (list.length === 0) return;
    const current = get(selectedGame);
    const currentIndex = current ? list.findIndex(g => isSamePath(g.gameDir, current.gameDir)) : -1;
    const nextIndex = (currentIndex + dir + list.length) % list.length;
    playTick();
    selectedGame.set(list[nextIndex]);
    focusedIndex = nextIndex;
  }

  export function addRotaryDelta(_delta) {}

  export function focusSearch() {
    setTimeout(() => {
      if (effectiveMode === 'catalog') {
        catalogInputEl?.focus();
      } else {
        searchInputEl?.focus();
      }
    }, 50);
  }
</script>

<div class="flex-1 flex flex-col min-h-0 w-full relative">
  <!-- Desktop Grid View Mode (Unified Solid Canvas) -->
  <div class="flex-1 flex flex-col min-h-0 pt-16 sm:pt-20 px-6 sm:px-10 pb-6 max-w-7xl mx-auto w-full animate-fade-in">
    <!-- Toolbar: Search, Filters & Sorters -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-5 border-b border-surface-border">
      {#if effectiveMode === 'local'}
        <!-- Left: Search Filter Bar -->
        <SearchFilterBar
          bind:this={searchInputEl}
          bind:query={$searchQuery}
          placeholder="Search library by title, appid, platform..."
          totalCount={$games.length}
          filteredCount={$filteredGames.length}
          className="flex-1 max-w-md"
        />

        <!-- Right: Filter & Sort Dropdowns -->
        <div class="flex items-center space-x-3 justify-between md:justify-end">
          <!-- Check Updates Button -->
          <button
            type="button"
            onclick={handleCheckLibraryUpdates}
            disabled={isCheckingUpdates}
            class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
            title="Check library games for Steam updates"
          >
            <svg class="w-3.5 h-3.5 {isCheckingUpdates ? 'animate-spin text-sky-400' : 'text-neutral-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span class="hidden sm:inline">{isCheckingUpdates ? 'Checking...' : 'Check Updates'}</span>
          </button>

          <!-- Status Filter Dropdown -->
          <div class="flex items-center space-x-2">
            <span class="text-2xs font-mono text-neutral-500 uppercase tracking-wider font-bold">Filter:</span>
            <div class="relative">
              <select
                bind:value={$activeFilter}
                aria-label="Filter games by status"
                class="bg-surface-card border border-surface-border text-neutral-200 rounded-lg px-3 py-1.5 text-xs outline-none cursor-pointer pr-8 appearance-none hover:border-neutral-500 transition-colors"
              >
                <option value="all">All</option>
                <option value="patched">Ready {$stats.patchedCount > 0 ? `(${$stats.patchedCount})` : ''}</option>
                <option value="clean">Unpatched {$stats.cleanCount > 0 ? `(${$stats.cleanCount})` : ''}</option>
                {#if $stats.updatesCount > 0}
                  <option value="updates">Updates ({$stats.updatesCount})</option>
                {/if}
              </select>
              <div class="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none text-neutral-400">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>
          </div>

          <!-- Sort Dropdown -->
          <div class="flex items-center space-x-2">
            <span class="text-2xs font-mono text-neutral-500 uppercase tracking-wider font-bold">Sort:</span>
            <div class="relative">
              <select
                bind:value={$sortBy}
                aria-label="Sort library games"
                class="bg-surface-card border border-surface-border text-neutral-200 rounded-lg px-3 py-1.5 text-xs outline-none cursor-pointer pr-8 appearance-none hover:border-neutral-500 transition-colors"
              >
                {#each sortOptions as opt}
                  <option value={opt.id}>{opt.label}</option>
                {/each}
              </select>
              <div class="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none text-neutral-400">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <!-- Catalog Mode: Search Bar -->
        <div class="relative flex-1 max-w-md flex items-center">
          <div class="absolute left-3.5 pointer-events-none text-neutral-400 flex items-center">
            <Icon name="globe" className="w-4 h-4" />
          </div>
          <input
            bind:this={catalogInputEl}
            bind:value={catalogQuery}
            type="text"
            placeholder="Search Steam, GOG, RAWG catalogs by title or AppID..."
            class="w-full bg-surface-input border border-surface-border focus:border-neutral-400 rounded-xl pl-10 pr-9 py-2 text-xs text-white outline-none font-mono transition-colors shadow-sm placeholder:text-neutral-500"
          />
          {#if catalogQuery}
            <button
              type="button"
              onclick={() => { catalogQuery = ''; catalogInputEl?.focus(); }}
              class="absolute right-2.5 text-neutral-400 hover:text-white p-1 rounded-lg transition-colors flex items-center justify-center cursor-pointer"
              aria-label="Clear search"
            >
              <Icon name="close" className="w-3.5 h-3.5" />
            </button>
          {/if}
        </div>

        <!-- Catalog Mode: Store Filter Dropdown & Status Indicator -->
        <div class="flex items-center space-x-3 justify-between md:justify-end">
          {#if isSearchingStore}
            <div class="flex items-center space-x-1.5 text-neutral-400 text-2xs font-mono">
              <span class="inline-block w-2 h-2 rounded-full bg-white animate-ping"></span>
              <span>Searching...</span>
            </div>
          {/if}

          <div class="flex items-center space-x-2">
            <span class="text-2xs font-mono text-neutral-500 uppercase tracking-wider font-bold">Store:</span>
            <div class="relative">
              <select
                bind:value={activeStoreFilter}
                aria-label="Filter by store"
                class="bg-surface-card border border-surface-border text-neutral-200 rounded-lg px-3 py-1.5 text-xs outline-none cursor-pointer pr-8 appearance-none hover:border-neutral-500 transition-colors"
              >
                <option value="all">All {storeResults.length > 0 ? `(${storeResults.length})` : ''}</option>
                <option value="steam">Steam {steamCount > 0 ? `(${steamCount})` : ''}</option>
                <option value="gog">GOG {gogCount > 0 ? `(${gogCount})` : ''}</option>
              </select>
              <div class="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none text-neutral-400">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Cards Grid -->
    <div class="flex-1 overflow-y-auto pt-6 pb-12 pr-1 custom-scrollbar">
      {#if effectiveMode === 'local'}
        {#if $filteredGames.length === 0}
          <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono space-y-2">
            <p>No games found matching "{$searchQuery}"</p>
            {#if $searchQuery}
              <button
                type="button"
                onclick={() => { $searchQuery = ''; }}
                class="text-xs text-neutral-300 hover:underline font-bold mt-1 cursor-pointer"
              >
                Clear search filter
              </button>
            {/if}
          </div>
        {:else}
          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 sm:gap-6">
            {#each $filteredGames as game, index (game.gameDir || index)}
              {@const pColors = getPlatformColors(game)}
              <CoverCard
                appId={game.appId}
                name={game.name}
                coverUrl={game.coverUrl}
                localCoverUrl={game.localCoverUrl}
                platform={game.isStub ? 'STUB' : pColors.label}
                isRunning={game.isRunning}
                isReady={game.isReady || game.isPatched}
                isStub={game.isStub}
                hasUpdate={Boolean(game.hasUpdate || game.hasUpdatesOrIssues || (game.versionStatus && !game.versionStatus.is_up_to_date))}
                size="full"
                className="w-full"
                onclick={() => openGameHub(game)}
              />
            {/each}
          </div>
        {/if}
      {:else}
        <!-- Catalog Mode Grid -->
        {#if !catalogQuery.trim()}
          <CatalogEmptyState onsuggest={(q) => handleCatalogSuggest(typeof q === 'string' ? q : q?.detail)} />
        {:else if isSearchingStore && storeResults.length === 0}
          <div class="h-64 flex flex-col items-center justify-center text-neutral-400 text-sm font-mono space-y-3">
            <Icon name="spinner" className="w-8 h-8 text-white" />
            <p>Querying store providers...</p>
          </div>
        {:else if storeResults.length === 0}
          <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono text-center space-y-2">
            <p>No store catalog matches found for "{catalogQuery}"</p>
            <button
              type="button"
              onclick={() => { catalogQuery = ''; catalogInputEl?.focus(); }}
              class="text-xs text-neutral-400 hover:text-white underline underline-offset-4 cursor-pointer"
            >
              Clear search query
            </button>
          </div>
        {:else if filteredStoreResults.length === 0}
          <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono text-center space-y-2">
            <p>No titles found from {activeStoreFilter.toUpperCase()} for "{catalogQuery}"</p>
            <button
              type="button"
              onclick={() => { activeStoreFilter = 'all'; }}
              class="text-xs text-neutral-400 hover:text-white underline underline-offset-4 cursor-pointer"
            >
              Show all results ({storeResults.length})
            </button>
          </div>
        {:else}
          <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6 pb-8">
            {#each filteredStoreResults as item, index (item.store + ':' + item.id || index)}
              <CatalogCard
                {item}
                isFocused={index === focusedCatalogIndex}
                localMatch={getMatchingLocalGame(item)}
                onselect={() => handleSelectCatalogItem(item)}
                onfocus={() => { focusedCatalogIndex = index; }}
              />
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>
