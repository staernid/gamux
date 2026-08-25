<script>
  import { selectedGame, queueApplyGame, showInspector, libraryPath, scanLibrary, importGame, activeView, runningGames, games, refreshGame } from '../../stores/library.js';
  import { tasksStore } from '../../stores/tasks.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { formatBytes } from '../../utils/format.js';
  import { isSteamApp } from '../../utils/game.js';
  import { AppAPI } from '../../api/app.js';
  import { modalStore } from '../../stores/modal.js';
  import PosterArt from '../common/PosterArt.svelte';

  let isLaunching = false;
  let isStopping = false;

  $: isRunning = $selectedGame?.gameDir ? $runningGames.has($selectedGame.gameDir) : false;

  async function handleLaunch() {
    if (!$selectedGame) return;
    isLaunching = true;
    try {
      await AppAPI.launchGame({
        game_dir: $selectedGame.gameDir,
        name: $selectedGame.name,
        app_id: $selectedGame.appId,
        platform: $selectedGame.platform,
        exe_path: $selectedGame.exePath,
        exe_args: $selectedGame.exeArgs,
        operating_mode: $selectedGame.operatingMode,
        runner: $selectedGame.runner,
      });
      notificationStore.items.add({
        type: 'success',
        title: 'Launching',
        message: `Starting ${$selectedGame.name}...`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Launch Failed',
        message: err?.message || String(err)
      });
    } finally {
      setTimeout(() => { isLaunching = false; }, 1200);
    }
  }

  async function handleStop() {
    if (!$selectedGame?.gameDir) return;
    isStopping = true;
    try {
      await AppAPI.stopGame($selectedGame.gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Game Stopped',
        message: `Stopped ${$selectedGame.name}`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Stop Failed',
        message: err?.message || String(err)
      });
    } finally {
      isStopping = false;
    }
  }

  function handlePrepare() {
    if (!$selectedGame) return;
    queueApplyGame($selectedGame);
  }

  async function handleUpdateGame() {
    if (!$selectedGame?.gameDir) return;
    const appIdNum = parseInt($selectedGame.appId, 10);
    if (appIdNum && !isNaN(appIdNum)) {
      try {
        await tasksStore.start({
          app_id: appIdNum,
          game_name: $selectedGame.name,
          target_dir: $selectedGame.gameDir,
          platform: $selectedGame.platform,
          auto_apply: true,
        });
      } catch (err) {
        // Handled in store
      }
    } else {
      tasksStore.enqueue({
        type: 'update',
        title: `Update: ${$selectedGame.name}`,
        gameName: $selectedGame.name,
        gameDir: $selectedGame.gameDir,
        run: async () => {
          await AppAPI.submitUpdate($selectedGame.gameDir);
          return `Successfully queued update for ${$selectedGame.name}`;
        },
        onSuccess: async () => {
          try {
            await refreshGame($selectedGame.gameDir);
          } catch {
            scanLibrary();
          }
        }
      });
    }
  }

  async function handleOpenFolder() {
    if (!$selectedGame?.gameDir) return;
    try {
      await AppAPI.openDirectory($selectedGame.gameDir);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Cannot Open Folder',
        message: err?.message || String(err)
      });
    }
  }

  async function handleBrowseLibrary() {
    const selected = await AppAPI.selectDirectory('Select Steam Games Library Folder', $libraryPath);
    if (selected) {
      $libraryPath = selected;
      scanLibrary(selected);
    }
  }
</script>

