<script context="module">
  // In-memory module cache for metadata responses across component remounts
  const metadataCache = {};

  export function clearDrawerMetadataCache() {
    for (const key of Object.keys(metadataCache)) {
      delete metadataCache[key];
    }
  }
</script>

<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { modalStore } from '../../../../stores/modal.js';
  import { getMetadataInfo, isSteamApp, getGameSource } from '../../../../utils/game.js';

  export let game = null;

  let metadata = null;
  let isLoading = false;
  let currentLookupKey = '';

  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };
  $: gameSource = game ? getGameSource(game) : 'Standalone';
  $: providerKey = (metaInfo.provider || '').toLowerCase();
  $: targetLookupId = metaInfo.id || '';
  $: lookupKey = (providerKey && providerKey !== 'none' && targetLookupId && targetLookupId !== '0')
    ? `${providerKey}:${targetLookupId}`
    : '';

  async function loadMetadata(prov, id, key) {
    if (!key || !prov || prov === 'none' || !id || id === '0') {
      metadata = null;
      currentLookupKey = '';
      return;
    }

    if (metadataCache[key]) {
      metadata = metadataCache[key];
      currentLookupKey = key;
      return;
    }

    isLoading = true;
    try {
      const res = await AppAPI.getGameMetadata(prov, id);
      if (res && (res.short_description || res.about_the_game || res.detailed_description || res.developers)) {
        metadataCache[key] = res;
        if (lookupKey === key) {
          metadata = res;
          currentLookupKey = key;
        }
      } else {
        if (lookupKey === key) {
          metadata = null;
          currentLookupKey = key;
        }
      }
    } catch {
      if (lookupKey === key) {
        metadata = null;
        currentLookupKey = key;
      }
    } finally {
      isLoading = false;
    }
  }

  $: if (lookupKey !== currentLookupKey && !isLoading) {
    loadMetadata(providerKey, targetLookupId, lookupKey);
  }

  onMount(() => {
    if (lookupKey) {
      loadMetadata(providerKey, targetLookupId, lookupKey);
    }
  });

  function cleanDescription(text) {
    if (!text) return '';
    return text
      .replace(/<[^>]*>/g, '')
      .replace(/&quot;/g, '"')
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&#39;/g, "'")
      .trim();
  }

  $: summaryText = metadata?.short_description
    ? cleanDescription(metadata.short_description)
    : metadata?.about_the_game
    ? cleanDescription(metadata.about_the_game)
    : game?.description
    ? cleanDescription(game.description)
    : '';

  $: developers = metadata?.developers || [];
  $: publishers = metadata?.publishers || [];
  $: genres = metadata?.genres || [];
  $: releaseDate = metadata?.release_date || '';
</script>

<div class="space-y-4 animate-fade-in text-slate-300">
  <!-- Game Summary Card -->
  <div class="bg-[#0c0c12]/70 border border-[#1a1a26] rounded-2xl p-6 shadow-sm space-y-4">
    {#if isLoading && !metadata}
      <!-- Soft Loading Skeleton -->
      <div class="space-y-2.5 animate-pulse py-1">
        <div class="h-4 bg-slate-800/60 rounded w-1/4"></div>
        <div class="h-3 bg-slate-800/40 rounded w-full"></div>
        <div class="h-3 bg-slate-800/40 rounded w-5/6"></div>
      </div>
    {:else if summaryText}
      <!-- Synopsis / Story Lore -->
      <div class="space-y-2">
        <p class="text-sm text-slate-300 leading-relaxed font-normal select-text">
          {summaryText}
        </p>
      </div>

      <!-- Compact Game Facts (Developer, Publisher, Release Date, Genres) -->
      {#if developers.length > 0 || publishers.length > 0 || releaseDate || genres.length > 0}
        <div class="flex flex-wrap items-center gap-x-4 gap-y-2 pt-3 border-t border-[#181824] text-xs text-slate-400">
          {#if developers.length > 0}
            <div class="flex items-center space-x-1.5">
              <span class="text-slate-500">Developer:</span>
              <span class="text-slate-300 font-medium">{developers.join(', ')}</span>
            </div>
          {/if}

          {#if publishers.length > 0 && publishers.join(',') !== developers.join(',')}
            <div class="flex items-center space-x-1.5">
              <span class="text-slate-500">Publisher:</span>
              <span class="text-slate-300 font-medium">{publishers.join(', ')}</span>
            </div>
          {/if}

          {#if releaseDate}
            <div class="flex items-center space-x-1.5">
              <span class="text-slate-500">Released:</span>
              <span class="text-slate-300 font-medium">{releaseDate}</span>
            </div>
          {/if}

          {#if game?.buildId && game.buildId !== '0'}
            <div class="flex items-center space-x-1.5 font-mono">
              <span class="text-slate-500">Build:</span>
              <span class="text-slate-300 font-medium">{game.buildId}</span>
            </div>
          {/if}

          {#if genres.length > 0}
            <div class="flex items-center space-x-1.5 ml-auto">
              <div class="flex flex-wrap items-center gap-1.5">
                {#each genres.slice(0, 4) as g}
                  <span class="px-2 py-0.5 rounded-md bg-[#161622] border border-[#242436] text-2xs font-medium text-slate-300">
                    {g}
                  </span>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      {/if}
    {:else}
      <!-- Standalone / Unlinked Title Information -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 py-1">
        <div class="space-y-1">
          <h4 class="text-sm font-bold text-slate-200">{game?.name || 'Local Title'}</h4>
          <p class="text-xs text-slate-400 max-w-lg leading-relaxed">
            No official store metadata linked yet. Link Steam or GOG metadata to display store descriptions, release dates, genres, and community resources.
          </p>
        </div>
        <div class="flex items-center space-x-2 flex-shrink-0">
          <button
            type="button"
            on:click={() => modalStore.open('metadata-link', { game })}
            class="px-3.5 py-2 rounded-xl bg-gradient-to-r from-sky-500 to-blue-600 hover:from-sky-400 hover:to-blue-500 text-white text-xs font-bold transition shadow-lg flex items-center space-x-1.5 cursor-pointer"
          >
            <span>Link Metadata</span>
          </button>
          <button
            type="button"
            on:click={() => modalStore.open('artwork-picker', { game })}
            class="px-3.5 py-2 rounded-xl bg-[#161622] hover:bg-[#202030] border border-[#28283c] text-slate-300 hover:text-white text-xs font-bold transition cursor-pointer"
          >
            <span>Artwork</span>
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
