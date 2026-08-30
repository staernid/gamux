<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tooltip = '';
  export let value = ''; // current local value ('' means inherit if hasInheritance)
  export let placeholder = ''; // standard placeholder
  export let globalPlaceholder = ''; // global default watermark (if in inherit mode)
  export let disabled = false;
  export let mono = false;
  export let type = 'text'; // 'text' | 'password'
  export let status = ''; // optional status badge (e.g. 'SET')
  export let stacked = false; // stacked vertical card layout
  export let id = `input-${Math.random().toString(36).slice(2, 9)}`;

  const dispatch = createEventDispatcher();

  $: hasInheritance = Boolean(globalPlaceholder);
  $: isInherited = hasInheritance && (!value || value.trim() === '');
  $: isOverridden = hasInheritance && !isInherited;
  $: effectiveDescription = description || tooltip;

  function handleInput(e) {
    if (disabled) return;
    value = e.target.value;
    dispatch('input', value);
  }

  function revertToInherit(e) {
    e.stopPropagation();
    if (disabled || !hasInheritance) return;
    value = '';
    dispatch('input', '');
    dispatch('change', '');
  }
</script>

{#if stacked}
  <div
    class="flex flex-col justify-between gap-2.5 p-3 rounded-xl transition-all h-full
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-[#13131c]'}
      {isOverridden ? 'bg-[#121622]/60 border border-sky-500/30' : 'bg-[#0d0d14] border border-[#1e1e2c]'}"
  >
    <!-- Top: Header with Label, Status, and Override Indicator -->
    <div class="space-y-1 min-w-0">
      <div class="flex items-center justify-between gap-1.5 min-w-0">
        <div class="flex items-center space-x-2 min-w-0 flex-1">
          <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {isOverridden ? 'bg-sky-400' : 'bg-transparent'}" aria-hidden="true"></div>
          <label for={id} class="text-xs font-semibold cursor-pointer truncate {isOverridden ? 'text-white font-bold' : 'text-slate-200'}">
            {label}
          </label>
        </div>

        <div class="flex items-center space-x-1.5 flex-shrink-0">
          {#if isOverridden}
            <span class="text-[10px] font-mono text-sky-400 font-medium">
              (Override)
            </span>
          {/if}
          {#if status}
            <span class="px-2 py-0.5 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 rounded-lg text-[10px] font-mono font-bold">
              {status}
            </span>
          {/if}
        </div>
      </div>

      {#if effectiveDescription}
        <p class="text-[11px] text-slate-400 leading-snug">
          {effectiveDescription}
        </p>
      {/if}
    </div>

    <!-- Bottom: Full-Width Input + Optional Slot + Reset -->
    <div class="flex items-center space-x-1.5 w-full pt-1">
      <div class="relative flex-1 min-w-0">
        <input
          {id}
          {type}
          {value}
          {disabled}
          placeholder={globalPlaceholder || placeholder || ''}
          on:input={handleInput}
          class="w-full bg-[#14141e] border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors {mono ? 'font-mono' : ''}
            {isOverridden
              ? 'border-sky-500/60 text-white font-medium focus:border-sky-400'
              : 'border-[#262638] text-slate-300 placeholder-slate-500 focus:border-slate-500'}"
        />
      </div>

      <slot name="action" />

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#181824] hover:bg-[#242438] border border-[#2c2c3e] text-slate-400 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title="Reset to global default"
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

    <!-- Right Side: Input + Optional Status/Slot + Reset Button -->
    <div class="flex items-center space-x-2 flex-shrink-0 w-full sm:w-auto">
      <div class="relative flex-1 sm:w-60">
        <input
          {id}
          {type}
          {value}
          {disabled}
          placeholder={globalPlaceholder || placeholder || ''}
          on:input={handleInput}
          class="w-full bg-[#14141e] border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors {mono ? 'font-mono' : ''}
            {isOverridden
              ? 'border-sky-500/60 text-white font-medium focus:border-sky-400'
              : 'border-[#262638] text-slate-300 placeholder-slate-500 focus:border-slate-500'}"
        />
      </div>

      {#if status}
        <span class="px-2 py-1 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 rounded-lg text-[10px] font-mono font-bold flex-shrink-0">
          {status}
        </span>
      {/if}

      <slot name="action" />

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#181824] hover:bg-[#242438] border border-[#2c2c3e] text-slate-400 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title="Reset to global default"
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


