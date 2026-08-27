<script>
  import { notificationStore } from '../../stores/notifications.js';

  const toasts = notificationStore.toasts;

  $: topCenterToasts = $toasts.filter(t => t.position === 'top-center');
  $: bottomCenterToasts = $toasts.filter(t => t.position === 'bottom-center');
  $: bottomRightToasts = $toasts.filter(t => !t.position || t.position === 'bottom-right');
</script>

<!-- Top Center: Transient OSD Pills (Sort switches, HUD notifications) -->
{#if topCenterToasts.length > 0}
  <div class="fixed top-14 left-1/2 -translate-x-1/2 z-50 flex flex-col items-center space-y-2 pointer-events-none">
    {#each topCenterToasts as toast (toast.id)}
      <div class="pointer-events-auto bg-[#14151f]/95 border border-white/20 text-white shadow-2xl backdrop-blur-xl px-5 py-2 rounded-full flex items-center space-x-2.5 animate-fade-in">
        {#if toast.icon === 'sort' || toast.type === 'osd'}
          <svg class="w-4 h-4 text-slate-300 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
      <div class="pointer-events-auto bg-[#14151f]/95 border border-white/20 text-white shadow-2xl backdrop-blur-xl px-5 py-2 rounded-full flex items-center space-x-2.5 animate-fade-in">
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
{/if}
