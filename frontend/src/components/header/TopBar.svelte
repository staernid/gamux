<script>
  import { activeView, selectedGame, games, runningGames, runningInstances, setGameRunning } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { configStore } from '../../stores/config.js';
  import { profilesStore } from '../../stores/profiles.js';
  import Icon from '../common/Icon.svelte';
  import TopBarProfileMenu from './TopBarProfileMenu.svelte';
  import { AppAPI } from '../../api/app.js';
  import { isSamePath } from '../../utils/game.js';

  const totalTasks = tasksStore.activeCount;
  const activeTransfersCount = tasksStore.activeTransfersCount;
  const isSettingsDirty = configStore.isDirty;

  $: activeOpsCount = $activeTransfersCount > 0 ? $activeTransfersCount : $totalTasks;

  // Build unified running instance display list:
  // If runningInstances has items, map them to game + profile entities.
  // Otherwise fall back to runningGames paths for backward compatibility.
  $: instancePillList = (() => {
    const insts = $runningInstances || [];
    const profs = $profilesStore.profiles || [];
    const allGames = $games || [];

    if (insts.length > 0) {
      // Find games that have active multiseat seats (p1, p2...)
      const gamesWithSeats = new Set();
      for (const inst of insts) {
        if (inst.seatId || inst.seat_id) {
          gamesWithSeats.add((inst.gameDir || inst.game_dir || '').toLowerCase());
        }
      }

      const seen = new Set();
      const unique = [];

      for (const inst of insts) {
        const gDir = inst.gameDir || inst.game_dir || '';
        const normDir = gDir.toLowerCase();
        const seatId = inst.seatId || inst.seat_id || '';
        const profId = inst.profileId || inst.profile_id || 'main';

        // Suppress generic orphan non-seat pill if this game has active seats
        if (!seatId && gamesWithSeats.has(normDir)) {
          continue;
        }

        const dedupKey = `${normDir}:${profId}:${seatId}`;
        if (seen.has(dedupKey)) {
          continue;
        }
        seen.add(dedupKey);

        const game = allGames.find(g => isSamePath(g.gameDir, gDir));
        const profile = profs.find(p => p.id === profId) || (profId === 'main' ? $profilesStore.activeProfile : null);
        const name = inst.gameName || inst.game_name || game?.name || 'Running Game';
        const isSolo = !seatId && (profId === 'main' || profId === '' || !profId);

        unique.push({
          instanceId: inst.instanceId || inst.instance_id || gDir,
          gameDir: gDir,
          game,
          name,
          profileId: profId,
          seatId,
          profile,
          isSolo,
          mode: inst.mode || '',
        });
      }

      return unique;
    }

    // Fallback if runningInstances hasn't received detailed items yet
    return Array.from($runningGames || []).map(dir => {
      const game = allGames.find(g => isSamePath(g.gameDir, dir));
      return {
        instanceId: dir,
        gameDir: dir,
        game,
        name: game?.name || 'Running Game',
        profileId: 'main',
        seatId: '',
        profile: $profilesStore.activeProfile,
        isSolo: true,
        mode: '',
      };
    });
  })();

  function getInitials(name) {
    if (!name) return 'P';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }

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

  async function handleStopInstance(item) {
    if (item.instanceId && typeof AppAPI.stopGameInstance === 'function') {
      await AppAPI.stopGameInstance(item.instanceId);
    } else {
      await AppAPI.stopGame(item.gameDir);
    }
    setGameRunning(item.gameDir, false, 0, 0, {
      profileId: item.profileId,
      instanceId: item.instanceId,
      seatId: item.seatId
    });
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

<!-- Top Right Controls (Active Running Game Instances & Profile Switcher) -->
<aside class="absolute top-4 right-6 z-40 select-none pointer-events-none flex items-start space-x-2.5">
  {#if instancePillList.length > 0}
    <div class="flex flex-col items-end space-y-2">
      {#each instancePillList as item (item.instanceId)}
        <div class="pointer-events-auto flex items-center space-x-2.5 bg-surface-header/95 border border-surface-border hover:border-surface-borderHighlight px-3 py-1.5 rounded-2xl shadow-2xl animate-fade-in transition-colors">
          <!-- Profile Avatar + Badge (shown for secondary / multiseat profiles; omitted for solo main) -->
          {#if !item.isSolo}
            <div class="flex items-center space-x-1.5 flex-shrink-0" title="{item.profile?.name || item.profileId} ({item.seatId || item.profileId})">
              {#if item.profile?.avatar_url}
                <img
                  src={item.profile.avatar_url}
                  alt={item.profile.name || item.profileId}
                  class="w-4 h-4 rounded-full object-cover border border-surface-border flex-shrink-0"
                />
              {:else}
                <div class="w-4 h-4 rounded-full bg-surface-input text-neutral-300 font-bold text-4xs uppercase flex items-center justify-center flex-shrink-0 border border-surface-border">
                  {getInitials(item.profile?.name || item.profileId)}
                </div>
              {/if}
              <span class="px-1.5 py-0.2 rounded-md bg-surface-input border border-surface-border text-neutral-300 text-3xs font-mono font-bold">
                {item.seatId ? item.seatId.toUpperCase() : item.profileId}
              </span>
            </div>
          {/if}

          <!-- Game Title Clickable Button -->
          <button
            type="button"
            on:click={() => {
              if (item.game) {
                navigateWithGuard('game-hub', () => { $selectedGame = item.game; $activeView = 'game-hub'; });
              }
            }}
            class="flex items-center space-x-2 text-xs text-neutral-200 hover:text-white font-bold transition-colors truncate max-w-[160px] sm:max-w-[220px] cursor-pointer"
            title="Jump to Game Hub ({item.name})"
          >
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse flex-shrink-0"></span>
            <span class="truncate">{item.name}</span>
          </button>

          <!-- Instance Stop Button -->
          <button
            type="button"
            on:click|stopPropagation={() => handleStopInstance(item)}
            class="px-2 py-0.5 rounded-xl bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 hover:text-white text-2xs font-mono font-bold border border-rose-900/40 hover:border-rose-700/60 transition-all flex items-center space-x-1 flex-shrink-0 active:scale-95 cursor-pointer shadow-sm"
            title="Stop Process (Terminate {item.name}{!item.isSolo ? ` - ${item.profileId}` : ''})"
          >
            <svg class="w-2.5 h-2.5 fill-current" viewBox="0 0 24 24">
              <rect x="4" y="4" width="16" height="16" rx="2" />
            </svg>
            <span>Stop</span>
          </button>
        </div>
      {/each}
    </div>
  {/if}

  <TopBarProfileMenu onNavigateToProfilesSettings={() => navigateWithGuard('settings')} />
</aside>

