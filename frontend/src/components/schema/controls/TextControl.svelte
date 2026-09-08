<script>
  import { createEventDispatcher } from 'svelte';

  export let property = {};
  export let value = '';
  export let disabled = false;
  export let onChange = null;
  export let onchange = null;

  const dispatch = createEventDispatcher();
  const inputId = `txt-${(property.key || '').replace(/[^a-zA-Z0-9_-]/g, '_') || Math.random().toString(36).slice(2, 8)}`;

  function handleInput(e) {
    if (disabled) return;
    const nextVal = e.target.value;
    value = nextVal;
    dispatch('change', nextVal);
    if (typeof onChange === 'function') onChange(nextVal);
    if (typeof onchange === 'function') onchange(nextVal);
  }
</script>

<div
  class="w-full sm:w-auto min-w-[14rem]"
  data-testid={`control-text-${property.key}`}
>
  <input
    id={inputId}
    type="text"
    value={value ?? ''}
    {disabled}
    placeholder={typeof property.default === 'string' ? property.default : ''}
    on:input={handleInput}
    aria-label={property.label || property.key}
    class="w-full bg-[#121215] text-neutral-200 text-xs rounded-lg px-3 py-2 border border-neutral-700
      focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
  />
</div>
