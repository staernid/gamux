<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tooltip = '';
  export let value = ''; // current local value ('' means inherit if hasInheritance)
  export let globalValue = undefined; // global default value (if in inherit mode)
  export let globalLabel = ''; // optional human-readable label for global value
  export let options = []; // Array<{ value: string, label: string }>
  export let disabled = false;
  export let stacked = false;
  export let id = `select-${Math.random().toString(36).slice(2, 9)}`;

  const dispatch = createEventDispatcher();

  $: hasInheritance = globalValue !== undefined && globalValue !== null && globalValue !== '';
  $: isInherited = hasInheritance && (!value || value === '');
  $: isOverridden = hasInheritance && !isInherited;
  $: resolvedGlobalLabel = globalLabel || options.find(o => o.value === globalValue)?.label || globalValue || 'None';
  $: effectiveDescription = description || tooltip;

  function handleChange(e) {
    if (disabled) return;
    value = e.target.value;
    dispatch('change', value);
  }

  function revertToInherit(e) {
    e.stopPropagation();
    if (disabled || !hasInheritance) return;
    value = '';
    dispatch('change', '');
  }
</script>

{#if stacked}
  <div
    class="flex flex-col justify-between gap-2.5 p-3 rounded-xl transition-all h-full
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-[#13131c]'}
      {isOverridden ? 'bg-[#121622]/60 border border-sky-500/30' : 'bg-[#0d0d14] border border-[#1e1e2c]'}"
  >
    <!-- Top: Header with Label and Override Indicator -->
    <div class="space-y-1 min-w-0">
      <div class="flex items-center justify-between gap-1.5 min-w-0">
        <div class="flex items-center space-x-2 min-w-0 flex-1">
          <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {isOverridden ? 'bg-sky-400' : 'bg-transparent'}" aria-hidden="true"></div>
          <label for={id} class="text-xs font-semibold cursor-pointer truncate {isOverridden ? 'text-white font-bold' : 'text-slate-200'}">
            {label}
          </label>
        </div>

        {#if isOverridden}
          <span class="text-[10px] font-mono text-sky-400 font-medium flex-shrink-0">
            (Override)
          </span>
        {/if}
      </div>

      {#if effectiveDescription}
        <p class="text-[11px] text-slate-400 leading-snug">
          {effectiveDescription}
        </p>
      {/if}
    </div>

    <!-- Bottom: Full-Width Select + Reset -->
    <div class="flex items-center space-x-1.5 w-full pt-1">
      <div class="relative flex-1 min-w-0">
        <select
          {id}
          {value}
          {disabled}
          on:change={handleChange}
          class="w-full bg-[#14141e] border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors appearance-none cursor-pointer pr-8
            {isOverridden
              ? 'border-sky-500/60 text-white font-medium focus:border-sky-400'
              : 'border-[#262638] text-slate-300 focus:border-slate-500'}"
        >
          {#if hasInheritance}
            <option value="">Default ({resolvedGlobalLabel})</option>
          {/if}
          {#each options as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <div class="absolute inset-y-0 right-0 flex items-center px-2.5 pointer-events-none text-slate-400">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#181824] hover:bg-[#242438] border border-[#2c2c3e] text-slate-400 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title={`Reset to default (${resolvedGlobalLabel})`}
          aria-label="Reset to global default"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      {/if}
    </div>
  </div>
{:else}
  <div
    class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-2.5 sm:p-3 rounded-xl transition-all
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-[#13131c]'}
      {isOverridden ? 'bg-[#121622]/60 border border-sky-500/30' : 'bg-[#0d0d14] border border-[#1e1e2c]'}"
  >
    <!-- Left Side: Label & Description with Override Dot -->
    <div class="flex items-center space-x-2.5 min-w-0 flex-1">
      <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {isOverridden ? 'bg-sky-400' : 'bg-transparent'}" aria-hidden="true"></div>

      <div class="space-y-0.5 min-w-0 flex-1">
        <div class="flex items-center space-x-2 flex-wrap">
          <label for={id} class="text-xs font-semibold cursor-pointer {isOverridden ? 'text-white font-bold' : 'text-slate-200'}">
            {label}
          </label>
          {#if isOverridden}
            <span class="text-[10px] font-mono text-sky-400 font-medium">
              (Override)
            </span>
          {/if}
        </div>

        {#if effectiveDescription}
          <p class="text-[11px] text-slate-400 leading-snug truncate">
            {effectiveDescription}
          </p>
        {/if}
      </div>
    </div>

    <!-- Right Side: Select Input + Reset Button -->
    <div class="flex items-center space-x-1.5 flex-shrink-0 w-full sm:w-auto">
      <div class="relative flex-1 sm:w-60">
        <select
          {id}
          {value}
          {disabled}
          on:change={handleChange}
          class="w-full bg-[#14141e] border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors appearance-none cursor-pointer pr-8
            {isOverridden
              ? 'border-sky-500/60 text-white font-medium focus:border-sky-400'
              : 'border-[#262638] text-slate-300 focus:border-slate-500'}"
        >
          {#if hasInheritance}
            <option value="">Default ({resolvedGlobalLabel})</option>
          {/if}
          {#each options as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <div class="absolute inset-y-0 right-0 flex items-center px-2.5 pointer-events-none text-slate-400">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#181824] hover:bg-[#242438] border border-[#2c2c3e] text-slate-400 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title={`Reset to default (${resolvedGlobalLabel})`}
          aria-label="Reset to global default"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      {/if}
    </div>
  </div>
{/if}


