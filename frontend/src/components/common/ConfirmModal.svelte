<script>
  import { onMount } from 'svelte';
  import { modalStore } from '../../stores/modal.js';

  let {
    title = 'Confirm Action',
    message = 'Are you sure you want to proceed?',
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    danger = false,
    onConfirm = null,
    onCancel = null
  ,
    onconfirm,
    oncancel
  } = $props();
  let confirmBtn = $state();
  let cancelBtn = $state();

  onMount(() => {
    if (danger) {
      if (cancelBtn && typeof cancelBtn.focus === 'function') {
        cancelBtn.focus();
      }
    } else if (confirmBtn && typeof confirmBtn.focus === 'function') {
      confirmBtn.focus();
    }
  });

  function handleConfirm() {
    (onconfirm || onConfirm)?.();
    modalStore.close();
  }

  function handleCancel() {
    (oncancel || onCancel)?.();
    modalStore.close();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      handleCancel();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  role="dialog"
  aria-modal="true"
  aria-labelledby="confirm-modal-title"
  aria-describedby="confirm-modal-desc"
  class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4 select-none animate-in fade-in duration-150"
>
  <div class="bg-surface-header border {danger ? 'border-rose-500/40' : 'border-surface-border'} rounded-2xl p-6 max-w-md w-full shadow-2xl space-y-4">
    <!-- Header with Icon -->
    <div class="flex items-start space-x-3.5">
      <div class="p-2.5 rounded-xl {danger ? 'bg-rose-500/15 text-rose-400 border border-rose-500/30' : 'bg-neutral-800 text-neutral-200 border border-neutral-700'} flex-shrink-0">
        {#if danger}
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        {:else}
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        {/if}
      </div>

      <div class="space-y-1 min-w-0 flex-1">
        <h3 id="confirm-modal-title" class="text-base font-black text-white tracking-tight">
          {title}
        </h3>
        <p id="confirm-modal-desc" class="text-xs text-neutral-300 leading-relaxed">
          {message}
        </p>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center justify-end space-x-2.5 pt-2 border-t border-surface-border">
      <button
        type="button"
        bind:this={cancelBtn}
        onclick={handleCancel}
        class="px-4 py-2 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-xs font-bold text-neutral-200 hover:text-white transition-colors cursor-pointer"
      >
        {cancelLabel}
      </button>

      <button
        type="button"
        bind:this={confirmBtn}
        onclick={handleConfirm}
        class="px-4 py-2 rounded-xl text-xs font-black transition-all shadow-lg flex items-center space-x-1.5 cursor-pointer {danger ? 'bg-rose-600 hover:bg-rose-500 text-white' : 'bg-white hover:bg-neutral-200 text-black'}"
      >
        {confirmLabel}
      </button>
    </div>
  </div>
</div>
