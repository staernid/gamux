<script>
  import { createEventDispatcher } from 'svelte';
  import { formatPlaytime } from '../../utils/game.js';
  import { formatBytes } from '../../utils/format.js';

  export let game = null;
  export let selected = false;
  export let isRunning = false;
  export let className = '';
  export let onclick = null;
  export let onplay = null;

  const dispatch = createEventDispatcher();

  $: isReady = game ? (game.isReady || game.isPatched) : false;
  $: isStub = game ? Boolean(game.isStub) : false;

  function handleClick() {
    dispatch('click', { game });
    if (typeof onclick === 'function') onclick({ game });
  }

  function handlePlay(e) {
    e.stopPropagation();
    dispatch('play', { game });
    if (typeof onplay === 'function') onplay({ game });
  }
</script>

{#if game}
  <div
    role="row"
    tabindex="0"
    on:click={handleClick}
    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleClick(); } }}
    class="group w-full h-10 px-4 flex items-center justify-between border-b border-[#18181b] hover:bg-[#18181b]/80 transition-colors text-xs font-mono select-none cursor-pointer focus:outline-none {selected ? 'bg-[#1c1c20] border-l-2 border-l-white' : ''} {className}"
    data-testid="compact-game-row"
  >
    <!-- Left: Status Pip + Title -->
    <div class="flex items-center space-x-3 min-w-0 flex-1 pr-4">
      {#if isRunning}
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse flex-shrink-0" title="Running"></span>
      {:else if !isReady && !isStub}
        <span class="w-2 h-2 rounded-full bg-amber-400 flex-shrink-0" title="Setup Required"></span>
      {:else}
        <span class="w-2 h-2 rounded-full bg-neutral-700 flex-shrink-0"></span>
      {/if}

      <span class="font-bold text-neutral-200 group-hover:text-white truncate" title={game.name}>
        {game.name}
      </span>
    </div>

    <!-- Center/Right Columns: Platform, Size, Playtime, Runner -->
    <div class="flex items-center space-x-6 text-neutral-400 text-3xs flex-shrink-0">
      <span class="w-16 uppercase text-neutral-500 truncate">{game.platform || 'win64'}</span>
      <span class="w-20 text-right">{game.diskSizeBytes ? formatBytes(game.diskSizeBytes) : '—'}</span>
      <span class="w-24 text-right">{game.playtimeSeconds ? formatPlaytime(game.playtimeSeconds) : '—'}</span>
      <span class="w-24 text-right text-neutral-500 truncate">{game.runner || 'UMU'}</span>

      <!-- Fast Play Trigger Button -->
      <button
        type="button"
        on:click={handlePlay}
        class="px-2.5 py-1 rounded-lg bg-neutral-800 hover:bg-white hover:text-black text-neutral-300 transition-colors text-4xs font-bold uppercase tracking-wider"
      >
        Play
      </button>
    </div>
  </div>
{/if}
