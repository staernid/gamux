<script>
  import { createEventDispatcher } from 'svelte';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { AppAPI } from '../../../api/app.js';
  import { notificationStore } from '../../../stores/notifications.js';

  export let game = null;

  const dispatch = createEventDispatcher();

  const sourceOptions = [
    { id: '', label: 'Auto-Detect', desc: 'Detect from files' },
    { id: 'Steam', label: 'Steam', desc: 'Steamworks + Goldberg' },
    { id: 'GOG', label: 'GOG', desc: 'GOG game layout' },
    { id: 'Epic', label: 'Epic', desc: 'Epic Games manifest' },
    { id: 'Standalone', label: 'Standalone', desc: 'Direct DRM-free' }
  ];

  let searchQuery = $drawerDraft.name || game?.name || '';
  let customAppId = $drawerDraft.appId || (game?.appId && game.appId !== '0' ? String(game.appId) : '') || '';
  let searchResults = [];
  let isSearching = false;

  $: activeStoreType = ($drawerDraft.sourceOverride || game?.source || game?.store || 'Steam').toLowerCase();
  $: storeToSearch = activeStoreType === 'gog' ? 'gog' : 'steam';
  $: hasCustomAppId = !!($drawerDraft.appId || (game?.appId && game.appId !== '0'));

  async function handleStoreSearch() {
    if (!searchQuery.trim()) return;
    isSearching = true;
    try {
      const results = await AppAPI.searchStoreGames(searchQuery.trim());
      searchResults = (results || []).filter(c => {
        const s = (c.store || '').toLowerCase();
        return s === storeToSearch;
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Search Failed',
        message: err?.message || String(err)
      });
      searchResults = [];
    } finally {
      isSearching = false;
    }
  }

  function handleSelectMatch(cand) {
    if (!cand) return;
    drawerDraft.setField('appId', String(cand.id));
    if (!$drawerDraft.sourceOverride) {
      drawerDraft.setField('sourceOverride', cand.store === 'gog' ? 'GOG' : 'Steam');
    }
    customAppId = String(cand.id);
    notificationStore.items.add({
      type: 'success',
      title: 'Store Matched',
      message: `Set physical store identity to ${cand.name} (AppID: ${cand.id})`
    });
  }

  function handleSetManualAppId() {
    if (!customAppId.trim()) return;
    drawerDraft.setField('appId', customAppId.trim());
    if (!$drawerDraft.sourceOverride) {
      drawerDraft.setField('sourceOverride', 'Steam');
    }
    notificationStore.items.add({
      type: 'info',
      title: 'AppID Set',
      message: `Set game AppID to ${customAppId.trim()}`
    });
  }

  function handleResetAppId() {
    drawerDraft.setField('appId', '');
    customAppId = '';
    notificationStore.items.add({
      type: 'info',
      title: 'AppID Cleared',
      message: 'Reset to auto-detected AppID'
    });
  }
</script>

