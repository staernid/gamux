<script>
    import Icon from '../../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').StoreSearchResult} item
   * @property {boolean} [isFocused]
   * @property {import('../../../types/domain').Game | null} [localMatch]
   * @property {((item: import('../../../types/domain').StoreSearchResult) => void) | null} [onselect]
   * @property {(() => void) | null} [onfocus]
   */

  /** @type {Props} */
  let {
    item,
    isFocused = false,
    localMatch = null,
    onselect = null,
    onfocus = null
  } = $props();

  let imgFailed = $state(false);

  let storeName = $derived((item?.store || 'steam').toUpperCase());
  let releaseYear = $derived(item?.release_date ? item.release_date.slice(0, 4) : '');
  let hasPoster = $derived(Boolean(item?.poster_url) && !imgFailed);

  function handleClick() {
    onselect?.(item);
  }

  function handleMouseEnter() {
    onfocus?.();
  }

  function handleImgError() {
    imgFailed = true;
  }
</script>

<button
  type="button"
  onclick={handleClick}
  onmouseenter={handleMouseEnter}
  class="contain-card group cursor-pointer flex flex-col items-center text-center transition-transform duration-150 transform {isFocused ? 'scale-105 z-10' : 'hover:scale-102 opacity-90 hover:opacity-100'} bg-transparent border-0 p-0 text-left outline-none"
  aria-label="{item?.name || 'Game'} - {storeName} {localMatch ? (localMatch.isInstalled ? '(Installed)' : '(In Library)') : ''}"
>
  <!-- Card Poster Container -->
  <div
    class="gpu-layer relative w-full aspect-[2/3] rounded-2xl overflow-hidden transition-colors duration-150 bg-surface-card border {isFocused ? 'border-white ring-2 ring-white/60 shadow-[0_8px_25px_-4px_rgba(255,255,255,0.25)]' : 'border-surface-border hover:border-surface-borderHighlight shadow-md'}"
  >
    {#if hasPoster}
      <img
        src={item.poster_url}
        alt={item.name}
        onerror={handleImgError}
        loading="lazy"
        class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
      />
    {:else}
      <!-- Sleek Fallback when Poster is Absent or Fails -->
      <div class="w-full h-full flex flex-col items-center justify-center p-3 text-center bg-surface-header border border-dashed border-surface-border text-neutral-500 font-mono">
        <Icon name="gamepad" className="w-8 h-8 mb-2 text-neutral-600 group-hover:text-neutral-400 transition-colors" />
        <span class="text-3xs line-clamp-3 font-semibold text-neutral-400 group-hover:text-neutral-200">
          {item?.name}
        </span>
      </div>
    {/if}

    <!-- Store Provider Badge (Top Left) -->
    <div class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-4xs font-mono font-black uppercase tracking-wider text-white shadow-md bg-black/85 border border-white/10">
      {storeName}
    </div>

    <!-- Ownership Badge (Top Right) -->
    {#if localMatch}
      <div
        class="absolute top-2 right-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider shadow-md {localMatch.isInstalled ? 'bg-emerald-500/90 text-neutral-950 border border-emerald-400/40' : 'bg-surface-elevated text-neutral-200 border border-surface-borderHighlight'}"
      >
        {localMatch.isInstalled ? 'Installed' : 'In Library'}
      </div>
    {/if}

    <!-- Visual Action Cue on Hover/Focus -->
    <div
      class="absolute inset-x-2 bottom-2 py-1 px-2 rounded-lg bg-surface-header border border-surface-borderHighlight text-neutral-200 text-4xs font-mono font-bold tracking-tight text-center shadow-lg transition-opacity duration-150 {isFocused ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}"
    >
      <span>✦ {localMatch ? 'Open in Library' : 'Open in Hub'}</span>
    </div>
  </div>

  <!-- Title & Metadata Subtitle -->
  <div class="mt-2.5 flex flex-col items-center max-w-full px-1 space-y-0.5 w-full">
    <span class="text-xs font-semibold text-neutral-200 group-hover:text-white line-clamp-2 leading-snug w-full text-center transition-colors">
      {item.name}
    </span>
    <span class="text-3xs font-mono text-neutral-500 truncate w-full text-center">
      #{item.id} {releaseYear ? `· ${releaseYear}` : ''}
    </span>
  </div>
</button>
