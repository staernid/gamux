<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tooltip = '';
  export let options = []; // Array of { value, label } or simple strings
  export let value = null;
  export let disabled = false;

  const dispatch = createEventDispatcher();

  $: normalizedOptions = options.map(opt => {
    if (typeof opt === 'object' && opt !== null) {
      return { value: opt.value, label: opt.label ?? String(opt.value) };
    }
    return { value: opt, label: String(opt) };
  });

  $: currentIndex = normalizedOptions.findIndex(o => o.value === value);
  $: currentLabel = currentIndex >= 0 ? normalizedOptions[currentIndex].label : String(value ?? '');
  $: effectiveTooltip = tooltip || description;

  function step(dir) {
    if (disabled || normalizedOptions.length === 0) return;
    let nextIdx = currentIndex + dir;
    if (nextIdx < 0) nextIdx = normalizedOptions.length - 1;
    if (nextIdx >= normalizedOptions.length) nextIdx = 0;

    const nextVal = normalizedOptions[nextIdx].value;
    value = nextVal;
    dispatch('change', nextVal);
  }

  function handleKeydown(e) {
    if (disabled) return;
    if (e.key === 'ArrowLeft' || e.key === 'h' || e.key === 'H') {
      e.preventDefault();
      step(-1);
    } else if (e.key === 'ArrowRight' || e.key === 'l' || e.key === 'L' || e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      step(1);
    }
  }
</script>

<div
  class="flex items-center justify-between gap-3 p-2.5 sm:p-3 bg-[#0d0d12] hover:bg-[#13131a] focus-within:bg-[#161622] border border-[#222230] focus-within:border-sky-500/70 rounded-xl transition-all select-none group"
  role="group"
  aria-label={label}
>
  <!-- Label & Description -->
  <div class="space-y-0.5 flex-1 min-w-[120px]">
    <div class="flex items-center space-x-1.5 flex-wrap">
      <span class="block font-bold text-slate-200 text-xs group-focus-within:text-white transition-colors truncate">{label}</span>
      {#if effectiveTooltip}
        <span
          class="text-slate-500 hover:text-slate-300 transition-colors cursor-help p-0.5 flex-shrink-0"
          title={effectiveTooltip}
          aria-label={effectiveTooltip}
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </span>
      {/if}
    </div>
    {#if description && !tooltip}
      <p class="text-[11px] text-slate-400 leading-tight truncate">{description}</p>
    {/if}
  </div>

  <!-- Unified Stepper Control Capsule (Single Focus Stop for Controller & Keyboard) -->
  <div
    class="flex items-center space-x-1 bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 focus:ring-2 focus:ring-sky-500/30 rounded-lg p-0.5 flex-shrink-0 outline-none cursor-pointer transition-all hover:border-[#3c3c52]"
    tabindex="0"
    role="slider"
    aria-label={`${label}: ${currentLabel}`}
    aria-valuetext={currentLabel}
    aria-valuenow={currentIndex}
    aria-valuemin={0}
    aria-valuemax={normalizedOptions.length - 1}
    aria-disabled={disabled}
    on:keydown={handleKeydown}
    on:click={() => step(1)}
  >
    <!-- Left Arrow Button (Mouse target only, tabindex -1) -->
    <button
      type="button"
      tabindex="-1"
      on:click|stopPropagation={() => step(-1)}
      {disabled}
      class="p-1 rounded text-slate-400 hover:text-white hover:bg-[#20202e] disabled:opacity-30 transition-colors outline-none"
      title="Previous Option (Left)"
      aria-label="Previous Option"
    >
      <svg class="w-3.5 h-3.5 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" />
      </svg>
    </button>

    <!-- Value Display -->
    <span class="px-2 py-0.5 text-xs font-mono font-bold text-sky-300 min-w-[32px] text-center truncate select-none">
      {currentLabel}
    </span>

    <!-- Right Arrow Button (Mouse target only, tabindex -1) -->
    <button
      type="button"
      tabindex="-1"
      on:click|stopPropagation={() => step(1)}
      {disabled}
      class="p-1 rounded text-slate-400 hover:text-white hover:bg-[#20202e] disabled:opacity-30 transition-colors outline-none"
      title="Next Option (Right)"
      aria-label="Next Option"
    >
      <svg class="w-3.5 h-3.5 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" />
      </svg>
    </button>
  </div>
</div>
