<script>
    import { formatPlaytime } from '../../utils/game.js';
  import { formatBytes } from '../../utils/format.js';

  /**
   * @typedef {Object} Props
   * @property {import('../../types/domain').Game} [game]
   * @property {boolean} [selected]
   * @property {boolean} [isRunning]
   * @property {string} [className]
   * @property {(() => void) | null} [onclick]
   * @property {(() => void) | null} [onplay]
   */

  /** @type {Props} */
  let {
    game = null,
    selected = false,
    isRunning = false,
    className = '',
    onclick = null,
    onplay = null
  } = $props();

  let isReady = $derived(game ? (game.isReady || game.isPatched) : false);
  let isStub = $derived(game ? Boolean(game.isStub) : false);

  function handleClick() {
    onclick?.({ game });
  }

  function handlePlay(e) {
    e.stopPropagation();
    onplay?.({ game });
  }
</script>

{#if game}
  <div
    role="row"
    tabindex="0"
    aria-label={name}
    onclick={handleClick}
    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleClick(); } }}
    class="group w-full h-10 px-4 flex items-center justify-between border-b border-surface-border hover:bg-surface-header/80 transition-colors text-xs font-mono select-none cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-neutral-400 {selected ? 'bg-surface-header border-l-2 border-l-white' : ''} {className}"
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
      <span class="w-24 text-right text-neutral-500 truncate">{game.platform === 'linux' ? 'Linux Native' : (game.runner || 'UMU')}</span>

      <!-- Fast Play Trigger Button -->
      <button
        type="button"
        onclick={handlePlay}
        class="px-2.5 py-1 rounded-lg bg-neutral-800 hover:bg-white hover:text-black text-neutral-300 transition-colors text-4xs font-bold uppercase tracking-wider"
      >
        Play
      </button>
    </div>
  </div>
{/if}
