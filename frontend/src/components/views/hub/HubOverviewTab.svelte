<script module>
  // In-memory module cache across tab switches within the game hub
  export const overviewNewsCache = new Map();
  export const overviewAchievementsCache = new Map();

  export function clearOverviewHubCache() {
    overviewNewsCache.clear();
    overviewAchievementsCache.clear();
  }
</script>

<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../api/app.js';
  import { modalStore } from '../../../stores/modal.js';
  import { isSteamApp } from '../../../utils/game.js';
  import { openExternalUrl as openLink } from '../../../utils/navigation.js';
  import HubAboutCard from './cards/HubAboutCard.svelte';
  import HubExtendedInfoCard from './cards/HubExtendedInfoCard.svelte';
  import AchievementRibbon from '../../blocks/AchievementRibbon.svelte';
  import PatchNotesFeed from '../../blocks/PatchNotesFeed.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null ,
    onnavigateTab
  } = $props();




  // Achievements State
  let achievements = $state([]);
  let isLoadingAchievements = false;
  let loadedAchievementsDir = $state('');

  // News State
  let newsItems = $state([]);
  let isLoadingNews = false;
  let loadedNewsAppId = $state('');







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
    onnavigateTab?.({ tab: 'community', subTab, newsIndex });
  }
  let name = $derived(game?.name || '');
  let appId = $derived(game?.appId || '');
  let gogId = $derived(game?.gogId || '');
  let isGOG = $derived(Boolean(gogId || (game?.store && game.store.toLowerCase().includes('gog'))));
  let effectiveAppId = $derived((game?.appId && isSteamApp(game.appId))
    ? String(game.appId).trim()
    : ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId) ? String(game.metadataId).trim() : ''));
  let isSteam = $derived(Boolean(effectiveAppId));
  let pcgwUrl = $derived(isSteam
    ? `https://www.pcgamingwiki.com/api/appid.php?appid=${encodeURIComponent(effectiveAppId)}`
    : isGOG && gogId
    ? `https://www.pcgamingwiki.com/api/gog.php?page=${encodeURIComponent(gogId)}`
    : name ? `https://www.pcgamingwiki.com/w/index.php?search=${encodeURIComponent(name)}` : '');
  let protonDbUrl = $derived(isSteam ? `https://www.protondb.com/app/${encodeURIComponent(effectiveAppId)}` : (name ? `https://www.protondb.com/search?q=${encodeURIComponent(name)}` : ''));
  let steamDbUrl = $derived(isSteam ? `https://steamdb.info/app/${encodeURIComponent(effectiveAppId)}/` : (name ? `https://steamdb.info/search/?a=app&q=${encodeURIComponent(name)}` : ''));
  let hltbUrl = $derived(name ? `https://howlongtobeat.com/?q=${encodeURIComponent(name)}` : '');
  // Restore cached items immediately to prevent UI flicker on tab return
  $effect(() => {
    if (effectiveAppId && overviewNewsCache.has(effectiveAppId)) {
      const cached = overviewNewsCache.get(effectiveAppId);
      if (cached && cached.length > 0 && newsItems.length === 0) {
        newsItems = cached;
      }
    } else if (newsItems.length === 0 && game?.newsItems && game.newsItems.length > 0) {
      newsItems = game.newsItems.slice(0, 3);
    }
  });

  $effect(() => {
    if (game?.gameDir && overviewAchievementsCache.has(game.gameDir)) {
      const cachedAch = overviewAchievementsCache.get(game.gameDir);
      if (cachedAch && cachedAch.length > 0 && achievements.length === 0) {
        achievements = cachedAch;
      }
    }
  });

  $effect(() => {
    if (game?.gameDir && game.gameDir !== loadedAchievementsDir) {
      loadAchievements();
    }
  });

  $effect(() => {
    if (effectiveAppId && effectiveAppId !== loadedNewsAppId) {
      loadNews();
    }
  });
</script>

<div class="space-y-6 animate-fade-in text-neutral-200">
  <!-- Update Available Spotlight Card -->
  {#if (game?.versionStatus && !game.versionStatus.is_up_to_date) || game?.hasUpdate}
    <div class="bg-gradient-to-r from-amber-500/10 via-amber-500/5 to-transparent border border-amber-500/30 rounded-2xl p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 shadow-lg">
      <div class="flex items-start sm:items-center space-x-3.5 min-w-0">
        <div class="p-2.5 rounded-xl bg-amber-500/20 text-amber-300 border border-amber-500/30 flex-shrink-0">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        </div>
        <div class="min-w-0">
          <div class="flex items-center space-x-2">
            <span class="text-xs font-bold text-amber-200">Update Available</span>
            {#if game.versionStatus?.latest_build_id}
              <span class="text-3xs font-mono px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">
                Build #{game.versionStatus.latest_build_id}
              </span>
            {/if}
          </div>
          <p class="text-2xs text-neutral-400 mt-0.5 truncate">
            {game.versionStatus?.latest_patch_title || game.versionStatus?.message || 'A newer build is available for this title.'}
          </p>
        </div>
      </div>
      <button
        type="button"
        onclick={() => modalStore.open('update', { game, versionStatus: game.versionStatus })}
        class="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-neutral-950 text-xs font-black transition-colors flex items-center space-x-1.5 flex-shrink-0 shadow-md cursor-pointer"
      >
        <span>View Update Details</span>
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- Game Summary Synopsis & Key Facts -->
  <HubAboutCard {game} />

  <!-- Quick Resources / External Knowledge Links -->
  {#if pcgwUrl || protonDbUrl || steamDbUrl || hltbUrl}
    <div class="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
      {#if pcgwUrl}
        <button
          type="button"
          onclick={() => openLink(pcgwUrl, 'PCGamingWiki')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>PCGamingWiki</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
      {#if protonDbUrl}
        <button
          type="button"
          onclick={() => openLink(protonDbUrl, 'ProtonDB')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>ProtonDB</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
      {#if steamDbUrl}
        <button
          type="button"
          onclick={() => openLink(steamDbUrl, 'SteamDB')}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer text-2xs font-semibold"
        >
          <span>SteamDB</span>
          <svg class="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
        </button>
      {/if}
      {#if hltbUrl}
        <button
          type="button"
          onclick={() => openLink(hltbUrl, 'HowLongToBeat')}
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
        onviewAll={() => navigateToCommunity('achievements')}
        onselect={() => navigateToCommunity('achievements')}
      />
    </div>
  {/if}

  <!-- Latest Updates & Patch Notes Feed -->
  {#if newsItems.length > 0}
    <div class="bg-[#121215] border border-surface-border rounded-2xl p-5 shadow-sm">
      <PatchNotesFeed
        {newsItems}
        onviewAll={() => navigateToCommunity('news')}
        onread={(val) => navigateToCommunity('news', val?.index ?? 0)}
      />
    </div>
  {/if}

  <!-- Extended Game Info & Statistics Card -->
  <HubExtendedInfoCard {game} />
</div>
