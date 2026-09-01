<script>
  import { createEventDispatcher } from 'svelte';
  import { libraryPath } from '../../../stores/library.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import { AppAPI } from '../../../api/app.js';

  export let isSubmitting = false;

  const dispatch = createEventDispatcher();

  let searchQuery = '';
  let selectedID = '';
  let selectedName = '';
  let selectedStore = 'steam';
  let selectedPlatform = 'win64';
  let targetDir = '';
  let isSearching = false;
  let searchResults = [];

  $: if (!targetDir && $libraryPath) {
    targetDir = $libraryPath;
  }

  async function handleSearch() {
    const q = searchQuery.trim();
    if (!q) return;

    if (!isNaN(q) && parseInt(q, 10) > 0) {
      selectedID = q;
      selectedName = `AppID ${q}`;
      selectedStore = 'steam';
      searchResults = [];
      const safeName = `AppID_${q}`;
      if ($libraryPath) {
        targetDir = `${$libraryPath}/${safeName}`;
      } else {
        targetDir = `./${safeName}`;
      }

      isSearching = true;
      try {
        const candidates = await AppAPI.searchStoreGames(q);
        const match = (candidates || []).find(c => String(c.id || c.AppID) === q);
        if (match) {
          selectCandidate(match);
        }
      } catch {
        // Ignored
      } finally {
        isSearching = false;
      }
      return;
    }

    isSearching = true;
    try {
      const candidates = await AppAPI.searchStoreGames(q);
      searchResults = (candidates || []).filter(c => !c.store || c.store.toLowerCase() === 'steam');
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
    selectedID = String(candidate.id || candidate.AppID);
    selectedName = candidate.name || candidate.Name;
    selectedStore = 'steam';
    searchQuery = selectedName;
    searchResults = [];
    const safeName = selectedName.replace(/[™®©]/g, '').replace(/[:/\\?*<>|"]/g, '').replace(/\s+/g, ' ').trim() || selectedName;
    const isMusic = candidate.type === 'music' || candidate.type === 'soundtrack';
    if (isMusic) {
      targetDir = `~/Music/${safeName}`;
    } else if ($libraryPath) {
      targetDir = `${$libraryPath}/${safeName}`;
    } else {
      targetDir = `./${safeName}`;
    }
  }

  function clearSelection() {
    selectedID = '';
    selectedName = '';
    searchQuery = '';
    searchResults = [];
  }

  async function browseTargetDir() {
    const selected = await AppAPI.selectDirectory('Select Download Destination Folder', targetDir || $libraryPath);
    if (selected) {
      targetDir = selected;
    }
  }

  function handleStartDownload() {
    const appID = parseInt(selectedID.trim(), 10);
    const dest = targetDir.trim();

    if (!appID || isNaN(appID)) {
      notificationStore.items.add({
        type: 'error',
        title: 'Identifier Required',
        message: 'Please enter or select a valid Steam AppID or Game Title.'
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

    dispatch('start', {
      app_id: appID,
      game_name: selectedName || `Game ${appID}`,
      target_dir: dest,
      platform: selectedPlatform,
      auto_apply: true,
      store_type: selectedStore || 'steam',
    });
  }
</script>

<div class="bg-[#121218] border border-[#262638] rounded-2xl p-5 shadow-2xl space-y-4 animate-fade-in">
  <div class="flex items-center justify-between border-b border-[#1e1e2c] pb-2.5">
    <h3 class="font-bold text-white text-xs uppercase tracking-wider">Download Title from Store Provider</h3>
    <button type="button" on:click={() => dispatch('close')} class="text-slate-400 hover:text-white text-xs cursor-pointer">Close</button>
  </div>

  <!-- Search Field -->
  <div class="relative">
    <label for="dlSearchInput" class="block text-slate-400 mb-1 font-bold text-[10px] uppercase tracking-wider font-mono">
      Search Game Title or Steam AppID
    </label>
    <div class="flex space-x-2">
      <input
        id="dlSearchInput"
        type="text"
        bind:value={searchQuery}
        placeholder="e.g. Hades, Balatro, Cyberpunk 2077..."
        on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
        class="flex-1 bg-[#09090d] border border-[#242434] rounded-xl px-3.5 py-2 text-slate-100 outline-none font-mono focus:border-cyan-400 text-xs"
      />
      <button
        type="button"
        on:click={handleSearch}
        disabled={isSearching || !searchQuery.trim()}
        class="bg-[#1c1c28] hover:bg-[#28283a] border border-[#2f2f42] text-slate-200 font-bold px-4 py-2 rounded-xl transition text-xs flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isSearching}
          <span>Searching...</span>
        {:else}
          <span>Search</span>
        {/if}
      </button>
    </div>

    <!-- Search Candidates Dropdown -->
    {#if searchResults.length > 0}
      <div class="absolute left-0 right-0 top-full mt-2 bg-[#14141e] border border-[#2a2a3c] rounded-xl shadow-2xl z-30 overflow-hidden divide-y divide-[#1e1e2c] max-h-60 overflow-y-auto">
        {#each searchResults as candidate}
          <button
            type="button"
            on:click={() => selectCandidate(candidate)}
            class="w-full text-left p-2.5 hover:bg-[#1e1e2c] flex items-center space-x-3 transition group cursor-pointer"
          >
            <div class="w-8 h-11 rounded bg-black/40 overflow-hidden flex-shrink-0">
              {#if candidate.poster_url}
                <img src={candidate.poster_url} alt="" class="w-full h-full object-cover" />
              {/if}
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-bold text-slate-100 group-hover:text-cyan-400 truncate text-xs">{candidate.name}</div>
              <div class="text-[10px] text-slate-500 font-mono mt-0.5">
                <span class="px-1 py-0.2 rounded bg-black/40 text-slate-400 uppercase">{candidate.store || 'steam'}</span>
                ID: {candidate.id}
              </div>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Config Details Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 pt-1">
    <div>
      <label for="dlDestFolderInput" class="block text-slate-400 mb-1 font-bold text-[10px] uppercase tracking-wider font-mono">Destination Folder</label>
      <div class="flex space-x-1.5">
        <input
          id="dlDestFolderInput"
          type="text"
          bind:value={targetDir}
          class="flex-1 bg-[#09090d] border border-[#242434] rounded-xl px-3 py-1.5 text-slate-200 outline-none font-mono text-xs"
        />
        <button
          type="button"
          on:click={browseTargetDir}
          class="bg-[#1c1c28] hover:bg-[#28283a] border border-[#2e2e40] text-slate-300 px-3 py-1.5 rounded-xl text-xs font-bold cursor-pointer"
        >
          Browse
        </button>
      </div>
    </div>

    <div>
      <label for="dlPlatformSelect" class="block text-slate-400 mb-1 font-bold text-[10px] uppercase tracking-wider font-mono">Platform Target</label>
      <select
        id="dlPlatformSelect"
        bind:value={selectedPlatform}
        class="w-full bg-[#09090d] border border-[#242434] rounded-xl px-3 py-2 text-slate-200 outline-none text-xs font-mono cursor-pointer"
      >
        <option value="win64">Windows (64-bit)</option>
        <option value="linux">Linux Native</option>
        <option value="macos">macOS Native</option>
        <option value="all">All Available Depots</option>
      </select>
    </div>
  </div>

  <div class="flex justify-end space-x-2.5 pt-2 border-t border-[#1e1e2c]">
    <button
      type="button"
      on:click={clearSelection}
      class="px-3.5 py-1.5 rounded-xl bg-transparent hover:bg-slate-800 text-slate-400 hover:text-white font-bold text-xs cursor-pointer"
    >
      Reset
    </button>
    <button
      type="button"
      on:click={handleStartDownload}
      disabled={!selectedID || isSubmitting}
      class="bg-white hover:bg-neutral-200 disabled:opacity-40 text-black font-black px-5 py-1.5 rounded-xl transition text-xs shadow-sm cursor-pointer"
    >
      {isSubmitting ? 'Starting...' : 'Start Download'}
    </button>
  </div>
</div>
