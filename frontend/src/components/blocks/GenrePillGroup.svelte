<script>
  
  /**
   * @typedef {Object} Props
   * @property {string[]} [genres]
   * @property {string | null} [selectedGenre]
   * @property {boolean} [clickable]
   * @property {number} [maxVisible]
   * @property {string} [className]
   * @property {((genre: string) => void) | null} [onselect]
   */

  /** @type {Props} */
  let {
    genres = [],
    selectedGenre = null,
    clickable = true,
    maxVisible = 6,
    className = '',
    onselect = null
  } = $props();

  let list = $derived((Array.isArray(genres) ? genres : (genres ? [genres] : []))
    .map(g => (typeof g === 'string' ? g.trim() : (g?.description || g?.name || '')))
    .filter(Boolean));

  let visibleList = $derived(maxVisible ? list.slice(0, maxVisible) : list);
  let remainingCount = $derived(maxVisible && list.length > maxVisible ? list.length - maxVisible : 0);

  function handleClick(genre) {
    if (!clickable) return;
    onselect?.({ genre });
  }
</script>

<div class="flex flex-wrap items-center gap-1.5 {className}" data-testid="genre-pill-group">
  {#each visibleList as genre}
    {@const isSelected = selectedGenre && selectedGenre.toLowerCase() === genre.toLowerCase()}
    <button
      type="button"
      disabled={!clickable}
      onclick={() => handleClick(genre)}
      class="px-2 py-0.5 rounded-lg text-3xs font-mono font-bold uppercase tracking-wider transition-all duration-150 border select-none {clickable ? 'cursor-pointer' : 'cursor-default'} {isSelected ? 'bg-neutral-100 text-neutral-950 border-white shadow-sm' : 'bg-surface-card/90 hover:bg-surface-cardHover text-neutral-300 border-surface-border hover:border-neutral-500'}"
    >
      {genre}
    </button>
  {/each}

  {#if remainingCount > 0}
    <span class="px-1.5 py-0.5 rounded-lg text-4xs font-mono text-neutral-500 bg-surface-header border border-surface-border">
      +{remainingCount}
    </span>
  {/if}
</div>
