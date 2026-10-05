<script>
  import { notificationStore } from '../../stores/notifications.js';
  import { activeView } from '../../stores/library.js';
  import { modalStore } from '../../stores/modal.js';
  import { configStore } from '../../stores/config.js';

  const toasts = notificationStore.toasts;
  const undoQueue = notificationStore.undoQueue;

  let topCenterToasts = $derived($toasts.filter(t => t.position === 'top-center'));
  let bottomCenterToasts = $derived($toasts.filter(t => t.position === 'bottom-center'));
  let bottomRightToasts = $derived($toasts.filter(t => !t.position || t.position === 'bottom-right'));

  function handleToastAction(toast) {
    if (typeof toast.onAction === 'function') {
      toast.onAction();
      notificationStore.toasts.dismiss(toast.id);
      return;
    }
    if (toast.actionView) {
      if ($activeView === 'settings' && configStore.isDirty && toast.actionView !== 'settings') {
        modalStore.open('confirm', {
          title: 'Unsaved Changes',
          message: 'You have unsaved changes in Settings. Do you want to discard them or keep editing?',
          confirmLabel: 'Discard Changes',
          cancelLabel: 'Keep Editing',
          danger: true,
          onConfirm: () => {
            configStore.setDirty(false);
            $activeView = toast.actionView;
            notificationStore.toasts.dismiss(toast.id);
          }
        });
        return;
      }
      $activeView = toast.actionView;
      notificationStore.toasts.dismiss(toast.id);
    }
  }
</script>

<!-- Top Center: Transient OSD Pills (Sort switches, HUD notifications) -->
{#if topCenterToasts.length > 0}
  <div class="fixed top-14 left-1/2 -translate-x-1/2 z-50 flex flex-col items-center space-y-2 pointer-events-none">
    {#each topCenterToasts as toast (toast.id)}
      <div class="pointer-events-auto bg-surface-header/95 border border-surface-border text-white shadow-2xl  px-5 py-2 rounded-full flex items-center space-x-2.5 animate-fade-in">
        {#if toast.icon === 'sort' || toast.type === 'osd'}
          <svg class="w-4 h-4 text-neutral-300 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
          </svg>
        {/if}
        <span class="text-xs font-bold tracking-wide">{toast.message || toast.title}</span>
      </div>
    {/each}
  </div>
{/if}

<!-- Bottom Center: Centered Transient HUD -->
{#if bottomCenterToasts.length > 0}
  <div class="fixed bottom-12 left-1/2 -translate-x-1/2 z-50 flex flex-col items-center space-y-2 pointer-events-none">
    {#each bottomCenterToasts as toast (toast.id)}
      <div class="pointer-events-auto bg-surface-header/95 border border-surface-border text-white shadow-2xl  px-5 py-2 rounded-full flex items-center space-x-2.5 animate-fade-in">
        <span class="text-xs font-bold tracking-wide">{toast.message || toast.title}</span>
      </div>
    {/each}
  </div>
{/if}

<!-- Bottom Right: Standard System Alerts & Task Results -->
{#if bottomRightToasts.length > 0}
  <div class="fixed bottom-6 right-6 z-50 flex flex-col space-y-3 pointer-events-none max-w-sm w-full">
    {#each bottomRightToasts as toast (toast.id)}
      {@const isError = toast.type === 'error'}
      {@const isSuccess = toast.type === 'success'}
      {@const bgClass = isError ? 'bg-[#181214]/95 border-rose-800 text-rose-100 shadow-rose-950/50' :
                        isSuccess ? 'bg-[#121614]/95 border-emerald-800 text-emerald-100 shadow-emerald-950/50' :
                        'bg-surface-input/95 border-surface-border text-neutral-100 shadow-2xl'}
      <div class="{bgClass} pointer-events-auto border rounded-2xl p-3.5 shadow-2xl  transition-all duration-300 flex items-start space-x-3 animate-fade-in">
        {#if isError}
          <svg class="w-4 h-4 text-rose-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        {:else if isSuccess}
          <svg class="w-4 h-4 text-emerald-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
        {:else}
          <svg class="w-4 h-4 text-neutral-300 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        {/if}
        <div class="flex-1 text-xs min-w-0">
          <strong class="block font-bold mb-0.5 text-white truncate">{toast.title}</strong>
          <p class="text-neutral-300 leading-relaxed break-words">{toast.message}</p>
          {#if toast.gameName}
            <span class="inline-block mt-1 text-3xs font-mono bg-black/40 text-neutral-300 px-1.5 py-0.5 rounded border border-neutral-700">
              {toast.gameName}
            </span>
          {/if}
          {#if toast.actionLabel || toast.actionView || toast.onAction}
            <button
              type="button"
              onclick={() => handleToastAction(toast)}
              class="mt-2 text-2xs font-bold text-neutral-200 hover:text-white transition-colors flex items-center space-x-1 cursor-pointer"
            >
              <span>{toast.actionLabel || (toast.actionView === 'tasks' ? 'View in Tasks' : 'Open')}</span>
              <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
              </svg>
            </button>
          {/if}
        </div>
        <button
          onclick={() => notificationStore.toasts.dismiss(toast.id)}
          class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-white/10 transition-colors cursor-pointer"
          aria-label="Dismiss toast"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}

<!-- Bottom Center: Undo Snackbars (Reversible mutations with countdown) -->
{#if $undoQueue && $undoQueue.length > 0}
  <div class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex flex-col items-center space-y-2 pointer-events-none" data-testid="undo-snackbar-container">
    {#each $undoQueue as item (item.id)}
      <div class="pointer-events-auto bg-[#18181b] border border-[#3f3f46] text-white shadow-2xl px-4 py-2.5 rounded-2xl flex items-center space-x-3 text-xs animate-fade-in">
        <span class="text-neutral-200 font-medium">{item.message}</span>
        <button
          type="button"
          data-testid="undo-action-btn"
          onclick={() => notificationStore.triggerUndo(item.id)}
          class="px-2.5 py-1 rounded-xl bg-white text-neutral-950 hover:bg-neutral-200 font-black text-2xs transition shadow cursor-pointer flex items-center space-x-1"
        >
          <span>Undo ({item.remaining}s)</span>
        </button>
        <button
          type="button"
          data-testid="dismiss-undo-btn"
          onclick={() => notificationStore.dismissUndo(item.id)}
          class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-white/10 transition-colors cursor-pointer"
          aria-label="Dismiss undo"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}

