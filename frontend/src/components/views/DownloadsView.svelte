<script>
  import { onMount } from 'svelte';
  import { libraryPath, scanLibrary, refreshGame } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes } from '../../utils/format.js';

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

  const {
    activeDownloads,
    queuedDownloads,
    pausedDownloads,
    completedDownloads,
    totalSpeedBps,
    activeTransfersCount,
    speedLimit,
  } = tasksStore;

  $: if (!targetDir && $libraryPath) {
    targetDir = $libraryPath;
  }

  onMount(() => {
    tasksStore.refresh();
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
      searchResults = candidates || [];
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
    selectedStore = candidate.store || 'steam';
    searchQuery = selectedName;
    searchResults = [];
    if ($libraryPath) {
      const safeName = selectedName.replace(/[^\w\s-]/g, '').trim();
      targetDir = `${$libraryPath}/${safeName}`;
    }
  }

  function handleSelectCandidate(c) {
    selectCandidate(c);
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
        auto_apply: autoApply,
        store_type: selectedStore,
      });

      showAddPanel = false;
      clearSelection();
    } catch (err) {
      // Handled in store
    } finally {
      isSubmitting = false;
    }
  }

  function formatETA(seconds) {
    if (!seconds || seconds <= 0) return '--';
    if (seconds < 60) return `${seconds}s`;
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    if (mins < 60) return `${mins}m ${secs}s`;
    const hrs = Math.floor(mins / 60);
    return `${hrs}h ${mins % 60}m`;
  }

  function handleSpeedLimitChange(e) {
    const limit = parseInt(e.target.value, 10);
    tasksStore.setLimit(limit);
  }
</script>

