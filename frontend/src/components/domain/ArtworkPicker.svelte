<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { selectedGame, games } from '../../stores/library.js';
  import { isSamePath } from '../../utils/game.js';
  import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../types/domain').Game} [game]
   * @property {string} [assetType] - 'cover' | 'banner' | 'hero' | 'logo'
   * @property {((type: string, url: string) => void) | null} [onapply]
   * @property {((game: import('../../types/domain').Game) => void) | null} [onupdated]
   */

  /** @type {Props} */
  let {
    game = $bindable(null),
    assetType = $bindable('cover'),
    onapply = null,
    onupdated = null
  ,
    onassetTypeChange
  } = $props();

  let artQuery = $state('');
  let artOptions = $state([]);
  let isLoadingArt = $state(false);
  let customArtUrl = $state('');
  let isApplyingArt = $state(false);
  let failedUrls = $state(new Set());
  let appliedMap = $state({});

  let gameName = $derived(game?.name || '');
  let gameDir = $derived(game?.gameDir || '');
  let appId = $derived(game?.appId || '');
  let numericSteamAppId = $derived((() => {
    const n = parseInt(String(appId), 10);
    return !isNaN(n) && n > 0 ? n : 0;
  })());

  let currentActiveArtUrl = $derived((() => {
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
  })());

  const assetTypes = [
    { id: 'cover', label: 'Cover Art', hint: '600×900' },
    { id: 'banner', label: 'Banner Header', hint: '460×215' },
    { id: 'hero', label: 'Hero', hint: '1920×620' },
    { id: 'logo', label: 'Logo', hint: 'PNG' }
  ];

  let lastKnownGameKey = '';
  // Fetch artwork candidate options when selected target game changes
  $effect(() => {
    const currentKey = `${gameDir}:${appId}:${gameName}`;
    if (currentKey !== lastKnownGameKey) {
      const wasInitial = lastKnownGameKey === '';
      lastKnownGameKey = currentKey;
      if (!wasInitial) {
        artQuery = gameName || (appId ? String(appId) : '');
        fetchArtworkOptions();
      }
    }
  });

  onMount(() => {
    artQuery = gameName || (appId ? String(appId) : '');
    fetchArtworkOptions();
  });

  export async function fetchArtworkOptions() {
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

  function selectAssetType(type) {
    if (assetType === type) return;
    assetType = type;
    onassetTypeChange?.(type);
    fetchArtworkOptions();
  }

  export async function handleApplyArtwork(imageUrlOrPath) {
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
      onapply?.(targetUrl);
      onupdated?.(updated);
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

  export async function handleBrowseLocalArt() {
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

  function handleCustomUrlApply() {
    if (!customArtUrl.trim() || isApplyingArt) return;
    handleApplyArtwork(customArtUrl.trim());
    customArtUrl = '';
  }

  function handleImageError(url, thumbUrl) {
    failedUrls = new Set([...failedUrls, url, thumbUrl].filter(Boolean));
  }
</script>

<div class="space-y-4">
  <!-- Asset Type Selector & Search Bar -->
  <div class="flex flex-col sm:flex-row items-center justify-between gap-3 bg-surface-card border border-surface-border p-3 rounded-2xl">
    <!-- Type Pills -->
    <div class="flex items-center bg-surface-input border border-surface-border rounded-xl p-1 gap-1 text-xs w-full sm:w-auto overflow-x-auto">
      {#each assetTypes as type}
        {@const isActive = assetType === type.id}
        <button
          type="button"
          onclick={() => selectAssetType(type.id)}
          class="px-3 py-1.5 rounded-lg font-bold transition-all cursor-pointer flex items-center space-x-1.5 shrink-0 {isActive ? 'bg-white text-neutral-950 shadow-sm' : 'text-neutral-400 hover:text-white hover:bg-surface-cardHover'}"
        >
          <span>{type.label}</span>
          <span class="text-3xs font-mono {isActive ? 'text-neutral-600' : 'text-neutral-500'}">({type.hint})</span>
        </button>
      {/each}
    </div>

    <!-- Search Input -->
    <div class="flex items-center space-x-2 w-full sm:w-auto sm:max-w-xs">
      <div class="relative flex-1">
        <input
          type="text"
          bind:value={artQuery}
          onkeydown={(e) => { if (e.key === 'Enter') fetchArtworkOptions(); }}
          placeholder="Search artwork..."
          class="w-full bg-surface-input border border-surface-border focus:border-neutral-400 rounded-xl px-3 py-1.5 text-xs text-white outline-none font-mono placeholder:text-neutral-500 transition-colors"
        />
      </div>
      <button
        type="button"
        onclick={fetchArtworkOptions}
        disabled={isLoadingArt}
        class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white text-xs font-bold rounded-xl transition-colors cursor-pointer shrink-0"
      >
        Search
      </button>
    </div>
  </div>

  <!-- Custom File & URL Actions -->
  <div class="bg-surface-card border border-surface-border rounded-xl p-2.5 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
    <!-- URL Input -->
    <div class="flex items-center gap-2 flex-1 min-w-0">
      <span class="text-3xs font-mono font-bold uppercase tracking-wider text-neutral-500 pl-1 shrink-0">URL:</span>
      <input
        type="url"
        bind:value={customArtUrl}
        placeholder="https://example.com/custom-artwork.jpg"
        class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-3 py-1 text-xs text-neutral-200 outline-none font-mono placeholder:text-neutral-600"
      />
      <button
        type="button"
        onclick={handleCustomUrlApply}
        disabled={!customArtUrl.trim() || isApplyingArt}
        class="px-3.5 py-1 bg-surface-card hover:bg-surface-cardHover disabled:opacity-40 border border-surface-border text-neutral-200 hover:text-white text-xs font-bold rounded-lg transition-colors shrink-0 cursor-pointer"
      >
        {isApplyingArt ? 'Applying...' : 'Apply'}
      </button>
    </div>

    <!-- Quick File Picker & Reset -->
    <div class="flex items-center space-x-2 shrink-0 pt-2 sm:pt-0 border-t sm:border-t-0 border-surface-border">
      <button
        type="button"
        onclick={handleBrowseLocalArt}
        class="px-3 py-1 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white text-xs font-bold rounded-lg transition-colors cursor-pointer flex items-center space-x-1.5"
      >
        <Icon name="folder" className="w-3.5 h-3.5" />
        <span>Browse File...</span>
      </button>

      <button
        type="button"
        onclick={() => handleApplyArtwork('none')}
        disabled={isApplyingArt}
        class="px-3 py-1 bg-rose-950/20 hover:bg-rose-900/40 border border-rose-800/30 text-rose-300 text-xs font-bold rounded-lg transition-colors cursor-pointer"
        title="Reset active asset to monogram fallback"
      >
        Reset Art
      </button>
    </div>
  </div>

  <!-- Artwork Options Grid -->
  {#if isLoadingArt}
    <div class="h-64 flex flex-col items-center justify-center text-neutral-400 space-y-3 font-mono text-xs">
      <Icon name="spinner" className="w-7 h-7 text-white" />
      <p>Discovering artwork options across providers...</p>
    </div>
  {:else if artOptions.length === 0}
    <div class="h-64 flex flex-col items-center justify-center text-neutral-500 text-xs font-mono space-y-2 text-center">
      <p>No {assetType} artwork candidates found for "{artQuery}".</p>
      <p class="text-3xs text-neutral-600">Try searching with a simplified game title or enter a custom image URL / local file above.</p>
    </div>
  {:else}
    <div class="grid {assetType === 'cover' ? 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5' : 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3'} gap-4 pb-8">
      {#each artOptions.filter(o => !failedUrls.has(o.url) && !failedUrls.has(o.thumb_url)) as opt, index (opt.url || index)}
        {@const isActive = Boolean(currentActiveArtUrl && (currentActiveArtUrl === opt.url || currentActiveArtUrl.includes(opt.url) || opt.url.includes(currentActiveArtUrl)))}
        <div class="group relative rounded-2xl overflow-hidden bg-surface-card border transition-all duration-200 flex flex-col {isActive ? 'border-emerald-500 ring-2 ring-emerald-500/40 shadow-lg shadow-emerald-500/10' : 'border-surface-border hover:border-neutral-500'}">
          <!-- Image Container -->
          <div class="relative w-full {assetType === 'cover' ? 'aspect-2/3' : assetType === 'banner' ? 'aspect-460/215' : assetType === 'hero' ? 'aspect-16/9' : 'aspect-video p-4 flex items-center justify-center'} bg-black/40 overflow-hidden">
            <img
              src={opt.thumb_url || opt.url}
              alt=""
              class="w-full h-full {assetType === 'logo' ? 'object-contain' : 'object-cover'} group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
              onerror={() => handleImageError(opt.url, opt.thumb_url)}
            />

            <!-- Provider Badge -->
            <div class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-4xs font-mono font-bold uppercase tracking-wider text-white shadow-md bg-black/85 border border-white/10">
              {opt.provider === 'steam' ? 'Steam' : opt.provider === 'gog' ? 'GOG' : opt.provider === 'rawg' ? 'RAWG' : opt.provider === 'steamgriddb' ? 'SGDB' : opt.provider}
            </div>

            <!-- Active Status Badge -->
            {#if isActive}
              <div class="absolute top-2 right-2 px-2 py-0.5 rounded-md text-4xs font-black uppercase tracking-wider text-neutral-950 bg-emerald-400 shadow-md flex items-center space-x-1">
                <Icon name="check" className="w-2.5 h-2.5" />
                <span>Active</span>
              </div>
            {/if}

            <!-- Dimensions Badge -->
            {#if opt.width && opt.height}
              <div class="absolute bottom-2 left-2 px-1.5 py-0.5 rounded bg-black/90 text-4xs font-mono text-neutral-300 border border-white/10 font-bold">
                {opt.width}×{opt.height}
              </div>
            {/if}
          </div>

          <!-- Bottom Card Action -->
          <div class="p-2.5 bg-surface-card border-t border-surface-border flex items-center justify-between">
            <span class="text-3xs font-mono text-neutral-400 capitalize">{opt.style || 'Custom'}</span>
            <button
              type="button"
              onclick={() => handleApplyArtwork(opt.url)}
              disabled={isApplyingArt || isActive}
              class="px-3 py-1 font-bold text-xs rounded-lg transition-colors cursor-pointer {isActive ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 cursor-default' : 'bg-white hover:bg-neutral-200 text-neutral-950'}"
            >
              {isActive ? 'Applied' : isApplyingArt ? 'Applying...' : 'Select'}
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
