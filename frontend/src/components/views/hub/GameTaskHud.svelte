<script>
  import { createEventDispatcher } from 'svelte';
  import { activeView } from '../../../stores/library.js';
  import { tasksStore, isDownloadTask } from '../../../stores/tasks.js';
  import { formatBytes, formatETA } from '../../../utils/format.js';

  export let task;

  const dispatch = createEventDispatcher();

  function getTaskBadgeLabel(t) {
    if (!t) return 'TASK';
    const op = (t.operation || t.type || '').toLowerCase();
    if (isDownloadTask(t)) return 'DOWNLOADING';
    if (op === 'apply') return 'APPLYING SETUP';
    if (op === 'verify') return 'VERIFYING';
    if (op === 'normalize') return 'NORMALIZING';
    if (op === 'update') return 'UPDATING';
    if (op === 'tool_update') return 'TOOL UPDATE';
    if (op === 'rollback') return 'ROLLBACK';
    return 'PROCESSING';
  }

  function handleCancel() {
    tasksStore.cancel(task.id);
  }

  function openTasksView() {
    $activeView = 'tasks';
  }
</script>

<div class="bg-gradient-to-r from-[#12121e] via-[#101018] to-[#12121e] border border-cyan-500/40 rounded-2xl p-4 shadow-xl space-y-2.5 animate-fade-in">
  <div class="flex items-center justify-between gap-3">
    <div class="flex items-center space-x-2.5 min-w-0">
      <span class="px-2 py-0.5 rounded-lg text-[10px] font-black uppercase tracking-wider bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 flex items-center space-x-1.5 flex-shrink-0">
        <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse"></span>
        <span>{getTaskBadgeLabel(task)}</span>
      </span>
      <h4 class="font-bold text-white text-xs truncate">
        {task.stage || task.title || 'Task executing'}
      </h4>
      {#if task.details}
        <span class="text-slate-500 text-xs hidden sm:inline">·</span>
        <span class="text-slate-400 text-xs font-mono truncate hidden sm:inline max-w-sm">
          {task.details}
        </span>
      {/if}
    </div>

    <!-- Task Controls -->
    <div class="flex items-center space-x-2 flex-shrink-0">
      <button
        type="button"
        on:click={openTasksView}
        class="px-2.5 py-1 rounded-xl bg-[#1c1c28] hover:bg-[#252538] text-slate-300 hover:text-white text-[11px] font-bold border border-[#2e2e42] transition flex items-center space-x-1 cursor-pointer"
        title="Open full Tasks view (2)"
      >
        <span>View Tasks</span>
        <span class="text-slate-500 font-mono text-[9px] px-1 py-0.2 rounded bg-black/40">2</span>
      </button>
      <button
        type="button"
        on:click={handleCancel}
        class="px-2.5 py-1 rounded-xl bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 hover:text-white text-[11px] font-bold border border-rose-900/40 hover:border-rose-700/60 transition flex items-center space-x-1 cursor-pointer active:scale-95"
        title="Cancel active task"
      >
        Cancel
      </button>
    </div>
  </div>

  <!-- Progress Bar -->
  <div class="space-y-1">
    <div class="w-full bg-[#0a0a0f] border border-[#1e1e2c] h-2 rounded-full overflow-hidden">
      <div
        class="bg-gradient-to-r from-cyan-500 to-blue-500 h-full rounded-full transition-all duration-300 relative"
        style="width: {task.percent || 0}%"
      >
        <div class="absolute inset-0 bg-white/20 animate-pulse"></div>
      </div>
    </div>
    <div class="flex items-center justify-between text-[10px] font-mono text-slate-400">
      <span>
        {#if task.totalBytes && task.totalBytes > 0}
          {formatBytes(task.currentBytes || 0)} / {formatBytes(task.totalBytes)} ({task.percent?.toFixed(0) || 0}%)
        {:else if task.totalItems && task.totalItems > 0}
          Step {task.currentItems || 0} of {task.totalItems} ({task.percent?.toFixed(0) || 0}%)
        {:else}
          {task.percent?.toFixed(0) || 0}%
        {/if}
      </span>
      <div class="flex items-center space-x-3">
        {#if task.speedBps > 0}
          <span class="text-cyan-400 font-bold">{formatBytes(task.speedBps)}/s</span>
        {/if}
        {#if task.etaSeconds > 0}
          <span>ETA: {formatETA(task.etaSeconds)}</span>
        {/if}
      </div>
    </div>
  </div>
</div>
