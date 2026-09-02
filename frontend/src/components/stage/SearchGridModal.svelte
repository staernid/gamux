<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import ModalShell from '../common/ModalShell.svelte';
  import { games, selectedGame, activeView, sortBy, getSortLabel } from '../../stores/library.js';
  import { getPlatformColors, isSamePath, formatPlaytime, formatLastPlayed, formatInstallDate, normalizeGame } from '../../utils/game.js';
  import { formatBytes } from '../../utils/format.js';
  import { AppAPI } from '../../api/app.js';

  const dispatch = createEventDispatcher();

  let searchInputEl;
  let query = '';
  let focusedIndex = 0;
  let gridContainerEl;
  let searchMode = 'library'; // 'library' | 'store'
  let storeResults = [];
  let isSearchingStore = false;
  let storeSearchTimeout = null;
  let lastStoreQuery = '';

  const quickSortOptions = [
    { id: 'name_asc', label: 'Name' },
    { id: 'last_played_desc', label: 'Last Played' },
    { id: 'playtime_desc', label: 'Playtime' },
    { id: 'install_desc', label: 'Installed' },
    { id: 'size_desc', label: 'Size' },
  ];

  $: allGames = $games || [];
  $: searchedGames = query.trim()
    ? allGames.filter(g => {
        const q = query.toLowerCase();
        const name = (g.name || '').toLowerCase();
        const appId = String(g.appId || '');
        return name.includes(q) || appId.includes(q);
      })
    : allGames;

  $: filtered = [...searchedGames].sort((a, b) => {
    if ($sortBy === 'name_asc') {
      return (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'name_desc') {
      return (b.name || '').localeCompare(a.name || '');
    } else if ($sortBy === 'last_played_desc') {
      const diff = (b.lastPlayed || 0) - (a.lastPlayed || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'last_played_asc') {
      const diff = (a.lastPlayed || 0) - (b.lastPlayed || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'playtime_desc') {
      const diff = (b.playtimeSeconds || 0) - (a.playtimeSeconds || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'playtime_asc') {
      const diff = (a.playtimeSeconds || 0) - (b.playtimeSeconds || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'install_desc') {
      const diff = (b.installedAt || 0) - (a.installedAt || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'install_asc') {
      const diff = (a.installedAt || 0) - (b.installedAt || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'size_desc') {
      const diff = (b.diskSizeBytes || 0) - (a.diskSizeBytes || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'size_asc') {
      const diff = (a.diskSizeBytes || 0) - (b.diskSizeBytes || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'appid') {
      const diff = (parseInt(a.appId, 10) || 0) - (parseInt(b.appId, 10) || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if ($sortBy === 'status') {
      const diff = (b.isPatched ? 1 : 0) - (a.isPatched ? 1 : 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    }
    return 0;
  });

  // Store search debounce effect
  $: if (searchMode === 'store') {
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

  function getCardMetaBadge(g, sortMode) {
    if (!g) return '';
    if (g.isStub) return 'In Library (Uninstalled)';
    if (g.isVirtual) return 'Store Title';
    if (sortMode === 'last_played_desc' || sortMode === 'last_played_asc') {
      return g.lastPlayed > 0 ? formatLastPlayed(g.lastPlayed) : 'Never';
    }
    if (sortMode === 'playtime_desc' || sortMode === 'playtime_asc') {
      return g.playtimeSeconds > 0 ? formatPlaytime(g.playtimeSeconds) : '0 hrs';
    }
    if (sortMode === 'install_desc' || sortMode === 'install_asc') {
      return g.installedAt > 0 ? formatInstallDate(g.installedAt) : '';
    }
    if (sortMode === 'size_desc' || sortMode === 'size_asc') {
      return g.diskSizeBytes > 0 ? formatBytes(g.diskSizeBytes) : '';
    }
    return '';
  }

  $: displayedItems = searchMode === 'library' ? filtered : storeResults;

  $: {
    if (focusedIndex >= displayedItems.length) {
      focusedIndex = Math.max(0, displayedItems.length - 1);
    }
  }

  onMount(() => {
    if (searchInputEl) {
      searchInputEl.focus();
    }
    if ($selectedGame && filtered.length > 0) {
      const idx = filtered.findIndex(g => isSamePath(g.gameDir, $selectedGame.gameDir));
      if (idx >= 0) focusedIndex = idx;
    }
  });

  onDestroy(() => {
    if (storeSearchTimeout) clearTimeout(storeSearchTimeout);
  });

  function selectAndClose(item) {
    if (!item) return;

    if (searchMode === 'library') {
      selectedGame.set(item);
      activeView.set('game-hub');
      dispatch('close');
      return;
    }

    // Store Catalog Mode
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
    dispatch('close');
  }


  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      dispatch('close');
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      if (displayedItems[focusedIndex]) {
        selectAndClose(displayedItems[focusedIndex]);
      }
      return;
    }

    // Grid keyboard navigation
    const cols = 6;
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      if (focusedIndex < displayedItems.length - 1) focusedIndex += 1;
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      if (focusedIndex > 0) focusedIndex -= 1;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (focusedIndex + cols < displayedItems.length) focusedIndex += cols;
      else focusedIndex = displayedItems.length - 1;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (focusedIndex - cols >= 0) focusedIndex -= cols;
      else focusedIndex = 0;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<ModalShell customHeader={true} padding="p-6 sm:p-10" on:close={() => dispatch('close')}>
  <!-- Top Search Bar Header -->
  <div class="flex flex-col gap-3.5 max-w-5xl w-full mx-auto mb-6 flex-shrink-0">
    <!-- Mode Switcher Tabs -->
    <div class="flex items-center justify-between border-b border-[#222232] pb-3">
      <div class="flex items-center space-x-2">
        <button
          type="button"
          on:click={() => { searchMode = 'library'; focusedIndex = 0; searchInputEl?.focus(); }}
          class="px-3.5 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center space-x-2 cursor-pointer {searchMode === 'library' ? 'bg-white text-neutral-950 shadow-md' : 'bg-[#18181b] border border-[#27272a] text-neutral-400 hover:text-white'}"
        >
          <span>📚 My Library</span>
          <span class="px-1.5 py-0.5 rounded text-3xs font-mono {searchMode === 'library' ? 'bg-neutral-200 text-neutral-900' : 'bg-[#222225] text-neutral-400'}">{allGames.length}</span>
        </button>

        <button
          type="button"
          on:click={() => { searchMode = 'store'; focusedIndex = 0; searchInputEl?.focus(); }}
          class="px-3.5 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center space-x-2 cursor-pointer {searchMode === 'store' ? 'bg-white text-neutral-950 shadow-md' : 'bg-[#18181b] border border-[#27272a] text-neutral-400 hover:text-white'}"
        >
          <span>🌐 Store Catalog</span>
          {#if isSearchingStore}
            <span class="inline-block w-2 h-2 rounded-full bg-neutral-400 animate-ping"></span>
          {/if}
        </button>
      </div>

      <!-- Stats & Close Hint -->
      <div class="flex items-center space-x-4 text-xs font-mono text-neutral-400">
        <span>{displayedItems.length} {displayedItems.length === 1 ? 'title' : 'titles'}</span>
        <button
          type="button"
          on:click={() => dispatch('close')}
          class="px-3 py-1.5 rounded-xl bg-[#18181b] hover:bg-[#222225] border border-[#27272a] text-neutral-300 hover:text-white transition-colors flex items-center space-x-1.5 cursor-pointer"
        >
          <span class="px-1.5 py-0.5 rounded bg-[#141417] border border-[#27272a] text-neutral-400 font-bold text-3xs">Esc / B</span>
          <span class="text-neutral-400">Close</span>
        </button>
      </div>
    </div>

    <!-- Search Input & Query Controls -->
    <div class="flex items-center justify-between gap-4">
      <div class="relative flex-1 max-w-xl flex items-center">
        <svg class="w-5 h-5 text-neutral-400 absolute left-4 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input
          bind:this={searchInputEl}
          bind:value={query}
          type="text"
          placeholder={searchMode === 'library' ? 'Type to filter library titles...' : 'Search Steam, GOG, RAWG catalogs by title or AppID...'}
          class="w-full bg-[#141417] border border-[#27272a] focus:border-neutral-400 rounded-2xl pl-12 pr-10 py-3 text-base text-white outline-none font-mono transition-all shadow-xl placeholder:text-neutral-500"
        />
        {#if query}
          <button
            type="button"
            on:click={() => { query = ''; searchInputEl?.focus(); }}
            class="absolute right-4 text-neutral-400 hover:text-white p-1 rounded-lg transition-colors flex items-center justify-center cursor-pointer"
            aria-label="Clear search"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        {/if}
      </div>

      {#if searchMode === 'library'}
        <!-- Quick Sort Chips Toolbar -->
        <div class="flex items-center gap-1.5 flex-wrap">
          <span class="text-3xs font-mono text-neutral-500 uppercase tracking-wider font-bold mr-1">Sort:</span>
          {#each quickSortOptions as opt}
            <button
              type="button"
              on:click={() => { $sortBy = opt.id; }}
              class="px-2.5 py-1 rounded-xl text-xs font-mono font-medium transition-all cursor-pointer {$sortBy === opt.id ? 'bg-[#222225] border border-[#3f3f46] text-white shadow-sm' : 'bg-[#141417] border border-[#27272a] text-neutral-400 hover:text-neutral-200'}"
            >
              {opt.label}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  <!-- Grid Content Area -->
  <div
    bind:this={gridContainerEl}
    class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-2 custom-scrollbar"
  >
    {#if searchMode === 'library'}
      <!-- Local Library Grid -->
      {#if filtered.length === 0}
        <div class="h-64 flex flex-col items-center justify-center text-slate-500 text-sm font-mono space-y-2">
          <p>No local library games matching "{query}"</p>
          <button
            type="button"
            on:click={() => { searchMode = 'store'; }}
            class="text-xs text-cyan-400 hover:underline font-bold mt-1"
          >
            Search online store catalog for "{query}" ➔
          </button>
        </div>
      {:else}
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4 pb-8">
          {#each filtered as game, index (game.gameDir || index)}
            {@const isFocused = index === focusedIndex}
            {@const pColors = getPlatformColors(game)}
            <button
              type="button"
              on:click={() => selectAndClose(game)}
              on:mouseenter={() => { focusedIndex = index; }}
              class="group cursor-pointer flex flex-col items-center text-center transition-all duration-150 transform {isFocused ? 'scale-105' : 'hover:scale-102 opacity-80 hover:opacity-100'} bg-transparent border-0 p-0 text-left"
            >
              <!-- Card Cover -->
              <div
                class="relative w-full aspect-[2/3] rounded-2xl overflow-hidden shadow-lg border transition-all duration-200"
                style="
                  {isFocused
                    ? `border: 2.5px solid ${pColors.border}; box-shadow: 0 8px 25px -4px ${pColors.glow};`
                    : 'border: 1px solid rgba(255, 255, 255, 0.1);'
                  }
                "
              >
                <PosterArt
                  appId={game.appId}
                  name={game.name}
                  posterUrl={game.coverUrl}
                  localCoverUrl={game.localCoverUrl}
                  className="w-full h-full object-cover"
                />

                <!-- Platform or Stub badge -->
                <div
                  class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-3xs font-mono font-black uppercase tracking-wider text-white shadow-md backdrop-blur-md"
                  style="background-color: {game.isStub ? '#3f3f46' : pColors.badgeBg};"
                >
                  {game.isStub ? 'STUB' : pColors.label}
                </div>
              </div>

              <!-- Title & Metadata Subtitle Below -->
              <div class="mt-2 flex flex-col items-center max-w-full px-1 space-y-0.5">
                <span class="text-xs font-semibold text-neutral-200 group-hover:text-white line-clamp-2">
                  {game.name}
                </span>
                {#if getCardMetaBadge(game, $sortBy)}
                  <span class="text-3xs font-mono text-neutral-400 truncate">
                    {getCardMetaBadge(game, $sortBy)}
                  </span>
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    {:else}
      <!-- Store Catalog Grid -->
      {#if !query.trim()}
        <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono space-y-2 text-center">
          <svg class="w-10 h-10 text-neutral-600 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
          <p class="font-bold text-neutral-300">Online Store Catalog Discovery</p>
          <p class="text-xs text-neutral-500 max-w-md">Type a title or Steam AppID in the search bar above to search live store providers (Steam, GOG, RAWG).</p>
        </div>
      {:else if isSearchingStore && storeResults.length === 0}
        <div class="h-64 flex flex-col items-center justify-center text-neutral-400 text-sm font-mono space-y-3">
          <svg class="animate-spin w-8 h-8 text-white" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <p>Querying store providers...</p>
        </div>
      {:else if storeResults.length === 0}
        <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono">
          <p>No store catalog matches found for "{query}"</p>
        </div>
      {:else}
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
                class="relative w-full aspect-[2/3] rounded-2xl overflow-hidden shadow-lg border transition-all duration-200"
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
                  <div class="w-full h-full bg-[#121218] flex items-center justify-center p-3 text-center text-3xs text-neutral-500 font-mono">
                    {item.name}
                  </div>
                {/if}

                <!-- Store Provider Badge (Top Left) -->
                <div class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-3xs font-mono font-black uppercase tracking-wider text-white shadow-md backdrop-blur-md bg-black/60 border border-white/10">
                  {storeName}
                </div>

                <!-- Ownership Badge (Top Right) -->
                {#if localMatch}
                  <div class="absolute top-2 right-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider shadow-md backdrop-blur-md {localMatch.isInstalled ? 'bg-emerald-500/90 text-neutral-950' : 'bg-neutral-800 text-neutral-200 border border-neutral-700'}">
                    {localMatch.isInstalled ? 'Installed' : 'In Library'}
                  </div>
                {/if}
              </div>

              <!-- Title & ID Subtitle Below -->
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
    {/if}
  </div>
</ModalShell>

