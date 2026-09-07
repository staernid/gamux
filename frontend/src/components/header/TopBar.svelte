<script>
  import { activeView, selectedGame, games, runningGames, setGameRunning } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { configStore } from '../../stores/config.js';
  import { AppAPI } from '../../api/app.js';
  import { isSamePath } from '../../utils/game.js';

  const totalTasks = tasksStore.activeCount;
  const activeTransfersCount = tasksStore.activeTransfersCount;
  const isSettingsDirty = configStore.isDirty;

  $: activeOpsCount = $activeTransfersCount > 0 ? $activeTransfersCount : $totalTasks;
  $: runningGameList = Array.from($runningGames || []).map(dir => $games.find(g => isSamePath(g.gameDir, dir))).filter(Boolean);

  const views = ['library', 'tasks', 'settings'];

  function navigateWithGuard(targetView, onProceed = null) {
    if (typeof onProceed === 'function') {
      onProceed();
    } else {
      $activeView = targetView;
    }
  }

  function cycleView(dir) {
    const currentBase = $activeView === 'game-hub' ? 'library' : $activeView;
    const idx = views.indexOf(currentBase);
    let next = idx + dir;
    if (next < 0) next = views.length - 1;
    if (next >= views.length) next = 0;
    navigateWithGuard(views[next]);
  }

  function handleShelfOrHubClick() {
    if ($activeView === 'game-hub') {
      navigateWithGuard('library');
    } else if ($activeView === 'library') {
      if ($selectedGame) {
        $activeView = 'game-hub';
      }
    } else {
      navigateWithGuard('library');
    }
  }
</script>

<header class="absolute top-4 left-1/2 -translate-x-1/2 z-40 select-none pointer-events-none flex items-center justify-center">
  <div class="pointer-events-auto flex items-center bg-surface-header/95 backdrop-blur-md border border-surface-border p-1 rounded-2xl shadow-2xl space-x-1">
    <!-- Left Bumper Key Prompt (Q / L1) -->
    <button
      type="button"
      on:click={() => cycleView(-1)}
      class="flex items-center space-x-1 text-neutral-500 hover:text-neutral-200 transition-colors text-2xs font-mono px-2 py-1.5 rounded-xl hover:bg-surface-input cursor-pointer"
      title="Previous View (Q / L1)"
      aria-label="Previous View (Q / L1)"
    >
      <span class="px-1 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-400 text-3xs font-bold">Q</span>
      <span class="text-neutral-600">/</span>
      <span class="text-neutral-400 font-semibold text-3xs">L1</span>
    </button>

    <div class="h-4 w-px bg-surface-border mx-0.5"></div>

    <!-- Centered Navigation Items -->
    <nav class="flex items-center space-x-1">
      <!-- Shelf / Game Hub (1) -->
      <button
        type="button"
        on:click={handleShelfOrHubClick}
        class="px-3 py-1.5 rounded-xl transition-all flex items-center space-x-2 cursor-pointer {$activeView === 'library' || $activeView === 'game-hub' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title={$activeView === 'game-hub' ? 'Back to Shelf (B / Esc)' : 'Shelf (1)'}
      >
        <svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
        </svg>
        <span class="text-xs font-semibold tracking-wide">{$activeView === 'game-hub' ? 'Back to Shelf' : 'Shelf'}</span>
      </button>

      <!-- Tasks (2) -->
      <button
        type="button"
        on:click={() => navigateWithGuard('tasks')}
        class="relative px-3 py-1.5 rounded-xl transition-all flex items-center space-x-2 cursor-pointer {$activeView === 'tasks' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title="Tasks (2)"
      >
        <svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        <span class="text-xs font-semibold tracking-wide">Tasks</span>
        {#if activeOpsCount > 0}
          <span class="px-1.5 py-0.5 text-3xs rounded-full bg-white text-neutral-950 font-black font-mono shadow">
            {activeOpsCount}
          </span>
        {/if}
      </button>

      <!-- Settings (3) -->
      <button
        type="button"
        on:click={() => navigateWithGuard('settings')}
        class="px-3 py-1.5 rounded-xl transition-all flex items-center space-x-2 cursor-pointer {$activeView === 'settings' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title="Settings (3)"
      >
        <svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
        <span class="text-xs font-semibold tracking-wide">Settings</span>
      </button>

      <!-- Command Palette & Search (Ctrl+K) -->
      <button
        type="button"
        on:click={() => modalStore.open('search-grid')}
        class="px-2.5 py-1.5 rounded-xl transition-all flex items-center space-x-1.5 text-neutral-400 hover:text-neutral-200 hover:bg-surface-input cursor-pointer"
        title="Search & Commands (Ctrl+K)"
        aria-label="Search & Commands (Ctrl+K)"
      >
        <svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <span class="text-xs font-semibold tracking-wide hidden sm:inline">Search</span>
        <span class="px-1.5 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-400 font-mono text-3xs font-bold">Ctrl+K</span>
      </button>
    </nav>

    <div class="h-4 w-px bg-surface-border mx-0.5"></div>

    <!-- Right Bumper Key Prompt (E / R1) -->
    <button
      type="button"
      on:click={() => cycleView(1)}
      class="flex items-center space-x-1 text-neutral-500 hover:text-neutral-200 transition-colors text-2xs font-mono px-2 py-1.5 rounded-xl hover:bg-surface-input cursor-pointer"
      title="Next View (E / R1)"
      aria-label="Next View (E / R1)"
    >
      <span class="text-neutral-400 font-semibold text-3xs">R1</span>
      <span class="text-neutral-600">/</span>
      <span class="px-1 py-0.5 rounded bg-surface-card border border-surface-border text-neutral-400 text-3xs font-bold">E</span>
    </button>
  </div>
</header>

<!-- Active Running Games Stack (Anchored to top-right, vertically stacked) -->
{#if runningGameList.length > 0}
  <aside class="absolute top-4 right-6 z-40 select-none pointer-events-none flex flex-col items-end space-y-2">
    {#each runningGameList as rGame (rGame.gameDir)}
      <div class="pointer-events-auto flex items-center space-x-2.5 bg-surface-header/95 border border-surface-border hover:border-surface-borderHighlight px-3 py-1.5 rounded-2xl shadow-2xl animate-fade-in transition-colors">
        <button
          type="button"
          on:click={() => navigateWithGuard('game-hub', () => { $selectedGame = rGame; $activeView = 'game-hub'; })}
          class="flex items-center space-x-2 text-xs text-neutral-200 hover:text-white font-bold transition-colors truncate max-w-[180px] sm:max-w-[240px] cursor-pointer"
          title="Jump to Game Hub ({rGame.name})"
        >
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse flex-shrink-0"></span>
          <span class="truncate">{rGame.name}</span>
        </button>
        <button
          type="button"
          on:click|stopPropagation={() => {
            AppAPI.stopGame(rGame.gameDir);
            setGameRunning(rGame.gameDir, false);
          }}
          class="px-2 py-0.5 rounded-xl bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 hover:text-white text-2xs font-mono font-bold border border-rose-900/40 hover:border-rose-700/60 transition-all flex items-center space-x-1 flex-shrink-0 active:scale-95 cursor-pointer shadow-sm"
          title="Stop Process (Terminate {rGame.name})"
        >
          <svg class="w-2.5 h-2.5 fill-current" viewBox="0 0 24 24">
            <rect x="4" y="4" width="16" height="16" rx="2" />
          </svg>
          <span>Stop</span>
        </button>
      </div>
    {/each}
  </aside>
{/if}
