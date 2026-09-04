<script>
  import { onMount } from 'svelte';
  import { libraryPath } from '../../stores/library.js';
  import { tasksStore, isDownloadTask } from '../../stores/tasks.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';
  import TaskCard from './tasks/TaskCard.svelte';
  import TaskQueueSection from './tasks/TaskQueueSection.svelte';
  import TaskHistorySection from './tasks/TaskHistorySection.svelte';

  let currentFilter = 'all'; // 'all' | 'downloads' | 'operations' | 'completed'

  let diskSpace = null;
  let expandedDepots = {};
  let speedHistory = [0];

  const {
    items,
    active,
    queued,
    paused,
    completed,
    activeDownloads,
    queuedDownloads,
    pausedDownloads,
    completedDownloads,
    activeOperations,
    queuedOperations,
    totalSpeedBps,
  } = tasksStore;

  async function loadDiskSpace(path) {
    if (!path) return;
    try {
      diskSpace = await AppAPI.getDiskSpace(path);
    } catch {
      // Ignored
    }
  }

  $: if ($libraryPath) {
    loadDiskSpace($libraryPath);
  }

  $: {
    const currentSpeed = $totalSpeedBps || 0;
    if (speedHistory.length > 25) {
      speedHistory = [...speedHistory.slice(1), currentSpeed];
    } else {
      speedHistory = [...speedHistory, currentSpeed];
    }
  }

  function generateSparklinePoints(data, width = 100, height = 22) {
    if (!data || data.length === 0) return '';
    const max = Math.max(...data, 1048576); // min 1 MB/s scale
    return data.map((val, idx) => {
      const x = (idx / Math.max(data.length - 1, 1)) * width;
      const y = height - (val / max) * (height - 4) - 2;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
  }

  onMount(() => {
    tasksStore.refresh();
    if ($libraryPath) {
      loadDiskSpace($libraryPath);
    }
  });

  function handleOpenDirectory(e) {
    const dir = typeof e === 'string' ? e : e.detail;
    if (dir) AppAPI.openDirectory(dir);
  }

  function toggleDepots(e) {
    const id = typeof e === 'string' ? e : e.detail;
    expandedDepots[id] = !expandedDepots[id];
  }

  // Filtered lists based on currentFilter
  $: displayedActive = (() => {
    if (currentFilter === 'downloads') return $activeDownloads;
    if (currentFilter === 'operations') return $activeOperations;
    if (currentFilter === 'completed') return [];
    return $active;
  })();

  $: displayedQueued = (() => {
    if (currentFilter === 'downloads') return [...$queuedDownloads, ...$pausedDownloads];
    if (currentFilter === 'operations') return $queuedOperations;
    if (currentFilter === 'completed') return [];
    return [...$queued, ...$paused];
  })();

  $: displayedCompleted = (() => {
    if (currentFilter === 'downloads') return $completedDownloads;
    if (currentFilter === 'operations') return $completed.filter(t => !isDownloadTask(t));
    return $completed;
  })();
</script>

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-8 max-w-4xl mx-auto w-full space-y-5 text-xs sm:text-sm animate-fade-in custom-scrollbar">
  <!-- Minimalist HUD Header -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-[#1c1c28]">
    <div>
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2.5">
        Tasks
        <span class="text-2xs font-mono font-medium px-2 py-0.5 rounded-full bg-[#181824] text-slate-400 border border-[#272738]">
          {$active.length} active
        </span>
      </h2>
    </div>

    <!-- Right Controls: Bandwidth HUD -->
    <div class="flex items-center space-x-2.5 w-full sm:w-auto justify-end">
      {#if $totalSpeedBps > 0}
        <div class="flex items-center space-x-2 px-2.5 py-1 rounded-xl bg-[#12121a] border border-[#222232]">
          <span class="font-mono text-xs font-bold text-cyan-400">{formatBytes($totalSpeedBps)}/s</span>
          {#if speedHistory.length > 2}
            <svg class="w-12 h-3.5 text-cyan-400/80" viewBox="0 0 100 22">
              <polyline
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                points={generateSparklinePoints(speedHistory, 100, 22)}
              />
            </svg>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  <!-- Sleek Minimalist Target Drive Capacity Bar -->
  {#if diskSpace}
    {@const usedPct = diskSpace.total_bytes ? Math.min(100, Math.max(0, (diskSpace.used_bytes / diskSpace.total_bytes) * 100)) : 0}
    <div class="px-3.5 py-2 rounded-xl bg-[#0e0e14] border border-[#1b1b26] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 text-xs">
      <div class="flex items-center space-x-2 text-slate-400 truncate max-w-sm sm:max-w-md">
        <span class="text-slate-500 font-mono text-3xs uppercase font-bold tracking-wider">Target Drive Capacity:</span>
        <span class="font-mono text-2xs text-slate-300 truncate font-medium">{diskSpace.path}</span>
      </div>
      <div class="flex items-center space-x-3 w-full sm:w-auto justify-between sm:justify-end">
        <div class="w-28 bg-[#161620] h-1.5 rounded-full overflow-hidden">
          <div class="bg-cyan-500/80 h-full rounded-full transition-all duration-500" style="width: {usedPct}%"></div>
        </div>
        <span class="font-mono text-2xs text-slate-400 whitespace-nowrap">
          <strong class="text-emerald-400 font-semibold">{formatBytes(diskSpace.free_bytes)} Free</strong>
          <span class="text-slate-600"> / {formatBytes(diskSpace.total_bytes)} Total</span>
        </span>
      </div>
    </div>
  {/if}

  <!-- Segmented Filter Pills -->
  <div class="flex items-center space-x-1.5 overflow-x-auto pb-1 text-xs">
    <button
      type="button"
      on:click={() => currentFilter = 'all'}
      class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 cursor-pointer {currentFilter === 'all' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
    >
      <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
      <span>All ({$items.length})</span>
    </button>
    <button
      type="button"
      on:click={() => currentFilter = 'downloads'}
      class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 cursor-pointer {currentFilter === 'downloads' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
    >
      <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
      </svg>
      <span>Transfers ({$activeDownloads.length + $queuedDownloads.length + $pausedDownloads.length})</span>
    </button>
    <button
      type="button"
      on:click={() => currentFilter = 'operations'}
      class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 cursor-pointer {currentFilter === 'operations' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
    >
      <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Operations ({$activeOperations.length + $queuedOperations.length})</span>
    </button>
    {#if $completed.length > 0}
      <button
        type="button"
        on:click={() => currentFilter = 'completed'}
        class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 cursor-pointer {currentFilter === 'completed' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
      >
        <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>Completed ({$completed.length})</span>
      </button>
    {/if}
  </div>

  <!-- Active Tasks Section -->
  <div class="space-y-3">
    {#if displayedActive.length === 0 && displayedQueued.length === 0 && displayedCompleted.length === 0}
      <div class="bg-[#101016] border border-[#1b1b26] rounded-2xl p-10 text-center text-slate-500 space-y-1 font-mono text-xs">
        <p class="font-bold text-slate-400 text-sm">No active tasks running.</p>
        <p class="text-2xs text-slate-600">Background downloads, file verifications, and setup operations will appear here.</p>
      </div>
    {:else}
      {#if displayedActive.length > 0}
        <div class="space-y-2.5">
          {#each displayedActive as tr (tr.id)}
            <TaskCard
              task={tr}
              {expandedDepots}
              on:pause={(e) => tasksStore.pause(e.detail)}
              on:cancel={(e) => tasksStore.cancel(e.detail)}
              on:toggleDepots={toggleDepots}
            />
          {/each}
        </div>
      {/if}

      <!-- Queued & Paused Tasks -->
      {#if displayedQueued.length > 0}
        <TaskQueueSection
          tasks={displayedQueued}
          on:resume={(e) => tasksStore.resume(e.detail)}
          on:pause={(e) => tasksStore.pause(e.detail)}
          on:cancel={(e) => tasksStore.cancel(e.detail)}
        />
      {/if}

      <!-- Completed History -->
      {#if displayedCompleted.length > 0}
        <TaskHistorySection
          tasks={displayedCompleted}
          on:openDirectory={handleOpenDirectory}
          on:dismiss={(e) => tasksStore.remove(e.detail)}
          on:clearAll={() => tasksStore.clearFinished()}
        />
      {/if}
    {/if}
  </div>
</div>
