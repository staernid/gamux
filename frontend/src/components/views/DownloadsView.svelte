<script>
  import { onMount, untrack } from 'svelte';
  import { libraryPath, games, selectedGame, activeView } from '../../stores/library.js';
  import { tasksStore, isDownloadTask } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';
  import { isSamePath } from '../../utils/game.js';
  import TaskCard from './tasks/TaskCard.svelte';
  import TaskQueueSection from './tasks/TaskQueueSection.svelte';
  import TaskHistorySection from './tasks/TaskHistorySection.svelte';

  let diskSpace = $state(null);
  let expandedDepots = $state({});
  let speedHistory = $state([0]);
  let showLimitMenu = $state(false);

  const SPEED_LIMIT_OPTIONS = [
    { label: 'Unlimited', value: 0 },
    { label: '50 MB/s', value: 50 * 1024 * 1024 },
    { label: '25 MB/s', value: 25 * 1024 * 1024 },
    { label: '10 MB/s', value: 10 * 1024 * 1024 },
    { label: '5 MB/s', value: 5 * 1024 * 1024 },
    { label: '2 MB/s', value: 2 * 1024 * 1024 },
  ];

  async function loadDiskSpace(path) {
    if (!path) return;
    try {
      diskSpace = await AppAPI.getDiskSpace(path);
    } catch {
      // Ignored
    }
  }

  $effect(() => {
    const p = $libraryPath;
    if (p) {
      untrack(() => loadDiskSpace(p));
    }
  });

  $effect(() => {
    const currentSpeed = tasksStore.totalSpeedBps || 0;
    untrack(() => {
      if (speedHistory.length > 25) {
        speedHistory = [...speedHistory.slice(1), currentSpeed];
      } else {
        speedHistory = [...speedHistory, currentSpeed];
      }
    });
  });

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

  function handleOpenGame(e) {
    const task = e.detail || e;
    if (!task) return;
    const allGames = $games || [];
    const game = allGames.find(g => (task.gameDir && isSamePath(g.gameDir, task.gameDir)) || (task.appId && String(g.appId || g.app_id) === String(task.appId)));
    if (game) {
      selectedGame.set(game);
      activeView.set('game-hub');
    } else if (task.gameDir) {
      AppAPI.openDirectory(task.gameDir);
    }
  }

  function toggleDepots(e) {
    const id = typeof e === 'string' ? e : e.detail;
    expandedDepots[id] = !expandedDepots[id];
  }

  function handleSetLimit(bps) {
    tasksStore.setLimit(bps);
    showLimitMenu = false;
  }

  function handlePauseAll() {
    tasksStore.pauseAll();
  }

  function handleResumeAll() {
    tasksStore.resumeAll();
  }

  // Strictly downloads only
  let displayedActive = $derived(tasksStore.activeDownloads);
  let displayedQueued = $derived([...tasksStore.queuedDownloads, ...tasksStore.pausedDownloads]);
  let displayedCompleted = $derived(tasksStore.completedDownloads);
