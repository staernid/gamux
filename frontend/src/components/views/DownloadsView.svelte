<script>
  import { onMount } from 'svelte';
  import { libraryPath } from '../../stores/library.js';
  import { tasksStore, isDownloadTask } from '../../stores/tasks.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes, formatETA } from '../../utils/format.js';
  import PosterArt from '../common/PosterArt.svelte';

  let searchQuery = '';
  let selectedID = '';
  let selectedName = '';
  let selectedStore = 'steam';
  let selectedPlatform = 'win64';
  let targetDir = '';
  let autoApply = true;
  let isSearching = false;
  let searchResults = [];
  let showAddPanel = false;
  let isSubmitting = false;

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
    activeTransfersCount,
    speedLimit,
  } = tasksStore;

  $: if (!targetDir && $libraryPath) {
    targetDir = $libraryPath;
  }

  async function loadDiskSpace(path) {
    if (!path) return;
    try {
      diskSpace = await AppAPI.getDiskSpace(path);
    } catch {
      // Ignored
    }
  }

  $: if (targetDir || $libraryPath) {
    loadDiskSpace(targetDir || $libraryPath);
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

  async function handleSearch() {
    const q = searchQuery.trim();
    if (!q) return;

    if (!isNaN(q) && parseInt(q, 10) > 0) {
      selectedID = q;
      selectedName = `AppID ${q}`;
      selectedStore = 'steam';
      searchResults = [];
      return;
    }

    isSearching = true;
    try {
      const candidates = await AppAPI.searchStoreGames(q);
      searchResults = (candidates || []).filter(c => !c.store || c.store.toLowerCase() === 'steam');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Search Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSearching = false;
    }
  }

  function selectCandidate(candidate) {
    selectedID = String(candidate.id || candidate.AppID);
    selectedName = candidate.name || candidate.Name;
    selectedStore = 'steam';
    searchQuery = selectedName;
    searchResults = [];
    const safeName = selectedName.replace(/[™®©]/g, '').replace(/[:/\\?*<>|"]/g, '').replace(/\s+/g, ' ').trim() || selectedName;
    const isMusic = candidate.type === 'music' || candidate.type === 'soundtrack';
    if (isMusic) {
      targetDir = `~/Music/${safeName}`;
    } else if ($libraryPath) {
      targetDir = `${$libraryPath}/${safeName}`;
    } else {
      targetDir = `./${safeName}`;
    }
  }


  function clearSelection() {
    selectedID = '';
    selectedName = '';
    searchQuery = '';
    searchResults = [];
  }

  async function browseTargetDir() {
    const selected = await AppAPI.selectDirectory('Select Download Destination Folder', targetDir || $libraryPath);
    if (selected) {
      targetDir = selected;
    }
  }

  async function startDownload() {
    if (isSubmitting) return;

    const appID = parseInt(selectedID.trim(), 10);
    const dest = targetDir.trim();

    if (!appID || isNaN(appID)) {
      notificationStore.items.add({
        type: 'error',
        title: 'Identifier Required',
        message: 'Please enter or select a valid Steam AppID or Game Title.'
      });
      return;
    }

    if (!dest) {
      notificationStore.items.add({
        type: 'error',
        title: 'Destination Required',
        message: 'Please select a destination folder for downloading.'
      });
      return;
    }

    try {
      isSubmitting = true;
      await tasksStore.start({
        app_id: appID,
        game_name: selectedName || `Game ${appID}`,
        target_dir: dest,
        platform: selectedPlatform,
        auto_apply: true,
        store_type: selectedStore || 'steam',
      });

      showAddPanel = false;
      clearSelection();
    } catch {
      // Handled in store
    } finally {
      isSubmitting = false;
    }
  }


  function handleSpeedLimitChange(e) {
    const limit = parseInt(e.target.value, 10);
    tasksStore.setLimit(limit);
  }

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

  function handleOpenDirectory(dir) {
    if (dir) AppAPI.openDirectory(dir);
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
        <span class="text-[11px] font-mono font-medium px-2 py-0.5 rounded-full bg-[#181824] text-slate-400 border border-[#272738]">
          {$active.length} active
        </span>
      </h2>
    </div>

    <!-- Right Controls: Bandwidth HUD + Speed Cap + New Download CTA -->
    <div class="flex items-center space-x-2.5 w-full sm:w-auto justify-between sm:justify-end">
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

      <div class="flex items-center space-x-1.5 bg-[#12121a] border border-[#222232] rounded-xl px-2.5 py-1 text-xs">
        <span class="text-[10px] text-slate-500 font-mono uppercase font-bold">Cap:</span>
        <select
          value={$speedLimit}
          on:change={handleSpeedLimitChange}
          class="bg-transparent text-slate-300 text-xs outline-none font-mono cursor-pointer"
        >
          <option value="0" class="bg-[#12121a] text-slate-200">Unlimited</option>
          <option value="5242880" class="bg-[#12121a] text-slate-200">5 MB/s</option>
          <option value="10485760" class="bg-[#12121a] text-slate-200">10 MB/s</option>
          <option value="26214400" class="bg-[#12121a] text-slate-200">25 MB/s</option>
          <option value="52428800" class="bg-[#12121a] text-slate-200">50 MB/s</option>
        </select>
      </div>

      <button
        type="button"
        on:click={() => showAddPanel = !showAddPanel}
        class="bg-[#1e1e2c] hover:bg-[#2a2a3c] text-white font-bold px-3 py-1.5 rounded-xl border border-[#323246] transition text-xs flex items-center space-x-1.5 shadow-sm"
      >
        <span class="text-slate-400 font-bold">+</span>
        <span>{showAddPanel ? 'Close' : 'New Download'}</span>
      </button>
    </div>
  </div>

  <!-- Sleek Minimalist Target Drive Capacity Bar -->
  {#if diskSpace}
    {@const usedPct = diskSpace.total_bytes ? Math.min(100, Math.max(0, (diskSpace.used_bytes / diskSpace.total_bytes) * 100)) : 0}
    <div class="px-3.5 py-2 rounded-xl bg-[#0e0e14] border border-[#1b1b26] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 text-xs">
      <div class="flex items-center space-x-2 text-slate-400 truncate max-w-sm sm:max-w-md">
        <span class="text-slate-500 font-mono text-[10px] uppercase font-bold tracking-wider">Target Drive Capacity:</span>
        <span class="font-mono text-[11px] text-slate-300 truncate font-medium">{diskSpace.path}</span>
      </div>
      <div class="flex items-center space-x-3 w-full sm:w-auto justify-between sm:justify-end">
        <div class="w-28 bg-[#161620] h-1.5 rounded-full overflow-hidden">
          <div class="bg-cyan-500/80 h-full rounded-full transition-all duration-500" style="width: {usedPct}%"></div>
        </div>
        <span class="font-mono text-[11px] text-slate-400 whitespace-nowrap">
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
      class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 {currentFilter === 'all' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
    >
      <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
      <span>All ({$items.length})</span>
    </button>
    <button
      type="button"
      on:click={() => currentFilter = 'downloads'}
      class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 {currentFilter === 'downloads' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
    >
      <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
      </svg>
      <span>Transfers ({$activeDownloads.length + $queuedDownloads.length + $pausedDownloads.length})</span>
    </button>
    <button
      type="button"
      on:click={() => currentFilter = 'operations'}
      class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 {currentFilter === 'operations' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
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
        class="px-3 py-1.5 rounded-xl font-bold transition flex items-center space-x-1.5 {currentFilter === 'completed' ? 'bg-[#222232] text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141e]'}"
      >
        <svg class="w-3.5 h-3.5 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>Completed ({$completed.length})</span>
      </button>
    {/if}
  </div>

  <!-- Collapsible Add Download Panel -->
  {#if showAddPanel}
    <div class="bg-[#121218] border border-[#262638] rounded-2xl p-5 shadow-2xl space-y-4 animate-fade-in">
      <div class="flex items-center justify-between border-b border-[#1e1e2c] pb-2.5">
        <h3 class="font-bold text-white text-xs uppercase tracking-wider">Download Title from Store Provider</h3>
        <button on:click={() => showAddPanel = false} class="text-slate-400 hover:text-white text-xs">Close</button>
      </div>

      <!-- Search Field -->
      <div class="relative">
        <label for="dlSearchInput" class="block text-slate-400 mb-1 font-bold text-[10px] uppercase tracking-wider font-mono">
          Search Game Title or Steam AppID
        </label>
        <div class="flex space-x-2">
          <input
            id="dlSearchInput"
            type="text"
            bind:value={searchQuery}
            placeholder="e.g. Hades, Balatro, Cyberpunk 2077..."
            on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
            class="flex-1 bg-[#09090d] border border-[#242434] rounded-xl px-3.5 py-2 text-slate-100 outline-none font-mono focus:border-cyan-400 text-xs"
          />
          <button
            on:click={handleSearch}
            disabled={isSearching || !searchQuery.trim()}
            class="bg-[#1c1c28] hover:bg-[#28283a] border border-[#2f2f42] text-slate-200 font-bold px-4 py-2 rounded-xl transition text-xs flex items-center space-x-1.5"
          >
            {#if isSearching}
              <span>Searching...</span>
            {:else}
              <span>Search</span>
            {/if}
          </button>
        </div>

        <!-- Search Candidates Dropdown -->
        {#if searchResults.length > 0}
          <div class="absolute left-0 right-0 top-full mt-2 bg-[#14141e] border border-[#2a2a3c] rounded-xl shadow-2xl z-30 overflow-hidden divide-y divide-[#1e1e2c] max-h-60 overflow-y-auto">
            {#each searchResults as candidate}
              <button
                type="button"
                on:click={() => selectCandidate(candidate)}
                class="w-full text-left p-2.5 hover:bg-[#1e1e2c] flex items-center space-x-3 transition group"
              >
                <div class="w-8 h-11 rounded bg-black/40 overflow-hidden flex-shrink-0">
                  {#if candidate.poster_url}
                    <img src={candidate.poster_url} alt="" class="w-full h-full object-cover" />
                  {/if}
                </div>
                <div class="flex-1 min-w-0">
                  <div class="font-bold text-slate-100 group-hover:text-cyan-400 truncate text-xs">{candidate.name}</div>
                  <div class="text-[10px] text-slate-500 font-mono mt-0.5">
                    <span class="px-1 py-0.2 rounded bg-black/40 text-slate-400 uppercase">{candidate.store || 'steam'}</span>
                    ID: {candidate.id}
                  </div>
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Config Details Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 pt-1">
        <div>
          <label for="dlDestFolderInput" class="block text-slate-400 mb-1 font-bold text-[10px] uppercase tracking-wider font-mono">Destination Folder</label>
          <div class="flex space-x-1.5">
            <input
              id="dlDestFolderInput"
              type="text"
              bind:value={targetDir}
              class="flex-1 bg-[#09090d] border border-[#242434] rounded-xl px-3 py-1.5 text-slate-200 outline-none font-mono text-xs"
            />
            <button
              on:click={browseTargetDir}
              class="bg-[#1c1c28] hover:bg-[#28283a] border border-[#2e2e40] text-slate-300 px-3 py-1.5 rounded-xl text-xs font-bold"
            >
              Browse
            </button>
          </div>
        </div>

        <div>
          <label for="dlPlatformSelect" class="block text-slate-400 mb-1 font-bold text-[10px] uppercase tracking-wider font-mono">Platform Target</label>
          <select
            id="dlPlatformSelect"
            bind:value={selectedPlatform}
            class="w-full bg-[#09090d] border border-[#242434] rounded-xl px-3 py-2 text-slate-200 outline-none text-xs font-mono"
          >
            <option value="win64">Windows (64-bit)</option>
            <option value="linux">Linux Native</option>
            <option value="macos">macOS Native</option>
            <option value="all">All Available Depots</option>
          </select>
        </div>
      </div>

      <div class="flex justify-end space-x-2.5 pt-2 border-t border-[#1e1e2c]">
        <button
          on:click={clearSelection}
          class="px-3.5 py-1.5 rounded-xl bg-transparent hover:bg-slate-800 text-slate-400 hover:text-white font-bold text-xs"
        >
          Reset
        </button>
        <button
          on:click={startDownload}
          disabled={!selectedID}
          class="bg-white hover:bg-neutral-200 disabled:opacity-40 text-black font-black px-5 py-1.5 rounded-xl transition text-xs shadow-sm"
        >
          Start Download
        </button>
      </div>
    </div>
  {/if}

  <!-- Active Tasks Section -->
  <div class="space-y-3">
    {#if displayedActive.length === 0 && displayedQueued.length === 0 && displayedCompleted.length === 0}
      <div class="bg-[#101016] border border-[#1b1b26] rounded-2xl p-10 text-center text-slate-500 space-y-1 font-mono text-xs">
        <p class="font-bold text-slate-400 text-sm">No active tasks running.</p>
        <p class="text-[11px] text-slate-600">Background downloads, file verifications, and setup operations will appear here.</p>
      </div>
    {:else}
      {#if displayedActive.length > 0}
        <div class="space-y-2.5">
          {#each displayedActive as tr (tr.id)}
            {@const isDl = isDownloadTask(tr)}
            <div class="bg-[#111118] border border-[#242436] rounded-2xl p-4 shadow-lg space-y-3 hover:border-[#2f2f48] transition">
              <div class="flex items-start justify-between gap-3">
                <div class="flex items-center space-x-3.5 min-w-0">
                  <!-- Poster Thumbnail -->
                  <div class="w-11 h-15 rounded-xl overflow-hidden bg-black/40 border border-[#222232] flex-shrink-0">
                    <PosterArt
                      appId={tr.app_id || tr.appID || tr.game_id}
                      name={tr.game_name || tr.title}
                      posterUrl={tr.poster_url || tr.posterUrl}
                      className="w-full h-full object-cover"
                    />
                  </div>

                  <div class="min-w-0 space-y-0.5">
                    <div class="flex items-center space-x-2 flex-wrap">
                      <span class="px-1.5 py-0.2 text-[9px] font-bold rounded bg-[#1c1c2a] text-cyan-300 uppercase font-mono border border-[#28283c]">
                        {tr.store || 'steam'}
                      </span>
                      <span class="px-1.5 py-0.2 text-[9px] font-bold rounded bg-[#181822] text-slate-400 uppercase font-mono border border-[#242434]">
                        {getTaskTypeLabel(tr.type)}
                      </span>
                      <h4 class="font-bold text-white text-sm truncate">{tr.game_name || tr.title || tr.game_id}</h4>
                    </div>
                    <p class="text-[11px] text-slate-400 font-mono truncate max-w-md">
                      {tr.details || tr.stage || 'In progress...'}
                    </p>
                  </div>
                </div>

                <!-- Controls -->
                <div class="flex items-center space-x-2 flex-shrink-0">
                  {#if isDl}
                    <button
                      type="button"
                      on:click={() => tasksStore.pause(tr.id)}
                      class="bg-[#181824] hover:bg-[#242436] border border-[#28283a] text-amber-400 font-bold px-3 py-1.5 rounded-xl text-xs transition"
                    >
                      Pause
                    </button>
                  {/if}
                  <button
                    type="button"
                    on:click={() => tasksStore.cancel(tr.id)}
                    class="bg-[#181824] hover:bg-rose-950/40 border border-[#28283a] hover:border-rose-900/50 text-slate-400 hover:text-rose-400 font-bold px-3 py-1.5 rounded-xl text-xs transition"
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
                    style="width: {tr.percent || 0}%"
                  ></div>
                </div>

                <!-- Metadata line -->
                <div class="flex items-center justify-between text-[11px] font-mono text-slate-400">
                  <span>
                    {#if tr.total_bytes && tr.total_bytes > 0}
                      {formatBytes(tr.bytes_downloaded || tr.current_bytes || 0)} / {formatBytes(tr.total_bytes)} ({tr.percent?.toFixed(0) || 0}%)
                    {:else if tr.total_items && tr.total_items > 0}
                      {tr.current_items || 0} / {tr.total_items} items ({tr.percent?.toFixed(0) || 0}%)
                    {:else}
                      {tr.percent?.toFixed(0) || 0}%
                    {/if}
                  </span>
                  {#if tr.speed_bps > 0}
                    <span class="text-cyan-400 font-bold">{formatBytes(tr.speed_bps)}/s</span>
                  {/if}
                  {#if tr.eta_seconds > 0}
                    <span>ETA: {formatETA(tr.eta_seconds)}</span>
                  {/if}
                </div>
              </div>

              <!-- Multi-Depot Chunk Breakdown (if available) -->
              {#if tr.depots && tr.depots.length > 0}
                <div class="pt-2 border-t border-[#1a1a26]">
                  <button
                    type="button"
                    on:click={() => expandedDepots[tr.id] = !expandedDepots[tr.id]}
                    class="flex items-center justify-between w-full text-xs font-bold text-slate-400 hover:text-slate-200 transition py-0.5"
                  >
                    <span class="flex items-center space-x-2 font-mono text-[11px]">
                      <span>Depots & Chunk Breakdown ({tr.depots.length})</span>
                      {#if tr.total_items}
                        <span class="text-[10px] text-cyan-400 bg-cyan-950/40 px-1.5 py-0.2 rounded border border-cyan-900/40">
                          {tr.current_items?.toLocaleString()} / {tr.total_items?.toLocaleString()} chunks
                        </span>
                      {/if}
                    </span>
                    <span class="text-[10px] text-slate-500 font-bold flex items-center space-x-1">
                      <span>{expandedDepots[tr.id] ? 'Hide' : 'Details'}</span>
                      <svg class="w-3 h-3 transform transition-transform {expandedDepots[tr.id] ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </span>
                  </button>

                  {#if expandedDepots[tr.id]}
                    <div class="mt-2 space-y-2 bg-[#09090e] p-3 rounded-xl border border-[#1b1b26] animate-fade-in">
                      {#each tr.depots as depot}
                        <div class="space-y-1 text-xs">
                          <div class="flex items-center justify-between text-[11px]">
                            <div class="flex items-center space-x-1.5 truncate max-w-xs">
                              <span class="font-mono text-slate-400 font-bold">#{depot.id}</span>
                              <span class="text-slate-200 font-medium truncate">{depot.name}</span>
                            </div>
                            <div class="flex items-center space-x-2 font-mono text-[10px]">
                              <span class="text-slate-400">{formatBytes(depot.downloaded_bytes)} / {formatBytes(depot.total_bytes)}</span>
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
          {/each}
        </div>
      {/if}

      <!-- Queued & Paused Tasks -->
      {#if displayedQueued.length > 0}
        <div class="space-y-2.5 pt-2">
          <h3 class="font-bold text-slate-400 text-xs tracking-wider uppercase font-mono">
            Queue & Paused ({displayedQueued.length})
          </h3>

          <div class="bg-[#111118] border border-[#20202c] rounded-2xl divide-y divide-[#1b1b26] overflow-hidden">
            {#each displayedQueued as tr (tr.id)}
              <div class="p-3.5 flex items-center justify-between hover:bg-[#151520] transition">
                <div class="flex items-center space-x-3 min-w-0">
                  <span class="px-1.5 py-0.5 text-[9px] font-bold rounded bg-[#1c1c28] text-slate-400 uppercase font-mono border border-[#262638]">
                    {tr.status}
                  </span>
                  <div class="min-w-0">
                    <span class="font-bold text-white text-xs truncate block">{tr.game_name || tr.title || tr.game_id}</span>
                    <p class="text-[10px] text-slate-500 font-mono mt-0.5 truncate">{tr.details || tr.stage || 'Waiting'}</p>
                  </div>
                </div>

                <div class="flex items-center space-x-2 flex-shrink-0">
                  {#if tr.status === 'paused'}
                    <button
                      type="button"
                      on:click={() => tasksStore.resume(tr.id)}
                      class="bg-[#181824] hover:bg-[#242436] text-cyan-400 border border-[#28283a] text-xs px-3 py-1 rounded-xl font-bold transition"
                    >
                      Resume
                    </button>
                  {:else if isDownloadTask(tr)}
                    <button
                      type="button"
                      on:click={() => tasksStore.pause(tr.id)}
                      class="bg-[#181824] hover:bg-[#242436] text-amber-400 border border-[#28283a] text-xs px-3 py-1 rounded-xl font-bold transition"
                    >
                      Pause
                    </button>
                  {/if}
                  <button
                    type="button"
                    on:click={() => tasksStore.cancel(tr.id)}
                    class="text-slate-500 hover:text-rose-400 text-xs px-2 py-1 font-bold transition"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Completed History -->
      {#if displayedCompleted.length > 0}
        <div class="space-y-2.5 pt-2">
          <div class="flex items-center justify-between">
            <h3 class="font-bold text-slate-400 text-xs tracking-wider uppercase font-mono">
              Completed History ({displayedCompleted.length})
            </h3>
            <button
              type="button"
              on:click={() => tasksStore.clearFinished()}
              class="text-[11px] font-bold text-slate-500 hover:text-slate-300 transition"
            >
              Clear All
            </button>
          </div>

          <div class="bg-[#111118] border border-[#20202c] rounded-2xl divide-y divide-[#1b1b26] overflow-hidden">
            {#each displayedCompleted as tr (tr.id)}
              <div class="p-3.5 flex items-center justify-between hover:bg-[#151520] transition gap-3">
                <div class="flex items-center space-x-3 min-w-0">
                  <span class="w-2 h-2 rounded-full flex-shrink-0 {tr.status === 'completed' ? 'bg-emerald-400' : 'bg-rose-500'}"></span>
                  <div class="min-w-0">
                    <span class="font-bold text-white text-xs truncate block">{tr.game_name || tr.title || tr.game_id}</span>
                    <p class="text-[10px] text-slate-500 font-mono mt-0.5 truncate">{tr.details || tr.status}</p>
                  </div>
                </div>

                <div class="flex items-center space-x-2 flex-shrink-0">
                  {#if tr.game_dir}
                    <button
                      type="button"
                      on:click={() => handleOpenDirectory(tr.game_dir)}
                      class="text-slate-400 hover:text-white text-xs px-2.5 py-1 rounded-lg bg-[#181824] border border-[#272738] font-medium"
                    >
                      Folder
                    </button>
                  {/if}
                  <button
                    type="button"
                    on:click={() => tasksStore.remove(tr.id)}
                    class="text-slate-500 hover:text-slate-300 text-xs px-2 py-1"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>
