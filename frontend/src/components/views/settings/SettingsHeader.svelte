<script>
  /**
   * @typedef {Object} Props
   * @property {boolean} [isDirty]
   * @property {boolean} [isSaving]
   * @property {boolean} [saveSuccess]
   * @property {string|null} [settingsLoadError]
   * @property {() => void} [onretry]
   */

  /** @type {Props} */
  let {
    isDirty = false,
    isSaving = false,
    saveSuccess = false,
    settingsLoadError = null,
    onretry = undefined
  } = $props();
</script>

<div class="space-y-4">
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-surface-border">
    <div class="flex items-center space-x-3">
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2.5">
        System Settings
      </h2>
      {#if isDirty && !isSaving}
        <span class="text-xs font-mono text-amber-400 bg-amber-500/10 border border-amber-500/30 px-2.5 py-0.5 rounded-full flex items-center space-x-1.5 animate-fade-in">
          <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
          <span>Unsaved Changes (Ctrl+S)</span>
        </span>
      {:else if isSaving}
        <span class="text-xs font-mono text-neutral-300 bg-surface-header border border-surface-border px-2.5 py-0.5 rounded-full flex items-center space-x-1.5 animate-pulse">
          <svg class="animate-spin w-3 h-3 text-neutral-400" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Saving...</span>
        </span>
      {:else if saveSuccess}
        <span class="text-xs font-mono font-semibold text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 px-2.5 py-0.5 rounded-full flex items-center space-x-1.5 animate-fade-in">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <span>Saved</span>
        </span>
      {/if}
    </div>
  </div>

  {#if settingsLoadError}
    <div class="p-3 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between text-xs">
      <span class="text-rose-400 font-medium">{settingsLoadError}</span>
      <button
        type="button"
        onclick={onretry}
        class="px-2.5 py-1 rounded-lg bg-surface-button hover:bg-surface-buttonHover border border-surface-border text-xs text-white cursor-pointer"
      >
        Retry
      </button>
    </div>
  {/if}
</div>