</script>

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-8 max-w-4xl mx-auto w-full space-y-5 text-xs sm:text-sm animate-fade-in custom-scrollbar">
  <!-- Minimalist HUD Header -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-surface-border">
    <div class="flex items-center space-x-3">
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2.5">
        Downloads
        <span class="text-2xs font-mono font-medium px-2 py-0.5 rounded-full bg-surface-header text-neutral-300 border border-surface-border">
          {tasksStore.activeDownloads.length} active
        </span>
      </h2>
    </div>

    <!-- Right Controls: Grouped into Interactive Tools and Telemetry HUD -->
    <div class="flex items-center space-x-2 flex-wrap gap-y-2 w-full sm:w-auto justify-end">
      <!-- Interactive Action: Master Pause/Resume All Button -->
      {#if tasksStore.activeDownloads.length > 0}
        <button
          type="button"
          onclick={handlePauseAll}
          class="px-2.5 py-1 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-2xs font-mono font-bold text-amber-400 hover:text-amber-300 transition cursor-pointer flex items-center space-x-1"
          title="Pause all active downloads"
        >
          <svg class="w-2.5 h-2.5 fill-current" viewBox="0 0 24 24">
            <rect x="6" y="5" width="4" height="14" rx="1" />
            <rect x="14" y="5" width="4" height="14" rx="1" />
          </svg>
          <span>Pause All</span>
        </button>
      {:else if tasksStore.pausedDownloads.length > 0}
        <button
          type="button"
          onclick={handleResumeAll}
          class="px-2.5 py-1 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-2xs font-mono font-bold text-emerald-400 hover:text-emerald-300 transition cursor-pointer flex items-center space-x-1"
          title="Resume all paused downloads"
        >
          <svg class="w-2.5 h-2.5 fill-current" viewBox="0 0 24 24">
            <path d="M8 5v14l11-7z" />
          </svg>
          <span>Resume All</span>
        </button>
      {/if}

      <!-- Interactive Action: Speed Limit Dropdown -->
      <div class="relative">
        <button
          type="button"
          onclick={() => showLimitMenu = !showLimitMenu}
          class="flex items-center space-x-1 px-2.5 py-1 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-2xs font-mono text-neutral-300 hover:text-white transition cursor-pointer"
          title="Bandwidth Limit"
        >
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          <span>{tasksStore.speedLimit > 0 ? `${formatBytes(tasksStore.speedLimit)}/s` : 'Max'}</span>
          <svg class="w-2.5 h-2.5 text-neutral-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {#if showLimitMenu}
          <button type="button" class="fixed inset-0 z-40 cursor-default bg-transparent border-0 w-full h-full" aria-label="Close bandwidth menu" onclick={() => showLimitMenu = false}></button>
          <div class="absolute right-0 mt-1 w-36 bg-surface-header border border-surface-border rounded-xl shadow-2xl py-1 z-50 text-xs font-mono animate-fade-in">
            <div class="px-2.5 py-1 text-4xs uppercase tracking-wider text-neutral-500 font-bold border-b border-surface-borderSubtle">
              Bandwidth Cap
            </div>
            {#each SPEED_LIMIT_OPTIONS as opt}
              <button
                type="button"
                onclick={() => handleSetLimit(opt.value)}
                class="w-full text-left px-2.5 py-1 text-2xs flex items-center justify-between hover:bg-surface-cardHover transition cursor-pointer {tasksStore.speedLimit === opt.value ? 'text-emerald-400 font-bold' : 'text-neutral-300'}"
              >
                <span>{opt.label}</span>
                {#if tasksStore.speedLimit === opt.value}
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Subtle Visual Divider -->
      <div class="h-4 w-px bg-surface-border hidden sm:block mx-1"></div>

      <!-- Telemetry HUD: Drive Space Pill -->
      {#if diskSpace}
        <div
          class="flex items-center space-x-1.5 px-2.5 py-1 rounded-xl bg-surface-card border border-surface-border text-2xs font-mono text-neutral-300"
          title="Drive {diskSpace.path}: {formatBytes(diskSpace.free_bytes)} free of {formatBytes(diskSpace.total_bytes)}"
        >
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 flex-shrink-0"></span>
          <span class="font-bold text-neutral-200">{formatBytes(diskSpace.free_bytes)}</span>
          <span class="text-neutral-500">Free</span>
        </div>
      {/if}

      <!-- Telemetry HUD: Bandwidth & Sparkline -->
      <div class="flex items-center space-x-2 px-2.5 py-1 rounded-xl bg-surface-card border border-surface-border min-w-[70px]">
        <span class="font-mono text-xs font-bold text-neutral-200">
          {tasksStore.totalSpeedBps > 0 ? `${formatBytes(tasksStore.totalSpeedBps)}/s` : '0 B/s'}
        </span>
        {#if tasksStore.totalSpeedBps > 0 && speedHistory.length > 2}
          <svg class="w-12 h-3.5 text-neutral-400 flex-shrink-0" viewBox="0 0 100 22" aria-hidden="true">
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
    </div>
  </div>

  <!-- Downloads Content Area -->
  <div class="space-y-4">
    {#if displayedActive.length === 0 && displayedQueued.length === 0 && displayedCompleted.length === 0}
      <!-- Living Empty State -->
      <div class="bg-surface-card border border-surface-border rounded-2xl p-12 text-center text-neutral-400 space-y-4">
        <div class="w-12 h-12 mx-auto rounded-2xl bg-surface-input border border-surface-border flex items-center justify-center text-neutral-400">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        </div>
        <div class="space-y-1">
          <p class="font-bold text-white text-sm">No downloads active</p>
          <p class="text-xs text-neutral-400 max-w-sm mx-auto leading-relaxed">
            Active downloads, depot transfers, and install queues will appear here.
          </p>
        </div>
        <div class="pt-2 flex items-center justify-center space-x-3">
          <button
            type="button"
            onclick={() => modalStore.open('catalog')}
            class="px-4 py-2 rounded-xl bg-white text-neutral-950 font-bold hover:bg-neutral-200 transition shadow cursor-pointer text-xs"
          >
            Browse Catalog
          </button>
          <button
            type="button"
            onclick={() => activeView.set('library')}
            class="px-4 py-2 rounded-xl bg-surface-header hover:bg-surface-cardHover text-neutral-200 border border-surface-border font-bold transition cursor-pointer text-xs"
          >
            Go to Shelf
          </button>
        </div>
      </div>
    {:else}
      <!-- Active Downloads (Hero) -->
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

      <!-- Queued & Paused Downloads -->
      {#if displayedQueued.length > 0}
        <TaskQueueSection
          tasks={displayedQueued}
          on:resume={(e) => tasksStore.resume(e.detail)}
          on:pause={(e) => tasksStore.pause(e.detail)}
          on:cancel={(e) => tasksStore.cancel(e.detail)}
          on:reorder={(e) => tasksStore.reorder(e.detail)}
        />
      {/if}

      <!-- Completed Downloads History -->
      {#if displayedCompleted.length > 0}
        <TaskHistorySection
          tasks={displayedCompleted}
          on:openGame={handleOpenGame}
          on:openDirectory={handleOpenDirectory}
          on:dismiss={(e) => tasksStore.remove(e.detail)}
          on:clearAll={() => tasksStore.clearFinished()}
        />
      {/if}
    {/if}
  </div>
</div>
