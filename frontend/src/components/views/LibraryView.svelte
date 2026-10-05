<script>
  import { get } from 'svelte/store';
  import CoverCard from '../blocks/CoverCard.svelte';
  import SearchFilterBar from '../blocks/SearchFilterBar.svelte';
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
    isSamePath
  } from '../../utils/game.js';
  import { playTick, playSelect } from '../../utils/audio.js';

  let searchInputEl = $state();
  let focusedIndex = 0;

  const sortOptions = [
    { id: 'name_asc', label: 'Name (A-Z)' },
    { id: 'name_desc', label: 'Name (Z-A)' },
    { id: 'last_played_desc', label: 'Last Played' },
    { id: 'playtime_desc', label: 'Playtime' },
    { id: 'install_desc', label: 'Installed' },
    { id: 'size_desc', label: 'Size' }
  ];

  function openGameHub(game) {
    if (!game) return;
    selectedGame.set(game);
    playSelect();
    activeView.set('game-hub');
  }

  export function step(dir) {
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
      searchInputEl?.focus();
    }, 50);
  }
</script>

<div class="flex-1 flex flex-col min-h-0 w-full relative">
  <!-- Desktop Grid View Mode -->
  <div class="flex-1 flex flex-col min-h-0 pt-16 sm:pt-20 px-6 sm:px-10 pb-6 max-w-7xl mx-auto w-full animate-fade-in">
    <!-- Toolbar: Search, Filters & Sorters -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-5 border-b border-surface-border">
      <!-- Left: Search Filter Bar -->
      <SearchFilterBar
        bind:this={searchInputEl}
        bind:query={$searchQuery}
        placeholder="Search library by title, appid, platform..."
        totalCount={$games.length}
        filteredCount={$filteredGames.length}
        className="flex-1 max-w-md"
      />

        <!-- Middle: Status Filter Chips -->
        <div class="flex items-center space-x-1.5 overflow-x-auto py-0.5">
          <button
            type="button"
            onclick={() => { $activeFilter = 'all'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {$activeFilter === 'all' ? 'bg-surface-cardHover text-white font-bold shadow-sm border border-surface-borderHighlight' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
          >
            All
          </button>
          <button
            type="button"
            onclick={() => { $activeFilter = 'patched'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all flex items-center space-x-1.5 cursor-pointer {$activeFilter === 'patched' ? 'bg-surface-cardHover text-white font-bold shadow-sm border border-surface-borderHighlight' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
          >
            <span>Ready</span>
            {#if $stats.patchedCount > 0}
              <span class="px-1 py-0.2 rounded text-3xs {$activeFilter === 'patched' ? 'bg-neutral-800 text-neutral-200 border border-neutral-700' : 'bg-neutral-800 text-neutral-300'}">{$stats.patchedCount}</span>
            {/if}
          </button>
          <button
            type="button"
            onclick={() => { $activeFilter = 'clean'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all flex items-center space-x-1.5 cursor-pointer {$activeFilter === 'clean' ? 'bg-surface-cardHover text-white font-bold shadow-sm border border-surface-borderHighlight' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
          >
            <span>Unpatched</span>
            {#if $stats.cleanCount > 0}
              <span class="px-1 py-0.2 rounded text-3xs {$activeFilter === 'clean' ? 'bg-neutral-800 text-neutral-200 border border-neutral-700' : 'bg-neutral-800 text-neutral-300'}">{$stats.cleanCount}</span>
            {/if}
          </button>
          {#if $stats.updatesCount > 0}
            <button
              type="button"
              onclick={() => { $activeFilter = 'updates'; }}
              class="px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-all flex items-center space-x-1.5 cursor-pointer {$activeFilter === 'updates' ? 'bg-surface-cardHover text-white font-bold shadow-sm border border-surface-borderHighlight' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
            >
              <span>Updates</span>
              <span class="px-1 py-0.2 rounded text-3xs bg-amber-500/20 text-amber-300 font-bold border border-amber-500/30">{$stats.updatesCount}</span>
            </button>
          {/if}
        </div>

        <!-- Right: Sort Dropdown -->
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
                onclick={() => { $searchQuery = ''; }}
                class="text-xs text-neutral-300 hover:underline font-bold mt-1"
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
                playtimeSeconds={game.playtimeSeconds}
                lastPlayed={game.lastPlayed}
                diskSizeBytes={game.diskSizeBytes}
                size="full"
                className="w-full"
                onclick={() => openGameHub(game)}
              />
            {/each}
          </div>
        {/if}
    </div>
  </div>
</div>
