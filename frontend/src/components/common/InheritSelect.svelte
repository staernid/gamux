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
  $: isInherited = hasInheritance && (!value || value === '' || value === globalValue);
  $: isOverridden = hasInheritance && !isInherited;
  $: selectedValue = hasInheritance ? (isOverridden ? value : globalValue) : value;
  $: resolvedGlobalLabel = globalLabel || options.find(o => o.value === globalValue)?.label || globalValue || '';
  $: effectiveDescription = description || tooltip;

  function handleChange(e) {
    if (disabled) return;
    const selectedVal = e.target.value;
    if (hasInheritance) {
      value = (selectedVal === globalValue) ? '' : selectedVal;
    } else {
      value = selectedVal;
    }
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
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-[#18181b]'}
      {isOverridden ? 'bg-[#18181b] border border-neutral-600' : 'bg-[#101013] border border-[#27272a]'}"
  >
    <!-- Top: Header with Label and Override Indicator -->
    <div class="space-y-1 min-w-0">
      <div class="flex items-center justify-between gap-1.5 min-w-0">
        <div class="flex items-center space-x-2 min-w-0 flex-1">
          <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {isOverridden ? 'bg-white' : 'bg-transparent'}" aria-hidden="true"></div>
          <label for={id} class="text-xs font-semibold cursor-pointer truncate {isOverridden ? 'text-white font-bold' : 'text-neutral-200'}">
            {label}
          </label>
        </div>

        {#if isOverridden}
          <span class="text-3xs font-mono text-neutral-300 font-medium flex-shrink-0">
            (Override)
          </span>
        {/if}
      </div>

      {#if effectiveDescription}
        <p class="text-xs text-neutral-400 leading-relaxed">
          {effectiveDescription}
        </p>
      {/if}
    </div>

    <!-- Bottom: Full-Width Select + Reset -->
    <div class="flex items-center space-x-1.5 w-full pt-1">
      <div class="relative flex-1 min-w-0">
        <select
          {id}
          value={selectedValue}
          {disabled}
          on:change={handleChange}
          class="w-full bg-[#141417] border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors appearance-none cursor-pointer pr-8
            {isOverridden
              ? 'border-neutral-500 text-white font-medium focus:border-white'
              : 'border-[#27272a] text-neutral-200 focus:border-neutral-500'}"
        >
          {#each options as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <div class="absolute inset-y-0 right-0 flex items-center px-2.5 pointer-events-none text-neutral-400">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#202024] hover:bg-[#27272a] border border-[#3f3f46] text-neutral-300 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title={`Reset to global default (${resolvedGlobalLabel})`}
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
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-[#18181b]'}
      {isOverridden ? 'bg-[#18181b] border border-neutral-600' : 'bg-[#101013] border border-[#27272a]'}"
  >
    <!-- Left Side: Label & Description with Override Dot -->
    <div class="flex items-start sm:items-center space-x-2.5 min-w-0 flex-1">
      <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 mt-1 sm:mt-0 {isOverridden ? 'bg-white' : 'bg-transparent'}" aria-hidden="true"></div>

      <div class="space-y-0.5 min-w-0 flex-1">
        <div class="flex items-center space-x-2 flex-wrap">
          <label for={id} class="text-xs font-semibold cursor-pointer {isOverridden ? 'text-white font-bold' : 'text-neutral-200'}">
            {label}
          </label>
          {#if isOverridden}
            <span class="text-3xs font-mono text-neutral-300 font-medium">
              (Override)
            </span>
          {/if}
        </div>

        {#if effectiveDescription}
          <p class="text-xs text-neutral-400 leading-relaxed">
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
          value={selectedValue}
          {disabled}
          on:change={handleChange}
          class="w-full bg-[#141417] border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors appearance-none cursor-pointer pr-8
            {isOverridden
              ? 'border-neutral-500 text-white font-medium focus:border-white'
              : 'border-[#27272a] text-neutral-200 focus:border-neutral-500'}"
        >
          {#each options as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <div class="absolute inset-y-0 right-0 flex items-center px-2.5 pointer-events-none text-neutral-400">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#202024] hover:bg-[#27272a] border border-[#3f3f46] text-neutral-300 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title={`Reset to global default (${resolvedGlobalLabel})`}
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


