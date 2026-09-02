<script>
  import { createEventDispatcher, onMount } from 'svelte';

  export let title = '';
  export let badge = '';
  export let subtitle = '';
  export let maxWidth = 'max-w-5xl';
  export let padding = 'p-5 sm:p-8';
  export let showClose = true;
  export let closeLabel = 'Done';
  export let closeKeyHint = 'Esc';
  export let customHeader = false;

  const dispatch = createEventDispatcher();
  let dialogEl;

  onMount(() => {
    if (dialogEl && typeof dialogEl.focus === 'function') {
      dialogEl.focus();
    }
  });

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      dispatch('close');
    }
  }

  function handleClose() {
    dispatch('close');
  }
</script>

<div
  bind:this={dialogEl}
  role="dialog"
  aria-modal="true"
  tabindex="-1"
  on:keydown={handleKeydown}
  class="fixed inset-0 z-50 bg-[#09090b]/95 backdrop-blur-2xl flex flex-col {padding} select-none animate-in fade-in duration-150 outline-none"
>
  {#if !customHeader && (title || $$slots.header || $$slots.headerActions)}
    <!-- Standard Modal Header Bar -->
    <div class="flex items-center justify-between {maxWidth} w-full mx-auto mb-5 flex-shrink-0">
      <div class="min-w-0 pr-4">
        {#if $$slots.header}
          <slot name="header" />
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
      <div class="flex items-center space-x-2 flex-shrink-0">
        <slot name="headerActions" />

        {#if showClose}
          <button
            type="button"
            on:click={handleClose}
            class="px-4 py-1.5 rounded-xl bg-[#18181b] hover:bg-[#222225] border border-[#27272a] text-neutral-200 hover:text-white text-xs font-mono transition-colors flex items-center space-x-1.5 cursor-pointer"
          >
            {#if closeKeyHint}
              <span class="px-1.5 py-0.5 rounded bg-[#121214] border border-[#27272a] text-neutral-400 font-bold text-3xs">
                {closeKeyHint}
              </span>
            {/if}
            <span>{closeLabel}</span>
          </button>
        {/if}
      </div>
    </div>
  {/if}

  <slot />
</div>
