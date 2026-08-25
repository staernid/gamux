<script>
  import { activeView, isScanning, scanLibrary } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';

  const totalTasks = tasksStore.activeCount;
  const activeTransfersCount = tasksStore.activeTransfersCount;

  const views = ['library', 'downloads', 'activity', 'settings'];

  function cycleView(dir) {
    const idx = views.indexOf($activeView);
    let next = idx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    $activeView = views[next];
  }
</script>

<header class="w-full pt-4 pb-2 flex items-center justify-center flex-shrink-0 select-none z-30 pointer-events-auto">
  <!-- Centered Floating Pill Navigation -->
  <div class="flex items-center space-x-2 bg-[#12121a]/80 backdrop-blur-xl border border-[#222232] px-2 py-1.5 rounded-2xl shadow-2xl">
    <!-- Left Bumper Chip (Q / L1) -->
    <button
      type="button"
      on:click={() => cycleView(-1)}
      class="px-2 py-1 rounded-xl bg-[#181824] hover:bg-[#222232] text-slate-400 hover:text-white transition-colors flex items-center space-x-1 text-[10px] font-mono font-bold border border-[#27273a]"
      title="Previous View (Q / L1)"
    >
      <span class="text-slate-200">Q</span>
      <span class="text-slate-500">/</span>
      <span class="text-slate-400">L1</span>
    </button>

    <!-- View Icons & Refresh Action -->
    <nav class="flex items-center space-x-1">
      <!-- Shelf / Library (1) -->
      <button
        type="button"
        on:click={() => { $activeView = 'library'; }}
        class="p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'library' ? 'bg-[#28283c] text-white shadow-lg' : 'text-slate-400 hover:text-slate-200'}"
        title="Shelf (1)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="currentColor" viewBox="0 0 20 20">
          <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
        </svg>
      </button>

      <!-- Downloads (2) -->
      <button
        type="button"
        on:click={() => { $activeView = 'downloads'; }}
        class="relative p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'downloads' || $activeView === 'transfers' || $activeView === 'downloader' ? 'bg-[#28283c] text-white shadow-lg' : 'text-slate-400 hover:text-slate-200'}"
        title="Downloads (2)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        {#if $activeTransfersCount > 0}
          <span class="absolute -top-1 -right-1 px-1.5 py-0.2 text-[9px] rounded-full bg-[#1a9fff] text-white font-bold font-mono shadow">
            {$activeTransfersCount}
          </span>
        {/if}
      </button>

      <!-- Activity (3) -->
      <button
        type="button"
        on:click={() => { $activeView = 'activity'; }}
        class="relative p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'activity' ? 'bg-[#28283c] text-white shadow-lg' : 'text-slate-400 hover:text-slate-200'}"
        title="Activity (3)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
        </svg>
        {#if $totalTasks > 0}
          <span class="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
        {/if}
      </button>

      <!-- Settings (4) -->
      <button
        type="button"
        on:click={() => { $activeView = 'settings'; }}
        class="p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'settings' ? 'bg-[#28283c] text-white shadow-lg' : 'text-slate-400 hover:text-slate-200'}"
        title="Settings (4)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </button>

      <!-- Refresh (R) -->
      <button
        type="button"
        on:click={() => scanLibrary()}
        disabled={$isScanning}
        class="p-2 sm:px-2.5 sm:py-2 rounded-xl transition-all flex items-center justify-center text-slate-400 hover:text-sky-300 disabled:opacity-50"
        title="Refresh Library (R / F5)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5 {$isScanning ? 'animate-spin text-sky-400' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
      </button>
    </nav>

    <!-- Right Bumper Chip (E / R1) -->
    <button
      type="button"
      on:click={() => cycleView(1)}
      class="px-2 py-1 rounded-xl bg-[#181824] hover:bg-[#222232] text-slate-400 hover:text-white transition-colors flex items-center space-x-1 text-[10px] font-mono font-bold border border-[#27273a]"
      title="Next View (E / R1)"
    >
      <span class="text-slate-200">E</span>
      <span class="text-slate-500">/</span>
      <span class="text-slate-400">R1</span>
    </button>
  </div>
</header>
