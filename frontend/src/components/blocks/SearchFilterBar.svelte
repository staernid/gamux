<script>
    import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {string} [query]
   * @property {string} [placeholder]
   * @property {number} [totalCount]
   * @property {number} [filteredCount]
   * @property {string} [className]
   * @property {((q: string) => void) | null} [oninput]
   */

  /** @type {Props} */
  let {
    query = $bindable(''),
    placeholder = 'Search titles, genres, or developers...',
    totalCount = 0,
    filteredCount = 0,
    className = '',
    oninput = null
  } = $props();
  let inputEl = $state();

  function handleInput(e) {
    query = e.target.value;
    oninput?.({ query });
  }

  function handleClear() {
    query = '';
    oninput?.({ query: '' });
    if (inputEl) inputEl.focus();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      if (query) {
        e.preventDefault();
        handleClear();
      }
    }
  }

  export function focus() {
    if (inputEl) inputEl.focus();
  }
</script>

<div
  class="relative flex items-center bg-[#121214] border border-[#27272a] focus-within:border-neutral-400 rounded-2xl px-3.5 py-2 transition-colors shadow-inner select-none {className}"
  data-testid="search-filter-bar"
>
  <!-- Search Magnifier Icon -->
  <Icon name="search" className="w-4 h-4 text-neutral-500 mr-2.5 flex-shrink-0" />

  <!-- Input Field -->
  <input
    bind:this={inputEl}
    type="text"
    value={query}
    oninput={handleInput}
    onkeydown={handleKeydown}
    {placeholder}
    class="bg-transparent border-0 outline-none text-xs text-neutral-100 placeholder-neutral-500 w-full min-w-0 font-normal"
  />

  <!-- Count Pill & Clear Button -->
  <div class="flex items-center space-x-2 ml-2 flex-shrink-0">
    {#if totalCount > 0}
      <span class="text-4xs font-mono text-neutral-500 bg-[#18181b] px-2 py-0.5 rounded-md border border-[#27272a]">
        {filteredCount !== undefined ? filteredCount : totalCount} / {totalCount}
      </span>
    {/if}

    {#if query}
      <button
        type="button"
        onclick={handleClear}
        class="text-neutral-400 hover:text-white p-0.5 rounded-full hover:bg-neutral-800 transition cursor-pointer"
        title="Clear Search (Esc)"
        aria-label="Clear Search"
      >
        <Icon name="close" className="w-3.5 h-3.5" />
      </button>
    {/if}
  </div>
</div>
