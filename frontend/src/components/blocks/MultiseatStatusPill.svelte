<script>
    import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [isCoop]
   * @property {Record<string, unknown> | null} [session]
   * @property {string} [p1Gamepad]
   * @property {string} [p2Gamepad]
   * @property {string} [className]
   */

  /** @type {Props} */
  let {
    isCoop = false,
    session = null,
    p1Gamepad = 'auto',
    p2Gamepad = 'auto',
    className = ''
  ,
    onclick
  } = $props();

  let isRunning = $derived(Boolean(session?.is_running));
</script>

<button
  type="button"
  onclick={() => onclick?.()}
  class="inline-flex items-center space-x-2 px-3 py-1.5 rounded-xl bg-[#121214]/90 hover:bg-[#1c1c20] border border-[#27272a] hover:border-neutral-500 text-3xs font-mono transition-all shadow-sm cursor-pointer select-none {className}"
  data-testid="multiseat-status-pill"
  title="Multiseat Co-op Session Management"
>
  <Icon name="users" className="w-3.5 h-3.5 text-neutral-400" />

  {#if isRunning}
    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
    <span class="text-emerald-300 font-bold">2P Active</span>
  {:else if isCoop}
    <span class="text-neutral-200 font-bold">Co-op Armed</span>
    <span class="text-neutral-500">·</span>
    <span class="text-neutral-400">P1: {p1Gamepad} | P2: {p2Gamepad}</span>
  {:else}
    <span class="text-neutral-400">Co-op Standby</span>
  {/if}
</button>
