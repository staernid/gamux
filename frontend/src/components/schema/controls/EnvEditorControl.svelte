<script>
  import { createEventDispatcher } from 'svelte';

  export let property = {};
  export let value = {};
  export let disabled = false;
  export let hideHeader = false;
  export let onChange = null;
  export let onchange = null;

  const dispatch = createEventDispatcher();

  let newKey = '';
  let newValue = '';

  $: envMap = (value && typeof value === 'object' && !Array.isArray(value)) ? value : {};
  $: envKeys = Object.keys(envMap).sort();

  function notifyChange(nextVal) {
    dispatch('change', nextVal);
    if (typeof onChange === 'function') onChange(nextVal);
    if (typeof onchange === 'function') onchange(nextVal);
  }

  function addPair() {
    if (disabled) return;
    const k = newKey.trim();
    if (!k) return;
    const next = { ...envMap, [k]: newValue.trim() };
    newKey = '';
    newValue = '';
    value = next;
    notifyChange(next);
  }

  function removePair(k) {
    if (disabled) return;
    const next = { ...envMap };
    delete next[k];
    value = next;
    notifyChange(next);
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter') {
      e.preventDefault();
      addPair();
    }
  }
</script>

{#if hideHeader}
  <div class="w-full space-y-2" data-testid={`control-env-${property.key}`}>
    <!-- Key-Value Pair Listing -->
    {#if envKeys.length === 0}
      <div class="p-2 rounded bg-[#101013] border border-surface-border text-center">
        <p class="text-3xs text-neutral-500 font-mono">No environment variables defined</p>
      </div>
    {:else}
      <div class="space-y-1">
        {#each envKeys as k}
          <div class="flex items-center justify-between gap-2 p-1.5 rounded bg-[#101013] border border-surface-border">
            <div class="flex items-center space-x-2 min-w-0 flex-1 font-mono text-xs">
              <span class="font-bold text-neutral-200 truncate">{k}</span>
              <span class="text-neutral-500">=</span>
              <span class="text-neutral-300 truncate">{envMap[k]}</span>
            </div>
            <button
              type="button"
              {disabled}
              on:click={() => removePair(k)}
              class="p-0.5 text-neutral-500 hover:text-rose-400 transition-colors flex-shrink-0 cursor-pointer"
              aria-label={`Remove variable ${k}`}
            >
              <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Add New Variable -->
    <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-1.5">
      <input
        type="text"
        placeholder="VARIABLE_NAME"
        bind:value={newKey}
        {disabled}
        on:keydown={handleKeyDown}
        class="flex-1 bg-[#121215] text-neutral-200 text-xs font-mono rounded-lg px-2.5 py-1.5 border border-neutral-700
          focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors"
        aria-label="Variable Name"
      />
      <input
        type="text"
        placeholder="value"
        bind:value={newValue}
        {disabled}
        on:keydown={handleKeyDown}
        class="flex-1 bg-[#121215] text-neutral-200 text-xs font-mono rounded-lg px-2.5 py-1.5 border border-neutral-700
          focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors"
        aria-label="Variable Value"
      />
      <button
        type="button"
        {disabled}
        on:click={addPair}
        class="px-2.5 py-1.5 text-xs font-semibold bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 rounded-lg transition-colors whitespace-nowrap cursor-pointer"
        aria-label="Add Variable"
      >
        Add
      </button>
    </div>
  </div>
{:else}
  <!-- Standalone Mode -->
  <div
    class="p-3 rounded-lg bg-surface-card/30 border border-surface-border/50 transition-all space-y-3
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-surface-card/60 hover:border-surface-borderHighlight'}"
    data-testid={`control-env-${property.key}`}
  >
    <div class="space-y-0.5">
      <span class="block text-xs font-semibold text-neutral-200">
        {property.label || property.key}
      </span>
      {#if property.description}
        <p class="text-xs text-neutral-400 leading-relaxed">
          {property.description}
        </p>
      {/if}
    </div>

    <!-- Key-Value Pair Listing -->
    {#if envKeys.length === 0}
      <div class="p-2 rounded bg-[#101013] border border-surface-border text-center">
        <p class="text-xs text-neutral-500 font-mono">No environment variables defined</p>
      </div>
    {:else}
      <div class="space-y-1.5">
        {#each envKeys as k}
          <div class="flex items-center justify-between gap-2 p-2 rounded bg-[#101013] border border-surface-border">
            <div class="flex items-center space-x-2 min-w-0 flex-1 font-mono text-xs">
              <span class="font-bold text-neutral-200 truncate">{k}</span>
              <span class="text-neutral-500">=</span>
              <span class="text-neutral-300 truncate">{envMap[k]}</span>
            </div>
            <button
              type="button"
              {disabled}
              on:click={() => removePair(k)}
              class="p-1 text-neutral-500 hover:text-rose-400 transition-colors flex-shrink-0"
              aria-label={`Remove variable ${k}`}
            >
              <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Add New Variable -->
    <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-1">
      <input
        type="text"
        placeholder="VARIABLE_NAME"
        bind:value={newKey}
        {disabled}
        on:keydown={handleKeyDown}
        class="flex-1 bg-[#121215] text-neutral-200 text-xs font-mono rounded-lg px-3 py-2 border border-neutral-700
          focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors"
        aria-label="Variable Name"
      />
      <input
        type="text"
        placeholder="value"
        bind:value={newValue}
        {disabled}
        on:keydown={handleKeyDown}
        class="flex-1 bg-[#121215] text-neutral-200 text-xs font-mono rounded-lg px-3 py-2 border border-neutral-700
          focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors"
        aria-label="Variable Value"
      />
      <button
        type="button"
        {disabled}
        on:click={addPair}
        class="px-3 py-2 text-xs font-semibold bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 rounded-lg transition-colors whitespace-nowrap"
        aria-label="Add Variable"
      >
        Add
      </button>
    </div>
  </div>
{/if}
