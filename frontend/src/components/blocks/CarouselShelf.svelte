<script>
    import CoverCard from './CoverCard.svelte';
  import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [games]
   * @property {number} [selectedIndex]
   * @property {string} [cardSize]
   * @property {string} [className]
   * @property {any} [onselect]
   */

  /** @type {Props} */
  let {
    games = [],
    selectedIndex = $bindable(0),
    cardSize = 'md',
    className = '',
    onselect = null
  } = $props();
  let trackEl = $state();

  let list = $derived(Array.isArray(games) ? games : []);

  function handleCardClick(game, index) {
    selectedIndex = index;
    onselect?.({ game, index });scrollToIndex(index);
  }

  function scrollToIndex(idx) {
    if (!trackEl) return;
    const cards = trackEl.querySelectorAll('[data-testid="cover-card"]');
    if (cards[idx] && typeof cards[idx].scrollIntoView === 'function') {
      cards[idx].scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' });
    }
  }

  export function step(dir) {
    if (list.length === 0) return;
    let next = selectedIndex + dir;
    if (next < 0) next = 0;
    if (next >= list.length) next = list.length - 1;
    selectedIndex = next;
    onselect?.({ game: list[next], index: next });scrollToIndex(next);
  }
</script>

<div class="relative w-full overflow-hidden select-none {className}" data-testid="carousel-shelf">
  <!-- Left Scroll Button -->
  {#if selectedIndex > 0}
    <button
      type="button"
      onclick={() => step(-1)}
      class="absolute left-2 top-1/2 -translate-y-1/2 z-20 w-10 h-10 rounded-full bg-black/70 hover:bg-black/95 border border-white/20 hover:border-white text-white flex items-center justify-center backdrop-blur-md shadow-xl transition-all cursor-pointer"
      title="Scroll Left"
      aria-label="Scroll Left"
    >
      <Icon name="chevron-left" className="w-5 h-5 text-white" />
    </button>
  {/if}

  <!-- Track of CoverCards -->
  <div
    bind:this={trackEl}
    class="flex items-center space-x-4 sm:space-x-6 overflow-x-auto py-4 px-6 custom-scrollbar scroll-smooth"
  >
    {#each list as game, idx (game.gameDir || game.appId || idx)}
      <CoverCard
        appId={game.appId}
        name={game.name}
        coverUrl={game.coverUrl}
        localCoverUrl={game.localCoverUrl}
        platform={game.platform}
        isRunning={Boolean(game.isRunning)}
        isReady={Boolean(game.isReady || game.isPatched)}
        isStub={Boolean(game.isStub)}
        playtimeSeconds={game.playtimeSeconds || 0}
        lastPlayed={game.lastPlayed || 0}
        diskSizeBytes={game.diskSizeBytes || 0}
        selected={idx === selectedIndex}
        size={cardSize}
        onclick={() => handleCardClick(game, idx)}
      />
    {/each}
  </div>

  <!-- Right Scroll Button -->
  {#if selectedIndex < list.length - 1}
    <button
      type="button"
      onclick={() => step(1)}
      class="absolute right-2 top-1/2 -translate-y-1/2 z-20 w-10 h-10 rounded-full bg-black/70 hover:bg-black/95 border border-white/20 hover:border-white text-white flex items-center justify-center backdrop-blur-md shadow-xl transition-all cursor-pointer"
      title="Scroll Right"
      aria-label="Scroll Right"
    >
      <Icon name="chevron-right" className="w-5 h-5 text-white" />
    </button>
  {/if}
</div>
