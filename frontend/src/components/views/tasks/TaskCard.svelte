<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { formatBytes, formatETA } from '../../../utils/format.js';
  import { isDownloadTask } from '../../../stores/tasks.js';
  import PosterArt from '../../common/PosterArt.svelte';

  export let task;
  export let expandedDepots = {};
  export let onpause = null;
  export let oncancel = null;
  export let ontoggleDepots = null;

  const dispatch = createEventDispatcher();

  $: isDl = isDownloadTask(task);
  $: isIndeterminate = !isDl && (!task.percent || task.percent === 0);

  let confirmingCancel = false;
  let cancelTimeout = null;

  function getTaskTypeLabel(type) {
    switch (type) {
      case 'download': return 'DOWNLOAD';
      case 'apply': return 'SETUP & DRM';
      case 'verify': return 'INTEGRITY';
      case 'update': return 'UPDATE';
      case 'tool_update': return 'TOOL UPDATE';
      case 'normalize': return 'NORMALIZE';
      case 'rollback': return 'ROLLBACK';
      case 'workshop': return 'WORKSHOP';
      default: return 'TASK';
    }
  }

  function toggleDepots() {
    if (typeof ontoggleDepots === 'function') ontoggleDepots(task.id);
    dispatch('toggleDepots', task.id);
  }

  function handlePause() {
    if (typeof onpause === 'function') onpause(task.id);
    dispatch('pause', task.id);
  }

  function handleCancel() {
    if (!confirmingCancel) {
      confirmingCancel = true;
      clearTimeout(cancelTimeout);
      cancelTimeout = setTimeout(() => {
        confirmingCancel = false;
      }, 3000);
      return;
    }
    clearTimeout(cancelTimeout);
    confirmingCancel = false;
    if (typeof oncancel === 'function') oncancel(task.id);
    dispatch('cancel', task.id);
  }

  onDestroy(() => {
    if (cancelTimeout) {
      clearTimeout(cancelTimeout);
    }
  });
</script>

