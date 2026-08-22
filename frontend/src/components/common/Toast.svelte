<script>
  import { notificationStore } from '../../stores/notifications.js';

  const toasts = notificationStore.toasts;
</script>

<div class="fixed bottom-6 right-6 z-50 flex flex-col space-y-3 pointer-events-none max-w-sm w-full">
  {#each $toasts as toast (toast.id)}
    {@const isError = toast.type === 'error'}
    {@const isSuccess = toast.type === 'success'}
    {@const bgClass = isError ? 'bg-rose-950/95 border-rose-800 text-rose-100 shadow-rose-950/50' :
                      isSuccess ? 'bg-emerald-950/95 border-emerald-800 text-emerald-100 shadow-emerald-950/50' :
                      'bg-[#121a28]/95 border-cyan-800/80 text-slate-100 shadow-cyan-950/40'}
    <div class="{bgClass} pointer-events-auto border rounded-2xl p-3.5 shadow-2xl backdrop-blur-xl transition-all duration-300 flex items-start space-x-3 animate-fade-in">
      {#if isError}
        <svg class="w-4 h-4 text-rose-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      {:else if isSuccess}
        <svg class="w-4 h-4 text-emerald-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
      {:else}
        <svg class="w-4 h-4 text-cyan-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      {/if}
      <div class="flex-1 text-xs min-w-0">
        <strong class="block font-bold mb-0.5 text-white truncate">{toast.title}</strong>
        <p class="text-slate-300 leading-snug break-words">{toast.message}</p>
        {#if toast.gameName}
          <span class="inline-block mt-1 text-[10px] font-mono bg-black/40 text-cyan-300 px-1.5 py-0.5 rounded">
            {toast.gameName}
          </span>
        {/if}
      </div>
      <button
        on:click={() => notificationStore.toasts.dismiss(toast.id)}
        class="text-slate-400 hover:text-white text-xs font-bold px-1"
        aria-label="Dismiss toast"
      >
        ✕
      </button>
    </div>
  {/each}
</div>
