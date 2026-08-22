<script>
  import { libraryPath, scanLibrary, games, activeView } from '../../stores/library.js';
  import { taskQueue } from '../../stores/queue.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { normalizeGame } from '../../utils/game.js';
  import { AppAPI } from '../../api/app.js';

  let searchQuery = '';
  let selectedAppID = '';
  let selectedPlatform = 'win64';
  let targetDir = '';
  let autoApply = true;
  let isSearching = false;
  let searchResults = [];

  $: if (!targetDir && $libraryPath) {
    targetDir = $libraryPath;
  }

  async function handleSearch() {
    const q = searchQuery.trim();
    if (!q) return;

    if (!isNaN(q) && parseInt(q, 10) > 0) {
      selectedAppID = q;
      searchResults = [];
      return;
    }

    isSearching = true;
    try {
      const candidates = await AppAPI.searchSteamGames(q);
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
    selectedAppID = String(candidate.AppID);
    searchQuery = candidate.Name;
    searchResults = [];
    if ($libraryPath) {
      const safeName = candidate.Name.replace(/[^\w\s-]/g, '').trim();
      targetDir = `${$libraryPath}/${safeName}`;
    }
  }

  async function browseTargetDir() {
    const selected = await AppAPI.selectDirectory('Select Download Destination Folder', targetDir || $libraryPath);
    if (selected) {
      targetDir = selected;
    }
  }

  function startDownload() {
    const appID = parseInt(selectedAppID.trim(), 10);
    const dest = targetDir.trim();

    if (!appID || isNaN(appID)) {
      notificationStore.items.add({
        type: 'error',
        title: 'Invalid AppID',
        message: 'Please enter or search for a valid Steam AppID.'
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

    // Return to shelf or activity
    activeView.set('activity');

    taskQueue.enqueue({
      type: 'download',
      title: `Download: AppID ${appID}`,
      gameName: searchQuery || `Steam AppID ${appID}`,
      gameDir: dest,
      run: async () => {
        const res = await AppAPI.downloadGame({
          app_id: appID,
          target_dir: dest,
          platform: selectedPlatform,
          auto_apply: autoApply,
        });
        return typeof res === 'string' ? res : `Downloaded game files for AppID ${appID}`;
      },
      onSuccess: async () => {
        try {
          const raw = await AppAPI.inspectGameEx(dest, 0);
          const game = normalizeGame(raw);
          if (game) {
            games.update(list => {
              const idx = list.findIndex(g => g.GameDir === game.GameDir);
              if (idx >= 0) {
                const copy = [...list];
                copy[idx] = game;
                return copy;
              }
              return [game, ...list];
            });
          }
        } catch {
          if ($libraryPath) {
            scanLibrary();
          }
        }
      }
    });
  }
</script>

<div class="flex-1 overflow-y-auto p-6 sm:p-10 flex flex-col justify-center max-w-2xl mx-auto w-full animate-fade-in text-xs sm:text-sm">
  <div class="bg-[#121218] border border-[#242434] p-6 sm:p-8 rounded-2xl shadow-2xl space-y-5">
    <div class="border-b border-[#20202c] pb-4 flex items-center justify-between">
      <div>
        <h2 class="text-xl font-black text-white">SteamPipe Downloader</h2>
        <p class="text-xs text-slate-400 mt-0.5">Acquire official Steam depots directly from CDNs with auto-decryption</p>
      </div>
      <button
        type="button"
        on:click={() => { $activeView = 'library'; }}
        class="text-xs font-bold text-slate-400 hover:text-white px-2.5 py-1 rounded-lg bg-[#1a1a24] border border-[#2a2a38]"
      >
        ✕ Shelf
      </button>
    </div>

    <!-- Search Input -->
    <div class="relative">
      <label for="dlSearchInput" class="block text-slate-300 mb-1 font-bold text-xs uppercase tracking-wider">
        Search Steam Title or Enter AppID
      </label>
      <div class="flex space-x-2">
        <input
          id="dlSearchInput"
          type="text"
          bind:value={searchQuery}
          placeholder="e.g. Hades or 1145360"
          on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
          class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3.5 py-2.5 text-slate-100 outline-none font-mono focus:border-white text-xs sm:text-sm"
        />
        <button
          type="button"
          on:click={handleSearch}
          disabled={isSearching}
          class="bg-[#20202c] hover:bg-[#2c2c3c] text-white px-4 py-2.5 rounded-xl font-bold transition-colors disabled:opacity-50 border border-[#343446]"
        >
          {isSearching ? '...' : 'Search'}
        </button>
      </div>

      <!-- Search Results Dropdown -->
      {#if searchResults.length > 0}
        <div class="absolute left-0 right-0 top-16 bg-[#161622] border border-[#303042] rounded-xl shadow-2xl max-h-52 overflow-y-auto z-50 divide-y divide-[#222230]">
          {#each searchResults as candidate}
            <button
              type="button"
              on:click={() => selectCandidate(candidate)}
              class="w-full text-left p-3 hover:bg-[#202030] flex items-center justify-between text-xs transition-colors"
            >
              <span class="font-bold text-white truncate mr-2">{candidate.Name}</span>
              <span class="font-mono text-slate-400 flex-shrink-0">AppID {candidate.AppID}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Target AppID & Platform -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
      <div>
        <label for="dlAppIdInput" class="block text-slate-300 mb-1 font-bold text-xs uppercase tracking-wider">
          Target Steam AppID
        </label>
        <input
          id="dlAppIdInput"
          type="text"
          bind:value={selectedAppID}
          placeholder="123456"
          class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3.5 py-2 text-slate-100 outline-none font-mono font-bold focus:border-white"
        />
      </div>
      <div>
        <label for="dlPlatformSel" class="block text-slate-300 mb-1 font-bold text-xs uppercase tracking-wider">
          Architecture / Platform
        </label>
        <select
          id="dlPlatformSel"
          bind:value={selectedPlatform}
          class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3.5 py-2 text-slate-100 outline-none focus:border-white"
        >
          <option value="win64">Windows 64-bit (Wine/Proton)</option>
          <option value="linux">Linux Native Binary</option>
          <option value="all">All Available Depots</option>
        </select>
      </div>
    </div>

    <!-- Destination Folder -->
    <div>
      <label for="dlTargetDirInput" class="block text-slate-300 mb-1 font-bold text-xs uppercase tracking-wider">
        Destination Folder
      </label>
      <div class="flex space-x-2">
        <input
          id="dlTargetDirInput"
          type="text"
          bind:value={targetDir}
          placeholder="/home/user/Games/MyGame"
          class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3.5 py-2 text-slate-100 outline-none font-mono focus:border-white text-xs"
        />
        <button
          type="button"
          on:click={browseTargetDir}
          class="bg-[#20202c] hover:bg-[#2c2c3c] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#343446] flex-shrink-0"
        >
          Browse
        </button>
      </div>
    </div>

    <!-- Checkbox Option -->
    <div class="pt-1">
      <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
        <input type="checkbox" bind:checked={autoApply} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
        <span>Automatically apply Goldberg Emulator & Lutris registration upon completion</span>
      </label>
    </div>

    <!-- Actions -->
    <div class="flex items-center justify-end space-x-3 pt-3 border-t border-[#20202c]">
      <button
        type="button"
        on:click={() => { $activeView = 'library'; }}
        class="text-xs font-bold text-slate-400 hover:text-white px-3 py-2"
      >
        Cancel
      </button>
      <button
        type="button"
        on:click={startDownload}
        class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-xs px-6 py-2.5 rounded-xl transition-colors shadow-lg"
      >
        Start Download
      </button>
    </div>
  </div>
</div>
