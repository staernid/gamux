<script>
  import { createEventDispatcher } from 'svelte';

  export let title = '';
  export let overrideCount = 0;
  export let isSaving = false;
  export let saveSuccess = false;
  export let onsave = undefined;
  export let onreset = undefined;

  const dispatch = createEventDispatcher();

  function handleSave() {
    dispatch('save');
    if (typeof onsave === 'function') onsave();
  }

  function handleReset() {
    dispatch('reset');
    if (typeof onreset === 'function') onreset();
  }
</script>

<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 bg-surface-card border border-surface-border rounded-xl p-2.5 shadow-xl">
  <div class="flex items-center space-x-2 px-2">
    <span class="text-xs font-bold text-neutral-200">{title}</span>
  </div>

  <!-- Status Badge & Save Actions -->
  <div class="flex items-center space-x-2 flex-shrink-0">
    {#if overrideCount > 0}
      <span class="px-2 py-0.5 rounded-lg text-xs font-mono font-bold bg-neutral-800 text-neutral-200 border border-neutral-700 flex items-center space-x-1.5">
        <span class="w-1.5 h-1.5 rounded-full bg-neutral-300"></span>
        <span>{overrideCount} Override{overrideCount === 1 ? '' : 's'}</span>
      </span>
    {:else}
      <span class="px-2 py-0.5 rounded-lg text-xs font-mono text-neutral-400 bg-surface-header border border-surface-border">
        Inheriting Defaults
      </span>
    {/if}

    {#if saveSuccess}
      <span class="text-xs font-mono font-bold text-emerald-400 flex items-center space-x-1 animate-fade-in">
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
        <span>Saved</span>
      </span>
    {/if}

    <button
      type="button"
      on:click={handleReset}
      class="p-1.5 bg-surface-cardHover hover:bg-surface-elevated border border-surface-borderHighlight text-neutral-400 hover:text-white rounded-lg text-xs transition-colors cursor-pointer"
      title="Reset all settings to system defaults"
      aria-label="Reset all settings to system defaults"
    >
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
    </button>

    <button
      type="button"
      on:click={handleSave}
      disabled={isSaving}
      class="px-3.5 py-1.5 rounded-lg bg-white hover:bg-neutral-200 active:bg-neutral-300 text-neutral-950 font-bold text-xs shadow-md transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
    >
      {#if isSaving}
        <svg class="animate-spin w-3.5 h-3.5 text-neutral-950" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
        </svg>
        <span>Saving...</span>
      {:else}
        <span>Save</span>
      {/if}
    </button>
  </div>
</div>
