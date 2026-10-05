<script>
  import { onMount } from 'svelte';
  import { hasMetadataMatch } from '../../../utils/game.js';
  import { AppAPI } from '../../../api/app.js';
  import { notificationStore } from '../../../stores/notifications.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null ,
    onlinkMeta,
    onunlinkMeta,
    onclose
  } = $props();

  // svelte-ignore state_referenced_locally
  const initialProvider = (game?.metadataProvider || game?.metadata_provider || game?.store || 'steam').toLowerCase();
  // svelte-ignore state_referenced_locally
  let metaTargetStore = $state(initialProvider === 'gog' ? 'gog' : initialProvider === 'rawg' ? 'rawg' : (initialProvider === 'none' || initialProvider === 'standalone') ? 'none' : 'steam');
  // svelte-ignore state_referenced_locally
  let metaSearchQuery = $state(game?.name || '');
  // svelte-ignore state_referenced_locally
  let metaCustomId = $state(game?.metadataId || game?.metadata_id || (game?.appId && game.appId !== '0' ? String(game.appId) : '') || '');
  let metaSearchResults = $state([]);
  let isSearchingMeta = $state(false);
  let isLinkingMeta = false;

  const metadataProviders = [
    { id: 'steam', display_name: 'Steam', badge: 'AppID' },
    { id: 'gog', display_name: 'GOG.com', badge: 'GOG ID' },
    { id: 'rawg', display_name: 'RAWG Database', badge: 'Open DB' },
    { id: 'none', display_name: 'None', badge: 'Unlinked' }
  ];

  let hasMatch = $derived(game ? hasMetadataMatch(game) : false);

  onMount(() => {
    if (metaTargetStore !== 'none' && metaSearchQuery.trim()) {
      handleMetaSearch();
    }
  });

  async function handleMetaSearch(storeOverride = null) {
    if (!metaSearchQuery.trim()) return;
    const storeToMatch = (storeOverride || metaTargetStore || 'steam').toLowerCase();
    isSearchingMeta = true;
    try {
      const results = await AppAPI.searchStoreGames(metaSearchQuery.trim());
      metaSearchResults = (results || []).filter(c => {
        const storeLower = (c.store || '').toLowerCase();
        return storeLower === storeToMatch;
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Search Failed',
        message: err?.message || String(err)
      });
      metaSearchResults = [];
    } finally {
      isSearchingMeta = false;
    }
  }

  function handleApplyMetaLink(idToUse = null) {
    const targetId = idToUse !== null ? String(idToUse) : (metaTargetStore === 'none' || metaTargetStore === 'standalone' ? '0' : metaCustomId.trim());
    onlinkMeta?.({ targetStore: metaTargetStore, targetId });
  }

  function handleUnlinkMeta() {
    onunlinkMeta?.();
  }
</script>

