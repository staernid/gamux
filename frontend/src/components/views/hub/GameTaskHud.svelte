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

<div class="bg-surface-header border border-surface-border rounded-2xl p-4 shadow-xl space-y-2.5 animate-fade-in">
  <div class="flex items-center justify-between gap-3">
    <div class="flex items-center space-x-2.5 min-w-0">
      <span class="px-2 py-0.5 rounded-lg text-3xs font-black uppercase tracking-wider bg-neutral-800 text-neutral-200 border border-neutral-700 flex items-center space-x-1.5 flex-shrink-0">
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
        <span>{getTaskBadgeLabel(task)}</span>
      </span>
      <h4 class="font-bold text-white text-xs truncate">
        {task.stage || task.title || 'Task executing'}
      </h4>
      {#if task.details}
        <span class="text-neutral-500 text-xs hidden sm:inline">·</span>
        <span class="text-neutral-400 text-xs font-mono truncate hidden sm:inline max-w-sm">
          {task.details}
        </span>
      {/if}
    </div>

    <!-- Task Controls -->
    <div class="flex items-center space-x-2 flex-shrink-0">
      <button
        type="button"
        on:click={openTasksView}
        class="px-2.5 py-1 rounded-xl bg-surface-card hover:bg-surface-cardHover text-neutral-300 hover:text-white text-2xs font-bold border border-surface-border transition flex items-center space-x-1 cursor-pointer"
        title="Open full Tasks view (2)"
      >
        <span>View Tasks</span>
        <span class="text-neutral-500 font-mono text-3xs px-1 py-0.5 rounded bg-black/40">2</span>
      </button>
      <button
        type="button"
        on:click={handleCancel}
        class="px-2.5 py-1 rounded-xl bg-rose-950/30 hover:bg-rose-900/50 text-rose-300 hover:text-white text-2xs font-bold border border-rose-900/40 transition flex items-center space-x-1 cursor-pointer active:scale-95"
        title="Cancel active task"
      >
        Cancel
      </button>
    </div>
  </div>

  <!-- Progress Bar -->
  <div class="space-y-1">
    <div class="w-full bg-surface-input border border-surface-border h-2 rounded-full overflow-hidden">
      <div
        class="bg-white h-full rounded-full transition-all duration-300 relative"
        style="width: {task.percent || 0}%"
      >
        <div class="absolute inset-0 bg-neutral-200/30 animate-pulse"></div>
      </div>
    </div>
    <div class="flex items-center justify-between text-3xs font-mono text-neutral-400">
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
          <span class="text-neutral-200 font-bold">{formatBytes(task.speedBps)}/s</span>
        {/if}
        {#if task.etaSeconds > 0}
          <span>ETA: {formatETA(task.etaSeconds)}</span>
        {/if}
      </div>
    </div>
  </div>
</div>
