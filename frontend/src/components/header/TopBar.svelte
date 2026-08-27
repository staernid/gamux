<script>
  import { activeView, selectedGame } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';

  const totalTasks = tasksStore.activeCount;
  const activeTransfersCount = tasksStore.activeTransfersCount;

  $: activeOpsCount = $activeTransfersCount > 0 ? $activeTransfersCount : $totalTasks;

  const views = ['library', 'tasks', 'settings'];

  function cycleView(dir) {
    const currentBase = $activeView === 'game-hub' ? 'library' : ($activeView === 'downloads' || $activeView === 'transfers' ? 'tasks' : $activeView);
    const idx = views.indexOf(currentBase);
    let next = idx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    $activeView = views[next];
  }

  function handleShelfOrHubClick() {
    if ($activeView === 'game-hub') {
      $activeView = 'library';
    } else if ($activeView === 'library') {
      if ($selectedGame) {
        $activeView = 'game-hub';
      }
    } else {
      $activeView = 'library';
    }
  }
</script>

<header class="absolute top-4 left-1/2 -translate-x-1/2 z-40 select-none pointer-events-none flex items-center justify-center">
  <div class="flex items-center space-x-2.5 pointer-events-auto">
    <!-- Left Bumper Key Prompt (Q / L1) -->
    <button
      type="button"
      on:click={() => cycleView(-1)}
      class="flex items-center space-x-1 text-slate-500 hover:text-slate-200 transition-colors text-[11px] font-mono group px-1"
      title="Previous View (Q / L1)"
    >
      <span class="px-1.5 py-0.5 rounded bg-[#161622]/90 backdrop-blur-md border border-[#262638] text-slate-400 group-hover:text-white text-[10px] font-bold shadow-sm">
        Q
      </span>
      <span class="text-slate-600">/</span>
      <span class="text-slate-400 font-semibold group-hover:text-slate-200">L1</span>
    </button>

    <!-- Centered Floating Pill Navigation Dock -->
    <nav class="flex items-center space-x-1 bg-[#12121a]/85 backdrop-blur-xl border border-[#222232] px-1.5 py-1 rounded-2xl shadow-2xl">
      <!-- Shelf / Game Hub (1) -->
      <button
        type="button"
        on:click={handleShelfOrHubClick}
        class="p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'library' || $activeView === 'game-hub' ? 'bg-[#252538] text-white shadow-md' : 'text-slate-400 hover:text-slate-200'}"
        title={$activeView === 'game-hub' ? 'Back to Shelf (B / Esc)' : 'Shelf (1)'}
      >
        {#if $activeView === 'game-hub'}
          <!-- Game Hub Symbolic Icon (Gamepad / Hub) -->
          <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5 text-cyan-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="6" width="20" height="12" rx="4" />
            <path d="M6 12h4" />
            <path d="M8 10v4" />
            <line x1="15" y1="11" x2="15.01" y2="11" />
            <line x1="18" y1="13" x2="18.01" y2="13" />
          </svg>
        {:else}
          <!-- Shelf / Library Grid Icon -->
          <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="currentColor" viewBox="0 0 20 20">
            <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
          </svg>
        {/if}
      </button>

      <!-- Tasks (2) -->
      <button
        type="button"
        on:click={() => { $activeView = 'tasks'; }}
        class="relative p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'tasks' || $activeView === 'downloads' || $activeView === 'transfers' || $activeView === 'downloader' ? 'bg-[#252538] text-white shadow-md' : 'text-slate-400 hover:text-slate-200'}"
        title="Tasks (2)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        {#if activeOpsCount > 0}
          <span class="absolute -top-1 -right-1 px-1.5 py-0.2 text-[9px] rounded-full bg-cyan-500 text-slate-950 font-black font-mono shadow">
            {activeOpsCount}
          </span>
        {/if}
      </button>

      <!-- Settings (3) -->
      <button
        type="button"
        on:click={() => { $activeView = 'settings'; }}
        class="p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center {$activeView === 'settings' ? 'bg-[#252538] text-white shadow-md' : 'text-slate-400 hover:text-slate-200'}"
        title="Settings (3)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </button>
    </nav>

    <!-- Right Bumper Key Prompt (E / R1) -->
    <button
      type="button"
      on:click={() => cycleView(1)}
      class="flex items-center space-x-1 text-slate-500 hover:text-slate-200 transition-colors text-[11px] font-mono group px-1"
      title="Next View (E / R1)"
    >
      <span class="text-slate-400 font-semibold group-hover:text-slate-200">R1</span>
      <span class="text-slate-600">/</span>
      <span class="px-1.5 py-0.5 rounded bg-[#161622]/90 backdrop-blur-md border border-[#262638] text-slate-400 group-hover:text-white text-[10px] font-bold shadow-sm">
        E
      </span>
    </button>
  </div>
</header>
