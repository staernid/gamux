<script>
  import {
    activeView,
    libraryPath,
    isScanning,
    scanLibrary,
    importGame,
    searchQuery,
    pathPopover,
    stats
  } from '../../stores/library.js';
  import { taskQueue } from '../../stores/queue.js';
  import { AppAPI } from '../../api/app.js';

  const activeTask = taskQueue.activeTask;
  const totalTasks = taskQueue.totalCount;

  let searchInputEl;

  async function handleBrowseLibrary() {
    const selected = await AppAPI.selectDirectory('Select Steam Games Library Folder', $libraryPath);
    if (selected) {
      $libraryPath = selected;
      $pathPopover = false;
      scanLibrary(selected);
    }
  }

  function handleScan() {
    scanLibrary();
  }
</script>

<header class="h-14 bg-[#0d0d12] border-b border-[#1f1f2a] px-4 sm:px-6 flex items-center justify-between flex-shrink-0 select-none z-30 transition-all">
  <!-- Left: Brand & Library Selector -->
  <div class="flex items-center space-x-3 sm:space-x-4 min-w-0">
    <!-- Emblem -->
    <div class="flex items-center space-x-2 font-black text-sm tracking-tight text-white flex-shrink-0">
      <span class="w-7 h-7 rounded-xl bg-white text-black flex items-center justify-center font-mono font-black text-xs shadow-md">
        GX
      </span>
      <span class="hidden sm:inline font-black text-slate-100 text-sm">gamux</span>
    </div>

    <span class="text-slate-700 hidden sm:inline">/</span>

    <!-- Library Folder Switcher -->
    <div class="relative">
      <button
        type="button"
        on:click={() => { $pathPopover = !$pathPopover; }}
        class="flex items-center space-x-2 bg-[#14141c] hover:bg-[#1c1c26] border border-[#232330] hover:border-slate-500 rounded-xl px-3 py-1.5 text-xs sm:text-sm text-slate-200 transition-colors max-w-[180px] sm:max-w-[240px] truncate"
        title={$libraryPath || 'No library folder selected'}
      >
        <svg class="w-4 h-4 text-slate-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <span class="truncate font-mono font-medium text-xs sm:text-sm">
          {$libraryPath ? $libraryPath.split('/').filter(Boolean).pop() || $libraryPath : 'Choose Library...'}
        </span>
        <svg class="w-3.5 h-3.5 text-slate-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {#if $pathPopover}
        <div class="absolute left-0 top-11 w-80 sm:w-96 panel-elevated rounded-2xl shadow-2xl p-4 z-50 animate-fade-in space-y-3 border border-[#323244] bg-[#14141c]">
          <div class="flex items-center justify-between">
            <h4 class="text-xs font-bold uppercase tracking-wider text-slate-300">Steam Library Directory</h4>
            <button type="button" on:click={() => { $pathPopover = false; }} class="text-slate-400 hover:text-white text-xs">✕</button>
          </div>
          <div class="space-y-2">
            <div class="flex space-x-2">
              <input
                type="text"
                bind:value={$libraryPath}
                placeholder="/path/to/steamapps/common"
                class="w-full bg-[#0a0a0e] border border-[#282836] rounded-xl px-3 py-1.5 text-xs text-slate-100 font-mono outline-none focus:border-white"
              />
              <button
                type="button"
                on:click={handleBrowseLibrary}
                class="bg-[#20202c] hover:bg-[#2c2c3c] text-white px-3 py-1.5 rounded-xl text-xs font-bold border border-[#36364a] flex-shrink-0"
              >
                Browse
              </button>
            </div>
            <div class="flex justify-between items-center pt-1">
              <button
                type="button"
                on:click={importGame}
                class="text-xs text-slate-400 hover:text-slate-200 underline font-mono"
              >
                Import Single Folder...
              </button>
              <button
                type="button"
                on:click={() => {
                  $pathPopover = false;
                  scanLibrary();
                }}
                class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-4 py-1.5 rounded-xl transition-colors shadow-sm"
              >
                Scan Now
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Scan Button -->
    <button
      type="button"
      on:click={handleScan}
      class="text-slate-400 hover:text-white p-1.5 rounded-lg hover:bg-[#181822] transition-colors"
      title="Rescan Library"
    >
      <svg class="w-4 h-4 {$isScanning ? 'animate-spin text-white' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
    </button>
  </div>

  <!-- Center: Tactile Icon Navigation (1-4) -->
  <nav class="flex items-center space-x-1.5 bg-[#14141c] p-1 rounded-xl border border-[#22222e]">
    <!-- Shelf (1) -->
    <button
      type="button"
      on:click={() => { $activeView = 'library'; }}
      class="px-3 py-1.5 rounded-lg transition-all flex items-center space-x-2 {$activeView === 'library' ? 'bg-[#252535] text-white shadow' : 'text-slate-400 hover:text-slate-200'}"
      title="Shelf (1)"
    >
      <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="currentColor" viewBox="0 0 20 20">
        <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
      </svg>
      <span class="text-xs font-mono font-bold text-slate-400">1</span>
    </button>

    <!-- Downloader (2) -->
    <button
      type="button"
      on:click={() => { $activeView = 'downloader'; }}
      class="px-3 py-1.5 rounded-lg transition-all flex items-center space-x-2 {$activeView === 'downloader' ? 'bg-[#252535] text-white shadow' : 'text-slate-400 hover:text-slate-200'}"
      title="Downloader (2)"
    >
      <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
      </svg>
      <span class="text-xs font-mono font-bold text-slate-400">2</span>
    </button>

    <!-- Activity (3) -->
    <button
      type="button"
      on:click={() => { $activeView = 'activity'; }}
      class="px-3 py-1.5 rounded-lg transition-all flex items-center space-x-2 {$activeView === 'activity' ? 'bg-[#252535] text-white shadow' : 'text-slate-400 hover:text-slate-200'}"
      title="Activity (3)"
    >
      <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
      </svg>
      {#if $totalTasks > 0}
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
      {/if}
      <span class="text-xs font-mono font-bold text-slate-400">3</span>
    </button>

    <!-- Settings (4) -->
    <button
      type="button"
      on:click={() => { $activeView = 'settings'; }}
      class="px-3 py-1.5 rounded-lg transition-all flex items-center space-x-2 {$activeView === 'settings' ? 'bg-[#252535] text-white shadow' : 'text-slate-400 hover:text-slate-200'}"
      title="Settings (4)"
    >
      <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span class="text-xs font-mono font-bold text-slate-400">4</span>
    </button>
  </nav>

  <!-- Right: Search Box & Active Telemetry -->
  <div class="flex items-center space-x-3">
    {#if $activeTask}
      <div class="hidden md:flex items-center space-x-2 bg-[#161622] border border-[#2b2b3c] px-3 py-1 rounded-xl text-xs font-mono">
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
        <span class="text-slate-200 truncate max-w-[130px]">{$activeTask.title}</span>
        <span class="text-emerald-400 font-bold">{$activeTask.percent}%</span>
      </div>
    {:else if $stats.totalCount > 0}
      <span class="hidden md:inline text-xs font-mono text-slate-400">
        {$stats.totalCount} {$stats.totalCount === 1 ? 'game' : 'games'}
      </span>
    {/if}

    <!-- Search Input (Activates on /) -->
    <div class="relative flex items-center">
      <svg class="w-3.5 h-3.5 text-slate-500 absolute left-3 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        bind:this={searchInputEl}
        bind:value={$searchQuery}
        type="text"
        placeholder="Search (/)"
        class="bg-[#14141c] border border-[#22222e] rounded-xl pl-8 pr-6 py-1.5 text-xs sm:text-sm text-slate-200 outline-none w-28 sm:w-40 focus:w-52 focus:border-white transition-all font-mono placeholder:text-slate-500"
      />
      {#if $searchQuery}
        <button
          type="button"
          on:click={() => { $searchQuery = ''; }}
          class="absolute right-2 text-slate-400 hover:text-white text-xs font-bold"
        >
          ✕
        </button>
      {/if}
    </div>
  </div>
</header>
