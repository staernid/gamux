<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { selectedGame, games } from '../../stores/library.js';
  import { isSamePath } from '../../utils/game.js';

  export let game = null;
  export let initialTab = 'metadata'; // 'metadata' | 'artwork'

  const dispatch = createEventDispatcher();

  // Active view tab: 'metadata' | 'artwork'
  let activeTab = initialTab || 'metadata';

  // ---------------------------------------------------------------------------
  // Metadata Tab State
  // ---------------------------------------------------------------------------
  let targetStore = 'steam'; // 'steam' | 'gog' | 'rawg' | 'standalone'
  let providers = [
    { id: 'steam', display_name: 'Steam', badge: 'AppID' },
    { id: 'gog', display_name: 'GOG.com', badge: 'GOG ID' },
    { id: 'rawg', display_name: 'RAWG Database', badge: 'Open DB' },
    { id: 'standalone', display_name: 'Standalone / DRM-Free', badge: 'Direct' },
  ];
  let metaQuery = '';
  let customId = '';
  let metaCandidates = [];
  let isSearchingMeta = false;
  let isLinkingMeta = false;
  let selectedMetaCandidate = null;

  // ---------------------------------------------------------------------------
  // Artwork Tab State
  // ---------------------------------------------------------------------------
  let assetType = 'cover'; // 'cover' | 'banner' | 'hero' | 'logo'
  let artQuery = '';
  let artOptions = [];
  let isLoadingArt = false;
  let customArtUrl = '';
  let isApplyingArt = false;
  let failedUrls = new Set();
  let appliedMap = {};

  $: gameName = game?.name || '';
  $: gameDir = game?.gameDir || '';
  $: appId = game?.appId || '';
  $: numericSteamAppId = (() => {
    const n = parseInt(String(appId), 10);
    return !isNaN(n) && n > 0 ? n : 0;
  })();

  $: currentActiveArtUrl = (() => {
    if (appliedMap[assetType]) return appliedMap[assetType];
    if (!game) return '';
    switch (assetType) {
      case 'cover':
        return game.coverUrl || game.localCoverUrl || '';
      case 'hero':
        return game.heroUrl || game.localHeroUrl || '';
      case 'banner':
        return game.bannerUrl || game.localBannerUrl || '';
      case 'logo':
        return game.logoUrl || game.localLogoUrl || '';
      default:
        return '';
    }
  })();

  onMount(() => {
    // 1. Initialize Metadata state
    const providerKey = (game?.metadataProvider || game?.metadata_provider || game?.store || 'steam').toLowerCase();
    if (providerKey === 'gog') targetStore = 'gog';
    else if (providerKey === 'rawg') targetStore = 'rawg';
    else if (providerKey === 'standalone' || providerKey === 'custom') targetStore = 'standalone';
    else targetStore = 'steam';

    metaQuery = gameName || '';
    customId = String(game?.metadataId || game?.metadata_id || game?.appId || '');
    if (customId === '0') customId = '';

    AppAPI.listMetadataProviders()
      .then(fetched => {
        if (Array.isArray(fetched) && fetched.length > 0) {
          const list = fetched
            .filter(p => p.id !== 'steamgriddb')
            .map(p => ({
              id: p.id,
              display_name: p.display_name,
              badge: p.id === 'steam' ? 'AppID' : p.id === 'gog' ? 'GOG ID' : p.id === 'rawg' ? 'Open DB' : 'Direct',
            }));
          list.push({ id: 'standalone', display_name: 'Standalone / DRM-Free', badge: 'Direct' });
          providers = list;
        }
      })
      .catch(() => {});

    if (targetStore !== 'standalone' && metaQuery.trim()) {
      handleSearchMeta();
    }

    // 2. Initialize Artwork query
    artQuery = gameName || (appId ? String(appId) : '');
    fetchArtworkOptions();
  });

  // ---------------------------------------------------------------------------
  // Metadata Actions
  // ---------------------------------------------------------------------------
  async function handleSearchMeta() {
    if (!metaQuery.trim()) return;
    isSearchingMeta = true;
    selectedMetaCandidate = null;
    try {
      const results = await AppAPI.searchStoreGames(metaQuery.trim());
      metaCandidates = (results || []).filter(c => {
        const storeLower = (c.store || '').toLowerCase();
        return storeLower === targetStore.toLowerCase();
      });
    } catch (err) {
      console.warn('Failed to search store games:', err);
      metaCandidates = [];
    } finally {
      isSearchingMeta = false;
    }
  }

  function handleSelectMetaCandidate(cand) {
    selectedMetaCandidate = cand;
    customId = cand.id;
  }

  async function handleApplyMetaLink() {
    if (!gameDir) return;
    isLinkingMeta = true;
    try {
      const idToLink = targetStore === 'standalone' ? '0' : customId.trim();
      const updated = await AppAPI.linkGameMetadata(gameDir, targetStore, idToLink);
      notificationStore.items.add({
        type: 'success',
        title: 'Metadata Linked',
        message: `Linked ${gameName} to ${targetStore.toUpperCase()} ${idToLink ? `(#${idToLink})` : ''}`,
      });
      if (updated) {
        game = { ...game, ...updated };
        selectedGame.update(curr => (curr && isSamePath(curr.gameDir, gameDir) ? { ...curr, ...updated } : curr));
      }
      // Re-query artwork for new metadata
      artQuery = updated?.name || gameName;
      fetchArtworkOptions();
      dispatch('updated', updated);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Linking Failed',
        message: err?.message || String(err),
      });
    } finally {
      isLinkingMeta = false;
    }
  }

  async function handleUnlinkMeta() {
    if (!gameDir) return;
    isLinkingMeta = true;
    try {
      const updated = await AppAPI.unlinkGameMetadata(gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Metadata Reset',
        message: `Restored auto-detected identity for ${gameName}.`,
      });
      if (updated) {
        game = { ...game, ...updated };
        selectedGame.update(curr => (curr && isSamePath(curr.gameDir, gameDir) ? { ...curr, ...updated } : curr));
      }
      artQuery = gameName;
      fetchArtworkOptions();
      dispatch('updated', updated);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Unlink Failed',
        message: err?.message || String(err),
      });
    } finally {
      isLinkingMeta = false;
    }
  }

  // ---------------------------------------------------------------------------
  // Artwork Actions
  // ---------------------------------------------------------------------------
  async function fetchArtworkOptions() {
    if (!gameDir && !numericSteamAppId && !gameName) return;
    isLoadingArt = true;
    artOptions = [];
    failedUrls.clear();
    const rawQuery = artQuery.trim();
    const isCustomSearch = Boolean(rawQuery && rawQuery !== gameName && rawQuery !== String(numericSteamAppId));

    try {
      artOptions =
        (await AppAPI.queryArtworkOptions({
          game_dir: gameDir,
          title: rawQuery || gameName,
          steam_app_id: isCustomSearch ? 0 : numericSteamAppId,
          gog_id: game?.gog_id || game?.GOGID || '',
          sgdb_id: game?.sgdb_id || game?.SGDBID || '',
          asset_type: assetType,
        })) || [];
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Fetch Artwork',
        message: err?.message || String(err),
      });
      artOptions = [];
    } finally {
      isLoadingArt = false;
    }
  }

  function handleSelectAssetType(type) {
    if (assetType === type) return;
    assetType = type;
    fetchArtworkOptions();
  }

  async function handleApplyArtwork(imageUrlOrPath) {
    if (!gameDir || !imageUrlOrPath) return;
    isApplyingArt = true;
    try {
      const updated = await AppAPI.applyCustomArtwork(gameDir, assetType, imageUrlOrPath);
      appliedMap[assetType] = imageUrlOrPath;
      appliedMap = { ...appliedMap };

      notificationStore.items.add({
        type: 'success',
        title: 'Artwork Applied',
        message: `Set ${assetType} artwork for ${gameName}`,
      });

      if (updated) {
        const coverUrl = updated.cover_url ?? updated.CoverURL ?? updated.custom_cover_url ?? '';
        const localCoverUrl = updated.local_cover_url ?? updated.LocalCoverURL ?? updated.local_cover ?? '';
        const heroUrl = updated.hero_url ?? updated.HeroURL ?? updated.custom_hero_url ?? '';
        const localHeroUrl = updated.local_hero_url ?? updated.LocalHeroURL ?? updated.local_hero ?? '';
        const bannerUrl = updated.banner_url ?? updated.BannerURL ?? updated.custom_banner_url ?? '';
        const localBannerUrl = updated.local_banner_url ?? updated.LocalBannerURL ?? updated.local_banner ?? '';
        const logoUrl = updated.logo_url ?? updated.LogoURL ?? updated.custom_logo_url ?? '';
        const localLogoUrl = updated.local_logo_url ?? updated.LocalLogoURL ?? updated.local_logo ?? '';

        const patch = {
          local_cover: updated.local_cover ?? updated.LocalCover,
          local_cover_url: localCoverUrl,
          localCoverUrl,
          cover_url: coverUrl,
          coverUrl,
          custom_cover_url: coverUrl,
          local_hero: updated.local_hero ?? updated.LocalHero,
          local_hero_url: localHeroUrl,
          localHeroUrl,
          hero_url: heroUrl,
          heroUrl,
          custom_hero_url: heroUrl,
          local_banner: updated.local_banner ?? updated.LocalBanner,
          local_banner_url: localBannerUrl,
          localBannerUrl,
          banner_url: bannerUrl,
          bannerUrl,
          custom_banner_url: bannerUrl,
          local_logo: updated.local_logo ?? updated.LocalLogo,
          local_logo_url: localLogoUrl,
          localLogoUrl,
          logo_url: logoUrl,
          logoUrl,
          custom_logo_url: logoUrl,
        };

        game = { ...game, ...patch };
        selectedGame.update(current => {
          if (!current || !isSamePath(current.gameDir, gameDir)) return current;
          return { ...current, ...patch };
        });
        games.update(list =>
          list.map(g => {
            if (isSamePath(g.gameDir, gameDir)) {
              return { ...g, ...patch };
            }
            return g;
          })
        );
      }
      dispatch('updated', updated);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Apply Artwork',
        message: err?.message || String(err),
      });
    } finally {
      isApplyingArt = false;
    }
  }

  async function handleBrowseLocalArt() {
    try {
      const selected = await AppAPI.selectFile('Select Image File', 'Images', '*.jpg;*.jpeg;*.png;*.webp');
      if (selected) {
        await handleApplyArtwork(selected);
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'File Selection Failed',
        message: err?.message || String(err),
      });
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
  class="fixed inset-0 z-50 bg-[#09090c]/95 backdrop-blur-2xl flex flex-col p-5 sm:p-8 select-none animate-in fade-in duration-150 outline-none"
>
  <!-- Main Modal Header Bar -->
  <div class="flex items-center justify-between max-w-5xl w-full mx-auto mb-5 flex-shrink-0">
    <div class="min-w-0 pr-4">
      <div class="flex items-center space-x-2.5">
        <span class="text-xs font-bold px-2 py-0.5 rounded bg-sky-500/20 text-sky-300 border border-sky-500/30 uppercase tracking-wider">
          Metadata & Artwork Hub
        </span>
        <h2 class="text-lg sm:text-xl font-black text-white tracking-tight truncate">
          {gameName}
        </h2>
      </div>
      <p class="text-xs text-slate-400 mt-1">Manage store database bindings, official box art, and custom graphics.</p>
    </div>

    <!-- Header Actions & Close Button -->
    <div class="flex items-center space-x-2 flex-shrink-0">
      {#if activeTab === 'metadata'}
        <button
          type="button"
          on:click={handleUnlinkMeta}
          disabled={isLinkingMeta}
          class="px-3 py-1.5 rounded-xl bg-[#201518] hover:bg-[#301c22] border border-rose-500/30 text-rose-300 hover:text-rose-200 text-xs font-bold transition-colors cursor-pointer"
          title="Reset metadata overrides to auto-detected origin"
        >
          Reset to Auto
        </button>
      {:else}
        <button
          type="button"
          on:click={() => handleApplyArtwork('none')}
          disabled={isApplyingArt}
          class="px-3 py-1.5 rounded-xl bg-amber-950/40 hover:bg-amber-900/60 border border-amber-700/50 text-amber-300 text-xs font-bold transition-colors cursor-pointer"
          title="Reset active asset to monogram fallback"
        >
          Reset Art
        </button>
        <button
          type="button"
          on:click={handleBrowseLocalArt}
          class="px-3 py-1.5 rounded-xl bg-[#1e1e2c] hover:bg-[#2a2a3e] border border-[#34344c] text-white text-xs font-bold transition-colors cursor-pointer"
        >
          Browse File...
        </button>
      {/if}

      <button
        type="button"
        on:click={() => dispatch('close')}
        class="px-4 py-1.5 rounded-xl bg-[#1a1a24] hover:bg-[#252535] border border-[#2b2b3c] text-slate-300 hover:text-white text-xs font-mono transition-colors flex items-center space-x-1.5 cursor-pointer"
      >
        <span class="px-1.5 py-0.5 rounded bg-[#161622] border border-[#262638] text-slate-400 font-bold text-[10px]">Esc</span>
        <span>Done</span>
      </button>
    </div>
  </div>

  <!-- Primary Tab Selector (Store Link vs Artwork Gallery) -->
  <div class="max-w-5xl w-full mx-auto mb-4 flex items-center justify-between border-b border-[#1f1f2e] pb-3 flex-shrink-0 gap-4">
    <div class="flex items-center space-x-2">
      <button
        type="button"
        on:click={() => { activeTab = 'metadata'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'metadata' ? 'bg-sky-600 text-white shadow-lg shadow-sky-950/50 border border-sky-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        <span>1. Store & Metadata Link</span>
      </button>

      <button
        type="button"
        on:click={() => { activeTab = 'artwork'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'artwork' ? 'bg-purple-600 text-white shadow-lg shadow-purple-950/50 border border-purple-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>2. Artwork Gallery & Custom Art</span>
      </button>
    </div>

    <!-- Active Source Indicator -->
    <div class="hidden sm:flex items-center space-x-2 text-[11px] font-mono text-slate-400 bg-[#12121c] border border-[#222232] px-3 py-1.5 rounded-xl">
      <span class="text-slate-500">Source:</span>
      <span class="text-slate-200 font-bold uppercase">{game?.source || game?.store || 'Steam'}</span>
      {#if appId && appId !== '0'}
        <span class="text-sky-400">#{appId}</span>
      {/if}
    </div>
  </div>

  <!-- Content Container -->
  <div class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-1 custom-scrollbar space-y-4">
    <!-- =================================================================== -->
    <!-- TAB 1: METADATA LINK -->
    <!-- =================================================================== -->
    {#if activeTab === 'metadata'}
      <!-- Store Provider Pills -->
      <div class="flex items-center gap-2 overflow-x-auto custom-scrollbar pb-1">
        {#each providers as prov}
          {@const isActive = targetStore === prov.id}
          <button
            type="button"
            on:click={() => { targetStore = prov.id; if (targetStore !== 'standalone' && metaQuery.trim()) handleSearchMeta(); }}
            class="px-3.5 py-1.5 rounded-xl font-bold text-xs transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {isActive ? 'bg-sky-600 text-white shadow-md border border-sky-400/40' : 'bg-[#14141e] border border-[#262638] text-slate-400 hover:text-white'}"
          >
            <span>{prov.display_name}</span>
            <span class="text-[10px] bg-black/30 px-1.5 py-0.5 rounded font-mono">{prov.badge}</span>
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
              on:click={handleApplyMetaLink}
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
                on:keydown={(e) => { if (e.key === 'Enter') handleSearchMeta(); }}
                placeholder={`Search ${targetStore.toUpperCase()} catalogue...`}
                class="w-full bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-xl px-4 py-2 text-xs text-white outline-none font-mono placeholder:text-slate-500"
              />
            </div>
            <button
              type="button"
              on:click={handleSearchMeta}
              disabled={isSearchingMeta}
              class="w-full sm:w-auto px-5 py-2 bg-[#1e1e2c] hover:bg-[#28283c] text-white text-xs font-bold rounded-xl border border-[#34344c] transition-colors flex-shrink-0 cursor-pointer"
            >
              {isSearchingMeta ? 'Searching...' : 'Search'}
            </button>
          </div>

          <!-- Manual ID Input & Apply Button -->
          <div class="flex items-center gap-3 pt-2 border-t border-[#1e1e2c]">
            <span class="text-[11px] font-bold text-slate-400 flex-shrink-0">
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
              on:click={handleApplyMetaLink}
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
            <span class="text-[10px] font-bold uppercase tracking-wider text-slate-400 block px-1">
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
                      on:click={() => handleSelectMetaCandidate(cand)}
                      class="px-3 py-1 bg-[#1e1e2c] hover:bg-[#28283c] text-slate-200 hover:text-white text-xs font-bold rounded-lg border border-[#323246] transition-colors cursor-pointer"
                    >
                      Select ID
                    </button>
                    <button
                      type="button"
                      on:click={() => { handleSelectMetaCandidate(cand); handleApplyMetaLink(); }}
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

    <!-- =================================================================== -->
    <!-- TAB 2: ARTWORK GALLERY -->
    <!-- =================================================================== -->
    {:else if activeTab === 'artwork'}
      <!-- Asset Type & Search Controls -->
      <div class="flex flex-col sm:flex-row items-center justify-between gap-3 bg-[#12121c] border border-[#262638] p-3 rounded-2xl">
        <div class="flex items-center bg-[#0a0a0f] border border-[#222232] rounded-xl p-1 gap-1 text-xs w-full sm:w-auto overflow-x-auto">
          <button
            type="button"
            on:click={() => handleSelectAssetType('cover')}
            class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'cover' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
          >
            Cover Art (600×900)
          </button>
          <button
            type="button"
            on:click={() => handleSelectAssetType('banner')}
            class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'banner' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
          >
            Banner Header
          </button>
          <button
            type="button"
            on:click={() => handleSelectAssetType('hero')}
            class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'hero' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
          >
            Hero (1920×620)
          </button>
          <button
            type="button"
            on:click={() => handleSelectAssetType('logo')}
            class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'logo' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
          >
            Logo (PNG)
          </button>
        </div>

        <div class="flex items-center space-x-2 w-full sm:w-auto sm:max-w-xs">
          <input
            type="text"
            bind:value={artQuery}
            on:keydown={(e) => { if (e.key === 'Enter') fetchArtworkOptions(); }}
            placeholder="Search artwork..."
            class="flex-1 bg-[#0a0a0f] border border-[#222232] focus:border-sky-500 rounded-xl px-3 py-1.5 text-xs text-white outline-none font-mono placeholder:text-slate-500"
          />
          <button
            type="button"
            on:click={fetchArtworkOptions}
            class="px-3 py-1.5 bg-[#1e1e2c] hover:bg-[#2a2a3e] text-slate-200 text-xs font-bold rounded-xl border border-[#34344c] transition-colors cursor-pointer flex-shrink-0"
          >
            Search
          </button>
        </div>
      </div>

      <!-- Custom Image URL Input Bar -->
      <div class="bg-[#12121c] border border-[#262638] rounded-xl p-2.5 flex items-center gap-3">
        <span class="text-[11px] font-bold uppercase tracking-wider text-slate-400 pl-1 flex-shrink-0">Custom URL:</span>
        <input
          type="url"
          bind:value={customArtUrl}
          placeholder="https://example.com/custom-artwork.jpg"
          class="flex-1 bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-lg px-3 py-1 text-xs text-slate-100 outline-none font-mono placeholder:text-slate-600"
        />
        <button
          type="button"
          on:click={() => handleApplyArtwork(customArtUrl.trim())}
          disabled={!customArtUrl.trim() || isApplyingArt}
          class="px-4 py-1 bg-sky-600 hover:bg-sky-500 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition-colors flex-shrink-0 shadow cursor-pointer"
        >
          {isApplyingArt ? 'Applying...' : 'Apply URL'}
        </button>
      </div>

      <!-- Artwork Grid Area -->
      {#if isLoadingArt}
        <div class="h-64 flex flex-col items-center justify-center text-slate-400 space-y-3 font-mono text-xs">
          <div class="w-7 h-7 border-2 border-purple-500 border-t-transparent rounded-full animate-spin"></div>
          <p>Discovering artwork options across providers...</p>
        </div>
      {:else if artOptions.length === 0}
        <div class="h-64 flex flex-col items-center justify-center text-slate-500 text-xs font-mono space-y-2 text-center">
          <p>No {assetType} artwork candidates found for "{artQuery}".</p>
          <p class="text-[11px] text-slate-600">Try searching with a simplified game title or enter a custom image URL above.</p>
        </div>
      {:else}
        <div class="grid {assetType === 'cover' ? 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5' : 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3'} gap-4 pb-8">
          {#each artOptions.filter(o => !failedUrls.has(o.url) && !failedUrls.has(o.thumb_url)) as opt, index (opt.url || index)}
            {@const isActive = Boolean(currentActiveArtUrl && (currentActiveArtUrl === opt.url || currentActiveArtUrl.includes(opt.url) || opt.url.includes(currentActiveArtUrl)))}
            <div class="group relative rounded-2xl overflow-hidden bg-[#14141e] border transition-all duration-200 flex flex-col {isActive ? 'border-emerald-500 ring-2 ring-emerald-500/40 shadow-lg shadow-emerald-500/10' : 'border-[#262638] hover:border-slate-400'}">
              <div class="relative w-full {assetType === 'cover' ? 'aspect-[2/3]' : assetType === 'banner' ? 'aspect-[460/215]' : assetType === 'hero' ? 'aspect-[16/9]' : 'aspect-video p-4 flex items-center justify-center'} bg-black/40 overflow-hidden">
                <img
                  src={opt.thumb_url || opt.url}
                  alt=""
                  class="w-full h-full {assetType === 'logo' ? 'object-contain' : 'object-cover'} group-hover:scale-105 transition-transform duration-300"
                  loading="lazy"
                  on:error={() => { failedUrls = new Set([...failedUrls, opt.url, opt.thumb_url].filter(Boolean)); }}
                />
                <div class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-[9px] font-black uppercase tracking-wider text-white shadow-md backdrop-blur-md {opt.provider === 'steam' ? 'bg-blue-600/90' : opt.provider === 'gog' ? 'bg-purple-600/90' : opt.provider === 'rawg' ? 'bg-amber-600/90' : 'bg-emerald-600/90'}">
                  {opt.provider === 'steam' ? 'Steam' : opt.provider === 'gog' ? 'GOG' : opt.provider === 'rawg' ? 'RAWG' : opt.provider === 'steamgriddb' ? 'SteamGridDB' : opt.provider}
                </div>
                {#if isActive}
                  <div class="absolute top-2 right-2 px-2 py-0.5 rounded-md text-[9px] font-black uppercase tracking-wider text-emerald-100 bg-emerald-600 shadow-md flex items-center space-x-1">
                    <svg class="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Active</span>
                  </div>
                {/if}
                {#if opt.width && opt.height}
                  <div class="absolute bottom-2 left-2 px-1.5 py-0.5 rounded bg-black/75 backdrop-blur-sm text-[9px] font-mono text-slate-300 border border-white/10 font-bold">
                    {opt.width}×{opt.height}
                  </div>
                {/if}
              </div>

              <div class="p-2.5 bg-[#0f0f16] border-t border-[#1e1e2c] flex items-center justify-between">
                <span class="text-[10px] font-mono text-slate-400 capitalize">{opt.style || 'Custom'}</span>
                <button
                  type="button"
                  on:click={() => handleApplyArtwork(opt.url)}
                  disabled={isApplyingArt || isActive}
                  class="px-3 py-1 font-bold text-xs rounded-lg transition-colors shadow cursor-pointer {isActive ? 'bg-emerald-600/30 text-emerald-300 border border-emerald-500/40 cursor-default' : 'bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white'}"
                >
                  {isActive ? 'Applied' : isApplyingArt ? 'Applying...' : 'Select'}
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>
