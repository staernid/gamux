<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';

  export let game = null;

  const dispatch = createEventDispatcher();

  let targetStore = 'Steam'; // 'Steam' | 'GOG' | 'Standalone'
  let query = '';
  let customId = '';
  let candidates = [];
  let isSearching = false;
  let isLinking = false;
  let selectedCandidate = null;

  $: gameName = game?.name || '';
  $: gameDir = game?.gameDir || '';

  onMount(() => {
    const currStore = game?.store || 'Steam';
    if (currStore === 'GOG') targetStore = 'GOG';
    else if (currStore === 'Standalone' || currStore === 'Custom') targetStore = 'Standalone';
    else targetStore = 'Steam';

    query = gameName || '';
    customId = String(game?.appId || '');
    if (customId === '0') customId = '';

    if (targetStore !== 'Standalone' && query.trim()) {
      handleSearch();
    }
  });

  async function handleSearch() {
    if (!query.trim()) return;
    isSearching = true;
    selectedCandidate = null;
    try {
      const results = await AppAPI.searchStoreGames(query.trim());
      candidates = (results || []).filter(c => {
        if (targetStore === 'Steam') return c.store === 'steam';
        if (targetStore === 'GOG') return c.store === 'gog';
        return true;
      });
    } catch (err) {
      console.warn('Failed to search store games:', err);
      candidates = [];
    } finally {
      isSearching = false;
    }
  }

  function handleSelectCandidate(cand) {
    selectedCandidate = cand;
    customId = cand.id;
  }

  async function handleApplyLink() {
    if (!gameDir) return;
    isLinking = true;
    try {
      const idToLink = targetStore === 'Standalone' ? '0' : customId.trim();
      const updated = await AppAPI.linkGameMetadata(gameDir, targetStore, idToLink);
      notificationStore.items.add({
        type: 'success',
        title: 'Metadata Linked',
        message: `Linked ${gameName} to ${targetStore} ${idToLink ? `(#${idToLink})` : ''}`
      });
      dispatch('updated', updated);
      dispatch('close');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Linking Failed',
        message: err?.message || String(err)
      });
    } finally {
      isLinking = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      dispatch('close');
    }
  }
</script>

<div
  role="dialog"
  aria-modal="true"
  tabindex="-1"
  on:keydown={handleKeydown}
  class="fixed inset-0 z-50 bg-[#09090c]/92 backdrop-blur-2xl flex flex-col p-6 sm:p-10 select-none animate-in fade-in duration-150 outline-none"
