<script>
  
  let {
    label = '',
    description = '',
    tooltip = '',
    options = [],
    min = undefined,
    max = undefined,
    stepSize = 1,
    value = $bindable(null),
    disabled = false
  ,
    onchange
  } = $props();

  let resolvedOptions = $derived.by(() => {
    if (Array.isArray(options) && options.length > 0) {
      return options;
    }
    if (min !== undefined && max !== undefined && Number(min) <= Number(max)) {
      const generated = [];
      const s = Number(stepSize) || 1;
      for (let v = Number(min); v <= Number(max); v += s) {
        generated.push({ value: v, label: String(v) });
      }
      return generated;
    }
    return [];
  });

  let normalizedOptions = $derived(
    resolvedOptions.map(opt => {
      if (typeof opt === 'object' && opt !== null) {
        return { value: opt.value, label: opt.label ?? String(opt.value) };
      }
      return { value: opt, label: String(opt) };
    })
  );

  let currentIndex = $derived(normalizedOptions.findIndex(o => o.value === value || Number(o.value) === Number(value)));
  let currentLabel = $derived(currentIndex >= 0 ? normalizedOptions[currentIndex].label : String(value ?? ''));
  let effectiveTooltip = $derived(tooltip || description);

  function step(dir) {
    if (disabled || normalizedOptions.length === 0) return;
    let nextIdx = currentIndex + dir;
    if (nextIdx < 0) nextIdx = normalizedOptions.length - 1;
    if (nextIdx >= normalizedOptions.length) nextIdx = 0;

    const nextVal = normalizedOptions[nextIdx].value;
    value = nextVal;
    onchange?.(nextVal);
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
  class="flex items-start sm:items-center justify-between gap-3 p-2.5 sm:p-3 bg-surface-input hover:bg-surface-card focus-within:bg-surface-card border border-surface-border focus-within:border-neutral-500 rounded-xl transition-all select-none group"
  role="group"
  aria-label={label}
>
  <!-- Label & Description -->
  <div class="space-y-0.5 flex-1 min-w-[120px]">
    <div class="flex items-center space-x-1.5 flex-wrap">
      <span class="block font-bold text-neutral-200 text-xs group-focus-within:text-white transition-colors">{label}</span>
      {#if effectiveTooltip}
        <span
          class="text-neutral-400 hover:text-white transition-colors cursor-help p-0.5 flex-shrink-0"
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
      <p class="text-xs text-neutral-400 leading-relaxed">{description}</p>
    {/if}
  </div>

  <!-- Unified Stepper Control Capsule (Single Focus Stop for Controller & Keyboard) -->
  <div
    class="flex items-center space-x-1 bg-surface-input border border-surface-border focus:border-white focus:ring-1 focus:ring-white/20 rounded-lg p-0.5 flex-shrink-0 outline-none transition-all hover:border-neutral-500"
    tabindex="0"
    role="slider"
    aria-label={`${label}: ${currentLabel}`}
    aria-valuetext={currentLabel}
    aria-valuenow={currentIndex}
    aria-valuemin={0}
    aria-valuemax={normalizedOptions.length - 1}
    aria-disabled={disabled}
    onkeydown={handleKeydown}
  >
    <!-- Left Arrow Button (Mouse target only, tabindex -1) -->
    <button
      type="button"
      tabindex="-1"
      onclick={(e) => { e.stopPropagation(); step(-1); }}
      {disabled}
      class="p-1 rounded text-neutral-400 hover:text-white hover:bg-surface-cardHover disabled:opacity-30 transition-colors outline-none cursor-pointer"
      title="Previous Option (Left)"
      aria-label="Previous Option"
    >
      <svg class="w-3.5 h-3.5 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" />
      </svg>
    </button>

    <!-- Value Display -->
    <span class="px-2 py-0.5 text-xs font-mono font-bold text-white min-w-[32px] text-center truncate select-none cursor-default">
      {currentLabel}
    </span>

    <!-- Right Arrow Button (Mouse target only, tabindex -1) -->
    <button
      type="button"
      tabindex="-1"
      onclick={(e) => { e.stopPropagation(); step(1); }}
      {disabled}
      class="p-1 rounded text-neutral-400 hover:text-white hover:bg-surface-cardHover disabled:opacity-30 transition-colors outline-none cursor-pointer"
      title="Next Option (Right)"
      aria-label="Next Option"
    >
      <svg class="w-3.5 h-3.5 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" />
      </svg>
    </button>
  </div>
</div>
