<script>
  /**
   * @typedef {Object} Props
   * @property {number} [playerCount]
   * @property {string} [label]
   * @property {string} [className]
   */

  /** @type {Props} */
  let { playerCount = 0, label = 'playing right now', className = '' } = $props();

  let count = $derived(typeof playerCount === 'number' && !isNaN(playerCount) ? playerCount : 0);
</script>

{#if count > 0}
  <div
    class="inline-flex items-center space-x-1.5 px-2.5 py-1 rounded-full bg-[#121214] border border-[#27272a] text-3xs font-mono text-neutral-300 shadow-sm select-none {className}"
    data-testid="live-community-badge"
    title="{count.toLocaleString()} players currently in-game"
  >
    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shadow-sm"></span>
    <span class="sr-only">{count.toLocaleString()} {label}</span>
    <span class="font-bold text-neutral-200">{count.toLocaleString()}</span>
    <span class="text-neutral-400">{label}</span>
  </div>
{/if}
