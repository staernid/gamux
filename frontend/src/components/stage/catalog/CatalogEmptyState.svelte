<script>
    import Icon from '../../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [onsuggest]
   */

  /** @type {Props} */
  let { onsuggest = null } = $props();

  const suggestions = [
    { label: 'Hades II', query: 'Hades II' },
    { label: 'Balatro', query: 'Balatro' },
    { label: 'Cyberpunk 2077', query: 'Cyberpunk 2077' },
    { label: 'Roguelike', query: 'Roguelike' },
    { label: 'Deckbuilder', query: 'Deckbuilder' },
    { label: 'Co-op', query: 'Co-op' },
    { label: 'Sci-Fi', query: 'Sci-Fi' },
  ];

  function handleSelect(query) {
    onsuggest?.(query);
  }
</script>

<div class="h-80 flex flex-col items-center justify-center text-neutral-500 font-mono text-center px-4">
  <!-- Glowing Icon Badge -->
  <div class="w-14 h-14 rounded-2xl bg-surface-card border border-surface-border flex items-center justify-center text-neutral-400 mb-3 shadow-md">
    <Icon name="globe" className="w-7 h-7" />
  </div>

  <h3 class="text-sm font-bold text-neutral-200 tracking-tight">
    Online Store Catalog Discovery
  </h3>

  <p class="text-xs text-neutral-400 max-w-md leading-relaxed mt-1">
    Search Steam, GOG, and RAWG for uninstalled titles, explore metadata, or match store entries.
  </p>

  <!-- Quick Discovery Suggestions -->
  <div class="mt-6 flex flex-col items-center space-y-2">
    <span class="text-3xs font-mono font-bold uppercase tracking-wider text-neutral-500">
      Quick Discovery Seeds
    </span>
    <div class="flex flex-wrap justify-center gap-1.5 max-w-lg">
      {#each suggestions as s}
        <button
          type="button"
          onclick={() => handleSelect(s.query)}
          class="px-2.5 py-1 rounded-lg text-2xs font-mono bg-surface-card hover:bg-surface-cardHover border border-surface-border hover:border-surface-borderHighlight text-neutral-300 hover:text-white transition-all cursor-pointer shadow-sm active:scale-95"
        >
          {s.label}
        </button>
      {/each}
    </div>
  </div>

  <!-- Keyboard Shortcuts Legend -->
  <div class="mt-8 flex items-center space-x-3 text-3xs text-neutral-500">
    <span class="flex items-center space-x-1">
      <kbd class="px-1.5 py-0.5 rounded bg-surface-header border border-surface-border text-neutral-400 font-mono">↑↓←→</kbd>
      <span>Navigate</span>
    </span>
    <span>·</span>
    <span class="flex items-center space-x-1">
      <kbd class="px-1.5 py-0.5 rounded bg-surface-header border border-surface-border text-neutral-400 font-mono">Enter</kbd>
      <span>Select</span>
    </span>
    <span>·</span>
    <span class="flex items-center space-x-1">
      <kbd class="px-1.5 py-0.5 rounded bg-surface-header border border-surface-border text-neutral-400 font-mono">Esc</kbd>
      <span>Close</span>
    </span>
  </div>
</div>
