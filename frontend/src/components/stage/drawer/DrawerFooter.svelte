<script>
  export let isPatched = false;
  export let onRollback = () => {};
  export let onResetDefaults = () => {};
  export let onCancel = () => {};
  export let onSave = () => {};
</script>

<div class="px-5 py-3 border-t border-[#20202c] bg-[#15151e] flex items-center justify-between">
  <div class="flex items-center space-x-3">
    {#if isPatched}
      <button
        type="button"
        on:click={onRollback}
        class="text-rose-400 hover:text-rose-300 font-bold text-xs"
      >
        Rollback Setup
      </button>
    {/if}
    <button
      type="button"
      on:click={onResetDefaults}
      class="text-amber-400 hover:text-amber-300 font-bold text-xs"
      title="Reset custom overrides and inherit global defaults"
    >
      Reset to Defaults
    </button>
  </div>

  <div class="flex items-center space-x-2">
    <button
      type="button"
      on:click={onCancel}
      class="text-slate-400 hover:text-white px-3 py-1.5 font-bold text-xs"
    >
      Cancel
    </button>
    <button
      type="button"
      on:click={onSave}
      class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-xs px-6 py-2 rounded-xl transition-colors shadow-sm"
    >
      Save
    </button>
  </div>
</div>
