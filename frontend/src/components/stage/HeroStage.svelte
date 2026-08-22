<script>
  import { selectedGame, queueApplyGame, showInspector, libraryPath, scanLibrary, importGame, activeView } from '../../stores/library.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { formatBytes } from '../../utils/format.js';
  import { isSteamApp } from '../../utils/game.js';
  import { AppAPI } from '../../api/app.js';
  import PosterArt from '../common/PosterArt.svelte';

  let isLaunching = false;

  async function handleLaunch() {
    if (!$selectedGame) return;
    const slug = $selectedGame.AppID || $selectedGame.Name.toLowerCase().replace(/[^\w-]/g, '-');
    isLaunching = true;
    try {
      await AppAPI.launchLutrisGame(slug);
      notificationStore.items.add({
        type: 'success',
        title: 'Launching',
        message: `Starting ${$selectedGame.Name}...`
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

  function handlePrepare() {
    if (!$selectedGame) return;
    queueApplyGame($selectedGame);
  }

  async function handleOpenFolder() {
    if (!$selectedGame?.GameDir) return;
    try {
      await AppAPI.openFolder($selectedGame.GameDir);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Open Folder Failed',
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
          appId={$selectedGame.AppID}
          name={$selectedGame.Name}
          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
        />

        {#if isSteamApp($selectedGame.AppID)}
          <span class="absolute bottom-2 left-2 bg-black/85 backdrop-blur-sm font-mono text-[10px] text-slate-300 px-2 py-0.5 rounded-md font-bold border border-white/10">
            #{$selectedGame.AppID}
          </span>
        {/if}
      </div>

      <!-- Right: Primary Metadata & Concise Handheld Actions -->
      <div class="flex-1 flex flex-col justify-between space-y-5 min-w-0 text-center md:text-left">
        <div>
          <!-- Status Pill Strip -->
          <div class="flex items-center justify-center md:justify-start space-x-2.5 mb-2">
            {#if $selectedGame.isPatched && $selectedGame.lutris_registered}
              <span class="inline-flex items-center space-x-1.5 bg-emerald-950/40 border border-emerald-700/50 text-emerald-300 text-xs px-2.5 py-0.5 rounded-full font-bold">
                <span class="pip pip-ready"></span>
                <span>Ready</span>
              </span>
            {:else if $selectedGame.isPatched}
              <span class="inline-flex items-center space-x-1.5 bg-yellow-950/40 border border-yellow-700/50 text-yellow-300 text-xs px-2.5 py-0.5 rounded-full font-bold">
                <span class="pip pip-update"></span>
                <span>Patched</span>
              </span>
            {:else}
              <span class="inline-flex items-center space-x-1.5 bg-slate-900 border border-slate-700 text-slate-400 text-xs px-2.5 py-0.5 rounded-full font-medium">
                <span class="pip pip-clean"></span>
                <span>Original</span>
              </span>
            {/if}

            {#if $selectedGame.hasUpdatesOrIssues}
              <span class="inline-flex items-center space-x-1 bg-rose-950/40 border border-rose-700/50 text-rose-300 text-xs px-2.5 py-0.5 rounded-full font-bold">
                Update Pending
              </span>
            {/if}
          </div>

          <!-- Game Title -->
          <h1 class="text-2xl sm:text-3xl lg:text-4xl font-black text-white tracking-tight leading-tight" title={$selectedGame.Name}>
            {$selectedGame.Name}
          </h1>

          <!-- Quick Specs -->
          <div class="flex flex-wrap items-center justify-center md:justify-start gap-x-4 gap-y-1.5 text-xs text-slate-400 font-mono mt-2.5">
            <span class="text-slate-200 bg-[#161622] px-2.5 py-0.5 rounded-md border border-[#262638]">
              {$selectedGame.Platform === 'linux' ? '🐧 Linux Native' : '🪟 Windows'}
            </span>
            <span>Size: <strong class="text-white font-semibold">{formatBytes($selectedGame.DiskSizeBytes)}</strong></span>
            {#if $selectedGame.DLCCount}
              <span>DLCs: <strong class="text-slate-200">{$selectedGame.DLCCount}</strong></span>
            {/if}
            {#if $selectedGame.AchievementCount}
              <span>Achievements: <strong class="text-slate-200">{$selectedGame.AchievementCount}</strong></span>
            {/if}
          </div>

          <!-- Directory Path -->
          <div class="text-[11px] font-mono text-slate-500 truncate max-w-xl mt-2 select-text" title={$selectedGame.GameDir}>
            {$selectedGame.GameDir}
          </div>
        </div>

        <!-- Chunky Action Buttons -->
        <div class="flex flex-wrap items-center justify-center md:justify-start gap-3 pt-2">
          {#if $selectedGame.lutris_registered}
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
          on:click={() => { $activeView = 'downloader'; }}
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
