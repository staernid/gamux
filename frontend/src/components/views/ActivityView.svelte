<script>
  import { activeView } from '../../stores/library.js';
  import { taskQueue } from '../../stores/queue.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { formatTimeAgo } from '../../utils/format.js';

  const activeTask = taskQueue.activeTask;
  const queue = taskQueue.queue;
  const totalTasks = taskQueue.totalCount;

  const items = notificationStore.items;
  const unreadCount = notificationStore.unreadCount;

  function handleClearActivity() {
    notificationStore.items.clearAll();
  }

  function markAllRead() {
    notificationStore.items.markAllRead();
  }
</script>

<div class="flex-1 overflow-y-auto p-6 sm:p-10 max-w-3xl mx-auto w-full space-y-6 text-xs sm:text-sm animate-fade-in">
  <!-- Header -->
  <div class="border-b border-[#20202c] pb-4 flex items-center justify-between">
    <div class="flex items-center space-x-3">
      <div>
        <h2 class="text-xl font-black text-white">Operations & Activity</h2>
        <p class="text-xs text-slate-400 mt-0.5">Live background tasks, downloads, and system execution logs</p>
      </div>
      {#if $totalTasks > 0}
        <span class="text-xs font-mono bg-white text-black px-2.5 py-0.5 rounded-full font-black shadow-sm">
          {$totalTasks} active
        </span>
      {/if}
    </div>

    <div class="flex items-center space-x-2">
      {#if $items.length > 0}
        <button
          type="button"
          on:click={markAllRead}
          class="text-xs font-bold bg-[#181822] hover:bg-[#222230] text-slate-300 px-3 py-1.5 rounded-lg border border-[#2a2a3a] transition-colors"
        >
          Mark Read
        </button>
        <button
          type="button"
          on:click={handleClearActivity}
          class="text-xs font-bold bg-[#181822] hover:bg-[#222230] text-slate-300 px-3 py-1.5 rounded-lg border border-[#2a2a3a] transition-colors"
        >
          Clear History
        </button>
      {/if}
      <button
        type="button"
        on:click={() => { $activeView = 'library'; }}
        class="text-xs font-bold text-slate-400 hover:text-white px-3 py-1.5 rounded-lg bg-[#1a1a24] border border-[#2a2a38]"
      >
        ✕ Shelf
      </button>
    </div>
  </div>

  <div class="space-y-4">
    <!-- Active In-Progress Task -->
    {#if $activeTask}
      <div class="bg-[#161622] border border-[#2e2e42] p-5 rounded-2xl space-y-3 shadow-lg">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-2.5 min-w-0">
            <span class="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-ping flex-shrink-0"></span>
            <strong class="font-bold text-white truncate text-sm sm:text-base">{$activeTask.title}</strong>
            {#if $activeTask.gameName}
              <span class="text-[11px] font-mono bg-black/40 text-slate-300 px-2 py-0.5 rounded">
                {$activeTask.gameName}
              </span>
            {/if}
          </div>
          <span class="font-mono font-black text-emerald-400 text-sm sm:text-base">{$activeTask.percent}%</span>
        </div>

        <div class="w-full bg-black/60 h-2 rounded-full overflow-hidden">
          <div
            class="bg-emerald-400 h-full transition-all duration-150"
            style="width: {$activeTask.percent}%"
          ></div>
        </div>

        <div class="flex items-center justify-between text-xs font-mono text-slate-400 pt-0.5">
          <span class="truncate max-w-[280px]">{$activeTask.stepText}</span>
          <span class="truncate text-slate-500 max-w-[200px]">{$activeTask.details}</span>
        </div>
      </div>
    {/if}

    <!-- Queued Tasks -->
    {#if $queue.length > 0}
      <div class="space-y-1.5">
        <span class="block text-[11px] uppercase font-bold text-slate-500 tracking-wider">
          Queued ({$queue.length})
        </span>
        <div class="space-y-1.5">
          {#each $queue as t, idx}
            <div class="bg-[#121218] border border-[#22222e] p-3.5 rounded-xl flex items-center justify-between text-xs">
              <div class="flex items-center space-x-2.5 min-w-0">
                <span class="text-slate-500 font-mono font-bold">#{idx + 1}</span>
                <span class="text-slate-200 font-bold truncate">{t.title}</span>
                {#if t.gameName}
                  <span class="font-mono text-slate-400 bg-black/40 px-1.5 py-0.5 rounded text-[10px]">
                    {t.gameName}
                  </span>
                {/if}
              </div>
              <span class="text-slate-500 font-mono">Waiting</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- History Stream -->
    {#if $items.length > 0}
      <div class="space-y-2">
        <span class="block text-[11px] uppercase font-bold text-slate-500 tracking-wider pt-1">
          Recent Notifications
        </span>
        <div class="space-y-2">
          {#each $items as item (item.id)}
            {@const isError = item.type === 'error'}
            {@const isSuccess = item.type === 'success'}
            {@const isWarning = item.type === 'warning'}
            {@const borderClass = isError ? 'border-rose-900/50 bg-rose-950/20' :
                                  isSuccess ? 'border-emerald-900/50 bg-emerald-950/20' :
                                  isWarning ? 'border-amber-900/50 bg-amber-950/20' :
                                  'border-[#22222e] bg-[#121218]'}
            <div class="p-3.5 rounded-xl border {borderClass} space-y-1 transition-all text-xs">
              <div class="flex items-center justify-between">
                <div class="flex items-center space-x-2 min-w-0">
                  <span class="w-1.5 h-1.5 rounded-full {isError ? 'bg-rose-400' : isSuccess ? 'bg-emerald-400' : isWarning ? 'bg-amber-400' : 'bg-slate-400'}"></span>
                  <strong class="font-bold text-white truncate">{item.title}</strong>
                  {#if item.gameName}
                    <span class="text-[10px] font-mono bg-black/40 text-slate-400 px-1.5 py-0.5 rounded">
                      {item.gameName}
                    </span>
                  {/if}
                </div>
                <span class="text-[10px] text-slate-500 font-mono flex-shrink-0">
                  {formatTimeAgo(item.timestamp)}
                </span>
              </div>
              <p class="text-slate-300 pl-3.5 leading-relaxed">{item.message}</p>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Empty State -->
    {#if !$activeTask && $queue.length === 0 && $items.length === 0}
      <div class="p-12 text-center text-slate-500 space-y-1 font-mono text-xs">
        <p class="font-bold text-slate-400">No active tasks or notifications</p>
        <p class="text-[11px] text-slate-600">Downloads, applies, and alerts will appear in this unified stream.</p>
      </div>
    {/if}
  </div>
</div>
