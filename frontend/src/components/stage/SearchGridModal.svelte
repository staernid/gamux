<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import ModalShell from '../common/ModalShell.svelte';
  import { games, selectedGame, activeView, sortBy, getSortLabel, scanLibrary, importGame } from '../../stores/library.js';
  import { getPlatformColors, isSamePath, formatPlaytime, formatLastPlayed, formatInstallDate, normalizeGame } from '../../utils/game.js';
  import { formatBytes } from '../../utils/format.js';
  import { AppAPI } from '../../api/app.js';

  const dispatch = createEventDispatcher();

  export let initialQuery = '';
  export let initialMode = 'library';

  let searchInputEl;
  let query = initialQuery || '';
  let focusedIndex = 0;
  let gridContainerEl;
  let searchMode = initialMode || 'library'; // 'library' | 'store'
  let storeResults = [];
  let isSearchingStore = false;
  let storeSearchTimeout = null;
  let lastStoreQuery = '';

  const availableCommands = [
    {
      id: 'nav-library',
      title: 'Go to Shelf (Library)',
      description: 'View installed and local game titles',
      category: 'Navigation',
      shortcut: '1',
      icon: 'shelf',
      keywords: ['shelf', 'library', 'games', 'home'],
      action: () => {
        $activeView = 'library';
        dispatch('close');
      },
    },
    {
      id: 'nav-tasks',
      title: 'Go to Tasks',
      description: 'View active downloads, transfers, and background jobs',
      category: 'Navigation',
      shortcut: '2',
      icon: 'tasks',
      keywords: ['tasks', 'downloads', 'queue', 'transfers', 'jobs'],
      action: () => {
        $activeView = 'tasks';
        dispatch('close');
      },
    },
    {
      id: 'nav-settings',
      title: 'Go to Settings',
      description: 'Configure library paths, runners, and preferences',
      category: 'Navigation',
      shortcut: '3',
      icon: 'settings',
      keywords: ['settings', 'config', 'preferences', 'runners', 'wine', 'proton'],
      action: () => {
        $activeView = 'settings';
        dispatch('close');
      },
    },
    {
      id: 'act-scan',
      title: 'Scan Library Folders',
      description: 'Rescan game directories for newly added titles',
      category: 'Actions',
      shortcut: 'Scan',
      icon: 'scan',
      keywords: ['scan', 'refresh', 'rescan', 'reload', 'detect'],
      action: () => {
        scanLibrary();
        dispatch('close');
      },
    },
    {
      id: 'act-import',
      title: 'Import Game Folder...',
      description: 'Select an external game directory to import',
      category: 'Actions',
      shortcut: 'Import',
      icon: 'import',
      keywords: ['import', 'add', 'folder', 'directory'],
      action: () => {
        dispatch('close');
        importGame();
      },
    },
    {
      id: 'act-store',
      title: 'Search Online Store Catalog',
      description: 'Search Steam, GOG, and RAWG for uninstalled titles',
      category: 'Catalog',
      shortcut: 'Store',
      icon: 'store',
      keywords: ['store', 'catalog', 'steam', 'gog', 'rawg', 'online'],
      action: () => {
        searchMode = 'store';
        focusedIndex = 0;
        searchInputEl?.focus();
      },
    },
  ];

  $: isCommandMode = searchMode === 'library' && query.trim().startsWith('>');
  $: commandQuery = isCommandMode ? query.trim().slice(1).trim().toLowerCase() : query.trim().toLowerCase();

  $: matchingCommands = searchMode === 'library' && (isCommandMode || commandQuery.length >= 2)
    ? availableCommands.filter(cmd => {
        if (!commandQuery) return true;
        if (cmd.title.toLowerCase().includes(commandQuery)) return true;
        if (cmd.description.toLowerCase().includes(commandQuery)) return true;
        if (cmd.category.toLowerCase().includes(commandQuery)) return true;
        return cmd.keywords.some(k => k.includes(commandQuery));
      })
    : [];

  let focusedCommandIndex = 0;
  $: {
    if (focusedCommandIndex >= matchingCommands.length) {
      focusedCommandIndex = Math.max(0, matchingCommands.length - 1);
    }
  }

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

    if (isCommandMode) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (focusedCommandIndex < matchingCommands.length - 1) focusedCommandIndex += 1;
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (focusedCommandIndex > 0) focusedCommandIndex -= 1;
      } else if (e.key === 'Enter') {
        e.preventDefault();
        if (matchingCommands[focusedCommandIndex]) {
          matchingCommands[focusedCommandIndex].action();
        }
      }
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      if (displayedItems[focusedIndex]) {
        selectAndClose(displayedItems[focusedIndex]);
      } else if (matchingCommands.length > 0) {
        matchingCommands[0].action();
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
    <div class="flex items-center justify-between border-b border-surface-border pb-3">
      <div class="flex items-center space-x-2">
        <button
          type="button"
          on:click={() => { searchMode = 'library'; focusedIndex = 0; searchInputEl?.focus(); }}
          class="px-3.5 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center space-x-2 cursor-pointer {searchMode === 'library' ? 'bg-white text-neutral-950 shadow-md' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
        >
          <span>📚 My Library</span>
          <span class="px-1.5 py-0.5 rounded text-3xs font-mono {searchMode === 'library' ? 'bg-neutral-200 text-neutral-900' : 'bg-surface-cardHover text-neutral-400'}">{allGames.length}</span>
        </button>

        <button
          type="button"
          on:click={() => { searchMode = 'store'; focusedIndex = 0; searchInputEl?.focus(); }}
          class="px-3.5 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center space-x-2 cursor-pointer {searchMode === 'store' ? 'bg-white text-neutral-950 shadow-md' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
        >
          <span>🌐 Store Catalog</span>
          {#if isSearchingStore}
            <span class="inline-block w-2 h-2 rounded-full bg-neutral-400 animate-ping"></span>
          {/if}
        </button>
      </div>

      <!-- Stats & Close Hint -->
      <div class="flex items-center space-x-4 text-xs font-mono text-neutral-400">
        <span class="hidden sm:inline-block text-neutral-500">
          Tip: Type <kbd class="px-1.5 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-300 font-bold text-3xs">&gt;</kbd> for commands
        </span>
        <span>{displayedItems.length} {displayedItems.length === 1 ? 'title' : 'titles'}</span>
        <button
          type="button"
          on:click={() => dispatch('close')}
          class="px-3 py-1.5 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition-colors flex items-center space-x-1.5 cursor-pointer"
        >
          <span class="px-1.5 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-400 font-bold text-3xs">Esc / B</span>
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
          placeholder={searchMode === 'library' ? (isCommandMode ? 'Type a command name (e.g. settings, scan, tasks)...' : 'Type to filter library titles or run commands (e.g. >settings, scan)...') : 'Search Steam, GOG, RAWG catalogs by title or AppID...'}
          class="w-full bg-surface-input border border-surface-border focus:border-neutral-400 rounded-2xl pl-12 pr-10 py-3 text-base text-white outline-none font-mono transition-all shadow-xl placeholder:text-neutral-500"
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
              class="px-2.5 py-1 rounded-xl text-xs font-mono font-medium transition-all cursor-pointer {$sortBy === opt.id ? 'bg-surface-cardHover border border-surface-borderHighlight text-white shadow-sm' : 'bg-surface-input border border-surface-border text-neutral-400 hover:text-neutral-200'}"
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
    {#if isCommandMode}
      <!-- Command Palette Dedicated List -->
      <div class="flex flex-col space-y-2 pb-8 max-w-2xl mx-auto w-full">
        <div class="text-3xs font-mono uppercase tracking-wider text-neutral-500 font-bold px-1 mb-1">
          Commands &amp; Actions ({matchingCommands.length})
        </div>
        {#if matchingCommands.length === 0}
          <div class="h-40 flex items-center justify-center text-neutral-500 text-sm font-mono">
            No commands matching "{commandQuery}"
          </div>
        {:else}
          {#each matchingCommands as cmd, idx (cmd.id)}
            {@const isCmdFocused = idx === focusedCommandIndex}
            <button
              type="button"
              on:click={cmd.action}
              on:mouseenter={() => { focusedCommandIndex = idx; }}
              class="w-full flex items-center justify-between p-3.5 rounded-2xl border transition-all text-left cursor-pointer {isCmdFocused ? 'bg-surface-cardHover border-surface-borderHighlight text-white shadow-md' : 'bg-surface-card border-surface-border text-neutral-300 hover:border-neutral-600'}"
            >
              <div class="flex items-center space-x-3.5 min-w-0">
                <div class="w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0 {isCmdFocused ? 'bg-white text-neutral-950' : 'bg-surface-input border border-surface-border text-neutral-400'}">
                  {#if cmd.icon === 'shelf'}
                    <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" /></svg>
                  {:else if cmd.icon === 'tasks'}
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
                  {:else if cmd.icon === 'settings'}
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                  {:else if cmd.icon === 'scan'}
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                  {:else if cmd.icon === 'import'}
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" /></svg>
                  {:else}
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" /></svg>
                  {/if}
                </div>
                <div class="min-w-0">
                  <div class="text-sm font-semibold truncate {isCmdFocused ? 'text-white' : 'text-neutral-200'}">
                    {cmd.title}
                  </div>
                  <div class="text-2xs text-neutral-400 font-mono truncate">
                    {cmd.description}
                  </div>
                </div>
              </div>
              <div class="flex items-center space-x-2 flex-shrink-0 pl-3">
                <span class="text-3xs font-mono uppercase text-neutral-500">{cmd.category}</span>
                {#if cmd.shortcut}
                  <span class="px-2 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-400 font-mono text-3xs font-bold">
                    {cmd.shortcut}
                  </span>
                {/if}
              </div>
            </button>
          {/each}
        {/if}
      </div>
    {:else if searchMode === 'library'}
      <!-- Suggested Commands Strip if searching keyword -->
      {#if matchingCommands.length > 0}
        <div class="mb-5 pb-4 border-b border-surface-border">
          <div class="text-3xs font-mono uppercase tracking-wider text-neutral-500 font-bold mb-2">
            Suggested Commands
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            {#each matchingCommands.slice(0, 3) as cmd}
              <button
                type="button"
                on:click={cmd.action}
                class="flex items-center justify-between p-2.5 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border hover:border-surface-borderHighlight text-left transition-all cursor-pointer group"
              >
                <div class="flex items-center space-x-2.5 min-w-0">
                  <span class="text-xs font-semibold text-neutral-200 group-hover:text-white truncate">{cmd.title}</span>
                </div>
                <span class="text-3xs font-mono px-1.5 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-400 font-bold">{cmd.shortcut || '↵'}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Local Library Grid -->
      {#if filtered.length === 0}
        <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono space-y-2">
          <p>No local library games matching "{query}"</p>
          <button
            type="button"
            on:click={() => { searchMode = 'store'; }}
            class="text-xs text-neutral-300 hover:text-white hover:underline font-bold mt-1 bg-surface-card px-3 py-1.5 rounded-xl border border-surface-border hover:bg-surface-cardHover transition-colors cursor-pointer"
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
                  <div class="w-full h-full bg-surface-card border border-surface-border flex items-center justify-center p-3 text-center text-3xs text-neutral-500 font-mono">
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

