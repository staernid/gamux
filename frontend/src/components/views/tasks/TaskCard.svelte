<script>
  import { createEventDispatcher } from 'svelte';
  import { formatBytes, formatETA } from '../../../utils/format.js';
  import { isDownloadTask } from '../../../stores/tasks.js';
  import PosterArt from '../../common/PosterArt.svelte';

  export let task;
  export let expandedDepots = {};

  const dispatch = createEventDispatcher();

  $: isDl = isDownloadTask(task);

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
    dispatch('toggleDepots', task.id);
  }

  function handlePause() {
    dispatch('pause', task.id);
  }

  function handleCancel() {
    dispatch('cancel', task.id);
  }
</script>

<div class="bg-[#111118] border border-[#242436] rounded-2xl p-4 shadow-lg space-y-3 hover:border-[#2f2f48] transition">
  <div class="flex items-start justify-between gap-3">
    <div class="flex items-center space-x-3.5 min-w-0">
      <!-- Poster Thumbnail -->
      <div class="w-11 h-15 rounded-xl overflow-hidden bg-black/40 border border-[#222232] flex-shrink-0">
        <PosterArt
          appId={task.appId}
          name={task.gameName}
          posterUrl={task.posterUrl}
          className="w-full h-full object-cover"
        />
      </div>

      <div class="min-w-0 space-y-0.5">
        <div class="flex items-center space-x-2 flex-wrap">
          <span class="px-1.5 py-0.2 text-[9px] font-bold rounded bg-[#1c1c2a] text-cyan-300 uppercase font-mono border border-[#28283c]">
            {task.store || 'steam'}
          </span>
          <span class="px-1.5 py-0.2 text-[9px] font-bold rounded bg-[#181822] text-slate-400 uppercase font-mono border border-[#242434]">
            {getTaskTypeLabel(task.type)}
          </span>
          <h4 class="font-bold text-white text-sm truncate">{task.gameName || task.title}</h4>
        </div>
        <p class="text-[11px] text-slate-400 font-mono truncate max-w-md">
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
          class="bg-[#181824] hover:bg-[#242436] border border-[#28283a] text-amber-400 font-bold px-3 py-1.5 rounded-xl text-xs transition cursor-pointer"
        >
          Pause
        </button>
      {/if}
      <button
        type="button"
        on:click={handleCancel}
        class="bg-[#181824] hover:bg-rose-950/40 border border-[#28283a] hover:border-rose-900/50 text-slate-400 hover:text-rose-400 font-bold px-3 py-1.5 rounded-xl text-xs transition cursor-pointer"
      >
        Cancel
      </button>
    </div>
  </div>

  <!-- Progress Bar -->
  <div class="space-y-1.5">
    <div class="w-full bg-[#0a0a0f] border border-[#1c1c28] h-2 rounded-full overflow-hidden">
      <div
        class="bg-cyan-400 h-full rounded-full transition-all duration-300"
        style="width: {task.percent || 0}%"
      ></div>
    </div>

    <!-- Metadata line -->
    <div class="flex items-center justify-between text-[11px] font-mono text-slate-400">
      <span>
        {#if task.totalBytes && task.totalBytes > 0}
          {formatBytes(task.currentBytes || 0)} / {formatBytes(task.totalBytes)} ({task.percent?.toFixed(0) || 0}%)
        {:else if task.totalItems && task.totalItems > 0}
          {task.currentItems || 0} / {task.totalItems} items ({task.percent?.toFixed(0) || 0}%)
        {:else}
          {task.percent?.toFixed(0) || 0}%
        {/if}
      </span>
      {#if task.speedBps > 0}
        <span class="text-cyan-400 font-bold">{formatBytes(task.speedBps)}/s</span>
      {/if}
      {#if task.etaSeconds > 0}
        <span>ETA: {formatETA(task.etaSeconds)}</span>
      {/if}
    </div>
  </div>

  <!-- Multi-Depot Chunk Breakdown (if available) -->
  {#if task.depots && task.depots.length > 0}
    <div class="pt-2 border-t border-[#1a1a26]">
      <button
        type="button"
        on:click={toggleDepots}
        class="flex items-center justify-between w-full text-xs font-bold text-slate-400 hover:text-slate-200 transition py-0.5 cursor-pointer"
      >
        <span class="flex items-center space-x-2 font-mono text-[11px]">
          <span>Depots & Chunk Breakdown ({task.depots.length})</span>
          {#if task.totalItems}
            <span class="text-[10px] text-cyan-400 bg-cyan-950/40 px-1.5 py-0.2 rounded border border-cyan-900/40">
              {task.currentItems?.toLocaleString()} / {task.totalItems?.toLocaleString()} chunks
            </span>
          {/if}
        </span>
        <span class="text-[10px] text-slate-500 font-bold flex items-center space-x-1">
          <span>{expandedDepots[task.id] ? 'Hide' : 'Details'}</span>
          <svg class="w-3 h-3 transform transition-transform {expandedDepots[task.id] ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </span>
      </button>

      {#if expandedDepots[task.id]}
        <div class="mt-2 space-y-2 bg-[#09090e] p-3 rounded-xl border border-[#1b1b26] animate-fade-in">
          {#each task.depots as depot}
            <div class="space-y-1 text-xs">
              <div class="flex items-center justify-between text-[11px]">
                <div class="flex items-center space-x-1.5 truncate max-w-xs">
                  <span class="font-mono text-slate-400 font-bold">#{depot.id}</span>
                  <span class="text-slate-200 font-medium truncate">{depot.name}</span>
                </div>
                <div class="flex items-center space-x-2 font-mono text-[10px]">
                  <span class="text-slate-400">{formatBytes(depot.downloaded_bytes || depot.downloadedBytes || 0)} / {formatBytes(depot.total_bytes || depot.totalBytes || 0)}</span>
                  <span class="font-bold {depot.percent >= 100 ? 'text-emerald-400' : 'text-cyan-400'}">
                    {depot.percent?.toFixed(0)}%
                  </span>
                </div>
              </div>

              <div class="w-full bg-[#12121a] h-1.5 rounded-full overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-300 {depot.percent >= 100 ? 'bg-emerald-400' : 'bg-cyan-500'}"
                  style="width: {depot.percent}%"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