<div class="flex-1 overflow-y-auto p-4 sm:p-8 max-w-5xl mx-auto w-full space-y-6 text-xs sm:text-sm animate-fade-in">
  <!-- Top Downloads Navigation & Telemetry Hub -->
  <div class="bg-[#121218] border border-[#242434] rounded-2xl p-5 shadow-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <div class="flex items-center space-x-4">
      <div class="w-12 h-12 rounded-xl bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
        <svg class="w-6 h-6 animate-pulse" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
      </div>
      <div>
        <h2 class="text-lg font-black text-white flex items-center gap-2">
          Downloads
          <span class="text-[10px] px-2 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 font-mono">
            {$activeTransfersCount} active
          </span>
        </h2>
        <p class="text-xs text-slate-400">
          Total Bandwidth: <span class="font-mono text-cyan-400 font-bold">{formatBytes($totalSpeedBps)}/s</span>
        </p>
      </div>
    </div>

    <!-- Controls: Speed Limiter & Add Download Trigger -->
    <div class="flex items-center space-x-3 w-full sm:w-auto justify-between sm:justify-end">
      <div class="flex items-center space-x-2">
        <span class="text-[11px] text-slate-400 uppercase font-bold tracking-wider">Speed Cap:</span>
        <select
          value={$speedLimit}
          on:change={handleSpeedLimitChange}
          class="bg-[#0c0c10] border border-[#2b2b3d] text-slate-200 text-xs rounded-xl px-2.5 py-1.5 outline-none font-mono focus:border-cyan-500"
        >
          <option value="0">Unlimited</option>
          <option value="5242880">5 MB/s</option>
          <option value="10485760">10 MB/s</option>
          <option value="26214400">25 MB/s</option>
          <option value="52428800">50 MB/s</option>
        </select>
      </div>

      <button
        on:click={() => showAddPanel = !showAddPanel}
        class="bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold px-3.5 py-2 rounded-xl flex items-center space-x-1.5 transition text-xs shadow-lg shadow-cyan-500/10"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>New Download</span>
      </button>
    </div>
  </div>

  <!-- Collapsible Add Download Panel -->
  {#if showAddPanel}
    <div class="bg-[#121218] border border-[#2b2b3d] rounded-2xl p-6 shadow-2xl space-y-4 animate-fade-in">
      <div class="flex items-center justify-between border-b border-[#20202c] pb-3">
        <h3 class="font-bold text-white text-sm">Download Title from Store Provider</h3>
        <button on:click={() => showAddPanel = false} class="text-slate-400 hover:text-white text-xs">Close</button>
      </div>

      <!-- Search Field -->
      <div class="relative">
        <label for="dlSearchInput" class="block text-slate-300 mb-1 font-bold text-[11px] uppercase tracking-wider">
          Search Game Title or Steam AppID
        </label>
        <div class="flex space-x-2">
          <input
            id="dlSearchInput"
            type="text"
            bind:value={searchQuery}
            placeholder="e.g. Hades, Balatro, Cyberpunk 2077..."
            on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
            class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3.5 py-2.5 text-slate-100 outline-none font-mono focus:border-cyan-400 text-xs"
          />
          <button
            on:click={handleSearch}
            disabled={isSearching || !searchQuery.trim()}
            class="bg-[#20202e] hover:bg-[#2c2c3e] border border-[#34344a] text-slate-200 font-bold px-4 py-2 rounded-xl transition text-xs flex items-center space-x-1.5"
          >
            {#if isSearching}
              <svg class="animate-spin h-3.5 w-3.5 text-slate-300" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
              </svg>
              <span>Searching...</span>
            {:else}
              <span>Search</span>
            {/if}
          </button>
        </div>

        <!-- Search Candidates Dropdown -->
        {#if searchResults.length > 0}
          <div class="absolute left-0 right-0 top-full mt-2 bg-[#171722] border border-[#303046] rounded-xl shadow-2xl z-30 overflow-hidden divide-y divide-[#20202e]">
            {#each searchResults as candidate}
              <button
                type="button"
                on:click={() => selectCandidate(candidate)}
                class="w-full text-left p-3 hover:bg-[#202030] flex items-center space-x-3 transition group"
              >
                {#if candidate.poster_url}
                  <img src={candidate.poster_url} alt="" class="w-10 h-14 object-cover rounded bg-black/40" />
                {/if}
                <div class="flex-1 min-w-0">
                  <div class="font-bold text-slate-100 group-hover:text-cyan-400 truncate text-xs">{candidate.name}</div>
                  <div class="text-[10px] text-slate-400 font-mono mt-0.5">
                    <span class="px-1.5 py-0.5 rounded bg-black/50 text-slate-300 uppercase">{candidate.store || 'steam'}</span>
                    ID: {candidate.id}
                  </div>
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Config Details Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-2">
        <div>
          <label for="dlDestFolderInput" class="block text-slate-300 mb-1 font-bold text-[11px] uppercase tracking-wider">Destination Folder</label>
          <div class="flex space-x-2">
            <input
              id="dlDestFolderInput"
              type="text"
              bind:value={targetDir}
              class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-200 outline-none font-mono text-xs"
            />
            <button
              on:click={browseTargetDir}
              class="bg-[#1e1e2c] hover:bg-[#2a2a3e] border border-[#303046] text-slate-300 px-3 py-2 rounded-xl text-xs font-bold"
            >
              Browse
            </button>
          </div>
        </div>

        <div>
          <label for="dlPlatformSelect" class="block text-slate-300 mb-1 font-bold text-[11px] uppercase tracking-wider">Platform Target</label>
          <select
            id="dlPlatformSelect"
            bind:value={selectedPlatform}
            class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-200 outline-none text-xs"
          >
            <option value="win64">Windows (64-bit)</option>
            <option value="linux">Linux Native</option>
            <option value="all">All Available Depots</option>
          </select>
        </div>

        <div>
          <label for="dlAutoApplyCheckbox" class="block text-slate-300 mb-1 font-bold text-[11px] uppercase tracking-wider">Post-Setup</label>
          <label class="flex items-center space-x-2 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 cursor-pointer">
            <input id="dlAutoApplyCheckbox" type="checkbox" bind:checked={autoApply} class="rounded accent-cyan-400" />
            <span class="text-xs text-slate-300">Auto-Apply DRM / Setup</span>
          </label>
        </div>
      </div>

      <div class="flex justify-end space-x-3 pt-3 border-t border-[#20202c]">
        <button
          on:click={clearSelection}
          class="px-4 py-2 rounded-xl bg-transparent hover:bg-slate-800 text-slate-400 hover:text-white font-bold text-xs"
        >
          Reset
        </button>
        <button
          on:click={startDownload}
          disabled={!selectedID}
          class="bg-cyan-500 hover:bg-cyan-400 disabled:opacity-40 text-slate-950 font-bold px-6 py-2 rounded-xl transition text-xs shadow-lg shadow-cyan-500/10"
        >
          Start Download
        </button>
      </div>
    </div>
  {/if}

  <!-- Active Downloads Section -->
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h3 class="font-black text-slate-200 text-xs tracking-wider uppercase flex items-center gap-2">
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
        Active Downloads ({$activeDownloads.length})
      </h3>
    </div>

    {#if $activeDownloads.length === 0}
      <div class="bg-[#121218]/50 border border-[#20202c] rounded-2xl p-8 text-center text-slate-500">
        <svg class="w-8 h-8 mx-auto mb-2 opacity-40" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
        </svg>
        <p class="text-xs">No active downloads running.</p>
      </div>
    {:else}
      <div class="space-y-3">
        {#each $activeDownloads as tr (tr.id)}
          <div class="bg-[#121218] border border-cyan-500/30 rounded-2xl p-5 shadow-xl space-y-3">
            <div class="flex items-start justify-between">
              <div>
                <div class="flex items-center space-x-2">
                  <span class="px-1.5 py-0.5 text-[9px] font-bold rounded bg-cyan-500/20 text-cyan-300 uppercase font-mono">
                    {tr.store || 'steam'}
                  </span>
                  <h4 class="font-black text-white text-sm">{tr.game_name || tr.title || tr.game_id}</h4>
                </div>
                <p class="text-[11px] text-slate-400 font-mono mt-0.5 truncate max-w-md">{tr.target_dir}</p>
              </div>

              <!-- Controls -->
              <div class="flex items-center space-x-2">
                <button
                  type="button"
                  on:click={() => tasksStore.pause(tr.id)}
                  class="bg-[#1e1e2c] hover:bg-[#2c2c3e] border border-[#303044] text-amber-400 font-bold px-3 py-1.5 rounded-lg text-xs transition"
                >
                  Pause
                </button>
                <button
                  type="button"
                  on:click={() => tasksStore.cancel(tr.id)}
                  class="bg-[#1e1e2c] hover:bg-rose-950/40 border border-[#303044] hover:border-rose-800 text-rose-400 font-bold px-3 py-1.5 rounded-lg text-xs transition"
                >
                  Cancel
                </button>
              </div>
            </div>

            <!-- Progress Bar -->
            <div class="space-y-1.5">
              <div class="w-full bg-[#0c0c10] border border-[#20202e] h-2.5 rounded-full overflow-hidden">
                <div
                  class="bg-gradient-to-r from-cyan-500 to-emerald-400 h-full rounded-full transition-all duration-300"
                  style="width: {tr.percent || 0}%"
                ></div>
              </div>

              <!-- Metadata info line -->
              <div class="mt-2.5 flex items-center justify-between text-xs text-stone-400">
                  <span>{formatBytes(tr.bytes_downloaded)} / {formatBytes(tr.total_bytes)} ({tr.percent?.toFixed(0) || 0}%)</span>
                  {#if tr.speed_bps > 0}
                    <span class="text-cyan-400 font-bold">{formatBytes(tr.speed_bps)}/s</span>
                  {/if}
                  {#if tr.eta_seconds > 0}
                    <span>ETA: {formatETA(tr.eta_seconds)}</span>
                  {/if}
                </div>
              </div>
            </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Queued & Paused Transfers -->
  {#if $queuedDownloads.length > 0 || $pausedDownloads.length > 0}
    <div class="space-y-3 pt-2">
      <h3 class="font-black text-slate-400 text-xs tracking-wider uppercase">
        Queue & Paused ({$queuedDownloads.length + $pausedDownloads.length})
      </h3>

      <div class="bg-[#121218] border border-[#20202c] rounded-2xl divide-y divide-[#1b1b26] overflow-hidden">
        {#each [...$queuedDownloads, ...$pausedDownloads] as tr (tr.id)}
          <div class="p-4 flex items-center justify-between hover:bg-[#151520] transition">
            <div>
              <div class="flex items-center space-x-2">
                <span class="px-1.5 py-0.5 text-[9px] font-bold rounded bg-slate-800 text-slate-300 uppercase font-mono">
                  {tr.status}
                </span>
                <span class="font-bold text-white text-xs">{tr.game_name || tr.title || tr.game_id}</span>
              </div>
              <p class="text-[10px] text-slate-500 font-mono mt-0.5">{tr.details || tr.stage}</p>
            </div>

            <div class="flex items-center space-x-2">
              {#if tr.status === 'paused'}
                <button
                  type="button"
                  on:click={() => tasksStore.resume(tr.id)}
                  class="bg-cyan-500/20 hover:bg-cyan-500/30 text-cyan-400 border border-cyan-500/40 text-xs px-3 py-1 rounded-lg font-bold"
                >
                  Resume
                </button>
              {:else}
                <button
                  type="button"
                  on:click={() => tasksStore.pause(tr.id)}
                  class="bg-[#1e1e2c] hover:bg-[#2b2b3c] text-amber-400 text-xs px-3 py-1 rounded-lg font-bold"
                >
                  Pause
                </button>
              {/if}
              <button
                type="button"
                on:click={() => tasksStore.cancel(tr.id)}
                class="text-rose-400 hover:text-rose-300 text-xs px-2 py-1 font-bold"
              >
                Cancel
              </button>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Finished / Completed History -->
  {#if $completedDownloads.length > 0}
    <div class="space-y-3 pt-2">
      <div class="flex items-center justify-between">
        <h3 class="font-black text-slate-400 text-xs tracking-wider uppercase">
          Completed History ({$completedDownloads.length})
        </h3>
        <button
          type="button"
          on:click={() => tasksStore.clearFinished()}
          class="text-[11px] font-bold text-slate-500 hover:text-slate-300 transition"
        >
          Clear All
        </button>
      </div>

      <div class="bg-[#121218] border border-[#20202c] rounded-2xl divide-y divide-[#1b1b26] overflow-hidden">
        {#each $completedDownloads as tr (tr.id)}
          <div class="p-4 flex items-center justify-between hover:bg-[#151520] transition">
            <div class="flex items-center space-x-3">
              <span class="w-2 h-2 rounded-full {tr.status === 'completed' ? 'bg-emerald-400' : 'bg-rose-500'}"></span>
              <div>
                <span class="font-bold text-white text-xs">{tr.game_name || tr.title || tr.game_id}</span>
                <p class="text-[10px] text-slate-500 font-mono mt-0.5">{tr.details || tr.status}</p>
              </div>
            </div>

            <div class="flex items-center space-x-2">
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
</div>
