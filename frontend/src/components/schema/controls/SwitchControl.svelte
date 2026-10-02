<script>
    import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [property]
   * @property {boolean} [value]
   * @property {boolean} [disabled]
   * @property {boolean} [hideHeader]
   * @property {any} [onChange]
   * @property {any} [onchange]
   */

  /** @type {Props} */
  let {
    property = {},
    value = $bindable(false),
    disabled = false,
    hideHeader = false,
    onChange = null,
    onchange = null
  } = $props();

  let isChecked = $derived(Boolean(value));

  function handleChange(nextVal) {
    value = nextVal;
    (onchange || onChange)?.(nextVal);
  }

  function toggle() {
    if (disabled) return;
    handleChange(!isChecked);
  }

  function handleKeyDown(e) {
    if (disabled) return;
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      toggle();
    }
  }
</script>

{#if hideHeader}
  <div
    class="flex items-center"
    data-testid={`control-switch-${property.key}`}
  >
    <div
      class="w-9 h-5 rounded-full transition-colors relative flex items-center p-0.5 cursor-pointer select-none
        {isChecked ? 'bg-neutral-200' : 'bg-neutral-800 border border-neutral-700'}
        {disabled ? 'opacity-40 pointer-events-none' : 'hover:opacity-90'}"
      role="switch"
      tabindex="0"
      aria-checked={isChecked}
      aria-label={property.label || 'Toggle switch'}
      onclick={toggle}
      onkeydown={handleKeyDown}
    >
      <div
        class="w-3.5 h-3.5 rounded-full shadow-sm transition-transform transform
          {isChecked ? 'bg-black translate-x-4' : 'bg-neutral-400 translate-x-0.5'}"
      ></div>
    </div>
  </div>
{:else}
  <!-- Standalone Mode: Reuses SeamlessSwitch -->
  <div data-testid={`control-switch-${property.key}`}>
    <SeamlessSwitch
      label={property.label || property.key}
      description={property.description || ''}
      value={isChecked}
      {disabled}
      onchange={(val) => handleChange(val)}
    />
  </div>
{/if}