<div class="flex-1 flex flex-col justify-center px-6 sm:px-12 lg:px-16 py-6 relative overflow-hidden bg-gradient-to-b from-[#0e0e14] to-[#09090c]">
  {#if $selectedGame}
    <!-- Focused Game Showcase -->
    <div class="max-w-5xl mx-auto w-full flex flex-col md:flex-row items-center md:items-start gap-8 lg:gap-12 animate-fade-in">
      <!-- Left: Game Capsule Artwork -->
      <div class="w-56 sm:w-64 lg:w-72 aspect-[2/3] rounded-2xl overflow-hidden bg-[#161622] border-2 border-[#2a2a3c] shadow-2xl flex-shrink-0 relative group">
        <PosterArt
          appId={$selectedGame.appId}
          name={$selectedGame.name}
          posterUrl={$selectedGame.coverUrl}
          localCoverUrl={$selectedGame.localCoverUrl}
          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
        />

        <!-- Hover Overlay for Changing Artwork -->
        <button
          type="button"
          on:click={() => { modalStore.open('artwork-picker', { game: $selectedGame }); }}
          class="absolute inset-0 bg-black/65 opacity-0 group-hover:opacity-100 transition-opacity flex flex-col items-center justify-center p-4 text-center cursor-pointer backdrop-blur-[2px] z-10"
          title="Change Artwork"
        >
          <span class="w-10 h-10 rounded-full bg-sky-600/90 text-white flex items-center justify-center text-lg mb-2 shadow-lg">
            🎨
          </span>
          <span class="text-xs font-bold text-white">Change Cover</span>
          <span class="text-[10px] text-slate-300 font-mono mt-0.5">SteamGridDB / Custom</span>
        </button>

        {#if isSteamApp($selectedGame.appId)}
          <span class="absolute bottom-2 left-2 bg-black/85 backdrop-blur-sm font-mono text-[10px] text-slate-300 px-2 py-0.5 rounded-md font-bold border border-white/10 pointer-events-none z-20">
            #{$selectedGame.appId}
          </span>
        {/if}
      </div>

      <!-- Right: Primary Metadata & Concise Handheld Actions -->
      <div class="flex-1 flex flex-col justify-between space-y-5 min-w-0 text-center md:text-left">
        <div>
          <!-- Status Pill Strip -->
          <div class="flex items-center justify-center md:justify-start space-x-2.5 mb-2">
            <button
              type="button"
              on:click={() => { modalStore.open('metadata-link', { game: $selectedGame }); }}
              class="inline-flex items-center space-x-1.5 text-xs px-2.5 py-0.5 rounded-full font-bold transition-all hover:scale-105 cursor-pointer {
                ($selectedGame.store === 'GOG')
                  ? 'bg-purple-950/50 border border-purple-600/60 text-purple-300 hover:bg-purple-900/60'
                  : ($selectedGame.store === 'Standalone' || $selectedGame.store === 'Custom')
                  ? 'bg-emerald-950/50 border border-emerald-600/60 text-emerald-300 hover:bg-emerald-900/60'
                  : 'bg-blue-950/50 border border-blue-600/60 text-blue-300 hover:bg-blue-900/60'
              }"
              title="Click to Change Store / Metadata Provider"
            >
              <span>{$selectedGame.store || 'Steam'}</span>
              <span class="text-[9px] opacity-70">▾</span>
            </button>

            {#if $selectedGame.isReady || $selectedGame.isPatched}
              <span class="inline-flex items-center space-x-1.5 bg-emerald-950/40 border border-emerald-700/50 text-emerald-300 text-xs px-2.5 py-0.5 rounded-full font-bold">
                <span class="pip pip-ready"></span>
                <span>{$selectedGame.operatingMode === 'standalone' ? 'Ready (Direct)' : ($selectedGame.lutrisRegistered ? 'Ready (Lutris)' : 'Ready (Prepared)')}</span>
              </span>
            {:else}
              <span class="inline-flex items-center space-x-1.5 bg-slate-900 border border-slate-700 text-slate-400 text-xs px-2.5 py-0.5 rounded-full font-medium">
                <span class="pip pip-clean"></span>
                <span>Original</span>
              </span>
            {/if}

            {#if $selectedGame.hasUpdate || $selectedGame.hasUpdatesOrIssues}
              <span class="inline-flex items-center space-x-1 bg-amber-950/40 border border-amber-700/50 text-amber-300 text-xs px-2.5 py-0.5 rounded-full font-bold">
                Update Available
              </span>
            {/if}
          </div>

          <!-- Game Title -->
          <h1 class="text-2xl sm:text-3xl lg:text-4xl font-black text-white tracking-tight leading-tight" title={$selectedGame.name}>
            {$selectedGame.name}
          </h1>

          <!-- Quick Specs -->
          <div class="flex flex-wrap items-center justify-center md:justify-start gap-x-4 gap-y-1.5 text-xs text-slate-400 font-mono mt-2.5">
            <span class="text-slate-200 bg-[#161622] px-2.5 py-0.5 rounded-md border border-[#262638]">
              {$selectedGame.platform === 'linux' ? '🐧 Linux Native' : '🪟 Windows'}
            </span>
            <span>Size: <strong class="text-white font-semibold">{formatBytes($selectedGame.diskSizeBytes)}</strong></span>
            {#if $selectedGame.dlcCount}
              <span>DLCs: <strong class="text-white font-semibold">{$selectedGame.dlcCount}</strong></span>
            {/if}
            {#if $selectedGame.achievementCount}
              <span>Achievements: <strong class="text-slate-200">{$selectedGame.achievementCount}</strong></span>
            {/if}
          </div>

          <!-- Directory Path -->
          <div class="text-[11px] font-mono text-slate-500 truncate max-w-xl mt-2 select-text" title={$selectedGame.gameDir}>
            {$selectedGame.gameDir}
          </div>
        </div>

        <!-- Chunky Action Buttons -->
        <div class="flex flex-wrap items-center justify-center md:justify-start gap-3 pt-2">
          {#if isRunning}
            <!-- Running / STOP button -->
            <button
              type="button"
              on:click={handleStop}
              disabled={isStopping}
              class="bg-rose-600 hover:bg-rose-500 active:bg-rose-700 text-white font-black text-sm sm:text-base px-8 py-3 rounded-2xl shadow-xl shadow-rose-950/50 flex items-center space-x-2.5 transition-transform active:scale-95 disabled:opacity-50"
            >
              <span class="w-3 h-3 rounded-sm bg-white inline-block animate-pulse"></span>
              <span>{isStopping ? 'Stopping...' : 'Stop Game'}</span>
              <span class="text-[10px] font-mono bg-black/30 px-1.5 py-0.5 rounded font-bold ml-1">Running</span>
            </button>
          {:else if $selectedGame.isReady || ($selectedGame.isPatched && ($selectedGame.lutrisRegistered || $selectedGame.operatingMode === 'standalone'))}
            <!-- Primary LAUNCH button -->
            <button
              type="button"
              on:click={handleLaunch}
              disabled={isLaunching}
              class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-sm sm:text-base px-8 py-3 rounded-2xl shadow-xl flex items-center space-x-2.5 transition-transform active:scale-95 disabled:opacity-50"
            >
              <svg class="w-5 h-5 fill-current" viewBox="0 0 24 24">
                <path d="M8 5v14l11-7z"/>
              </svg>
              <span>{isLaunching ? 'Starting...' : 'Launch'}</span>
              <span class="text-[10px] font-mono bg-black/10 px-1.5 py-0.5 rounded font-bold ml-1">↵</span>
            </button>
          {:else}
            <!-- Primary PREPARE button -->
            <button
              type="button"
              on:click={handlePrepare}
              class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-sm sm:text-base px-8 py-3 rounded-2xl shadow-xl flex items-center space-x-2.5 transition-transform active:scale-95"
            >
              <svg class="w-5 h-5 text-black" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <span>Prepare</span>
              <span class="text-[10px] font-mono bg-black/10 px-1.5 py-0.5 rounded font-bold ml-1">↵</span>
            </button>
          {/if}

          {#if $selectedGame.hasUpdate || $selectedGame.hasUpdatesOrIssues}
            <!-- UPDATE GAME button -->
            <button
              type="button"
              on:click={handleUpdateGame}
              class="bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-black text-sm sm:text-base px-6 py-3 rounded-2xl shadow-xl shadow-emerald-950/50 flex items-center space-x-2 transition-transform active:scale-95"
              title="Download delta depot files and apply updates"
            >
              <svg class="w-5 h-5 fill-current" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v7.586l2.293-2.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 111.414-1.414L9 11.586V4a1 1 0 011-1zM4 16a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1z" clip-rule="evenodd" />
              </svg>
              <span>Update Game</span>
            </button>
          {/if}

          <!-- Cog Icon (Manage Game Drawer / X) -->
          <button
            type="button"
            on:click={() => { $showInspector = true; }}
            class="bg-[#181822] hover:bg-[#222230] text-slate-200 hover:text-white p-3 rounded-2xl border border-[#2b2b3c] shadow-sm flex items-center justify-center relative transition-colors active:scale-95"
            title="Manage Game (X)"
          >
            <svg class="w-5 h-5 text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <span class="absolute -top-1 -right-1 text-[9px] font-mono text-slate-400 bg-[#252535] px-1 rounded border border-[#333348]">X</span>
          </button>

          <!-- Open Directory -->
          <button
            type="button"
            on:click={handleOpenFolder}
            class="bg-[#181822] hover:bg-[#222230] text-slate-300 hover:text-white p-3 rounded-2xl border border-[#2b2b3c] transition-colors"
            title="Open Directory in File Manager"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </button>
        </div>
      </div>
    </div>

  {:else}
    <!-- Clean Welcoming Empty Stage -->
    <div class="max-w-md mx-auto text-center space-y-4 animate-fade-in">
      <div class="w-16 h-16 rounded-3xl bg-[#14141e] border border-[#262638] flex items-center justify-center mx-auto text-slate-400 shadow-xl">
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 10l-2 1m0 0l-2-1m2 1v2.5M20 7l-2 1m2-1l-2-1m2 1v2.5M14 4l-2-1-2 1M4 7l2-1M4 7l2 1M4 7v2.5M12 21l-2-1m2 1l2-1m-2 1v-2.5M6 18l-2-1v-2.5M18 18l2-1v-2.5" />
        </svg>
      </div>

      <h2 class="text-xl font-black text-white">Shelf Empty</h2>
      <p class="text-xs text-slate-400 leading-relaxed">
        Select a game folder or download games to start managing.
      </p>

      <div class="flex flex-wrap items-center justify-center gap-2.5 pt-2">
        <button
          type="button"
          on:click={handleBrowseLibrary}
          class="bg-[#1c1c28] hover:bg-[#262636] border border-[#303042] text-white font-bold text-xs px-4 py-2.5 rounded-xl transition-colors"
        >
          Select Folder
        </button>
        <button
          type="button"
          on:click={() => { $activeView = 'downloads'; }}
          class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-4 py-2.5 rounded-xl transition-colors shadow-lg"
        >
          Download Game
        </button>
        <button
          type="button"
          on:click={importGame}
          class="bg-[#14141c] hover:bg-[#1e1e28] border border-[#262634] text-slate-300 font-bold text-xs px-4 py-2.5 rounded-xl transition-colors"
        >
          Import Folder
        </button>
      </div>
    </div>
  {/if}
</div>
