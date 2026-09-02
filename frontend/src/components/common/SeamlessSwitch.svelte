<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tooltip = '';
  export let value = false; // boolean or null (for inherit mode)
  export let inheritedValue = undefined; // boolean global default if in inherit mode
  export let disabled = false;

  const dispatch = createEventDispatcher();

  $: hasInheritance = inheritedValue !== undefined && inheritedValue !== null;
  $: isInherited = hasInheritance && (value === null || value === undefined);
  $: effectiveValue = isInherited ? Boolean(inheritedValue) : Boolean(value);
  $: isOverridden = hasInheritance && !isInherited;
  $: effectiveDescription = description || tooltip;

  function toggleSwitch() {
    if (disabled) return;
    if (hasInheritance) {
      if (isInherited) {
        value = !inheritedValue;
      } else {
        value = !value;
      }
    } else {
      value = !value;
    }
    dispatch('change', value);
  }

  function revertToInherit(e) {
    e.stopPropagation();
    if (disabled || !hasInheritance) return;
    value = null;
    dispatch('change', null);
  }

  function handleKeyDown(e) {
    if (disabled) return;
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      toggleSwitch();
    } else if ((e.key === 'Backspace' || e.key === 'Delete') && isOverridden) {
      e.preventDefault();
      revertToInherit(e);
    }
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  class="flex items-start sm:items-center justify-between gap-3 p-2.5 sm:p-3 rounded-xl transition-all select-none
    {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-[#18181b]'}
    {isOverridden ? 'bg-[#18181b] border border-neutral-600' : 'bg-[#101013] border border-[#27272a]'}"
  role="switch"
  tabindex="0"
  aria-checked={effectiveValue}
  aria-label={label || 'Setting toggle'}
  on:click={toggleSwitch}
  on:keydown={handleKeyDown}
>
  <!-- Left Side: Label & Description with Override Dot -->
  <div class="flex items-start sm:items-center space-x-2.5 min-w-0 flex-1">
    <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 mt-1 sm:mt-0 {isOverridden ? 'bg-white' : 'bg-transparent'}" aria-hidden="true"></div>

    <div class="space-y-0.5 min-w-0 flex-1">
      <div class="flex items-center space-x-2 flex-wrap">
        <span class="text-xs font-semibold {isOverridden ? 'text-white font-bold' : 'text-neutral-200'}">
          {label}
        </span>
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

  <!-- Right Side: Reset Button + Switch -->
  <div class="flex items-center space-x-2 flex-shrink-0">
    {#if isOverridden}
      <button
        type="button"
        on:click={revertToInherit}
        class="p-1.5 rounded-lg bg-[#202024] hover:bg-[#27272a] border border-[#3f3f46] text-neutral-300 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer"
        title={`Reset to default (${inheritedValue ? 'On' : 'Off'})`}
        aria-label="Reset to global default"
      >
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    {/if}

    <!-- The Switch Track & Thumb -->
    <div
      class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full transition-colors duration-200 ease-in-out border
        {isInherited
          ? (effectiveValue ? 'bg-neutral-700 border-neutral-600' : 'bg-neutral-900/80 border-neutral-800')
          : (effectiveValue ? 'bg-white border-white' : 'bg-neutral-900 border-neutral-700')}"
      aria-hidden="true"
    >
      <span
        class="pointer-events-none inline-block h-3.5 w-3.5 transform rounded-full transition duration-200 ease-in-out mt-[2px] ml-[2px]
          {effectiveValue ? 'translate-x-4' : 'translate-x-0'}
          {isInherited
            ? (effectiveValue ? 'bg-neutral-300' : 'bg-neutral-500')
            : (effectiveValue ? 'bg-neutral-950 shadow-sm' : 'bg-neutral-400')}"
      ></span>
    </div>
  </div>
</div>