<div class="bg-[#12121a] border border-purple-500/40 rounded-xl p-4 space-y-3.5 animate-in fade-in duration-150 mt-2">
  <div class="flex items-center justify-between border-b border-[#222232] pb-2">
    <div class="flex items-center space-x-2">
      <span class="text-xs font-bold text-white uppercase tracking-wider font-mono">Match Online Store Metadata</span>
      <span class="text-3xs text-slate-400 font-mono">(Does not modify physical game files)</span>
    </div>
    <div class="flex items-center space-x-2">
      {#if hasMatch}
        <button
          type="button"
          onclick={handleUnlinkMeta}
          disabled={isLinkingMeta}
          class="text-rose-400 hover:text-rose-300 text-xs font-mono px-2 py-0.5 rounded bg-rose-500/10 border border-rose-500/20 cursor-pointer"
        >
          Reset to Auto
        </button>
      {/if}
      <button type="button" onclick={() => onclose?.()} class="text-slate-400 hover:text-white text-xs font-mono cursor-pointer">Close</button>
    </div>
  </div>

  <!-- Provider Selection Pills -->
  <div class="flex items-center gap-2 overflow-x-auto pb-1">
    {#each metadataProviders as prov}
      {@const isAct = metaTargetStore === prov.id}
      <button
        type="button"
        onclick={() => { metaTargetStore = prov.id; if (metaTargetStore !== 'none' && metaTargetStore !== 'standalone' && metaSearchQuery.trim()) handleMetaSearch(prov.id); }}
        class="px-3 py-1 rounded-lg text-xs font-bold transition flex items-center space-x-1.5 cursor-pointer {isAct ? 'bg-purple-600 text-white shadow border border-purple-400/40' : 'bg-[#0a0a0f] border border-[#222232] text-slate-400 hover:text-white'}"
      >
        <span>{prov.display_name}</span>
        <span class="text-4xs bg-black/30 px-1 rounded font-mono">{prov.badge}</span>
      </button>
    {/each}
  </div>

  {#if metaTargetStore === 'none' || metaTargetStore === 'standalone'}
    <div class="p-3 bg-[#0a0a0f] border border-[#1e1e2c] rounded-lg text-xs text-slate-300 flex items-center justify-between">
      <p class="text-2xs text-slate-400">Unlinks external store metadata. Game will display local library title and fallback artwork.</p>
      <button
        type="button"
        onclick={() => handleApplyMetaLink('0')}
        disabled={isLinkingMeta}
        class="px-4 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs rounded-lg transition cursor-pointer"
      >
        {isLinkingMeta ? 'Unlinking...' : 'Set None (Unlinked)'}
      </button>
    </div>
  {:else}
    <!-- Live Search Bar -->
    <div class="flex flex-col sm:flex-row items-center gap-2">
      <div class="relative flex-1 w-full">
        <input
          type="text"
          bind:value={metaSearchQuery}
          placeholder={`Search ${metaTargetStore.toUpperCase()} catalogue (e.g. ${game?.name || ''})...`}
          onkeydown={(e) => { if (e.key === 'Enter') handleMetaSearch(); }}
          class="w-full bg-[#0a0a0f] border border-[#222232] focus:border-purple-500 rounded-lg px-3.5 py-1.5 text-xs text-white outline-none font-mono placeholder:text-slate-500"
        />
      </div>
      <button
        type="button"
        onclick={handleMetaSearch}
        disabled={isSearchingMeta || !metaSearchQuery.trim()}
        class="w-full sm:w-auto px-4 py-1.5 bg-[#1c1c28] hover:bg-[#28283a] text-white text-xs font-bold rounded-lg border border-[#2f2f42] transition flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isSearchingMeta}
          <div class="w-3 h-3 border-2 border-purple-400 border-t-transparent rounded-full animate-spin"></div>
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
    {#if metaSearchResults.length > 0}
      <div class="space-y-1.5 max-h-60 overflow-y-auto pr-1">
        <span class="text-3xs font-bold uppercase tracking-wider text-slate-400 block">
          Found Candidates ({metaSearchResults.length})
        </span>
        <div class="grid grid-cols-1 gap-1.5">
          {#each metaSearchResults as cand}
            <div class="flex items-center justify-between p-2.5 rounded-lg bg-[#0a0a0f] border border-[#1e1e2c] hover:border-purple-500/50 transition">
              <div class="flex items-center space-x-2.5 min-w-0">
                {#if cand.poster_url}
                  <img src={cand.poster_url} alt="" class="w-8 h-11 object-cover rounded bg-black/40 flex-shrink-0" />
                {/if}
                <div class="min-w-0">
                  <div class="text-xs font-bold text-white truncate">{cand.name}</div>
                  <div class="flex items-center space-x-2 text-3xs font-mono text-slate-400">
                    <span class="text-purple-300">ID: {cand.id}</span>
                    {#if cand.release_date}
                      <span>• {cand.release_date}</span>
                    {/if}
                  </div>
                </div>
              </div>
              <button
                type="button"
                onclick={() => handleApplyMetaLink(cand.id)}
                disabled={isLinkingMeta}
                class="px-3 py-1 bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold rounded-lg transition flex-shrink-0 cursor-pointer"
              >
                Link Metadata
              </button>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Manual ID Fallback Row -->
    <div class="flex items-center gap-2 pt-2 border-t border-[#1e1e2c]">
      <span class="text-3xs font-bold text-slate-400 font-mono flex-shrink-0">
        {metaTargetStore === 'steam' ? 'Steam AppID:' : metaTargetStore === 'gog' ? 'GOG ID / Slug:' : 'RAWG ID / Slug:'}
      </span>
      <input
        type="text"
        bind:value={metaCustomId}
        placeholder={metaTargetStore === 'steam' ? 'e.g. 1091500' : metaTargetStore === 'gog' ? 'e.g. the_witcher_3' : 'e.g. portal-2'}
        class="flex-1 bg-[#0a0a0f] border border-[#222232] focus:border-purple-500 rounded-lg px-2.5 py-1 text-xs text-white outline-none font-mono placeholder:text-slate-600"
      />
      <button
        type="button"
        onclick={() => handleApplyMetaLink()}
        disabled={!metaCustomId.trim() || isLinkingMeta}
        class="px-3.5 py-1 bg-purple-600 hover:bg-purple-500 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition flex-shrink-0 cursor-pointer"
      >
        Link ID
      </button>
    </div>
  {/if}
</div>
