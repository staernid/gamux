<script>
  import { onMount } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {string} [title]
   * @property {string} [badge]
   * @property {string} [subtitle]
   * @property {string} [maxWidth]
   * @property {string} [padding]
   * @property {boolean} [showClose]
   * @property {string} [closeLabel]
   * @property {string} [closeKeyHint]
   * @property {boolean} [customHeader]
   * @property {() => void} [onclose]
   * @property {import('svelte').Snippet} [header]
   * @property {import('svelte').Snippet} [headerActions]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    title = '',
    badge = '',
    subtitle = '',
    maxWidth = 'max-w-5xl',
    padding = 'p-5 sm:p-8',
    showClose = true,
    closeLabel = 'Done',
    closeKeyHint = 'Esc',
    customHeader = false,
    onclose,
    header,
    headerActions,
    children
  } = $props();

  let dialogEl = $state();

  onMount(() => {
    if (dialogEl && typeof dialogEl.focus === 'function') {
      dialogEl.focus();
    }
  });

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      onclose?.();
    }
  }

  function handleClose() {
    onclose?.();
  }
</script>

<div
  bind:this={dialogEl}
  role="dialog"
  aria-modal="true"
  tabindex="-1"
  onkeydown={handleKeydown}
  class="fixed inset-0 z-50 bg-surface-base/95 dark:bg-surface-base/98 flex flex-col {padding} select-none animate-in fade-in duration-150 outline-none"
>
  {#if !customHeader && (title || header || headerActions)}
    <!-- Standard Modal Header Bar -->
    <div class="flex items-center justify-between {maxWidth} w-full mx-auto mb-5 shrink-0">
      <div class="min-w-0 pr-4">
        {#if header}
          {@render header?.()}
        {:else}
          <div class="flex items-center space-x-2.5">
            {#if badge}
              <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-neutral-800 text-neutral-200 border border-neutral-700 uppercase tracking-wider">
                {badge}
              </span>
            {/if}
            {#if title}
              <h2 class="text-lg sm:text-xl font-black text-white tracking-tight truncate">
                {title}
              </h2>
            {/if}
          </div>
          {#if subtitle}
            <p class="text-xs text-neutral-400 mt-1">{subtitle}</p>
          {/if}
        {/if}
      </div>

      <!-- Header Actions & Close Button -->
      <div class="flex items-center space-x-2 shrink-0">
        {@render headerActions?.()}

        {#if showClose}
          <button
            type="button"
            onclick={handleClose}
            class="px-4 py-1.5 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white text-xs font-mono transition-colors flex items-center space-x-1.5 cursor-pointer"
          >
            {#if closeKeyHint}
              <span class="px-1.5 py-0.5 rounded bg-surface-header border border-surface-border text-neutral-400 font-bold text-3xs">
                {closeKeyHint}
              </span>
            {/if}
            <span>{closeLabel}</span>
          </button>
        {/if}
      </div>
    </div>
  {/if}

  {@render children?.()}
</div>
