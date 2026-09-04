<script>
  import { createEventDispatcher } from 'svelte';

  export let globalEnv = {}; // { KEY: VALUE }
  export let customEnv = {}; // { KEY: VALUE }
  export let disabled = false;

  const dispatch = createEventDispatcher();

  let newKey = '';
  let newValue = '';

  $: globalKeys = Object.keys(globalEnv || {}).sort();
  $: customKeys = Object.keys(customEnv || {}).sort();

  function addVariable() {
    if (disabled) return;
    const k = newKey.trim().toUpperCase();
    if (!k) return;
    const updated = { ...(customEnv || {}), [k]: newValue.trim() };
    customEnv = updated;
    newKey = '';
    newValue = '';
    dispatch('change', updated);
  }

  function removeVariable(k) {
    if (disabled) return;
    const updated = { ...(customEnv || {}) };
    delete updated[k];
    customEnv = updated;
    dispatch('change', updated);
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') {
      e.preventDefault();
      addVariable();
    }
  }
</script>

<div class="space-y-4">
  <!-- Custom Game Variables & Overrides -->
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <span class="text-xs font-bold text-slate-200 uppercase tracking-wider">
        Game-Specific Variables & Overrides
      </span>
      <span class="text-3xs font-mono text-neutral-400">
        {customKeys.length} active
      </span>
    </div>

    {#if customKeys.length === 0}
      <div class="p-3 bg-[#101013] border border-surface-border rounded-xl text-center">
        <p class="text-xs text-neutral-500 font-mono">No custom variables configured for this game.</p>
      </div>
    {:else}
      <div class="space-y-1.5">
        {#each customKeys as k}
          {@const isOverridingGlobal = Boolean(globalEnv && globalEnv[k] !== undefined)}
          <div class="flex items-center justify-between gap-3 p-2.5 bg-[#101013] border border-surface-border rounded-xl">
            <div class="flex items-center space-x-2 min-w-0 flex-1">
              <span class="w-1.5 h-1.5 rounded-full bg-white flex-shrink-0" title="Active variable"></span>
              <span class="font-mono text-xs font-bold text-white truncate">{k}</span>
              <span class="text-neutral-500 text-xs font-mono">=</span>
              <span class="font-mono text-xs text-neutral-200 truncate">{customEnv[k]}</span>
              {#if isOverridingGlobal}
                <span class="px-1.5 py-0.5 rounded text-3xs font-mono bg-neutral-800 text-neutral-300 border border-neutral-700 flex-shrink-0">
                  Overrides Global ({globalEnv[k]})
                </span>
              {/if}
            </div>

            <button
              type="button"
              on:click={() => removeVariable(k)}
              {disabled}
              class="p-1 text-neutral-400 hover:text-rose-400 transition-colors cursor-pointer flex-shrink-0"
              title="Remove variable"
              aria-label={`Remove variable ${k}`}
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Add Variable Row -->
    {#if !disabled}
      <div class="flex items-center gap-2 pt-1">
        <input
          type="text"
          placeholder="VARIABLE_NAME"
          bind:value={newKey}
          on:keydown={handleKeydown}
          class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-xl px-3 py-1.5 text-xs text-white font-mono uppercase outline-none transition-colors"
        />
        <input
          type="text"
          placeholder="value"
          bind:value={newValue}
          on:keydown={handleKeydown}
          class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none transition-colors"
        />
        <button
          type="button"
          on:click={addVariable}
          class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-xl text-xs font-semibold transition-colors flex-shrink-0 cursor-pointer flex items-center space-x-1"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <span>Add</span>
        </button>
      </div>
    {/if}
  </div>

  <!-- Inherited Global Variables (Read-Only) -->
  {#if globalKeys.length > 0}
    <div class="pt-3 border-t border-surface-border space-y-2">
      <span class="text-xs font-semibold text-neutral-400 uppercase tracking-wider block">
        Inherited System Variables (Global)
      </span>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
        {#each globalKeys as gk}
          {@const isOverridden = Boolean(customEnv && customEnv[gk] !== undefined)}
          <div class="flex items-center justify-between p-2 rounded-lg bg-[#101013] border border-surface-border {isOverridden ? 'opacity-40 line-through' : ''}">
            <div class="flex items-center space-x-1.5 min-w-0">
              <span class="font-mono text-2xs text-neutral-400 truncate">{gk}</span>
              <span class="text-neutral-600 text-2xs font-mono">=</span>
              <span class="font-mono text-2xs text-neutral-200 truncate">{globalEnv[gk]}</span>
            </div>
            {#if isOverridden}
              <span class="text-3xs font-mono text-neutral-500 uppercase">Overridden</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>
