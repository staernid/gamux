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
    if ($activeView === 'settings' && $isSettingsDirty && targetView !== 'settings') {
      modalStore.open('confirm', {
        title: 'Unsaved Changes',
        message: 'You have unsaved changes in Settings. Do you want to discard them or keep editing?',
        confirmLabel: 'Discard Changes',
        cancelLabel: 'Keep Editing',
        danger: true,
        onConfirm: () => {
          configStore.setDirty(false);
          if (typeof onProceed === 'function') {
            onProceed();
          } else {
            $activeView = targetView;
          }
        }
      });
      return;
    }

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
  <div class="flex items-center space-x-2.5 pointer-events-auto">
    <!-- Left Bumper Key Prompt (Q / L1) -->
    <button
      type="button"
      on:click={() => cycleView(-1)}
      class="flex items-center space-x-1 text-neutral-500 hover:text-neutral-200 transition-colors text-2xs font-mono group px-1 cursor-pointer"
      title="Previous View (Q / L1)"
    >
      <span class="px-1.5 py-0.5 rounded bg-surface-card/90 border border-surface-border text-neutral-400 group-hover:text-white text-3xs font-bold shadow-sm">
        Q
      </span>
      <span class="text-neutral-500">/</span>
      <span class="text-neutral-400 font-semibold group-hover:text-neutral-200">L1</span>
    </button>

    <!-- Centered Floating Pill Navigation Dock -->
    <nav class="flex items-center space-x-1 bg-[#121214]/90 border border-[#27272a] px-1.5 py-1 rounded-2xl shadow-2xl">
      <!-- Shelf / Game Hub (1) -->
      <button
        type="button"
        on:click={handleShelfOrHubClick}
        class="p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center cursor-pointer {$activeView === 'library' || $activeView === 'game-hub' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title={$activeView === 'game-hub' ? 'Back to Shelf (B / Esc)' : 'Shelf (1)'}
      >
        {#if $activeView === 'game-hub'}
          <!-- Game Hub Symbolic Icon (Gamepad / Hub) -->
          <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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
        on:click={() => navigateWithGuard('tasks')}
        class="relative p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center cursor-pointer {$activeView === 'tasks' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title="Tasks (2)"
      >
        <svg class="w-4 h-4 sm:w-4.5 sm:h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        {#if activeOpsCount > 0}
          <span class="absolute -top-1 -right-1 px-1.5 py-0.5 text-3xs rounded-full bg-white text-neutral-950 font-black font-mono shadow">
            {activeOpsCount}
          </span>
        {/if}
      </button>

      <!-- Settings (3) -->
      <button
        type="button"
        on:click={() => navigateWithGuard('settings')}
        class="p-2 sm:px-3 sm:py-2 rounded-xl transition-all flex items-center justify-center cursor-pointer {$activeView === 'settings' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
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
      class="flex items-center space-x-1 text-neutral-500 hover:text-neutral-200 transition-colors text-2xs font-mono group px-1 cursor-pointer"
      title="Next View (E / R1)"
    >
      <span class="text-neutral-400 font-semibold group-hover:text-neutral-200">R1</span>
      <span class="text-neutral-500">/</span>
      <span class="px-1.5 py-0.5 rounded bg-surface-card/90 border border-surface-border text-neutral-400 group-hover:text-white text-3xs font-bold shadow-sm">
        E
      </span>
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
