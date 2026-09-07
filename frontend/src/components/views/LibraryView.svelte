<script>
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import PosterArt from '../common/PosterArt.svelte';
  import Carousel3D from '../stage/Carousel3D.svelte';
  import {
    games,
    filteredGames,
    selectedGame,
    activeView,
    sortBy,
    activeFilter,
    searchQuery,
    cycleSortOrder,
    getSortLabel,
    stats
  } from '../../stores/library.js';
  import { configStore } from '../../stores/config.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';
  import {
    getPlatformColors,
    getPlatformType,
    getGameSource,
    getMetadataInfo,
    isSamePath,
    formatPlaytime,
    formatLastPlayed,
    formatInstallDate
  } from '../../utils/game.js';
  import { playTick, playSelect, playNav } from '../../utils/audio.js';

  export let carouselRef = null;

  // View mode: 'grid' (default for desktop) or 'shelf' (3D cover flow)
  let libraryMode = (typeof localStorage !== 'undefined' && localStorage.getItem('gamux_library_mode')) || 'grid';

  let searchInputEl;
  let focusedIndex = 0;
  let isSearchFocused = false;

  $: activeColors = $selectedGame ? getPlatformColors($selectedGame) : null;
  $: platformType = $selectedGame ? getPlatformType($selectedGame) : '';
  $: gameSource = $selectedGame ? getGameSource($selectedGame) : '';
  $: metaInfo = $selectedGame ? getMetadataInfo($selectedGame) : { provider: 'None', id: '' };

  const sortOptions = [
    { id: 'name_asc', label: 'Name (A-Z)' },
    { id: 'name_desc', label: 'Name (Z-A)' },
    { id: 'last_played_desc', label: 'Last Played' },
    { id: 'playtime_desc', label: 'Playtime' },
    { id: 'install_desc', label: 'Installed' },
    { id: 'size_desc', label: 'Size' }
  ];

  function setLibraryMode(mode) {
    if (mode === libraryMode) return;
    playNav();
    libraryMode = mode;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('gamux_library_mode', mode);
    }
  }

  function openGameHub(game) {
    if (!game) return;
    selectedGame.set(game);
    playSelect();
    activeView.set('game-hub');
  }

  function handleCardKeydown(e, game, index) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      openGameHub(game);
    } else if (e.key === 'ArrowRight') {
      e.preventDefault();
      if (index < $filteredGames.length - 1) focusedIndex = index + 1;
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      if (index > 0) focusedIndex = index - 1;
    }
  }

  export function step(dir) {
    if (libraryMode === 'shelf' && carouselRef) {
      carouselRef.step(dir);
    } else {
      const list = $filteredGames || [];
      if (list.length === 0) return;
      const current = get(selectedGame);
      const currentIndex = current ? list.findIndex(g => isSamePath(g.gameDir, current.gameDir)) : -1;
      const nextIndex = (currentIndex + dir + list.length) % list.length;
      playTick();
      selectedGame.set(list[nextIndex]);
      focusedIndex = nextIndex;
    }
  }

  export function addRotaryDelta(delta) {
    if (libraryMode === 'shelf' && carouselRef) {
      carouselRef.addRotaryDelta(delta);
    }
  }

  export function focusSearch() {
    if (libraryMode !== 'grid') {
      setLibraryMode('grid');
    }
    setTimeout(() => {
      searchInputEl?.focus();
    }, 50);
  }
</script>

