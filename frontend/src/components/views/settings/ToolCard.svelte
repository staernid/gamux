<script>
  import { openExternalUrl } from '../../../utils/navigation.js';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').ToolStatus} tool
   * @property {boolean} [isUpdating]
   * @property {(data: { key: string, name: string }) => void} [onupdate]
   */

  /** @type {Props} */
  let {
    tool,
    isUpdating = false,
    onupdate = undefined
  } = $props();

  let isInstalled = $derived(Boolean(tool?.isInstalled));
</script>

<div class="bg-surface-header border border-surface-border p-3 rounded-xl flex flex-col justify-between gap-2.5">
  <div class="space-y-1 min-w-0">
    <div class="flex items-center justify-between gap-1.5">
      <span class="font-semibold text-xs text-white truncate">{tool.name}</span>
      {#if !isInstalled}
        <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold bg-amber-500/15 text-amber-300 border border-amber-500/30 shrink-0">
          Missing
        </span>
      {:else if tool.hasUpdate}
        <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 shrink-0">
          Update
        </span>
      {/if}
    </div>

    <div class="text-2xs font-mono text-neutral-400 truncate">
      {#if !isInstalled}
        <span class="text-amber-400/90 font-medium">Not Installed</span>
        {#if tool.latestTag}
          <span class="text-neutral-500">·</span>
          <span class="text-emerald-400">{tool.latestTag}</span>
        {/if}
      {:else}
        <span>{tool.installedTag || 'Installed'}</span>
        {#if tool.latestTag}
          <span class="text-neutral-500">·</span>
          <span class="text-emerald-400">{tool.latestTag}</span>
        {/if}
      {/if}
    </div>
  </div>

  <div class="flex items-center space-x-1.5 pt-1 border-t border-surface-border">
    {#if tool.releaseUrl}
      <button
        type="button"
        onclick={() => openExternalUrl(tool.releaseUrl, { title: tool.name || 'GitHub Release' })}
        class="text-neutral-400 hover:text-white text-xs px-2 py-1 rounded-lg border border-surface-border bg-surface-input transition-colors cursor-pointer"
        title="Open GitHub Release"
        aria-label="Open GitHub Release"
      >
        ↗
      </button>
    {/if}
    <button
      type="button"
      onclick={() => onupdate?.({ key: tool.key, name: tool.name })}
      disabled={isUpdating}
      class="flex-1 py-1.5 rounded-lg font-semibold text-xs transition-colors flex items-center justify-center space-x-1 cursor-pointer disabled:opacity-50 {isInstalled && !tool.hasUpdate ? 'bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-400 hover:text-white' : 'bg-neutral-800 hover:bg-neutral-700 text-white border border-neutral-600 shadow-sm'}"
    >
      {#if isUpdating}
        <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
        </svg>
        <span>{isInstalled ? (tool.hasUpdate ? 'Updating...' : 'Reinstalling...') : 'Installing...'}</span>
      {:else}
        <span>{isInstalled ? (tool.hasUpdate ? 'Update' : 'Reinstall') : 'Install'}</span>
      {/if}
    </button>
  </div>
</div>
