<script>
  import { formatBytes } from '../../../../utils/format.js';
  import { getCategoryLabel, getCategoryBadgeClass, getContentCategory } from '../../../../utils/depot.js';

  /**
   * @typedef {Object} Props
   * @property {any} depot
   * @property {boolean} [isInstalled]
   * @property {boolean} [isSelected]
   * @property {boolean} [isActionRunning]
   * @property {boolean} [canUninstall]
   * @property {boolean} [isDefaultLocale]
   * @property {() => void} [ontoggle]
   * @property {() => void} [ondownloadSoundtrack]
   * @property {() => void} [ondownload]
   * @property {() => void} [onuninstall]
   */

  /** @type {Props} */
  let {
    depot,
    isInstalled = false,
    isSelected = false,
    isActionRunning = false,
    canUninstall = false,
    isDefaultLocale = false,
    ontoggle = undefined,
    ondownloadSoundtrack = undefined,
    ondownload = undefined,
    onuninstall = undefined,
  } = $props();
</script>

<div class="p-3 bg-surface-card border border-surface-border hover:border-surface-borderHighlight rounded-xl flex items-center justify-between gap-3 transition-colors">
  <div class="flex items-center space-x-3 min-w-0">
    {#if !isInstalled}
      <!-- Pre-download selection checkbox -->
      <label class="flex items-center cursor-pointer flex-shrink-0" title="Select depot for download">
        <input
          type="checkbox"
          checked={isSelected}
          onchange={() => ontoggle?.()}
          class="w-4 h-4 rounded bg-surface-input border border-surface-border text-white cursor-pointer accent-white"
        />
      </label>
    {:else}
      <!-- Installed Status Indicator -->
      <div class="w-4 h-4 rounded-md flex items-center justify-center flex-shrink-0 {depot.installed ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-700/60' : 'bg-surface-header text-neutral-600 border border-surface-border'}">
        {#if depot.installed}
          <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
          </svg>
        {/if}
      </div>
    {/if}

    <div class="space-y-1 min-w-0">
      <div class="flex items-center space-x-2 flex-wrap gap-y-1">
        <span class="font-bold text-neutral-100 text-xs">{depot.name || `Depot ${depot.depot_id}`}</span>
        <span class="px-2 py-0.5 rounded-full text-3xs font-bold border {getCategoryBadgeClass(depot)}">
          {getCategoryLabel(depot)}
        </span>
        {#if depot.osarch}
          <span class="px-1.5 py-0.5 rounded text-3xs font-bold bg-surface-header text-neutral-400 border border-surface-border">
            {depot.osarch}-bit
          </span>
        {/if}
        {#if isDefaultLocale}
          <span class="px-1.5 py-0.5 rounded text-3xs font-bold bg-blue-950/40 text-blue-300 border border-blue-800/40">
            Active Locale
          </span>
        {/if}
      </div>
      <p class="text-2xs text-neutral-400 font-mono">
        {#if depot.file_count > 0}
          {depot.file_count} files • {formatBytes(depot.size || 0)}
        {:else if depot.size > 0}
          Estimated {formatBytes(depot.size)}
          {#if depot.download_size && depot.download_size !== depot.size}
            ({formatBytes(depot.download_size)} download)
          {/if}
        {:else}
          0 files • 0 B
        {/if}
      </p>
    </div>
  </div>

  <!-- Actions -->
  <div class="flex items-center space-x-2 flex-shrink-0">
    {#if depot.category === 'soundtrack'}
      <button
        type="button"
        onclick={() => ondownloadSoundtrack?.()}
        disabled={isActionRunning}
        class="px-2.5 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover text-neutral-200 hover:text-white border border-surface-border font-bold text-2xs transition-colors flex items-center space-x-1 cursor-pointer"
      >
        <span>🎵 Save to Music</span>
      </button>
    {:else if !depot.installed}
      <button
        type="button"
        onclick={() => ondownload?.()}
        disabled={isActionRunning}
        class="px-2.5 py-1.5 rounded-lg bg-white hover:bg-neutral-200 text-neutral-950 font-bold text-2xs transition-colors flex items-center space-x-1 cursor-pointer"
      >
        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        <span>Install</span>
      </button>
    {:else}
      <div class="flex items-center space-x-1.5">
        <span class="text-emerald-400 font-bold text-2xs px-2 py-1 bg-emerald-950/40 rounded-lg border border-emerald-800/40">
          Installed
        </span>
        {#if canUninstall}
          <button
            type="button"
            onclick={() => onuninstall?.()}
            disabled={isActionRunning}
            title="Remove depot files to free space"
            class="px-2 py-1 rounded-lg bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 border border-rose-800/50 font-bold text-3xs transition-colors cursor-pointer"
          >
            Uninstall
          </button>
        {/if}
      </div>
    {/if}
  </div>
</div>
