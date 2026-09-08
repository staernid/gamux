<script>
  import { activeView, selectedGame, games, runningGames, setGameRunning } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { configStore } from '../../stores/config.js';
  import Icon from '../common/Icon.svelte';
  import { AppAPI } from '../../api/app.js';
  import { isSamePath } from '../../utils/game.js';

  const totalTasks = tasksStore.activeCount;
  const activeTransfersCount = tasksStore.activeTransfersCount;
  const isSettingsDirty = configStore.isDirty;

  $: activeOpsCount = $activeTransfersCount > 0 ? $activeTransfersCount : $totalTasks;
  $: runningGameList = Array.from($runningGames || []).map(dir => $games.find(g => isSamePath(g.gameDir, dir))).filter(Boolean);

  function navigateWithGuard(targetView, onProceed = null) {
    if (typeof onProceed === 'function') {
      onProceed();
    } else {
      $activeView = targetView;
    }
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
  <div class="pointer-events-auto flex items-center bg-surface-header/95 backdrop-blur-md border border-surface-border p-1 rounded-2xl shadow-2xl">
    <!-- Centered Navigation Items (Icon-only) -->
    <nav class="flex items-center space-x-1">
      <!-- Shelf / Game Hub (1) -->
      <button
        type="button"
        on:click={handleShelfOrHubClick}
        class="p-2 rounded-xl transition-all flex items-center justify-center cursor-pointer {$activeView === 'library' || $activeView === 'game-hub' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title={$activeView === 'game-hub' ? 'Back to Shelf' : 'Shelf (1)'}
        aria-label={$activeView === 'game-hub' ? 'Back to Shelf' : 'Shelf (1)'}
      >
        <Icon name="shelf" className="w-4 h-4 flex-shrink-0" />
      </button>

      <!-- Tasks (2) -->
      <button
        type="button"
        on:click={() => navigateWithGuard('tasks')}
        class="relative p-2 rounded-xl transition-all flex items-center justify-center cursor-pointer {$activeView === 'tasks' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title="Tasks (2)"
        aria-label="Tasks (2)"
      >
        <Icon name="tasks" className="w-4 h-4 flex-shrink-0" />
        {#if activeOpsCount > 0}
          <span class="absolute -top-1 -right-1 px-1.5 py-0.2 min-w-[15px] text-center text-4xs rounded-full bg-white text-neutral-950 font-black font-mono shadow">
            {activeOpsCount}
          </span>
        {/if}
      </button>

      <!-- Settings (3) -->
      <button
        type="button"
        on:click={() => navigateWithGuard('settings')}
        class="p-2 rounded-xl transition-all flex items-center justify-center cursor-pointer {$activeView === 'settings' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
        title="Settings (3)"
        aria-label="Settings (3)"
      >
        <Icon name="settings" className="w-4 h-4 flex-shrink-0" />
      </button>

      <!-- Store Catalog (Wireframe Globe) -->
      <button
        type="button"
        on:click={() => modalStore.open('catalog')}
        class="p-2 rounded-xl transition-all flex items-center justify-center text-neutral-400 hover:text-neutral-200 hover:bg-surface-input cursor-pointer"
        title="Catalog"
        aria-label="Catalog"
      >
        <Icon name="globe" className="w-4 h-4 flex-shrink-0" />
      </button>
    </nav>
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
