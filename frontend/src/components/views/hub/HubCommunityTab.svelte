<script>
  import { run } from 'svelte/legacy';

  import { AppAPI } from '../../../api/app.js';
  import { isSteamApp } from '../../../utils/game.js';
  import HubAchievementsCard from './cards/HubAchievementsCard.svelte';
  import HubNewsCard from './cards/HubNewsCard.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   * @property {string} [initialSubTab]
   * @property {number} [initialNewsIndex]
   */

  /** @type {Props} */
  let { game = null, initialSubTab = 'achievements', initialNewsIndex = 0 } = $props();

  // svelte-ignore state_referenced_locally
  let activeCommunitySubTab = $state(initialSubTab || 'achievements'); // 'achievements' | 'news'
  // svelte-ignore state_referenced_locally
  let prevInitialSubTab = $state(initialSubTab);
  let newsItems = $state([]);
  let isLoadingNews = $state(false);
  // svelte-ignore state_referenced_locally
  let expandedNewsIndex = $state(initialNewsIndex ?? 0);
  // svelte-ignore state_referenced_locally
  let prevInitialNewsIndex = $state(initialNewsIndex);
  let loadedNewsAppID = $state('');
  let newsError = $state('');






  async function loadNews(force = false) {
    if (!effectiveAppId || (!force && loadedNewsAppID === effectiveAppId && newsItems.length > 0)) return;
    isLoadingNews = true;
    newsError = '';
    try {
      const items = await AppAPI.fetchNews(effectiveAppId, 15);
      newsItems = items || [];
      loadedNewsAppID = effectiveAppId;
      if (initialNewsIndex !== undefined && initialNewsIndex < newsItems.length) {
        expandedNewsIndex = initialNewsIndex;
      } else {
        expandedNewsIndex = 0;
      }
    } catch (err) {
      newsError = err?.message || 'Failed to load Steam community news.';
    } finally {
      isLoadingNews = false;
    }
  }

  function toggleNewsExpand(idx) {
    expandedNewsIndex = expandedNewsIndex === idx ? -1 : idx;
  }
  run(() => {
    if (initialSubTab !== prevInitialSubTab) {
      prevInitialSubTab = initialSubTab;
      activeCommunitySubTab = initialSubTab || 'achievements';
    }
  });
  run(() => {
    if (initialNewsIndex !== prevInitialNewsIndex) {
      prevInitialNewsIndex = initialNewsIndex;
      expandedNewsIndex = initialNewsIndex ?? 0;
    }
  });
  let effectiveAppId = $derived((game?.appId && isSteamApp(game.appId))
    ? String(game.appId).trim()
    : ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId) ? String(game.metadataId).trim() : ''));
  let isSteam = $derived(Boolean(effectiveAppId));
  run(() => {
    if (activeCommunitySubTab === 'news' && loadedNewsAppID !== effectiveAppId && !isLoadingNews && effectiveAppId) {
      loadNews();
    }
  });
</script>

<div class="bg-[#121215] border border-surface-border rounded-2xl p-6 shadow-sm space-y-6">
  <!-- Community Sub-Pills -->
  <div class="flex items-center space-x-2 bg-surface-base p-1.5 rounded-xl border border-[#222226] w-fit">
    <button
      type="button"
      onclick={() => { activeCommunitySubTab = 'achievements'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeCommunitySubTab === 'achievements' ? 'bg-[#222226] text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
    >
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
      </svg>
      <span>Achievements</span>
    </button>

    {#if isSteam}
      <button
        type="button"
        onclick={() => { activeCommunitySubTab = 'news'; }}
        class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeCommunitySubTab === 'news' ? 'bg-[#222226] text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
      >
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
        </svg>
        <span>News & Updates</span>
        {#if newsItems.length > 0}
          <span class="px-1.5 py-0.5 rounded-full bg-[#1c1c20] text-neutral-300 text-3xs font-mono border border-[#303036]">
            {newsItems.length}
          </span>
        {/if}
      </button>
    {/if}
  </div>

  {#if activeCommunitySubTab === 'achievements'}
    <HubAchievementsCard gameDir={game?.gameDir} />
  {:else if activeCommunitySubTab === 'news'}
    <HubNewsCard
      {isLoadingNews}
      {newsError}
      {newsItems}
      {expandedNewsIndex}
      onRefreshNews={() => loadNews(true)}
      onToggleExpand={toggleNewsExpand}
    />
  {/if}
</div>