<div class="bg-surface-card border border-surface-border rounded-2xl p-4 shadow-lg space-y-3 hover:border-surface-borderHighlight transition">
  <div class="flex items-start justify-between gap-3">
    <div class="flex items-center space-x-3.5 min-w-0">
      <!-- Poster Thumbnail -->
      <div class="w-11 h-15 rounded-xl overflow-hidden bg-surface-base border border-surface-border flex-shrink-0">
        <PosterArt
          appId={task.appId}
          name={task.gameName}
          posterUrl={task.posterUrl}
          className="w-full h-full object-cover"
        />
      </div>

      <div class="min-w-0 space-y-1">
        <div class="flex items-center space-x-2 flex-wrap gap-y-1">
          <span class="px-1.5 py-0.5 text-4xs font-bold rounded bg-surface-pill text-neutral-300 uppercase font-mono border border-surface-border">
            {task.store || 'steam'}
          </span>
          <span class="px-1.5 py-0.5 text-4xs font-bold rounded bg-surface-input text-neutral-400 uppercase font-mono border border-surface-borderSubtle">
            {getTaskTypeLabel(task.type)}
          </span>
          <h4
            class="font-bold text-white text-sm truncate"
            title={task.gameName || task.title}
          >
            {task.gameName || task.title}
          </h4>
        </div>
        <p
          class="text-2xs text-neutral-400 font-mono truncate max-w-lg"
          title={task.details || task.stage || 'In progress...'}
        >
          {task.details || task.stage || 'In progress...'}
        </p>
      </div>
    </div>

    <!-- Controls -->
    <div class="flex items-center space-x-2 flex-shrink-0">
      {#if isDl}
        <button
          type="button"
          on:click={handlePause}
          class="bg-surface-elevated hover:bg-surface-cardHover border border-surface-border text-amber-400 hover:text-amber-300 font-bold px-3 py-1.5 rounded-xl text-xs transition cursor-pointer"
          aria-label="Pause {task.gameName || task.title || 'task'}"
        >
          Pause
        </button>
      {/if}
      <button
        type="button"
        on:click={handleCancel}
        class="font-bold px-3 py-1.5 rounded-xl text-xs transition cursor-pointer {confirmingCancel
          ? 'bg-rose-600 hover:bg-rose-500 text-white border border-rose-500 shadow-md animate-pulse'
          : 'bg-surface-elevated hover:bg-rose-950/40 border border-surface-border hover:border-rose-900/50 text-neutral-400 hover:text-rose-400'}"
        aria-label={confirmingCancel ? `Confirm cancellation for ${task.gameName || task.title || 'task'}` : `Cancel ${task.gameName || task.title || 'task'}`}
      >
        {confirmingCancel ? 'Confirm Cancel?' : 'Cancel'}
      </button>
    </div>
  </div>

  <!-- Progress Bar -->
  <div class="space-y-1.5">
    <div
      class="w-full bg-surface-input border border-surface-border h-2 rounded-full overflow-hidden"
      role="progressbar"
      aria-valuenow={isIndeterminate ? undefined : (task.percent || 0)}
      aria-valuemin="0"
      aria-valuemax="100"
      aria-label="{task.gameName || task.title || 'Task'} progress"
    >
      {#if isIndeterminate}
        <div class="h-full w-full bg-neutral-600/40 animate-pulse rounded-full"></div>
      {:else}
        <div
          class="bg-white h-full rounded-full transition-all duration-300 relative"
          style="width: {Math.min(100, Math.max(0, task.percent || 0))}%"
        >
          <div class="absolute inset-0 bg-neutral-200/20 animate-pulse"></div>
        </div>
      {/if}
    </div>

    <!-- Metadata line -->
    <div class="flex items-center justify-between text-2xs font-mono text-neutral-400">
      <span>
        {#if task.totalBytes && task.totalBytes > 0}
          {formatBytes(task.currentBytes || 0)} / {formatBytes(task.totalBytes)} ({task.percent?.toFixed(0) || 0}%)
        {:else if task.totalItems && task.totalItems > 0}
          {task.currentItems || 0} / {task.totalItems} items ({task.percent?.toFixed(0) || 0}%)
        {:else if !isIndeterminate}
          {task.percent?.toFixed(0) || 0}%
        {:else}
          Processing...
        {/if}
      </span>
      {#if task.speedBps > 0}
        <span class="text-neutral-200 font-bold">{formatBytes(task.speedBps)}/s</span>
      {/if}
      {#if task.etaSeconds > 0}
        <span>ETA: {formatETA(task.etaSeconds)}</span>
      {/if}
    </div>
  </div>

  <!-- Multi-Depot Chunk Breakdown (if available) -->
  {#if task.depots && task.depots.length > 0}
    <div class="pt-2 border-t border-surface-borderSubtle">
      <button
        type="button"
        on:click={toggleDepots}
        class="flex items-center justify-between w-full text-xs font-bold text-neutral-400 hover:text-neutral-200 transition py-0.5 cursor-pointer"
        aria-expanded={expandedDepots[task.id] ? "true" : "false"}
        aria-controls="depots-{task.id}"
      >
        <span class="flex items-center space-x-2 font-mono text-2xs">
          <span>Depots & Chunk Breakdown ({task.depots.length})</span>
          {#if task.totalItems}
            <span class="text-3xs text-neutral-300 bg-surface-pill px-1.5 py-0.5 rounded border border-surface-border">
              {task.currentItems?.toLocaleString()} / {task.totalItems?.toLocaleString()} chunks
            </span>
          {/if}
        </span>
        <span class="text-3xs text-neutral-500 font-bold flex items-center space-x-1">
          <span>{expandedDepots[task.id] ? 'Hide' : 'Details'}</span>
          <svg
            class="w-3 h-3 transform transition-transform {expandedDepots[task.id] ? 'rotate-180' : ''}"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </span>
      </button>

      {#if expandedDepots[task.id]}
        <div
          id="depots-{task.id}"
          class="mt-2 space-y-2.5 bg-surface-input p-3 rounded-xl border border-surface-borderSubtle max-h-60 overflow-y-auto custom-scrollbar animate-fade-in"
        >
          {#each task.depots as depot (depot.id)}
            <div class="space-y-1 text-xs">
              <div class="flex items-center justify-between text-2xs">
                <div class="flex items-center space-x-1.5 truncate max-w-xs">
                  <span class="font-mono text-neutral-400 font-bold">#{depot.id}</span>
                  <span class="text-neutral-200 font-medium truncate" title={depot.name}>{depot.name}</span>
                </div>
                <div class="flex items-center space-x-2 font-mono text-3xs">
                  <span class="text-neutral-400">{formatBytes(depot.downloaded_bytes || depot.downloadedBytes || 0)} / {formatBytes(depot.total_bytes || depot.totalBytes || 0)}</span>
                  <span class="font-bold {depot.percent >= 100 ? 'text-emerald-400' : 'text-neutral-200'}">
                    {depot.percent?.toFixed(0)}%
                  </span>
                </div>
              </div>

              <div class="w-full bg-surface-base h-1.5 rounded-full overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-300 {depot.percent >= 100 ? 'bg-emerald-400' : 'bg-white'}"
                  style="width: {Math.min(100, Math.max(0, depot.percent || 0))}%"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
