<script>
  import { createEventDispatcher } from 'svelte';
  import { activeView } from '../../../stores/library.js';
  import { tasksStore, isDownloadTask } from '../../../stores/tasks.js';
  import { formatBytes, formatETA } from '../../../utils/format.js';

  export let task;

  const dispatch = createEventDispatcher();

  $: isPaused = task?.status === 'paused';
  $: isPausable = Boolean(task && (isDownloadTask(task) || ((task.capabilities || 0) & 2) !== 0 || isPaused));

  function getTaskBadgeLabel(t) {
    if (!t) return 'TASK';
    if (t.status === 'paused') return 'PAUSED';
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

  function handleTogglePause() {
    if (!task?.id) return;
    if (isPaused) {
      tasksStore.resume(task.id);
    } else {
      tasksStore.pause(task.id);
    }
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
      <span class="px-2 py-0.5 rounded-lg text-3xs font-black uppercase tracking-wider {isPaused ? 'bg-amber-950/60 text-amber-300 border border-amber-800/60' : 'bg-neutral-800 text-neutral-200 border border-neutral-700'} flex items-center space-x-1.5 flex-shrink-0">
        <span class="w-1.5 h-1.5 rounded-full {isPaused ? 'bg-amber-400' : 'bg-emerald-400 animate-pulse'}"></span>
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
      {#if isPausable}
        <button
          type="button"
          on:click={handleTogglePause}
          class="px-2.5 py-1 rounded-xl text-2xs font-bold border transition flex items-center space-x-1 cursor-pointer active:scale-95 {isPaused ? 'bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border-amber-500/40' : 'bg-surface-card hover:bg-surface-cardHover text-neutral-300 hover:text-white border-surface-border'}"
          title={isPaused ? "Resume task" : "Pause task"}
        >
          {#if isPaused}
            <svg class="w-3 h-3 fill-current" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
            </svg>
            <span>Resume</span>
          {:else}
            <svg class="w-3 h-3 fill-current" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
            <span>Pause</span>
          {/if}
        </button>
      {/if}
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
        class="{isPaused ? 'bg-amber-400' : 'bg-white'} h-full rounded-full transition-all duration-300 relative"
        style="width: {task.percent || 0}%"
      >
        {#if !isPaused}
          <div class="absolute inset-0 bg-neutral-200/30 animate-pulse"></div>
        {/if}
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
        {#if isPaused}
          <span class="text-amber-400 font-semibold">Transfer Paused</span>
        {:else}
          {#if task.speedBps > 0}
            <span class="text-neutral-200 font-bold">{formatBytes(task.speedBps)}/s</span>
          {/if}
          {#if task.etaSeconds > 0}
            <span>ETA: {formatETA(task.etaSeconds)}</span>
          {/if}
        {/if}
      </div>
    </div>
  </div>
</div>
