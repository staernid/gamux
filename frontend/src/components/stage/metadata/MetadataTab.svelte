<script>
  import { createEventDispatcher } from 'svelte';

  export let targetStore = 'steam';
  export let providers = [];
  export let metaQuery = '';
  export let customId = '';
  export let metaCandidates = [];
  export let isSearchingMeta = false;
  export let isLinkingMeta = false;
  export let selectedMetaCandidate = null;

  const dispatch = createEventDispatcher();

  function selectStore(storeId) {
    targetStore = storeId;
    dispatch('storeChange', storeId);
  }

  function handleSearch() {
    dispatch('search');
  }

  function handleSelectCandidate(cand) {
    dispatch('selectCandidate', cand);
  }

  function handleApplyLink() {
    dispatch('applyLink');
  }
</script>

<div class="space-y-4">
  <!-- Store Provider Pills -->
  <div class="flex items-center gap-2 overflow-x-auto custom-scrollbar pb-1">
    {#each providers as prov}
      {@const isActive = targetStore === prov.id}
      <button
        type="button"
        on:click={() => selectStore(prov.id)}
        class="px-3.5 py-1.5 rounded-xl font-bold text-xs transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {isActive ? 'bg-sky-600 text-white shadow-md border border-sky-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-white'}"
      >
        <span>{prov.display_name}</span>
        <span class="text-3xs bg-black/30 px-1.5 py-0.5 rounded font-mono">{prov.badge}</span>
      </button>
    {/each}
  </div>

  {#if targetStore === 'standalone'}
    <div class="bg-[#12121c] border border-[#262638] p-6 rounded-2xl space-y-4">
      <div class="flex items-start space-x-3.5">
        <div class="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 flex-shrink-0">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <div class="space-y-1">
          <h3 class="text-base font-bold text-white">Standalone / DRM-Free Mode</h3>
          <p class="text-xs text-slate-400 leading-relaxed">
            Configuring this game as Standalone executes the binary directly via your selected runner without injecting Steamworks emulators or looking up Steam DRM keys.
          </p>
        </div>
      </div>
      <div class="pt-3 border-t border-[#202030] flex justify-end">
        <button
          type="button"
          on:click={handleApplyLink}
          disabled={isLinkingMeta}
          class="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-bold text-xs rounded-xl transition-all shadow-lg cursor-pointer"
        >
          {isLinkingMeta ? 'Updating...' : 'Set as Standalone'}
        </button>
      </div>
    </div>
  {:else}
    <!-- Search Row -->
    <div class="bg-[#12121c] border border-[#262638] p-4 rounded-2xl space-y-3">
      <div class="flex flex-col sm:flex-row items-center gap-3">
        <div class="relative flex-1 w-full">
          <input
            type="text"
            bind:value={metaQuery}
            on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
            placeholder={`Search ${targetStore.toUpperCase()} catalogue...`}
            class="w-full bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-xl px-4 py-2 text-xs text-white outline-none font-mono placeholder:text-slate-500"
          />
        </div>
        <button
          type="button"
          on:click={handleSearch}
          disabled={isSearchingMeta}
          class="w-full sm:w-auto px-5 py-2 bg-[#1e1e2c] hover:bg-[#28283c] text-white text-xs font-bold rounded-xl border border-[#34344c] transition-colors flex-shrink-0 cursor-pointer"
        >
          {isSearchingMeta ? 'Searching...' : 'Search'}
        </button>
      </div>

      <!-- Manual ID Input & Apply Button -->
      <div class="flex items-center gap-3 pt-2 border-t border-[#1e1e2c]">
        <span class="text-2xs font-bold text-slate-400 flex-shrink-0">
          {targetStore === 'steam' ? 'Steam AppID:' : targetStore === 'gog' ? 'GOG ID / Slug:' : 'RAWG Game ID / Slug:'}
        </span>
        <input
          type="text"
          bind:value={customId}
          placeholder={targetStore === 'steam' ? 'e.g. 1091500' : targetStore === 'gog' ? 'e.g. the_witcher_3_wild_hunt' : 'e.g. portal-2 or 4200'}
          class="flex-1 bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-lg px-3 py-1 text-xs text-white outline-none font-mono placeholder:text-slate-600"
        />
        <button
          type="button"
          on:click={handleApplyLink}
          disabled={!customId.trim() || isLinkingMeta}
          class="px-5 py-1.5 bg-sky-600 hover:bg-sky-500 active:bg-sky-700 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition-colors flex-shrink-0 shadow cursor-pointer"
        >
          {isLinkingMeta ? 'Linking...' : 'Link Game'}
        </button>
      </div>
    </div>

    <!-- Candidate Results List -->
    {#if isSearchingMeta}
      <div class="h-40 flex flex-col items-center justify-center text-slate-400 space-y-2 font-mono text-xs">
        <div class="w-6 h-6 border-2 border-sky-500 border-t-transparent rounded-full animate-spin"></div>
        <p>Querying {targetStore} catalogue...</p>
      </div>
    {:else if metaCandidates.length > 0}
      <div class="space-y-2 pb-6">
        <span class="text-3xs font-bold uppercase tracking-wider text-slate-400 block px-1">
          Matching {targetStore} Titles ({metaCandidates.length})
        </span>
        <div class="grid grid-cols-1 gap-2">
          {#each metaCandidates as cand}
            {@const isSelected = selectedMetaCandidate?.id === cand.id || customId === cand.id}
            <div class="flex items-center justify-between p-3 rounded-xl bg-[#12121c] border transition-all duration-150 {isSelected ? 'border-sky-500 bg-sky-950/20' : 'border-[#222232] hover:border-slate-500'}">
              <div class="flex items-center space-x-3 min-w-0">
                {#if cand.poster_url}
                  <img src={cand.poster_url} alt="" class="w-10 h-14 object-cover rounded-lg bg-black/40 flex-shrink-0 border border-white/5" />
                {/if}
                <div class="min-w-0">
                  <h4 class="text-xs font-bold text-white truncate" title={cand.name}>{cand.name}</h4>
                  <div class="flex items-center space-x-2 text-3xs font-mono text-slate-400 mt-0.5">
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
                  class="px-3 py-1 bg-[#1e1e2c] hover:bg-[#28283c] text-slate-200 hover:text-white text-xs font-bold rounded-lg border border-[#323246] transition-colors cursor-pointer"
                >
                  Select ID
                </button>
                <button
                  type="button"
                  on:click={() => { handleSelectCandidate(cand); handleApplyLink(); }}
                  class="px-3.5 py-1 bg-sky-600 hover:bg-sky-500 text-white text-xs font-bold rounded-lg transition-colors shadow cursor-pointer"
                >
                  Link & Sync Art
                </button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>
