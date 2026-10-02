<script>
  import { onMount, onDestroy, createEventDispatcher, tick } from 'svelte';
  import ModalShell from '../common/ModalShell.svelte';
  import Icon from '../common/Icon.svelte';
  import CatalogCard from './catalog/CatalogCard.svelte';
  import CatalogEmptyState from './catalog/CatalogEmptyState.svelte';
  import { games, selectedGame, activeView } from '../../stores/library.js';
  import { normalizeGame } from '../../utils/game.js';
  import { AppAPI } from '../../api/app.js';

  const dispatch = createEventDispatcher();

  export let initialQuery = '';
  export let onclose = null;

  function handleClose() {
    dispatch('close');
    if (typeof onclose === 'function') onclose();
  }

  let searchInputEl;
  let query = initialQuery || '';
  let focusedIndex = 0;
  let gridContainerEl;
  let storeResults = [];
  let isSearchingStore = false;
  let storeSearchTimeout = null;
  let lastStoreQuery = '';
  let activeStoreFilter = 'all'; // 'all' | 'steam' | 'gog'

  $: allGames = $games || [];

  // Debounced Store Catalog Search
  $: {
    const q = query.trim();
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
  }

  // Filtered Store Results by Provider
  $: filteredResults = storeResults.filter(item => {
    if (activeStoreFilter === 'all') return true;
    const store = (item.store || 'steam').toLowerCase();
    return store === activeStoreFilter;
  });

  // Store counts for filter chips
  $: steamCount = storeResults.filter(item => (item.store || 'steam').toLowerCase() === 'steam').length;
  $: gogCount = storeResults.filter(item => (item.store || '').toLowerCase() === 'gog').length;

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

  $: {
    if (focusedIndex >= filteredResults.length) {
      focusedIndex = Math.max(0, filteredResults.length - 1);
    }
  }

  // Scroll focused card into view smoothly
  $: if (gridContainerEl && filteredResults.length > 0) {
    tick().then(() => {
      const activeCard = gridContainerEl?.querySelector(`[data-index="${focusedIndex}"]`);
      if (activeCard && typeof activeCard.scrollIntoView === 'function') {
        activeCard.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    });
  }

  onMount(() => {
    if (searchInputEl) {
      searchInputEl.focus();
    }
  });

  onDestroy(() => {
    if (storeSearchTimeout) clearTimeout(storeSearchTimeout);
  });

  function selectAndClose(item) {
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
    activeView.set('game-hub');
    handleClose();
  }

  function handleSuggest(e) {
    query = e.detail;
    if (searchInputEl) {
      searchInputEl.focus();
    }
  }

  function getColumnCount() {
    if (!gridContainerEl) return 6;
    const gridEl = gridContainerEl.querySelector('.catalog-grid');
    if (gridEl && typeof window !== 'undefined' && window.getComputedStyle) {
      const computedStyle = window.getComputedStyle(gridEl);
      const gridTemplateColumns = computedStyle.getPropertyValue('grid-template-columns');
      if (gridTemplateColumns) {
        const count = gridTemplateColumns.split(' ').filter(Boolean).length;
        if (count > 0) return count;
      }
    }
    const width = gridContainerEl.clientWidth || (typeof window !== 'undefined' ? window.innerWidth : 1024);
    if (width >= 1024) return 6;
    if (width >= 768) return 4;
    if (width >= 640) return 3;
    return 2;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      handleClose();
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      if (filteredResults[focusedIndex]) {
        selectAndClose(filteredResults[focusedIndex]);
      }
      return;
    }

    const cols = getColumnCount();
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      if (focusedIndex < filteredResults.length - 1) focusedIndex += 1;
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      if (focusedIndex > 0) focusedIndex -= 1;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (focusedIndex + cols < filteredResults.length) focusedIndex += cols;
      else focusedIndex = filteredResults.length - 1;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (focusedIndex - cols >= 0) focusedIndex -= cols;
      else focusedIndex = 0;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<ModalShell customHeader={true} padding="p-6 sm:p-10" on:close={handleClose}>
  <!-- Header Bar -->
  <div class="flex flex-col gap-3.5 max-w-5xl w-full mx-auto mb-4 flex-shrink-0">
    <div class="flex items-center justify-between border-b border-surface-border pb-3">
      <div class="flex items-center space-x-3">
        <div class="w-8 h-8 rounded-xl bg-surface-card border border-surface-border flex items-center justify-center text-white shadow-sm">
          <Icon name="globe" className="w-4 h-4" />
        </div>
        <div class="flex items-center space-x-2.5">
          <h2 class="text-base font-bold font-mono tracking-tight text-white">Online Catalog</h2>
          <span class="px-2 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider bg-surface-card border border-surface-border text-neutral-400">
            Steam · GOG · RAWG
          </span>
        </div>
        {#if isSearchingStore}
          <span class="inline-block w-2 h-2 rounded-full bg-white animate-ping" title="Searching catalogs..."></span>
        {/if}
      </div>

      <!-- Close Button & Live Result Count -->
      <div class="flex items-center space-x-3 text-xs font-mono text-neutral-400">
        {#if storeResults.length > 0}
          <span class="text-neutral-400">
            {filteredResults.length} {filteredResults.length === 1 ? 'title' : 'titles'}
            {#if activeStoreFilter !== 'all'}
              <span class="text-neutral-500">({storeResults.length} total)</span>
            {/if}
          </span>
        {/if}
        <button
          type="button"
          on:click={handleClose}
          class="px-3 py-1.5 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition-colors flex items-center space-x-1.5 cursor-pointer shadow-sm active:scale-95"
        >
          <span class="px-1.5 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-400 font-bold text-3xs">Esc</span>
          <span class="text-neutral-400">Close</span>
        </button>
      </div>
    </div>

    <!-- Search Input Field -->
    <div class="relative w-full flex items-center">
      <div class="absolute left-4 pointer-events-none text-neutral-400 flex items-center">
        <Icon name="globe" className="w-5 h-5" />
      </div>
      <input
        bind:this={searchInputEl}
        bind:value={query}
        type="text"
        placeholder="Search Steam, GOG, RAWG catalogs by title or AppID..."
        class="w-full bg-surface-input border border-surface-border focus:border-neutral-400 rounded-2xl pl-12 pr-10 py-3 text-base text-white outline-none font-mono transition-all shadow-xl placeholder:text-neutral-500"
      />
      {#if query}
        <button
          type="button"
          on:click={() => { query = ''; searchInputEl?.focus(); }}
          class="absolute right-4 text-neutral-400 hover:text-white p-1 rounded-lg transition-colors flex items-center justify-center cursor-pointer"
          aria-label="Clear search"
        >
          <Icon name="close" className="w-4 h-4" />
        </button>
      {/if}
    </div>

    <!-- Store Filter Chips (Dual Cadence & Low Cognitive Load) -->
    {#if storeResults.length > 0}
      <div class="flex items-center space-x-2 pt-0.5">
        <span class="text-3xs font-mono uppercase tracking-wider text-neutral-500 mr-1">Store:</span>
        <button
          type="button"
          on:click={() => { activeStoreFilter = 'all'; focusedIndex = 0; }}
          class="px-2.5 py-1 rounded-lg text-2xs font-mono transition-all cursor-pointer {activeStoreFilter === 'all' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight font-bold' : 'bg-surface-card text-neutral-400 hover:text-white border border-surface-border'}"
        >
          All ({storeResults.length})
        </button>
        {#if steamCount > 0}
          <button
            type="button"
            on:click={() => { activeStoreFilter = 'steam'; focusedIndex = 0; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-mono transition-all cursor-pointer {activeStoreFilter === 'steam' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight font-bold' : 'bg-surface-card text-neutral-400 hover:text-white border border-surface-border'}"
          >
            Steam ({steamCount})
          </button>
        {/if}
        {#if gogCount > 0}
          <button
            type="button"
            on:click={() => { activeStoreFilter = 'gog'; focusedIndex = 0; }}
            class="px-2.5 py-1 rounded-lg text-2xs font-mono transition-all cursor-pointer {activeStoreFilter === 'gog' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight font-bold' : 'bg-surface-card text-neutral-400 hover:text-white border border-surface-border'}"
          >
            GOG ({gogCount})
          </button>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Content Area -->
  <div
    bind:this={gridContainerEl}
    class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-2 custom-scrollbar"
  >
    {#if !query.trim()}
      <!-- Clean Empty Discovery State with Quick Suggestions -->
      <CatalogEmptyState on:suggest={handleSuggest} />
    {:else if isSearchingStore && storeResults.length === 0}
      <!-- Loading State -->
      <div class="h-64 flex flex-col items-center justify-center text-neutral-400 text-sm font-mono space-y-3">
        <Icon name="spinner" className="w-8 h-8 text-white" />
        <p>Querying store providers...</p>
      </div>
    {:else if storeResults.length === 0}
      <!-- No Matches Found -->
      <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono text-center space-y-2">
        <p>No store catalog matches found for "{query}"</p>
        <button
          type="button"
          on:click={() => { query = ''; searchInputEl?.focus(); }}
          class="text-xs text-neutral-400 hover:text-white underline underline-offset-4 cursor-pointer"
        >
          Clear search query
        </button>
      </div>
    {:else if filteredResults.length === 0}
      <!-- Filter Yielded No Matches -->
      <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono text-center space-y-2">
        <p>No titles found from {activeStoreFilter.toUpperCase()} for "{query}"</p>
        <button
          type="button"
          on:click={() => { activeStoreFilter = 'all'; }}
          class="text-xs text-neutral-400 hover:text-white underline underline-offset-4 cursor-pointer"
        >
          Show all results ({storeResults.length})
        </button>
      </div>
    {:else}
      <!-- Store Results Grid -->
      <div class="catalog-grid grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4 pb-8">
        {#each filteredResults as item, index (item.store + ':' + item.id || index)}
          <div data-index={index} class="w-full">
            <CatalogCard
              {item}
              isFocused={index === focusedIndex}
              localMatch={getMatchingLocalGame(item)}
              on:select={() => selectAndClose(item)}
              on:focus={() => { focusedIndex = index; }}
            />
          </div>
        {/each}
      </div>
    {/if}
  </div>
</ModalShell>
