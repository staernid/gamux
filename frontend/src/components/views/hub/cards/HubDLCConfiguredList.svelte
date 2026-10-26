<script>
  /**
   * @typedef {Object} Props
   * @property {Array<any>} [dlcs]
   * @property {string} [appId]
   * @property {boolean} [isSyncing]
   * @property {() => void} [onsync]
   */

  /** @type {Props} */
  let {
    dlcs = [],
    appId = '',
    isSyncing = false,
    onsync = undefined
  } = $props();
</script>

<div class="p-3.5 bg-surface-header border border-surface-border rounded-xl text-xs space-y-3">
  <div class="flex items-center justify-between">
    <div class="space-y-0.5">
      <strong class="text-neutral-200 block font-bold">Downloadable Content (DLC)</strong>
      <span class="text-neutral-400 text-2xs">
        {dlcs.length} DLCs configured for this game
      </span>
    </div>
    {#if appId && appId !== '0'}
      <button
        type="button"
        onclick={() => onsync?.()}
        disabled={isSyncing}
        class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border transition-colors font-bold text-2xs flex items-center space-x-1.5 cursor-pointer"
      >
        <svg class="w-3 h-3 {isSyncing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>Sync Steam DLCs</span>
      </button>
    {/if}
  </div>

  {#if dlcs.length > 0}
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
      {#each dlcs as dlc (dlc.appId)}
        <div class="p-2.5 bg-surface-card border border-surface-border rounded-lg flex items-center justify-between gap-2">
          <div class="min-w-0">
            <span class="font-bold text-neutral-200 text-xs block truncate">{dlc.name || `DLC ${dlc.appId}`}</span>
            <span class="text-3xs font-mono text-neutral-400">AppID #{dlc.appId}</span>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-bold bg-emerald-950/50 text-emerald-400 border border-emerald-800/40 flex-shrink-0">
            Active
          </span>
        </div>
      {/each}
    </div>
  {/if}
</div>