>
  <!-- Header Bar -->
  <div class="flex items-center justify-between max-w-4xl w-full mx-auto mb-6 flex-shrink-0">
    <div>
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2">
        <span>Change Metadata Provider:</span>
        <span class="text-sky-400 truncate max-w-md">{gameName}</span>
      </h2>
      <p class="text-xs text-slate-400 mt-0.5">Switch store origin, re-link Steam AppID / GOG metadata, or configure as a Standalone game.</p>
    </div>

    <!-- Close Hint -->
    <div class="flex items-center space-x-3">
      <button
        type="button"
        on:click={() => dispatch('close')}
        class="px-3 py-1.5 rounded-xl bg-[#1a1a24] hover:bg-[#252535] border border-[#2b2b3c] text-slate-300 hover:text-white text-xs font-mono transition-colors flex items-center space-x-1.5"
      >
        <span class="px-1.5 py-0.5 rounded bg-[#161622] border border-[#262638] text-slate-400 font-bold text-[10px]">Esc / B</span>
        <span class="text-slate-400">Close</span>
      </button>
    </div>
  </div>

  <!-- Store Provider Tabs -->
  <div class="max-w-4xl w-full mx-auto mb-6 flex items-center justify-start gap-2 flex-shrink-0">
    <button
      type="button"
      on:click={() => { targetStore = 'Steam'; if (query.trim()) handleSearch(); }}
      class="px-4 py-2 rounded-xl font-bold text-xs transition-all flex items-center space-x-2 {targetStore === 'Steam' ? 'bg-blue-600 text-white shadow-lg shadow-blue-900/40 border border-blue-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-white'}"
    >
      <span>Steam</span>
      <span class="text-[10px] bg-black/30 px-1.5 py-0.5 rounded font-mono">AppID</span>
    </button>
    <button
      type="button"
      on:click={() => { targetStore = 'GOG'; if (query.trim()) handleSearch(); }}
      class="px-4 py-2 rounded-xl font-bold text-xs transition-all flex items-center space-x-2 {targetStore === 'GOG' ? 'bg-purple-600 text-white shadow-lg shadow-purple-900/40 border border-purple-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-white'}"
    >
      <span>GOG.com</span>
      <span class="text-[10px] bg-black/30 px-1.5 py-0.5 rounded font-mono">GOG ID</span>
    </button>
    <button
      type="button"
      on:click={() => { targetStore = 'Standalone'; }}
      class="px-4 py-2 rounded-xl font-bold text-xs transition-all flex items-center space-x-2 {targetStore === 'Standalone' ? 'bg-emerald-600 text-white shadow-lg shadow-emerald-900/40 border border-emerald-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-white'}"
    >
      <span>Standalone / DRM-Free</span>
      <span class="text-[10px] bg-black/30 px-1.5 py-0.5 rounded font-mono">Direct</span>
    </button>
  </div>

  <!-- Main Modal Content Area -->
  <div class="flex-1 max-w-4xl w-full mx-auto overflow-y-auto pr-2 custom-scrollbar space-y-4">
    {#if targetStore === 'Standalone'}
      <!-- Standalone Explanation Card -->
      <div class="bg-[#12121c] border border-[#262638] p-6 rounded-2xl space-y-4">
        <div class="flex items-start space-x-3.5">
          <div class="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 flex-shrink-0">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <div class="space-y-1">
            <h3 class="text-base font-bold text-white">Standalone / Custom Non-Store Configuration</h3>
            <p class="text-xs text-slate-400 leading-relaxed">
              Configuring this installation as Standalone disables Steamworks emulator injections (Goldberg Emulator / SteamStub unpacking requirements). The game will be executed directly via your selected runner (Wine, Proton, or Native Linux).
            </p>
          </div>
        </div>

        <div class="pt-3 border-t border-[#202030] flex justify-end">
          <button
            type="button"
            on:click={handleApplyLink}
            disabled={isLinking}
            class="px-6 py-2.5 bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-bold text-xs rounded-xl transition-all shadow-lg shadow-emerald-950/50 disabled:opacity-50"
          >
            {isLinking ? 'Updating...' : 'Set as Standalone Game'}
          </button>
        </div>
      </div>
    {:else}
      <!-- Search & Manual ID Row -->
      <div class="bg-[#12121c] border border-[#262638] p-4 rounded-2xl space-y-3">
        <div class="flex flex-col sm:flex-row items-center gap-3">
          <div class="relative flex-1 w-full">
            <input
              type="text"
              bind:value={query}
              on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
              placeholder={`Search ${targetStore} title...`}
              class="w-full bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-xl px-4 py-2 text-xs text-white outline-none font-mono placeholder:text-slate-500"
            />
          </div>
          <button
            type="button"
            on:click={handleSearch}
            disabled={isSearching}
            class="w-full sm:w-auto px-5 py-2 bg-[#1e1e2c] hover:bg-[#28283c] text-white text-xs font-bold rounded-xl border border-[#34344c] transition-colors flex-shrink-0"
          >
            {isSearching ? 'Searching...' : 'Search'}
          </button>
        </div>

        <!-- Manual ID Input & Apply Button -->
        <div class="flex items-center gap-3 pt-2 border-t border-[#1e1e2c]">
          <span class="text-[11px] font-bold text-slate-400 flex-shrink-0">
            {targetStore === 'Steam' ? 'Steam AppID:' : 'GOG ID / Slug:'}
          </span>
          <input
            type="text"
            bind:value={customId}
            placeholder={targetStore === 'Steam' ? 'e.g. 1091500' : 'e.g. the_witcher_3_wild_hunt'}
            class="flex-1 bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-lg px-3 py-1 text-xs text-white outline-none font-mono placeholder:text-slate-600"
          />
          <button
            type="button"
            on:click={handleApplyLink}
            disabled={!customId.trim() || isLinking}
            class="px-5 py-1.5 bg-sky-600 hover:bg-sky-500 active:bg-sky-700 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition-colors flex-shrink-0 shadow"
          >
            {isLinking ? 'Linking...' : 'Link Game'}
          </button>
        </div>
      </div>

      <!-- Candidate Results List -->
      {#if isSearching}
        <div class="h-48 flex flex-col items-center justify-center text-slate-400 space-y-2 font-mono text-xs">
          <div class="w-6 h-6 border-2 border-sky-500 border-t-transparent rounded-full animate-spin"></div>
          <p>Querying {targetStore} catalogue...</p>
        </div>
      {:else if candidates.length > 0}
        <div class="space-y-2 pb-6">
          <span class="text-[10px] font-bold uppercase tracking-wider text-slate-400 block px-1">
            Matching {targetStore} Titles ({candidates.length})
          </span>
          <div class="grid grid-cols-1 gap-2">
            {#each candidates as cand}
              {@const isSelected = selectedCandidate?.id === cand.id || customId === cand.id}
              <div
                class="flex items-center justify-between p-3 rounded-xl bg-[#12121c] border transition-all duration-150 {isSelected ? 'border-sky-500 bg-sky-950/20' : 'border-[#222232] hover:border-slate-500'}"
              >
                <div class="flex items-center space-x-3 min-w-0">
                  {#if cand.poster_url}
                    <img src={cand.poster_url} alt="" class="w-10 h-14 object-cover rounded-lg bg-black/40 flex-shrink-0 border border-white/5" />
                  {/if}
                  <div class="min-w-0">
                    <h4 class="text-xs font-bold text-white truncate" title={cand.name}>{cand.name}</h4>
                    <div class="flex items-center space-x-2 text-[10px] font-mono text-slate-400 mt-0.5">
                      <span>ID: <strong class="text-slate-200">{cand.id}</strong></span>
                      {#if cand.release_date}
                        <span>• Released: {cand.release_date}</span>
                      {/if}
                    </div>
                  </div>
                </div>

                <div class="flex items-center space-x-2 flex-shrink-0 pl-3">
                  <button
                    type="button"
                    on:click={() => handleSelectCandidate(cand)}
                    class="px-3 py-1 bg-[#1e1e2c] hover:bg-[#28283c] text-slate-200 hover:text-white text-xs font-bold rounded-lg border border-[#323246] transition-colors"
                  >
                    Select ID
                  </button>
                  <button
                    type="button"
                    on:click={() => { handleSelectCandidate(cand); handleApplyLink(); }}
                    class="px-3.5 py-1 bg-sky-600 hover:bg-sky-500 text-white text-xs font-bold rounded-lg transition-colors shadow"
                  >
                    Link & Apply
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>
