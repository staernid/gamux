<script>
  import { createEventDispatcher } from 'svelte';

  export let genres = [];
  export let selectedGenre = null;
  export let clickable = true;
  export let maxVisible = 6;
  export let className = '';
  export let onselect = null;

  const dispatch = createEventDispatcher();

  $: list = (Array.isArray(genres) ? genres : (genres ? [genres] : []))
    .map(g => (typeof g === 'string' ? g.trim() : (g?.description || g?.name || '')))
    .filter(Boolean);

  $: visibleList = maxVisible ? list.slice(0, maxVisible) : list;
  $: remainingCount = maxVisible && list.length > maxVisible ? list.length - maxVisible : 0;

  function handleClick(genre) {
    if (!clickable) return;
    dispatch('select', { genre });
    if (typeof onselect === 'function') {
      onselect({ genre });
    }
  }
</script>

<div class="flex flex-wrap items-center gap-1.5 {className}" data-testid="genre-pill-group">
  {#each visibleList as genre}
    {@const isSelected = selectedGenre && selectedGenre.toLowerCase() === genre.toLowerCase()}
    <button
      type="button"
      disabled={!clickable}
      on:click={() => handleClick(genre)}
      class="px-2 py-0.5 rounded-lg text-3xs font-mono font-bold uppercase tracking-wider transition-all duration-150 border select-none {clickable ? 'cursor-pointer' : 'cursor-default'} {isSelected ? 'bg-neutral-100 text-neutral-950 border-white shadow-sm' : 'bg-[#18181b]/90 hover:bg-[#27272a] text-neutral-300 border-[#27272a] hover:border-neutral-500'}"
    >
      {genre}
    </button>
  {/each}

  {#if remainingCount > 0}
    <span class="px-1.5 py-0.5 rounded-lg text-4xs font-mono text-neutral-500 bg-[#121214] border border-[#27272a]">
      +{remainingCount}
    </span>
  {/if}
</div>
