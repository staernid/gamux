<script>
    import { selectDirectoryDialog } from '../../../utils/filesystem.js';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').PropertyDescriptor} [property]
   * @property {string | string[]} [value]
   * @property {boolean} [disabled]
   * @property {boolean} [hideHeader]
   * @property {((val: string | string[]) => void) | null} [onChange]
   * @property {((val: string | string[]) => void) | null} [onchange]
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
  let inputId = $derived(`path-${(property.key || '').replace(/[^a-zA-Z0-9_-]/g, '_') || uid}`);

  let isList = $derived(property.type === 'path_list' || Array.isArray(value));
  let pathList = $derived(Array.isArray(value) ? value : (value ? [value] : []));

  function notifyChange(nextVal) {
    (onchange || onChange)?.(nextVal);
  }

  async function browseSingle() {
    if (disabled) return;
    const selected = await selectDirectoryDialog(property.label || 'Select Directory', typeof value === 'string' ? value : '');
    if (selected) {
      value = selected;
      notifyChange(selected);
    }
  }

  async function browseAddToList() {
    if (disabled) return;
    const selected = await selectDirectoryDialog(property.label || 'Select Folder to Add', '');
    if (selected && !pathList.includes(selected)) {
      const next = [...pathList, selected];
      value = next;
      notifyChange(next);
    }
  }

  function removePath(p) {
    if (disabled) return;
    const next = pathList.filter(item => item !== p);
    value = next;
    notifyChange(next);
  }

  function handleInputChange(e) {
    if (disabled) return;
    const nextVal = e.target.value;
    value = nextVal;
    notifyChange(nextVal);
  }
</script>

{#if hideHeader}
  <div class="w-full sm:w-auto min-w-[14rem] space-y-2" data-testid={`control-path-${property.key}`}>
    {#if isList}
      <div class="flex items-center justify-end mb-1">
        <button
          type="button"
          {disabled}
          onclick={browseAddToList}
          class="px-2 py-1 text-4xs font-mono font-bold uppercase tracking-wider bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 rounded transition-colors cursor-pointer"
          aria-label="Add Folder"
        >
          + Add Folder
        </button>
      </div>

      {#if pathList.length === 0}
        <div class="p-2 rounded bg-[#101013] border border-surface-border text-center">
          <p class="text-3xs text-neutral-500 font-mono">No folders added</p>
        </div>
      {:else}
        <div class="space-y-1">
          {#each pathList as p}
            <div class="flex items-center justify-between gap-2 p-1.5 rounded bg-[#101013] border border-surface-border">
              <span class="text-3xs font-mono text-neutral-300 truncate select-all">{p}</span>
              <button
                type="button"
                {disabled}
                onclick={() => removePath(p)}
                class="p-0.5 text-neutral-500 hover:text-rose-400 transition-colors flex-shrink-0 cursor-pointer"
                aria-label={`Remove path ${p}`}
              >
                <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
                </svg>
              </button>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      <div class="flex items-center gap-2">
        <input
          id={inputId}
          type="text"
          value={typeof value === 'string' ? value : ''}
          {disabled}
          placeholder={typeof property.default === 'string' ? property.default : '/path/to/directory'}
          oninput={handleInputChange}
          aria-label={property.label || property.key}
          class="flex-1 bg-[#121215] text-neutral-200 text-xs font-mono rounded-lg px-3 py-2 border border-neutral-700
            focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors"
        />
        <button
          type="button"
          {disabled}
          onclick={browseSingle}
          class="flex-shrink-0 px-3 py-2 text-xs font-semibold bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 rounded-lg transition-colors cursor-pointer"
        >
          Browse
        </button>
      </div>
    {/if}
  </div>
{:else}
  <!-- Standalone Mode -->
  <div
    class="p-3 rounded-lg bg-surface-card/30 border border-surface-border/50 transition-all space-y-2
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-surface-card/60 hover:border-surface-borderHighlight'}"
    data-testid={`control-path-${property.key}`}
  >
    <div class="flex items-start sm:items-center justify-between gap-3">
      <div class="space-y-0.5 min-w-0 flex-1">
        <label for={inputId} class="block text-xs font-semibold text-neutral-200 cursor-pointer">
          {property.label || property.key}
        </label>
        {#if property.description}
          <p class="text-xs text-neutral-400 leading-relaxed">
            {property.description}
          </p>
        {/if}
      </div>

      {#if isList}
        <button
          type="button"
          {disabled}
          onclick={browseAddToList}
          class="flex-shrink-0 px-2.5 py-1 text-2xs font-semibold uppercase tracking-wider bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 rounded transition-colors"
          aria-label="Add Folder"
        >
          Add Folder
        </button>
      {/if}
    </div>

    {#if isList}
      {#if pathList.length === 0}
        <div class="p-2 rounded bg-[#101013] border border-surface-border text-center">
          <p class="text-xs text-neutral-500 font-mono">No folders added</p>
        </div>
      {:else}
        <div class="space-y-1">
          {#each pathList as p}
            <div class="flex items-center justify-between gap-2 p-2 rounded bg-[#101013] border border-surface-border">
              <span class="text-xs font-mono text-neutral-300 truncate select-all">{p}</span>
              <button
                type="button"
                {disabled}
                onclick={() => removePath(p)}
                class="p-1 text-neutral-500 hover:text-rose-400 transition-colors flex-shrink-0"
                aria-label={`Remove path ${p}`}
              >
                <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
                </svg>
              </button>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      <div class="flex items-center gap-2">
        <input
          id={inputId}
          type="text"
          value={typeof value === 'string' ? value : ''}
          {disabled}
          placeholder={typeof property.default === 'string' ? property.default : '/path/to/directory'}
          oninput={handleInputChange}
          class="flex-1 bg-[#121215] text-neutral-200 text-xs font-mono rounded-lg px-3 py-2 border border-neutral-700
            focus:outline-none focus:border-neutral-400 focus:ring-1 focus:ring-neutral-400 transition-colors"
        />
        <button
          type="button"
          {disabled}
          onclick={browseSingle}
          class="flex-shrink-0 px-3 py-2 text-xs font-semibold bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 rounded-lg transition-colors"
        >
          Browse
        </button>
      </div>
    {/if}
  </div>
{/if}
