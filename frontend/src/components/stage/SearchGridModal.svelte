<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import { games, selectedGame } from '../../stores/library.js';
  import { getPlatformColors, isSamePath } from '../../utils/game.js';

  const dispatch = createEventDispatcher();

  let searchInputEl;
  let query = '';
  let focusedIndex = 0;
  let gridContainerEl;

  $: allGames = $games || [];
  $: filtered = query.trim()
    ? allGames.filter(g => {
        const q = query.toLowerCase();
        const name = (g.name || '').toLowerCase();
        const appId = String(g.appId || '');
        return name.includes(q) || appId.includes(q);
      })
    : allGames;

  $: {
    if (focusedIndex >= filtered.length) {
      focusedIndex = Math.max(0, filtered.length - 1);
    }
  }

  onMount(() => {
    if (searchInputEl) {
      searchInputEl.focus();
    }
    // Set initial focus to current selected game if found
    if ($selectedGame && filtered.length > 0) {
      const idx = filtered.findIndex(g => isSamePath(g.gameDir, $selectedGame.gameDir));
      if (idx >= 0) focusedIndex = idx;
    }
  });

  function selectAndClose(game) {
    if (!game) return;
    selectedGame.set(game);
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
      if (filtered[focusedIndex]) {
        selectAndClose(filtered[focusedIndex]);
      }
      return;
    }

    // Grid keyboard navigation
    const cols = 6;
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      if (focusedIndex < filtered.length - 1) focusedIndex += 1;
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      if (focusedIndex > 0) focusedIndex -= 1;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (focusedIndex + cols < filtered.length) focusedIndex += cols;
      else focusedIndex = filtered.length - 1;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (focusedIndex - cols >= 0) focusedIndex -= cols;
      else focusedIndex = 0;
    }
  }
</script>

<div
  role="dialog"
  aria-modal="true"
  tabindex="-1"
  on:keydown={handleKeydown}
  class="fixed inset-0 z-50 bg-[#09090c]/92 backdrop-blur-2xl flex flex-col p-6 sm:p-10 select-none animate-in fade-in duration-150 outline-none"
>
  <!-- Top Search Bar Header -->
  <div class="flex items-center justify-between max-w-5xl w-full mx-auto mb-6 flex-shrink-0">
    <div class="relative flex-1 max-w-xl flex items-center">
      <svg class="w-5 h-5 text-slate-400 absolute left-4 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        bind:this={searchInputEl}
        bind:value={query}
        type="text"
        placeholder="Type to filter titles..."
        class="w-full bg-[#14141d] border border-[#262638] focus:border-[#1a9fff] rounded-2xl pl-12 pr-10 py-3 text-base text-white outline-none font-mono transition-all shadow-xl placeholder:text-slate-500"
      />
      {#if query}
        <button
          type="button"
          on:click={() => { query = ''; searchInputEl?.focus(); }}
          class="absolute right-4 text-slate-400 hover:text-white text-sm font-bold"
        >
          ✕
        </button>
      {/if}
    </div>

    <!-- Stats & Close Hint -->
    <div class="flex items-center space-x-4 pl-4 text-xs font-mono text-slate-400">
      <span>{filtered.length} {filtered.length === 1 ? 'game' : 'games'}</span>
      <button
        type="button"
        on:click={() => dispatch('close')}
        class="px-3 py-1.5 rounded-xl bg-[#1a1a24] hover:bg-[#252535] border border-[#2b2b3c] text-slate-300 hover:text-white transition-colors flex items-center space-x-1.5"
      >
        <span class="px-1.5 py-0.5 rounded bg-[#161622] border border-[#262638] text-slate-400 font-bold text-[10px]">Esc / B</span>
        <span class="text-slate-400">Close</span>
      </button>
    </div>
  </div>

  <!-- Grid Content Area -->
  <div
    bind:this={gridContainerEl}
    class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-2 custom-scrollbar"
  >
    {#if filtered.length === 0}
      <div class="h-64 flex flex-col items-center justify-center text-slate-500 text-sm font-mono">
        <p>No games matching "{query}"</p>
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

              <!-- Mini Platform badge -->
              <div
                class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-[9px] font-black uppercase tracking-wider text-white shadow-md backdrop-blur-md"
                style="background-color: {pColors.badgeBg};"
              >
                {pColors.label}
              </div>
            </div>

            <!-- Title Below -->
            <span class="mt-2 text-xs font-semibold text-slate-200 group-hover:text-white line-clamp-2 max-w-full px-1">
              {game.name}
            </span>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>
