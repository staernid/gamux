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
  import { openExternalUrl as openLink } from '../../../utils/navigation.js';
  import HubAboutCard from './cards/HubAboutCard.svelte';
  import AchievementRibbon from '../../blocks/AchievementRibbon.svelte';
  import PatchNotesFeed from '../../blocks/PatchNotesFeed.svelte';

  export let game = null;

  const dispatch = createEventDispatcher();

  $: name = game?.name || '';
  $: appId = game?.appId || '';
  $: gogId = game?.gogId || '';
  $: isGOG = Boolean(gogId || (game?.store && game.store.toLowerCase().includes('gog')));

  $: pcgwUrl = isSteam
    ? `https://www.pcgamingwiki.com/api/appid.php?appid=${encodeURIComponent(effectiveAppId)}`
    : isGOG && gogId
    ? `https://www.pcgamingwiki.com/api/gog.php?page=${encodeURIComponent(gogId)}`
    : name ? `https://www.pcgamingwiki.com/w/index.php?search=${encodeURIComponent(name)}` : '';

  $: protonDbUrl = isSteam ? `https://www.protondb.com/app/${encodeURIComponent(effectiveAppId)}` : (name ? `https://www.protondb.com/search?q=${encodeURIComponent(name)}` : '');
  $: steamDbUrl = isSteam ? `https://steamdb.info/app/${encodeURIComponent(effectiveAppId)}/` : (name ? `https://steamdb.info/search/?a=app&q=${encodeURIComponent(name)}` : '');
  $: hltbUrl = name ? `https://howlongtobeat.com/?q=${encodeURIComponent(name)}` : '';

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

  <!-- Quick Resources / External Knowledge Links -->
  {#if pcgwUrl || protonDbUrl || steamDbUrl || hltbUrl}
    <div class="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
      {#if pcgwUrl}
        <button
          type="button"
          on:click={() => openLink(pcgwUrl, 'PCGamingWiki')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>PCGamingWiki</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
      {#if protonDbUrl}
        <button
          type="button"
          on:click={() => openLink(protonDbUrl, 'ProtonDB')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>ProtonDB</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
      {#if steamDbUrl}
        <button
          type="button"
          on:click={() => openLink(steamDbUrl, 'SteamDB')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>SteamDB</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
      {#if hltbUrl}
        <button
          type="button"
          on:click={() => openLink(hltbUrl, 'HowLongToBeat')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>HowLongToBeat</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
    </div>
  {/if}

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