<div class="flex-1 flex flex-col min-h-0 w-full relative">
  {#if libraryMode === 'grid'}
    <!-- Desktop Grid View Mode -->
    <div class="flex-1 flex flex-col min-h-0 pt-16 sm:pt-20 px-6 sm:px-10 pb-6 max-w-7xl mx-auto w-full animate-fade-in">
      <!-- Toolbar: Search, Filters, Sorters & View Switcher -->
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-5 border-b border-surface-border">
        <!-- Left: Search input + Game Count -->
        <div class="flex items-center space-x-3 flex-1 max-w-md">
          <div class="relative flex-1">
            <input
              bind:this={searchInputEl}
              bind:value={$searchQuery}
              on:focus={() => { isSearchFocused = true; }}
              on:blur={() => { isSearchFocused = false; }}
              type="text"
              placeholder="Search library by title, appid, platform..."
              class="w-full bg-surface-card border border-surface-border focus:border-neutral-400 rounded-xl pl-10 pr-9 py-2 text-xs sm:text-sm text-white placeholder-neutral-500 outline-none transition-all shadow-inner"
            />
            <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-neutral-500">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
            {#if $searchQuery}
              <button
                type="button"
                on:click={() => { $searchQuery = ''; searchInputEl?.focus(); }}
                class="absolute inset-y-0 right-0 pr-3 flex items-center text-neutral-500 hover:text-white cursor-pointer"
                title="Clear search"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            {/if}
          </div>

          <div class="hidden sm:flex items-center space-x-1.5 text-xs text-neutral-400 font-mono flex-shrink-0">
            <span class="font-bold text-white">{$filteredGames.length}</span>
            <span class="text-neutral-600">/</span>
            <span>{$games.length}</span>
            <span class="text-neutral-500">titles</span>
          </div>
        </div>

        <!-- Middle: Status Filter Chips -->
        <div class="flex items-center space-x-1.5 overflow-x-auto py-0.5">
          <button
            type="button"
            on:click={() => { $activeFilter = 'all'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {$activeFilter === 'all' ? 'bg-white text-neutral-950 font-bold shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
          >
            All
          </button>
          <button
            type="button"
            on:click={() => { $activeFilter = 'patched'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all flex items-center space-x-1.5 cursor-pointer {$activeFilter === 'patched' ? 'bg-white text-neutral-950 font-bold shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
          >
            <span>Ready</span>
            {#if $stats.patchedCount > 0}
              <span class="px-1 py-0.2 rounded text-3xs {$activeFilter === 'patched' ? 'bg-neutral-200 text-neutral-950' : 'bg-neutral-800 text-neutral-300'}">{$stats.patchedCount}</span>
            {/if}
          </button>
          <button
            type="button"
            on:click={() => { $activeFilter = 'clean'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all flex items-center space-x-1.5 cursor-pointer {$activeFilter === 'clean' ? 'bg-white text-neutral-950 font-bold shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
          >
            <span>Unpatched</span>
            {#if $stats.cleanCount > 0}
              <span class="px-1 py-0.2 rounded text-3xs {$activeFilter === 'clean' ? 'bg-neutral-200 text-neutral-950' : 'bg-neutral-800 text-neutral-300'}">{$stats.cleanCount}</span>
            {/if}
          </button>
          {#if $stats.updatesCount > 0}
            <button
              type="button"
              on:click={() => { $activeFilter = 'updates'; }}
              class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all flex items-center space-x-1.5 cursor-pointer {$activeFilter === 'updates' ? 'bg-white text-neutral-950 font-bold shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
            >
              <span>Updates</span>
              <span class="px-1 py-0.2 rounded text-3xs bg-amber-500/20 text-amber-300 font-bold">{$stats.updatesCount}</span>
            </button>
          {/if}
        </div>

        <!-- Right: Sort Dropdown & Layout Mode Switcher -->
        <div class="flex items-center space-x-3 justify-between md:justify-end">
          <!-- Sort Dropdown -->
          <div class="flex items-center space-x-2">
            <span class="text-2xs font-mono text-neutral-500 uppercase tracking-wider font-bold">Sort:</span>
            <div class="relative">
              <select
                bind:value={$sortBy}
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

          <!-- View Mode Toggle (Grid vs Shelf) -->
          <div class="flex items-center bg-surface-card border border-surface-border rounded-lg p-0.5">
            <button
              type="button"
              on:click={() => setLibraryMode('grid')}
              class="p-1.5 rounded-md text-xs transition-colors flex items-center justify-center cursor-pointer {libraryMode === 'grid' ? 'bg-[#27272a] text-white shadow-sm' : 'text-neutral-500 hover:text-neutral-300'}"
              title="Desktop Grid View"
              aria-label="Desktop Grid View"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="3" width="7" height="7" rx="1" />
                <rect x="14" y="3" width="7" height="7" rx="1" />
                <rect x="14" y="14" width="7" height="7" rx="1" />
                <rect x="3" y="14" width="7" height="7" rx="1" />
              </svg>
            </button>
            <button
              type="button"
              on:click={() => setLibraryMode('shelf')}
              class="p-1.5 rounded-md text-xs transition-colors flex items-center justify-center cursor-pointer {libraryMode === 'shelf' ? 'bg-[#27272a] text-white shadow-sm' : 'text-neutral-500 hover:text-neutral-300'}"
              title="3D Cover Flow / Shelf Mode"
              aria-label="3D Cover Flow / Shelf Mode"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="2" y="5" width="20" height="14" rx="2" />
                <line x1="8" y1="5" x2="8" y2="19" />
                <line x1="16" y1="5" x2="16" y2="19" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- Game Cards Grid -->
      <div class="flex-1 overflow-y-auto pt-6 pb-12 pr-1 custom-scrollbar">
        {#if $filteredGames.length === 0}
          <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-sm font-mono space-y-2">
            <p>No games found matching "{$searchQuery}"</p>
            {#if $searchQuery}
              <button
                type="button"
                on:click={() => { $searchQuery = ''; }}
                class="text-xs text-neutral-300 hover:underline font-bold mt-1"
              >
                Clear search filter
              </button>
            {/if}
          </div>
        {:else}
          <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
            {#each $filteredGames as game, index (game.gameDir || index)}
              {@const pColors = getPlatformColors(game)}
              <button
                type="button"
                on:click={() => openGameHub(game)}
                on:keydown={(e) => handleCardKeydown(e, game, index)}
                class="group cursor-pointer flex flex-col items-center text-left transition-all duration-150 transform hover:-translate-y-1 bg-transparent border-0 p-0"
              >
                <!-- Poster Art Container -->
                <div class="relative w-full aspect-[2/3] rounded-xl overflow-hidden shadow-md group-hover:shadow-xl border border-surface-border group-hover:border-neutral-400 transition-all">
                  <PosterArt
                    appId={game.appId}
                    name={game.name}
                    posterUrl={game.coverUrl}
                    localCoverUrl={game.localCoverUrl}
                    className="w-full h-full object-cover"
                  />

                  <!-- Top Left Badge: Platform / Stub -->
                  <div
                    class="absolute top-2 left-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider text-white shadow-md backdrop-blur-md"
                    style="background-color: {game.isStub ? '#3f3f46' : pColors.badgeBg};"
                  >
                    {game.isStub ? 'STUB' : pColors.label}
                  </div>

                  <!-- Bottom Gradient Overlay with Playtime -->
                  {#if game.playtimeSeconds > 0}
                    <div class="absolute inset-x-0 bottom-0 pt-6 pb-2 px-2.5 bg-gradient-to-t from-black/90 via-black/50 to-transparent flex items-end justify-between text-4xs font-mono text-neutral-300">
                      <span>{formatPlaytime(game.playtimeSeconds)}</span>
                      {#if game.lastPlayed > 0}
                        <span class="text-neutral-400">{formatLastPlayed(game.lastPlayed)}</span>
                      {/if}
                    </div>
                  {/if}
                </div>

                <!-- Title & Technical Details -->
                <div class="w-full mt-2 space-y-0.5 min-w-0">
                  <h4 class="text-xs font-semibold text-neutral-200 group-hover:text-white truncate transition-colors" title={game.name}>
                    {game.name}
                  </h4>
                  <div class="flex items-center space-x-1.5 text-3xs font-mono text-neutral-500 truncate">
                    <span>{formatBytes(game.diskSizeBytes)}</span>
                    <span>·</span>
                    <span class="truncate">{game.runner || 'UMU'}</span>
                  </div>
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {:else}
    <!-- Shelf / Carousel Mode -->
    <div class="flex-1 flex flex-col min-h-0 relative items-center justify-between">
      <!-- Mode Toggle Pill Floating on Top Right -->
      <div class="absolute top-4 right-6 z-30 flex items-center bg-surface-card/90 border border-surface-border rounded-lg p-0.5 shadow-lg">
        <button
          type="button"
          on:click={() => setLibraryMode('grid')}
          class="p-1.5 rounded-md text-xs transition-colors flex items-center justify-center cursor-pointer text-neutral-500 hover:text-neutral-300"
          title="Switch to Desktop Grid View"
          aria-label="Desktop Grid View"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="7" height="7" rx="1" />
            <rect x="14" y="3" width="7" height="7" rx="1" />
            <rect x="14" y="14" width="7" height="7" rx="1" />
            <rect x="3" y="14" width="7" height="7" rx="1" />
          </svg>
        </button>
        <button
          type="button"
          on:click={() => setLibraryMode('shelf')}
          class="p-1.5 rounded-md text-xs transition-colors flex items-center justify-center cursor-pointer bg-[#27272a] text-white shadow-sm"
          title="3D Cover Flow / Shelf Mode"
          aria-label="3D Cover Flow / Shelf Mode"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="5" width="20" height="14" rx="2" />
            <line x1="8" y1="5" x2="8" y2="19" />
            <line x1="16" y1="5" x2="16" y2="19" />
          </svg>
        </button>
      </div>

      <!-- Center 3D 5-Item Carousel -->
      <Carousel3D bind:this={carouselRef} on:launch={(e) => openGameHub(e.detail)} />

      <!-- Bottom HUD: Left Command Glossary & Right Game Details -->
      <div class="w-full px-8 pb-6 flex items-end justify-between z-20 flex-shrink-0">
        <!-- Bottom Left: Command Glossary -->
        <div class="flex flex-col space-y-1.5 text-xs text-neutral-400 font-mono select-none">
          <div class="flex items-center space-x-3">
            <button
              type="button"
              on:click={() => openGameHub($selectedGame)}
              class="flex items-center space-x-1.5 hover:text-neutral-200 transition-colors group cursor-pointer"
              title="Open Game Hub (A or Enter)"
            >
              <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-300 group-hover:text-white text-3xs font-bold">
                A / ↵
              </span>
              <span class="group-hover:text-neutral-200">Game Hub</span>
            </button>

            <span class="text-neutral-700">·</span>

            <button
              type="button"
              on:click={() => cycleSortOrder()}
              class="flex items-center space-x-1.5 hover:text-neutral-200 transition-colors group cursor-pointer"
              title="Cycle library sort order (Y)"
            >
              <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-300 group-hover:text-white text-3xs font-bold">
                Y
              </span>
              <span class="group-hover:text-neutral-200">Sort: {getSortLabel($sortBy)}</span>
            </button>
          </div>

          <div class="flex items-center">
            <button
              type="button"
              on:click={() => setLibraryMode('grid')}
              class="flex items-center space-x-1.5 hover:text-neutral-200 transition-colors group cursor-pointer"
              title="Switch to Library Grid View (/)"
            >
              <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-300 group-hover:text-white text-3xs font-bold">
                /
              </span>
              <span class="group-hover:text-neutral-200">Search Grid</span>
            </button>
          </div>
        </div>

        <!-- Bottom Right: Game Details & Telemetry -->
        {#if $selectedGame}
          <div class="flex flex-col items-end space-y-0.5 text-2xs text-neutral-400 font-mono text-right max-w-sm truncate">
            <div>
              <span class="text-neutral-500">Source:</span>
              <span class="text-neutral-200 font-semibold">{gameSource}</span>
              {#if metaInfo.provider && metaInfo.provider !== 'None'}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-500">Meta:</span>
                <span class="text-neutral-300 font-semibold">{metaInfo.provider}{metaInfo.id ? ` #${metaInfo.id}` : ''}</span>
              {/if}
            </div>
            <div>
              <span class="text-neutral-500">Platform:</span>
              <span class="text-neutral-300 font-semibold">{$selectedGame.platform || 'Linux'}</span>
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-300">{$selectedGame.runner || $configStore?.default_runner || 'UMU'}</span>
              <span class="text-neutral-500">·</span>
              <span class="text-neutral-300 font-semibold">{formatBytes($selectedGame.diskSizeBytes)}</span>
              {#if $selectedGame.playtimeSeconds > 0}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-200 font-medium">{formatPlaytime($selectedGame.playtimeSeconds)}</span>
              {/if}
              {#if $selectedGame.lastPlayed > 0}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-400">Played {formatLastPlayed($selectedGame.lastPlayed)}</span>
              {/if}
              {#if $selectedGame.installedAt > 0}
                <span class="text-neutral-500">·</span>
                <span class="text-neutral-400">Installed {formatInstallDate($selectedGame.installedAt)}</span>
              {/if}
            </div>
            <div class="truncate max-w-full text-neutral-500" title={$selectedGame.gameDir}>
              <span class="text-neutral-500">Path:</span>
              <span class="text-neutral-500">{$selectedGame.gameDir}</span>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
