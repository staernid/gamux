<script>
    import { playTick } from '../../utils/audio.js';

  /**
   * @typedef {Object} Props
   * @property {any} [property]
   * @property {boolean} [isOverridden]
   * @property {any} [inheritedValue]
   * @property {string} [scope]
   * @property {boolean} [disabled]
   * @property {any} [onReset]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    property = {},
    isOverridden = false,
    inheritedValue = undefined,
    scope = 'global',
    disabled = false,
    onReset = null,
    children
  ,
    onreset = null
  } = $props();

  function formatInherited(val) {
    if (val === undefined || val === null) {
      if (property.default !== undefined && property.default !== null) {
        return formatInherited(property.default);
      }
      return 'Default';
    }
    if (typeof val === 'boolean') {
      return val ? 'Enabled' : 'Disabled';
    }
    if (typeof val === 'string') {
      if (!val) return 'Empty';
      if (property.options && Array.isArray(property.options)) {
        const opt = property.options.find(o => String(o.value) === String(val));
        if (opt) return opt.label || opt.value;
      }
      return val;
    }
    if (typeof val === 'number') {
      return String(val);
    }
    if (Array.isArray(val)) {
      return `${val.length} items`;
    }
    if (typeof val === 'object') {
      return 'Configured';
    }
    return String(val);
  }

  function handleReset() {
    if (disabled) return;
    playTick();
    onreset?.({ key: property.key });
    if (typeof onReset === 'function') onReset({ key: property.key });
  }
</script>

<div
  class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 rounded-lg transition-all
    {isOverridden && scope === 'game'
      ? 'bg-[#18181c] border border-neutral-600 shadow-xs'
      : 'bg-surface-card/30 border border-surface-border/50 hover:bg-surface-card/50 hover:border-surface-borderHighlight'}"
  data-testid={`property-row-${property.key}`}
>
  <!-- Label & Description -->
  <div class="space-y-1 min-w-0 flex-1 sm:max-w-md">
    <div class="flex items-center space-x-2 flex-wrap gap-y-1">
      <span class="block text-xs font-semibold text-neutral-200" data-testid={`property-label-${property.key}`}>
        {property.label || property.key}
      </span>

      {#if scope === 'game'}
        {#if isOverridden}
          <span
            class="px-1.5 py-0.2 rounded text-4xs font-mono font-bold uppercase tracking-wider bg-neutral-800 text-neutral-200 border border-neutral-600"
            data-testid={`property-pill-overridden-${property.key}`}
          >
            Overridden
          </span>
          <button
            type="button"
            onclick={handleReset}
            {disabled}
            class="text-4xs font-mono text-neutral-400 hover:text-white px-1.5 py-0.2 rounded bg-neutral-900 hover:bg-neutral-800 border border-neutral-700 transition cursor-pointer flex items-center space-x-1"
            title="Reset to inherited global setting"
            aria-label={`Reset ${property.label || property.key} to global default`}
            data-testid={`property-reset-btn-${property.key}`}
          >
            <span>↺</span>
            <span>Reset</span>
          </button>
        {:else}
          <span
            class="px-1.5 py-0.2 rounded text-4xs font-mono text-neutral-400 bg-[#121215] border border-neutral-800"
            data-testid={`property-pill-inherited-${property.key}`}
          >
            Inherited: {formatInherited(inheritedValue)}
          </span>
        {/if}
      {/if}
    </div>

    {#if property.description}
      <p class="text-xs text-neutral-400 leading-relaxed" data-testid={`property-desc-${property.key}`}>
        {property.description}
      </p>
    {/if}
  </div>

  <!-- Control Slot -->
  <div class="flex-shrink-0 w-full sm:w-auto min-w-[10rem] flex justify-end">
    {@render children?.()}
  </div>
</div>
