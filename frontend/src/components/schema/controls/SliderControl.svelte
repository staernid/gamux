<script>
  
  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').PropertyDescriptor} [property]
   * @property {number} [value]
   * @property {boolean} [disabled]
   * @property {((val: number) => void) | null} [onChange]
   * @property {((val: number) => void) | null} [onchange]
   */

  /** @type {Props} */
  let {
    property = {},
    value = $bindable(0),
    disabled = false,
    onChange = null,
    onchange = null
  } = $props();
  const uid = Math.random().toString(36).slice(2, 8);
  let inputId = $derived(`slider-${(property.key || '').replace(/[^a-zA-Z0-9_-]/g, '_') || uid}`);

  let numValue = $derived(Number(value ?? property.default ?? 0));
  let min = $derived(property.min ?? 0);
  let max = $derived(property.max ?? (numValue > 100 ? numValue * 2 : 100));
  let step = $derived(property.step ?? 1);

  function updateVal(next) {
    if (disabled) return;
    const n = Math.max(min, Math.min(max, Number(next)));
    value = n;
    (onchange || onChange)?.(n);
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
    oninput={(e) => updateVal(e.target.value)}
    class="flex-1 h-1.5 bg-surface-card rounded-lg appearance-none cursor-pointer accent-white disabled:opacity-40 disabled:cursor-not-allowed border border-surface-border"
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
      oninput={(e) => updateVal(e.target.value)}
      class="w-full bg-surface-input text-neutral-200 text-xs font-mono text-center rounded-lg px-2 py-1.5 border border-surface-border
        focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      aria-label={`${property.label || property.key} numerical value`}
    />
  </div>
</div>
