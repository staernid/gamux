<script>
  import { createEventDispatcher } from 'svelte';

  export let property = {};
  export let value = 0;
  export let disabled = false;
  export let onChange = null;
  export let onchange = null;

  const dispatch = createEventDispatcher();
  const inputId = `slider-${(property.key || '').replace(/[^a-zA-Z0-9_-]/g, '_') || Math.random().toString(36).slice(2, 8)}`;

  $: numValue = Number(value ?? property.default ?? 0);
  $: min = property.min ?? 0;
  $: max = property.max ?? (numValue > 100 ? numValue * 2 : 100);
  $: step = property.step ?? 1;

  function updateVal(next) {
    if (disabled) return;
    const n = Math.max(min, Math.min(max, Number(next)));
    value = n;
    dispatch('change', n);
    if (typeof onChange === 'function') onChange(n);
    if (typeof onchange === 'function') onchange(n);
  }
</script>

<div
  class="flex items-center space-x-3 w-full sm:w-auto min-w-[14rem]"
  data-testid={`control-number-${property.key}`}
>
  <input
    id={inputId}
    type="range"
    {min}
    {max}
    {step}
    value={numValue}
    {disabled}
    on:input={(e) => updateVal(e.target.value)}
    class="flex-1 h-1.5 bg-[#18181b] rounded-lg appearance-none cursor-pointer accent-white disabled:opacity-40 disabled:cursor-not-allowed border border-neutral-700"
    aria-label={property.label || property.key}
  />

  <div class="w-18 flex-shrink-0">
    <input
      type="number"
      {min}
      {max}
      {step}
      value={numValue}
      {disabled}
      on:input={(e) => updateVal(e.target.value)}
      class="w-full bg-[#121215] text-neutral-200 text-xs font-mono text-center rounded-lg px-2 py-1.5 border border-neutral-700
        focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      aria-label={`${property.label || property.key} numerical value`}
    />
  </div>
</div>
