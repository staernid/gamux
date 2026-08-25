<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { selectedGame, games } from '../../stores/library.js';
  import { isSamePath } from '../../utils/game.js';

  export let game = null;

  const dispatch = createEventDispatcher();

  let assetType = 'cover'; // 'cover' | 'banner' | 'hero' | 'logo'
  let query = '';
  let options = [];
  let isLoading = false;
  let customUrl = '';
  let isApplying = false;
  let selectedOption = null;
  let failedUrls = new Set();

  $: appId = game?.appId || '';
  $: numericSteamAppId = (() => {
    const n = parseInt(String(appId), 10);
    return (!isNaN(n) && n > 0) ? n : 0;
  })();
  $: gameName = game?.name || '';
  $: gameDir = game?.gameDir || '';

  onMount(() => {
    query = gameName || (appId ? String(appId) : '');
    fetchOptions();
  });

  async function fetchOptions() {
    if (!gameDir && !numericSteamAppId && !gameName) return;
    isLoading = true;
    options = [];
    failedUrls.clear();
    const rawQuery = query.trim();
    const isCustomSearch = Boolean(rawQuery && rawQuery !== gameName && rawQuery !== String(numericSteamAppId));

    try {
      options = await AppAPI.queryArtworkOptions({
        game_dir: gameDir,
        title: rawQuery || gameName,
        steam_app_id: isCustomSearch ? 0 : numericSteamAppId,
        gog_id: game?.gog_id || game?.GOGID || '',
        sgdb_id: game?.sgdb_id || game?.SGDBID || '',
        asset_type: assetType,
      }) || [];
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Fetch Artwork',
        message: err?.message || String(err)
      });
      options = [];
    } finally {
      isLoading = false;
    }
  }

  function handleSelectAssetType(type) {
    if (assetType === type) return;
    assetType = type;
    fetchOptions();
  }

  async function handleApply(imageUrlOrPath) {
    if (!gameDir || !imageUrlOrPath) return;
    isApplying = true;
    try {
      const updated = await AppAPI.applyCustomArtwork(gameDir, assetType, imageUrlOrPath);
      notificationStore.items.add({
        type: 'success',
        title: 'Artwork Updated',
        message: `Updated ${assetType} artwork for ${gameName}`
      });
      if (updated) {
        const coverUrl = updated.cover_url ?? updated.CoverURL ?? updated.custom_cover_url ?? '';
        const localCoverUrl = updated.local_cover_url ?? updated.LocalCoverURL ?? updated.local_cover ?? '';
        const patch = {
          local_cover: updated.local_cover ?? updated.LocalCover,
          local_cover_url: localCoverUrl,
          localCoverUrl,
          cover_url: coverUrl,
          coverUrl,
          custom_cover_url: coverUrl,
          local_banner: updated.local_banner ?? updated.LocalBanner,
          local_banner_url: updated.local_banner_url ?? updated.LocalBannerURL,
          banner_url: updated.banner_url ?? updated.BannerURL,
          custom_banner_url: updated.custom_banner_url ?? updated.CustomBannerURL,
        };
        selectedGame.update(current => {
          if (!current || !isSamePath(current.gameDir, gameDir)) return current;
          return { ...current, ...patch };
        });
        games.update(list => list.map(g => {
          if (isSamePath(g.gameDir, gameDir)) {
            return { ...g, ...patch };
          }
          return g;
        }));
      }
      dispatch('updated', updated);
      dispatch('close');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Apply Artwork',
        message: err?.message || String(err)
      });
    } finally {
      isApplying = false;
    }
  }

  async function handleBrowseLocal() {
    try {
      const selected = await AppAPI.selectFile('Select Image File', 'Images', '*.jpg;*.jpeg;*.png;*.webp');
      if (selected) {
        await handleApply(selected);
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'File Selection Failed',
        message: err?.message || String(err)
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
  class="fixed inset-0 z-50 bg-[#09090c]/92 backdrop-blur-2xl flex flex-col p-6 sm:p-10 select-none animate-in fade-in duration-150 outline-none"
>
  <!-- Header Bar -->
  <div class="flex items-center justify-between max-w-5xl w-full mx-auto mb-6 flex-shrink-0">
    <div>
      <h2 class="text-xl font-black text-white tracking-tight flex items-center gap-2">
        <span>Change Artwork:</span>
        <span class="text-sky-400 truncate max-w-md">{gameName}</span>
      </h2>
      <p class="text-xs text-slate-400 mt-0.5">Select high-quality posters and banners from SteamGridDB, Steam, or custom files.</p>
    </div>

    <!-- Action Buttons -->
    <div class="flex items-center space-x-2.5">
      <button
        type="button"
        on:click={() => handleApply('none')}
        disabled={isApplying}
        class="px-3.5 py-1.5 rounded-xl bg-amber-950/40 hover:bg-amber-900/60 border border-amber-700/50 text-amber-300 hover:text-amber-200 text-xs font-bold transition-colors flex items-center space-x-1.5 shadow-sm"
        title="Remove custom artwork and force fallback monogram tile"
      >
        <span>🔤 Reset to Monogram</span>
      </button>
      <button
        type="button"
        on:click={handleBrowseLocal}
        class="px-3.5 py-1.5 rounded-xl bg-[#1e1e2c] hover:bg-[#2a2a3e] border border-[#34344c] text-white text-xs font-bold transition-colors flex items-center space-x-1.5 shadow-sm"
      >
        <span>📁 Browse Local File...</span>
      </button>
      <button
        type="button"
        on:click={() => dispatch('close')}
        class="px-3 py-1.5 rounded-xl bg-[#1a1a24] hover:bg-[#252535] border border-[#2b2b3c] text-slate-300 hover:text-white text-xs font-mono transition-colors flex items-center space-x-1.5"
      >
        <span>ESC</span>
        <span class="text-slate-500">Close</span>
      </button>
    </div>
  </div>

  <!-- Asset Type Pill Selector -->
  <div class="max-w-5xl w-full mx-auto mb-4 flex items-center justify-between gap-4 flex-shrink-0">
    <div class="flex items-center bg-[#14141e] border border-[#262638] rounded-xl p-1 gap-1 text-xs">
      <button
        type="button"
        on:click={() => handleSelectAssetType('cover')}
        class="px-3.5 py-1.5 rounded-lg font-bold transition-colors {assetType === 'cover' ? 'bg-sky-600 text-white shadow-md' : 'text-slate-400 hover:text-white'}"
      >
        Cover Art (600×900)
      </button>
      <button
        type="button"
        on:click={() => handleSelectAssetType('banner')}
        class="px-3.5 py-1.5 rounded-lg font-bold transition-colors {assetType === 'banner' ? 'bg-sky-600 text-white shadow-md' : 'text-slate-400 hover:text-white'}"
      >
        Banner Header
      </button>
      <button
        type="button"
        on:click={() => handleSelectAssetType('hero')}
        class="px-3.5 py-1.5 rounded-lg font-bold transition-colors {assetType === 'hero' ? 'bg-sky-600 text-white shadow-md' : 'text-slate-400 hover:text-white'}"
      >
        Hero Background (1920×620)
      </button>
      <button
        type="button"
        on:click={() => handleSelectAssetType('logo')}
        class="px-3.5 py-1.5 rounded-lg font-bold transition-colors {assetType === 'logo' ? 'bg-sky-600 text-white shadow-md' : 'text-slate-400 hover:text-white'}"
      >
        Title Logo (PNG)
      </button>
    </div>

    <!-- Search Input for Alternate Title/AppID -->
    <div class="flex-1 max-w-sm flex items-center space-x-2">
      <input
        type="text"
        bind:value={query}
        on:keydown={(e) => { if (e.key === 'Enter') fetchOptions(); }}
        placeholder="Search title or ID on SteamGridDB..."
        class="w-full bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-xl px-3 py-1.5 text-xs text-white outline-none font-mono placeholder:text-slate-500"
      />
      <button
        type="button"
        on:click={fetchOptions}
        class="px-3 py-1.5 bg-[#202030] hover:bg-[#2c2c40] text-slate-200 hover:text-white text-xs font-bold rounded-xl border border-[#323248] transition-colors flex-shrink-0"
      >
        Search
      </button>
    </div>
  </div>

  <!-- Custom Image URL Input Bar -->
  <div class="max-w-5xl w-full mx-auto mb-5 bg-[#12121c] border border-[#262638] rounded-xl p-2.5 flex items-center gap-3 flex-shrink-0">
    <span class="text-[11px] font-bold uppercase tracking-wider text-slate-400 pl-1 flex-shrink-0">Custom URL:</span>
    <input
      type="url"
      bind:value={customUrl}
      placeholder="https://example.com/custom-artwork.jpg"
      class="flex-1 bg-[#0a0a0f] border border-[#1f1f2e] focus:border-sky-500 rounded-lg px-3 py-1 text-xs text-slate-100 outline-none font-mono placeholder:text-slate-600"
    />
    <button
      type="button"
      on:click={() => handleApply(customUrl.trim())}
      disabled={!customUrl.trim() || isApplying}
      class="px-4 py-1 bg-sky-600 hover:bg-sky-500 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition-colors flex-shrink-0 shadow"
    >
      {isApplying ? 'Applying...' : 'Apply URL'}
    </button>
  </div>

  <!-- Candidates Grid Area -->
  <div class="flex-1 max-w-5xl w-full mx-auto overflow-y-auto pr-2 custom-scrollbar">
    {#if isLoading}
      <div class="h-64 flex flex-col items-center justify-center text-slate-400 space-y-3 font-mono text-xs">
        <div class="w-7 h-7 border-2 border-sky-500 border-t-transparent rounded-full animate-spin"></div>
        <p>Querying SteamGridDB and Steam providers...</p>
      </div>
    {:else if options.length === 0}
      <div class="h-64 flex flex-col items-center justify-center text-slate-500 text-xs font-mono space-y-2 text-center">
        <p>No {assetType} artwork candidates found for "{query}".</p>
        <p class="text-[11px] text-slate-600">Try searching with a simplified game title or enter a custom image URL above.</p>
      </div>
    {:else}
      <div class="grid {assetType === 'cover' ? 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5' : 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3'} gap-4 pb-8">
        {#each options.filter(o => !failedUrls.has(o.url) && !failedUrls.has(o.thumb_url)) as opt, index (opt.url || index)}
          {@const isSelected = selectedOption?.url === opt.url}
          <div
            class="group relative rounded-2xl overflow-hidden bg-[#14141e] border transition-all duration-200 flex flex-col {isSelected ? 'border-sky-500 ring-2 ring-sky-500/40 scale-[1.02]' : 'border-[#262638] hover:border-slate-400'}"
          >
            <!-- Image Card -->
            <div class="relative w-full {assetType === 'cover' ? 'aspect-[2/3]' : assetType === 'banner' ? 'aspect-[460/215]' : assetType === 'hero' ? 'aspect-[16/9]' : 'aspect-video p-4 flex items-center justify-center'} bg-black/40 overflow-hidden">
              <img
                src={opt.thumb_url || opt.url}
                alt=""
                class="w-full h-full {assetType === 'logo' ? 'object-contain' : 'object-cover'} group-hover:scale-105 transition-transform duration-300"
                loading="lazy"
                on:error={() => {
                  failedUrls = new Set([...failedUrls, opt.url, opt.thumb_url].filter(Boolean));
                }}
              />

              <!-- Provider Tag -->
              <div class="absolute top-2 left-2 px-2 py-0.5 rounded-md text-[9px] font-black uppercase tracking-wider text-white shadow-md backdrop-blur-md {opt.provider === 'steam' ? 'bg-blue-600/90' : 'bg-emerald-600/90'}">
                {opt.provider === 'steam' ? 'Steam Official' : 'SteamGridDB'}
              </div>

              <!-- Dimensions Tag -->
              {#if opt.width && opt.height}
                <div class="absolute bottom-2 left-2 px-1.5 py-0.5 rounded bg-black/75 backdrop-blur-sm text-[9px] font-mono text-slate-300 border border-white/10 font-bold">
                  {opt.width}×{opt.height}
                </div>
              {/if}
            </div>

            <!-- Action Bar Below Card -->
            <div class="p-2.5 bg-[#0f0f16] border-t border-[#1e1e2c] flex items-center justify-between">
              <span class="text-[10px] font-mono text-slate-400 capitalize">{opt.style || 'Custom'}</span>
              <button
                type="button"
                on:click={() => handleApply(opt.url)}
                disabled={isApplying}
                class="px-3 py-1 bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs rounded-lg transition-colors shadow"
              >
                {isApplying ? 'Applying...' : 'Select'}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
