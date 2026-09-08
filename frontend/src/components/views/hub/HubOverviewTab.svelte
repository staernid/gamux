<script context="module">
  // In-memory module cache across tab switches within the game hub
  export const overviewNewsCache = new Map();
  export const overviewAchievementsCache = new Map();

  export function clearOverviewHubCache() {
    overviewNewsCache.clear();
    overviewAchievementsCache.clear();
  }
</script>

<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../../api/app.js';
  import { isSteamApp } from '../../../utils/game.js';
  import HubAboutCard from './cards/HubAboutCard.svelte';
  import AchievementRibbon from '../../blocks/AchievementRibbon.svelte';
  import PatchNotesFeed from '../../blocks/PatchNotesFeed.svelte';

  export let game = null;

  const dispatch = createEventDispatcher();

  // Achievements State
  let achievements = [];
  let isLoadingAchievements = false;
  let loadedAchievementsDir = '';

  // News State
  let newsItems = [];
  let isLoadingNews = false;
  let loadedNewsAppId = '';

  $: effectiveAppId = (game?.appId && isSteamApp(game.appId))
    ? String(game.appId).trim()
    : ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId) ? String(game.metadataId).trim() : '');

  $: isSteam = Boolean(effectiveAppId);

  // Restore cached items immediately to prevent UI flicker on tab return
  $: if (effectiveAppId && overviewNewsCache.has(effectiveAppId)) {
    const cached = overviewNewsCache.get(effectiveAppId);
    if (cached && cached.length > 0 && newsItems.length === 0) {
      newsItems = cached;
    }
  } else if (newsItems.length === 0 && game?.newsItems && game.newsItems.length > 0) {
    newsItems = game.newsItems.slice(0, 3);
  }

  $: if (game?.gameDir && overviewAchievementsCache.has(game.gameDir)) {
    const cachedAch = overviewAchievementsCache.get(game.gameDir);
    if (cachedAch && cachedAch.length > 0 && achievements.length === 0) {
      achievements = cachedAch;
    }
  }

  $: if (game?.gameDir && game.gameDir !== loadedAchievementsDir) {
    loadAchievements();
  }

  $: if (effectiveAppId && effectiveAppId !== loadedNewsAppId) {
    loadNews();
  }

  onMount(() => {
    if (game?.gameDir && game.gameDir !== loadedAchievementsDir) {
      loadAchievements();
    }
    if (effectiveAppId && effectiveAppId !== loadedNewsAppId) {
      loadNews();
    }
  });

  async function loadAchievements() {
    if (!game?.gameDir) return;
    loadedAchievementsDir = game.gameDir;
    isLoadingAchievements = achievements.length === 0;
    try {
      const res = (await AppAPI.getGameAchievements(game.gameDir)) || [];
      achievements = res;
      overviewAchievementsCache.set(game.gameDir, res);
    } catch {
      if (achievements.length === 0) {
        achievements = [];
      }
    } finally {
      isLoadingAchievements = false;
    }
  }

  async function loadNews() {
    if (!effectiveAppId) return;
    loadedNewsAppId = effectiveAppId;
    isLoadingNews = newsItems.length === 0;
    try {
      const items = await AppAPI.fetchNews(effectiveAppId, 3);
      if (items && items.length > 0) {
        // Never replace a richer news list with a smaller one
        if (newsItems.length === 0 || items.length >= newsItems.length) {
          newsItems = items.slice(0, 3);
          overviewNewsCache.set(effectiveAppId, newsItems);
        }
      }
    } catch {
      if (newsItems.length === 0) {
        newsItems = [];
      }
    } finally {
      isLoadingNews = false;
    }
  }

  function navigateToCommunity(subTab = 'achievements', newsIndex = 0) {
    dispatch('navigateTab', { tab: 'community', subTab, newsIndex });
  }
</script>

<div class="space-y-6 animate-fade-in text-neutral-200">
  <!-- Game Summary Synopsis & Key Facts -->
  <HubAboutCard {game} />

  <!-- Achievements Ribbon Spotlight -->
  {#if achievements.length > 0}
    <div class="bg-[#121215] border border-surface-border rounded-2xl p-5 shadow-sm">
      <AchievementRibbon
        {achievements}
        on:viewAll={() => navigateToCommunity('achievements')}
        on:select={() => navigateToCommunity('achievements')}
      />
    </div>
  {/if}

  <!-- Latest Updates & Patch Notes Feed -->
  {#if newsItems.length > 0}
    <div class="bg-[#121215] border border-surface-border rounded-2xl p-5 shadow-sm">
      <PatchNotesFeed
        {newsItems}
        on:viewAll={() => navigateToCommunity('news')}
        on:read={(e) => navigateToCommunity('news', e.detail?.index ?? 0)}
      />
    </div>
  {/if}
</div>
