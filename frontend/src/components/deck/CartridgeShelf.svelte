<script>
  import { onMount, tick } from 'svelte';
  import { filteredGames, selectedGame, activeFilter, sortBy, stats } from '../../stores/library.js';
  import PosterArt from '../common/PosterArt.svelte';

  let shelfReelEl;

  const filters = [
    { id: 'all', label: 'All', countKey: 'totalCount' },
    { id: 'patched', label: 'Ready', countKey: 'patchedCount' },
    { id: 'clean', label: 'Original', countKey: 'cleanCount' },
    { id: 'lutris', label: 'Lutris', countKey: 'lutrisCount' },
    { id: 'updates', label: 'Updates', countKey: 'updatesCount' },
  ];

  // Auto-scroll the active game cartridge into view whenever selectedGame updates
  $: if ($selectedGame && shelfReelEl) {
    scrollToActive();
  }

  async function scrollToActive() {
    await tick();
    if (!shelfReelEl || !$selectedGame) return;
    const activeEl = shelfReelEl.querySelector('.cartridge.active');
    if (activeEl) {
      activeEl.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' });
    }
  }

  function handleSelect(game) {
    $selectedGame = game;
  }
</script>

<div class="h-44 sm:h-52 bg-[#0c0c10] border-t border-[#1f1f2a] flex flex-col flex-shrink-0 z-20 select-none">
  <!-- Shelf Top Strip: Filters & Sorter -->
  <div class="h-8 px-4 sm:px-6 flex items-center justify-between border-b border-[#181822] text-xs">
    <!-- Filter Chips -->
    <div class="flex items-center space-x-1.5 overflow-x-auto no-scrollbar">
      {#each filters as f}
        {@const isActive = $activeFilter === f.id}
        {@const count = $stats[f.countKey] || 0}
        <button
          type="button"
          on:click={() => { $activeFilter = f.id; }}
          class="px-2 py-0.5 rounded-md transition-all flex items-center space-x-1 text-[11px] {isActive ? 'bg-[#222230] text-white font-bold' : 'text-slate-500 hover:text-slate-300'}"
        >
          {#if f.id === 'updates' && count > 0}
            <span class="pip pip-update"></span>
          {/if}
          <span>{f.label}</span>
          <span class="font-mono text-[10px] text-slate-500">({count})</span>
        </button>
      {/each}
    </div>

    <!-- Sort Dropdown -->
    <div class="flex items-center space-x-1 text-slate-500 text-[11px] font-mono">
      <span>Sort:</span>
      <select
        bind:value={$sortBy}
        class="bg-transparent text-slate-400 font-bold outline-none cursor-pointer hover:text-slate-200"
      >
        <option value="name_asc" class="bg-[#14141c] text-white">Name (A-Z)</option>
        <option value="name_desc" class="bg-[#14141c] text-white">Name (Z-A)</option>
        <option value="size_desc" class="bg-[#14141c] text-white">Size (Largest)</option>
        <option value="size_asc" class="bg-[#14141c] text-white">Size (Smallest)</option>
        <option value="status" class="bg-[#14141c] text-white">Status</option>
      </select>
    </div>
  </div>

  <!-- Cartridge Reel / Horizontal Dock -->
  <div
    bind:this={shelfReelEl}
    class="flex-1 px-4 sm:px-6 py-2.5 flex items-center space-x-3 sm:space-x-4 overflow-x-auto overflow-y-hidden"
  >
    {#if $filteredGames.length > 0}
      {#each $filteredGames as game (game.GameDir || game.Name)}
        {@const isSelected = $selectedGame?.GameDir === game.GameDir}
        <button
          type="button"
          on:click={() => handleSelect(game)}
          class="cartridge relative flex-shrink-0 w-24 sm:w-28 aspect-[2/3] overflow-hidden group cursor-pointer text-left focus:outline-none {isSelected ? 'active' : ''}"
          title={game.Name}
        >
          <!-- Capsule Art -->
          <PosterArt
            appId={game.AppID}
            name={game.Name}
            className="w-full h-full object-cover"
          />

          <!-- Overlay gradient on bottom -->
          <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent pointer-events-none"></div>

          <!-- Status Pip in Top-Right Corner -->
          <div class="absolute top-1.5 right-1.5 pointer-events-none">
            {#if game.hasUpdatesOrIssues}
              <span class="pip pip-update"></span>
            {:else if game.isPatched && game.lutris_registered}
              <span class="pip pip-ready"></span>
            {:else if game.isPatched}
              <span class="pip pip-update"></span>
            {:else}
              <span class="pip pip-clean"></span>
            {/if}
          </div>

          <!-- Compact Title on Bottom -->
          <div class="absolute bottom-1.5 left-1.5 right-1.5 pointer-events-none">
            <span class="block text-[10px] font-bold text-white truncate leading-tight drop-shadow-sm">
              {game.Name}
            </span>
          </div>
        </button>
      {/each}
    {:else}
      <div class="w-full text-center text-slate-500 font-mono text-xs py-4">
        No games match the current filter or search.
      </div>
    {/if}
  </div>
</div>
