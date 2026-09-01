<script>
  import { createEventDispatcher } from 'svelte';

  export let assetType = 'cover'; // 'cover' | 'banner' | 'hero' | 'logo'
  export let artQuery = '';
  export let artOptions = [];
  export let isLoadingArt = false;
  export let customArtUrl = '';
  export let isApplyingArt = false;
  export let failedUrls = new Set();
  export let currentActiveArtUrl = '';

  const dispatch = createEventDispatcher();

  function selectAssetType(type) {
    if (assetType === type) return;
    assetType = type;
    dispatch('selectAssetType', type);
  }

  function handleSearch() {
    dispatch('search');
  }

  function handleApply(url) {
    dispatch('apply', url);
  }

  function handleCustomUrlApply() {
    if (!customArtUrl.trim() || isApplyingArt) return;
    dispatch('apply', customArtUrl.trim());
  }

  function handleImageError(url, thumbUrl) {
    failedUrls = new Set([...failedUrls, url, thumbUrl].filter(Boolean));
  }
</script>

<div class="space-y-4">
  <!-- Asset Type & Search Controls -->
  <div class="flex flex-col sm:flex-row items-center justify-between gap-3 bg-[#12121c] border border-[#262638] p-3 rounded-2xl">
    <div class="flex items-center bg-[#0a0a0f] border border-[#222232] rounded-xl p-1 gap-1 text-xs w-full sm:w-auto overflow-x-auto">
      <button
        type="button"
        on:click={() => selectAssetType('cover')}
        class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'cover' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
      >
        Cover Art (600×900)
      </button>
      <button
        type="button"
        on:click={() => selectAssetType('banner')}
        class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'banner' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
      >
        Banner Header
      </button>
      <button
        type="button"
        on:click={() => selectAssetType('hero')}
        class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'hero' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
      >
        Hero (1920×620)
      </button>
      <button
        type="button"
        on:click={() => selectAssetType('logo')}
        class="px-3 py-1 rounded-lg font-bold transition-colors cursor-pointer {assetType === 'logo' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white'}"
      >
        Logo (PNG)
      </button>
    </div>

    <div class="flex items-center space-x-2 w-full sm:w-auto sm:max-w-xs">
      <input
        type="text"
        bind:value={artQuery}
        on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
        placeholder="Search artwork..."
        class="flex-1 bg-[#0a0a0f] border border-[#222232] focus:border-sky-500 rounded-xl px-3 py-1.5 text-xs text-white outline-none font-mono placeholder:text-slate-500"
      />
      <button
        type="button"
        on:click={handleSearch}
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
      on:click={handleCustomUrlApply}
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
              on:error={() => handleImageError(opt.url, opt.thumb_url)}
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
              on:click={() => handleApply(opt.url)}
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
</div>
