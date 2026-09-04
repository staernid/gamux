<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { selectedGame, games } from '../../stores/library.js';
  import { isSamePath } from '../../utils/game.js';
  import ModalShell from '../common/ModalShell.svelte';
  import MetadataTab from './metadata/MetadataTab.svelte';
  import ArtworkTab from './metadata/ArtworkTab.svelte';

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
  async function handleSearchMeta(storeOverride = null) {
    if (!metaQuery.trim()) return;
    const storeToMatch = (storeOverride || targetStore || 'steam').toLowerCase();
    isSearchingMeta = true;
    selectedMetaCandidate = null;
    try {
      const results = await AppAPI.searchStoreGames(metaQuery.trim());
      metaCandidates = (results || []).filter(c => {
        const storeLower = (c.store || '').toLowerCase();
        return storeLower === storeToMatch;
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

  function handleSelectAssetType(e) {
    const type = e.detail || e;
    if (assetType === type) return;
    assetType = type;
    fetchArtworkOptions();
  }

  async function handleApplyArtwork(imageUrlOrPath) {
    const targetUrl = typeof imageUrlOrPath === 'string' ? imageUrlOrPath : (imageUrlOrPath?.detail || '');
    if (!gameDir || !targetUrl) return;
    isApplyingArt = true;
    try {
      const updated = await AppAPI.applyCustomArtwork(gameDir, assetType, targetUrl);
      appliedMap[assetType] = targetUrl;
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
</script>

<ModalShell
  title={gameName}
  badge="Metadata & Artwork Hub"
  subtitle="Manage store database bindings, official box art, and custom graphics."
  on:close={() => dispatch('close')}
>
  <svelte:fragment slot="headerActions">
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
  </svelte:fragment>

  <!-- Primary Tab Selector (Store Link vs Artwork Gallery) -->
  <div class="max-w-5xl w-full mx-auto mb-4 flex items-center justify-between border-b border-surface-border pb-3 flex-shrink-0 gap-4">
    <div class="flex items-center space-x-2">
      <button
        type="button"
        on:click={() => { activeTab = 'metadata'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'metadata' ? 'bg-white text-neutral-950 shadow-md' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        <span>1. Store & Metadata Link</span>
      </button>

      <button
        type="button"
        on:click={() => { activeTab = 'artwork'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'artwork' ? 'bg-white text-neutral-950 shadow-md' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>2. Artwork Gallery & Custom Art</span>
      </button>
    </div>

    <!-- Active Source Indicator -->
    <div class="hidden sm:flex items-center space-x-2 text-2xs font-mono text-neutral-400 bg-surface-input border border-surface-border px-3 py-1.5 rounded-xl">
      <span class="text-neutral-500">Source:</span>
      <span class="text-neutral-200 font-bold uppercase">{game?.source || game?.store || 'Steam'}</span>
      {#if appId && appId !== '0'}
        <span class="text-neutral-300">#{appId}</span>
      {/if}
    </div>
  </div>

  <!-- Content Container -->
  <div class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-1 custom-scrollbar space-y-4">
    {#if activeTab === 'metadata'}
      <MetadataTab
        {game}
        bind:targetStore
        {providers}
        bind:metaQuery
        bind:customId
        {metaCandidates}
        {isSearchingMeta}
        {isLinkingMeta}
        {selectedMetaCandidate}
        on:storeChange={(e) => {
          targetStore = e.detail;
          if (targetStore !== 'standalone' && metaQuery.trim()) {
            handleSearchMeta(e.detail);
          }
        }}
        on:search={handleSearchMeta}
        on:selectCandidate={(e) => handleSelectMetaCandidate(e.detail)}
        on:applyLink={handleApplyMetaLink}
      />
    {:else if activeTab === 'artwork'}
      <ArtworkTab
        bind:assetType
        bind:artQuery
        {artOptions}
        {isLoadingArt}
        bind:customArtUrl
        {isApplyingArt}
        {failedUrls}
        {currentActiveArtUrl}
        on:selectAssetType={handleSelectAssetType}
        on:search={fetchArtworkOptions}
        on:apply={(e) => handleApplyArtwork(e.detail)}
      />
    {/if}
  </div>
</ModalShell>
