<script module>
  // In-memory module cache for metadata responses across component remounts
  const metadataCache = {};

  export function clearDrawerMetadataCache() {
    for (const key of Object.keys(metadataCache)) {
      delete metadataCache[key];
    }
  }
</script>

<script>
  import { AppAPI } from '../../../../api/app.js';
  import { modalStore } from '../../../../stores/modal.js';
  import { getMetadataInfo, isSteamApp, getGameSource, normalizeGameMetadata } from '../../../../utils/game.js';
  import { GameSynopsis, GenrePillGroup } from '../../../blocks/index.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null } = $props();

  let metadata = $state(null);
  let isLoading = $state(false);
  let currentLookupKey = $state('');

  let metaInfo = $derived(game ? getMetadataInfo(game) : { provider: 'None', id: '' });
  let gameSource = $derived(game ? getGameSource(game) : 'Standalone');
  let providerKey = $derived((metaInfo.provider || '').toLowerCase());
  let targetLookupId = $derived(metaInfo.id || '');
  let lookupKey = $derived((providerKey && providerKey !== 'none' && targetLookupId && targetLookupId !== '0')
    ? `${providerKey}:${targetLookupId}`
    : '');

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
      const meta = normalizeGameMetadata(res);
      if (meta && (meta.shortDescription || meta.aboutTheGame || meta.detailedDescription || meta.developers.length > 0)) {
        metadataCache[key] = meta;
        if (lookupKey === key) {
          metadata = meta;
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

  // Asynchronously fetch store metadata when lookup key changes
  $effect(() => {
    if (lookupKey !== currentLookupKey && !isLoading) {
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

  let summaryText = $derived(metadata?.shortDescription
    ? cleanDescription(metadata.shortDescription)
    : metadata?.aboutTheGame
    ? cleanDescription(metadata.aboutTheGame)
    : game?.description
    ? cleanDescription(game.description)
    : '');

  let developers = $derived(metadata?.developers || []);
  let publishers = $derived(metadata?.publishers || []);
  let genres = $derived(metadata?.genres || []);
  let releaseDate = $derived(metadata?.releaseDate || '');
</script>

<div class="space-y-4 animate-fade-in text-neutral-200">
  <!-- Game Summary Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-sm space-y-4">
    {#if isLoading && !metadata}
      <!-- Soft Loading Skeleton -->
      <div class="space-y-2.5 animate-pulse py-1">
        <div class="h-4 bg-neutral-800 rounded w-1/4"></div>
        <div class="h-3 bg-neutral-800/60 rounded w-full"></div>
        <div class="h-3 bg-neutral-800/60 rounded w-5/6"></div>
      </div>
    {:else if summaryText}
      <GameSynopsis
        description={summaryText}
        {developers}
        {publishers}
        {releaseDate}
        buildId={game?.buildId}
      />

      <div class="pt-2 border-t border-surface-border flex items-center justify-between gap-2 flex-wrap">
        {#if genres.length > 0}
          <GenrePillGroup {genres} maxPills={6} />
        {:else}
          <div></div>
        {/if}
        <button
          type="button"
          onclick={() => modalStore.open('metadata-link', { game })}
          class="text-3xs font-mono font-medium text-neutral-400 hover:text-white px-2.5 py-1 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border transition flex items-center gap-1.5 cursor-pointer ml-auto"
          title="Edit or unlink store metadata match"
        >
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
          <span>Edit Store Match</span>
        </button>
      </div>
    {:else}
      <!-- Standalone / Unlinked Title Information -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 py-1">
        <div class="space-y-1">
          <h4 class="text-sm font-bold text-white">{game?.name || 'Local Title'}</h4>
          <p class="text-xs text-neutral-400 max-w-lg leading-relaxed">
            No official store metadata linked yet. Link Steam or GOG metadata to display store descriptions, release dates, genres, and community resources.
          </p>
        </div>
        <div class="flex items-center space-x-2 flex-shrink-0">
          <button
            type="button"
            onclick={() => modalStore.open('metadata-link', { game })}
            class="px-3.5 py-2 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 text-xs font-bold transition shadow-lg flex items-center space-x-1.5 cursor-pointer"
          >
            <span>Link Metadata</span>
          </button>
          <button
            type="button"
            onclick={() => modalStore.open('artwork-picker', { game })}
            class="px-3.5 py-2 rounded-xl bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white text-xs font-bold transition cursor-pointer"
          >
            <span>Artwork</span>
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
