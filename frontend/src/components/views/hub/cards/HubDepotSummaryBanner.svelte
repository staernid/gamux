<script>
  import { formatBytes } from '../../../../utils/format.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [isInstalled]
   * @property {number} [installedCount]
   * @property {number} [totalDepotsCount]
   * @property {number} [totalInstalledSize]
   * @property {number} [selectedCount]
   * @property {number} [selectedDepotsSize]
   * @property {string} [appId]
   * @property {boolean} [isActionRunning]
   * @property {boolean} [isLoading]
   * @property {any} [activeTransfer]
   * @property {() => void} [ondownloadSelected]
   * @property {() => void} [onrefresh]
   */

  /** @type {Props} */
  let {
    isInstalled = false,
    installedCount = 0,
    totalDepotsCount = 0,
    totalInstalledSize = 0,
    selectedCount = 0,
    selectedDepotsSize = 0,
    appId = '',
    isActionRunning = false,
    isLoading = false,
    activeTransfer = null,
    ondownloadSelected = undefined,
    onrefresh = undefined,
  } = $props();
</script>

<div class="space-y-4">
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3.5 bg-surface-header border border-surface-border rounded-xl text-xs">
    <div class="space-y-0.5">
      <strong class="text-neutral-200 block font-bold">DLC & Content Depots</strong>
      <span class="text-neutral-400 text-2xs">
        {#if isInstalled}
          {installedCount} of {totalDepotsCount} depots installed ({formatBytes(totalInstalledSize)})
        {:else}
          {totalDepotsCount} total depots discovered • {selectedCount} selected for download ({formatBytes(selectedDepotsSize)})
        {/if}
      </span>
    </div>
    <div class="flex items-center space-x-2">
      {#if !isInstalled && appId && totalDepotsCount > 0}
        <button
          type="button"
          onclick={() => ondownloadSelected?.()}
          disabled={isActionRunning || selectedCount === 0}
          class="px-3 py-1.5 rounded-lg bg-white hover:bg-neutral-200 text-neutral-950 font-bold text-2xs transition-colors flex items-center space-x-1.5 cursor-pointer shadow-sm"
        >
          <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>Download Selected ({selectedCount})</span>
        </button>
      {/if}
      <button
        type="button"
        onclick={() => onrefresh?.()}
        disabled={isLoading}
        class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border transition-colors font-bold text-2xs flex items-center space-x-1.5 cursor-pointer"
      >
        <svg class="w-3 h-3 {isLoading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>Refresh</span>
      </button>
    </div>
  </div>

  {#if activeTransfer}
    <div class="p-3 bg-surface-header border border-surface-borderHighlight rounded-xl space-y-2">
      <div class="flex items-center justify-between text-xs">
        <div class="flex items-center space-x-2">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span class="font-bold text-neutral-100">{activeTransfer.stage || 'Downloading Depots'}</span>
        </div>
        <span class="font-mono text-neutral-200 font-bold">{activeTransfer.percent || 0}%</span>
      </div>
      <div class="w-full bg-black/40 rounded-full h-1.5 overflow-hidden">
        <div class="bg-white h-full transition-all duration-300" style="width: {activeTransfer.percent || 0}%"></div>
      </div>
      <div class="flex items-center justify-between text-2xs text-neutral-400 font-mono">
        <span>{formatBytes(activeTransfer.currentBytes || 0)} / {formatBytes(activeTransfer.totalBytes || 0)}</span>
        <span>{formatBytes(activeTransfer.speedBps || 0)}/s</span>
      </div>
    </div>
  {/if}
</div>
