<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import ModalShell from '../common/ModalShell.svelte';
  import Icon from '../common/Icon.svelte';
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
    if (focusedIndex >= storeResults.length) {
      focusedIndex = Math.max(0, storeResults.length - 1);
    }
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

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      handleClose();
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      if (storeResults[focusedIndex]) {
        selectAndClose(storeResults[focusedIndex]);
      }
      return;
    }

    const cols = 6;
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      if (focusedIndex < storeResults.length - 1) focusedIndex += 1;
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      if (focusedIndex > 0) focusedIndex -= 1;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (focusedIndex + cols < storeResults.length) focusedIndex += cols;
      else focusedIndex = storeResults.length - 1;
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
  <div class="flex flex-col gap-3.5 max-w-5xl w-full mx-auto mb-6 flex-shrink-0">
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
          <span class="inline-block w-2 h-2 rounded-full bg-white animate-ping"></span>
        {/if}
      </div>

      <!-- Close Button -->
      <div class="flex items-center space-x-3 text-xs font-mono text-neutral-400">
        {#if storeResults.length > 0}
          <span>{storeResults.length} {storeResults.length === 1 ? 'title' : 'titles'}</span>
        {/if}
        <button
          type="button"
          on:click={handleClose}
          class="px-3 py-1.5 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition-colors flex items-center space-x-1.5 cursor-pointer"
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
  </div>

  <!-- Content Area -->
  <div
    bind:this={gridContainerEl}
    class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-2 custom-scrollbar"
  >
    {#if !query.trim()}
      <!-- Clean Empty Discovery State with Wireframe Globe -->
      <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono space-y-3 text-center">
        <div class="w-14 h-14 rounded-2xl bg-surface-card/60 border border-surface-border flex items-center justify-center text-neutral-500 mb-1">
          <Icon name="globe" className="w-7 h-7" />
        </div>
        <p class="font-bold text-neutral-300">Online Store Catalog Discovery</p>
        <p class="text-xs text-neutral-500 max-w-md leading-relaxed">
          Search Steam, GOG, and RAWG for uninstalled titles, explore metadata, or match store entries.
        </p>
      </div>
    {:else if isSearchingStore && storeResults.length === 0}
      <!-- Loading State -->
      <div class="h-64 flex flex-col items-center justify-center text-neutral-400 text-sm font-mono space-y-3">
        <Icon name="spinner" className="w-8 h-8 text-white" />
        <p>Querying store providers...</p>
      </div>
    {:else if storeResults.length === 0}
      <!-- No Matches Found -->
      <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono text-center">
        <p>No store catalog matches found for "{query}"</p>
      </div>
    {:else}
      <!-- Store Results Grid -->
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4 pb-8">
        {#each storeResults as item, index (item.store + ':' + item.id || index)}
          {@const isFocused = index === focusedIndex}
          {@const localMatch = getMatchingLocalGame(item)}
          {@const storeName = (item.store || 'steam').toUpperCase()}
          <button
            type="button"
            on:click={() => selectAndClose(item)}
            on:mouseenter={() => { focusedIndex = index; }}
            class="group cursor-pointer flex flex-col items-center text-center transition-all duration-150 transform {isFocused ? 'scale-105' : 'hover:scale-102 opacity-85 hover:opacity-100'} bg-transparent border-0 p-0 text-left"
          >
            <!-- Card Cover -->
            <div
              class="relative w-full aspect-[2/3] rounded-2xl overflow-hidden shadow-lg border transition-all duration-200 bg-surface-card"
              style="
                {isFocused
                  ? 'border: 2px solid #ffffff; box-shadow: 0 8px 25px -4px rgba(255,255,255,0.25);'
                  : 'border: 1px solid rgba(255, 255, 255, 0.1);'
                }
              "
            >
              {#if item.poster_url}
                <img src={item.poster_url} alt={item.name} class="w-full h-full object-cover" />
              {:else}
                <div class="w-full h-full flex items-center justify-center p-3 text-center text-3xs text-neutral-500 font-mono">
                  {item.name}
                </div>
              {/if}

              <!-- Store Provider Badge (Top Left) -->
              <div class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-3xs font-mono font-black uppercase tracking-wider text-white shadow-md backdrop-blur-md bg-black/70 border border-white/10">
                {storeName}
              </div>

              <!-- Ownership Badge (Top Right) -->
              {#if localMatch}
                <div class="absolute top-2 right-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider shadow-md backdrop-blur-md {localMatch.isInstalled ? 'bg-emerald-500/90 text-neutral-950' : 'bg-neutral-800 text-neutral-200 border border-neutral-700'}">
                  {localMatch.isInstalled ? 'Installed' : 'In Library'}
                </div>
              {/if}
            </div>

            <!-- Title & ID Subtitle -->
            <div class="mt-2 flex flex-col items-center max-w-full px-1 space-y-0.5">
              <span class="text-xs font-semibold text-neutral-200 group-hover:text-white line-clamp-2">
                {item.name}
              </span>
              <span class="text-3xs font-mono text-neutral-500 truncate">
                #{item.id} {item.release_date ? `· ${item.release_date.slice(0, 4)}` : ''}
              </span>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</ModalShell>