<div class="bg-[#12121a] border border-cyan-500/40 rounded-xl p-4 space-y-3.5 animate-in fade-in duration-150 mt-2">
  <div class="flex items-center justify-between border-b border-[#222232] pb-2">
    <div class="flex items-center space-x-2">
      <span class="text-xs font-bold text-white uppercase tracking-wider font-mono">Configure Physical Installation Source & Store</span>
      <span class="text-3xs text-slate-400 font-mono">(Overrides detected source & AppID)</span>
    </div>
    <div class="flex items-center space-x-2">
      {#if hasCustomAppId}
        <button
          type="button"
          on:click={handleResetAppId}
          class="text-rose-400 hover:text-rose-300 text-xs font-mono px-2 py-0.5 rounded bg-rose-500/10 border border-rose-500/20 cursor-pointer"
        >
          Reset AppID
        </button>
      {/if}
      <button type="button" on:click={() => dispatch('close')} class="text-slate-400 hover:text-white text-xs font-mono cursor-pointer">Close</button>
    </div>
  </div>

  <div class="grid grid-cols-2 sm:grid-cols-5 gap-2">
    {#each sourceOptions as opt}
      {@const isSel = ($drawerDraft.sourceOverride || '') === opt.id}
      <button
        type="button"
        on:click={() => drawerDraft.setField('sourceOverride', opt.id)}
        class="p-2.5 rounded-lg border text-left transition flex flex-col justify-between gap-1 cursor-pointer {isSel ? 'bg-cyan-950/40 border-cyan-400 text-white shadow-sm' : 'bg-[#0a0a0f] border-[#222232] text-slate-400 hover:text-slate-200 hover:border-slate-600'}"
      >
        <span class="font-bold text-xs">{opt.label}</span>
        <span class="text-3xs text-slate-500 font-mono">{opt.desc}</span>
      </button>
    {/each}
  </div>

  {#if ($drawerDraft.sourceOverride || '').toLowerCase() !== 'standalone' && ($drawerDraft.sourceOverride || '').toLowerCase() !== 'epic'}
    <!-- Store Matching & AppID Search -->
    <div class="pt-2 border-t border-[#1e1e2c] space-y-2.5">
      <div class="flex items-center justify-between">
        <span class="text-3xs font-bold uppercase tracking-wider text-slate-400 font-mono">
          Match Store & AppID ({storeToSearch.toUpperCase()})
        </span>
        {#if $drawerDraft.appId || (game?.appId && game.appId !== '0')}
          <span class="text-3xs font-mono text-cyan-400 bg-cyan-950/40 border border-cyan-500/30 px-2 py-0.5 rounded">
            Current AppID: {$drawerDraft.appId || game?.appId}
          </span>
        {/if}
      </div>

      <!-- Live Search Bar -->
      <div class="flex flex-col sm:flex-row items-center gap-2">
        <div class="relative flex-1 w-full">
          <input
            type="text"
            bind:value={searchQuery}
            placeholder={`Search ${storeToSearch.toUpperCase()} catalogue for AppID (e.g. ${game?.name || ''})...`}
            on:keydown={(e) => { if (e.key === 'Enter') handleStoreSearch(); }}
            class="w-full bg-[#0a0a0f] border border-[#222232] focus:border-cyan-500 rounded-lg px-3.5 py-1.5 text-xs text-white outline-none font-mono placeholder:text-slate-500"
          />
        </div>
        <button
          type="button"
          on:click={handleStoreSearch}
          disabled={isSearching || !searchQuery.trim()}
          class="w-full sm:w-auto px-4 py-1.5 bg-[#1c1c28] hover:bg-[#28283a] text-white text-xs font-bold rounded-lg border border-[#2f2f42] transition flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50"
        >
          {#if isSearching}
            <div class="w-3 h-3 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin"></div>
            <span>Searching...</span>
          {:else}
            <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <span>Search</span>
          {/if}
        </button>
      </div>

      <!-- Candidate Results List -->
      {#if searchResults.length > 0}
        <div class="space-y-1.5 max-h-60 overflow-y-auto pr-1">
          <span class="text-3xs font-bold uppercase tracking-wider text-slate-400 block">
            Found Store Candidates ({searchResults.length})
          </span>
          <div class="grid grid-cols-1 gap-1.5">
            {#each searchResults as cand}
              <div class="flex items-center justify-between p-2.5 rounded-lg bg-[#0a0a0f] border border-[#1e1e2c] hover:border-cyan-500/50 transition">
                <div class="flex items-center space-x-2.5 min-w-0">
                  {#if cand.poster_url}
                    <img src={cand.poster_url} alt="" class="w-8 h-11 object-cover rounded bg-black/40 flex-shrink-0" />
                  {/if}
                  <div class="min-w-0">
                    <div class="text-xs font-bold text-white truncate">{cand.name}</div>
                    <div class="flex items-center space-x-2 text-3xs font-mono text-slate-400">
                      <span class="text-cyan-300">AppID: {cand.id}</span>
                      {#if cand.release_date}
                        <span>• {cand.release_date}</span>
                      {/if}
                    </div>
                  </div>
                </div>
                <button
                  type="button"
                  on:click={() => handleSelectMatch(cand)}
                  class="px-3 py-1 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-bold rounded-lg transition flex-shrink-0 cursor-pointer"
                >
                  Match Store
                </button>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Manual AppID Input Row -->
      <div class="flex items-center gap-2 pt-2 border-t border-[#1e1e2c]">
        <span class="text-3xs font-bold text-slate-400 font-mono flex-shrink-0">
          {storeToSearch === 'steam' ? 'Steam AppID:' : 'GOG ID / Slug:'}
        </span>
        <input
          type="text"
          bind:value={customAppId}
          placeholder={storeToSearch === 'steam' ? 'e.g. 1091500' : 'e.g. the_witcher_3'}
          class="flex-1 bg-[#0a0a0f] border border-[#222232] focus:border-cyan-500 rounded-lg px-2.5 py-1 text-xs text-white outline-none font-mono placeholder:text-slate-600"
        />
        <button
          type="button"
          on:click={handleSetManualAppId}
          disabled={!customAppId.trim()}
          class="px-3.5 py-1 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition flex-shrink-0 cursor-pointer"
        >
          Set AppID
        </button>
      </div>
    </div>
  {/if}

  <div class="p-3 bg-[#0a0a0f] border border-[#1e1e2c] rounded-lg text-xs text-slate-400 space-y-1">
    <div class="font-bold text-slate-300">
      Current Source: <span class="text-cyan-400 font-mono font-bold">{$drawerDraft.sourceOverride || game?.source || game?.store || 'Standalone'}</span>
      {#if $drawerDraft.appId || (game?.appId && game.appId !== '0')}
        <span class="text-cyan-300 font-mono font-bold ml-1">(AppID: {$drawerDraft.appId || game?.appId})</span>
      {/if}
      {#if !$drawerDraft.sourceOverride}
        <span class="text-3xs text-slate-500 font-normal ml-1">(Auto-detected)</span>
      {/if}
    </div>
    <p class="text-2xs text-slate-400">
      {$drawerDraft.sourceOverride === 'Steam' 
        ? 'Steam mode enables Goldberg emulator injection and Steamworks API mocking.' 
        : $drawerDraft.sourceOverride === 'GOG' 
        ? 'GOG mode enables GOG registry & offline info detection.'
        : $drawerDraft.sourceOverride === 'Standalone'
        ? 'Standalone mode executes binary directly via Proton/Wine without injecting Steamworks emulators.'
        : 'Auto-detect inspects binaries, steam_api DLLs, and manifests to configure runtime hooks.'}
    </p>
  </div>
</div>
