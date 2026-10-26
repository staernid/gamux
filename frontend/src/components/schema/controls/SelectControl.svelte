<script>
  import SelectControl from './SelectControl.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').PropertyDescriptor} [property]
   * @property {string | number | boolean} [value]
   * @property {boolean} [disabled]
   * @property {boolean} [hideHeader]
   * @property {((val: string | number | boolean) => void) | null} [onChange]
   * @property {((val: string | number | boolean) => void) | null} [onchange]
   */

  /** @type {Props} */
  let {
    property = {},
    value = $bindable(''),
    disabled = false,
    hideHeader = false,
    onChange = null,
    onchange = null
  } = $props();
  const uid = Math.random().toString(36).slice(2, 8);
  let selectId = $derived(`select-${(property.key || '').replace(/[^a-zA-Z0-9_-]/g, '_') || uid}`);

  let options = $derived(property.options || []);

  function handleChange(e) {
    if (disabled) return;
    let selected = e.target.value;
    if (typeof property.default === 'number' && !isNaN(Number(selected))) {
      selected = Number(selected);
    }
    value = selected;
    (onchange || onChange)?.(selected);
  }
</script>

{#if hideHeader}
  <div class="relative w-full sm:w-auto min-w-[12rem]" data-testid={`control-select-${property.key}`}>
    <select
      id={selectId}
      value={value ?? ''}
      {disabled}
      aria-label={property.label || property.key}
      onchange={handleChange}
      class="w-full bg-surface-input text-neutral-200 text-xs rounded-lg px-3 py-2 border border-surface-border
        focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors
        appearance-none pr-8 cursor-pointer disabled:cursor-not-allowed disabled:opacity-40"
    >
      {#each options as opt}
        <option value={opt.value}>
          {opt.label || opt.value}
        </option>
      {/each}
    </select>

    <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-neutral-400">
      <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
      </svg>
    </div>
  </div>
{:else}
  <!-- Standalone Mode -->
  <div
    class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 rounded-lg bg-surface-card/30 border border-surface-border/50 transition-all
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-surface-card/60 hover:border-surface-borderHighlight'}"
    data-testid={`control-select-${property.key}`}
  >
    <div class="space-y-0.5 min-w-0 flex-1 sm:max-w-md">
      <label for={selectId} class="block text-xs font-semibold text-neutral-200 cursor-pointer">
        {property.label || property.key}
      </label>
      {#if property.description}
        <p class="text-xs text-neutral-400 leading-relaxed">
          {property.description}
        </p>
      {/if}
    </div>

    <div class="flex-shrink-0 w-full sm:w-auto min-w-[12rem]">
      <SelectControl
        {property}
        {value}
        {disabled}
        hideHeader={true}
        onchange={(val) => {
          value = val;
          (onchange || onChange)?.(val);
        }}
      />
    </div>
  </div>
{/if}
