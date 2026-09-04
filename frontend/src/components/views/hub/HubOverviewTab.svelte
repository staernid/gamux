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
  import DrawerAboutTab from '../../stage/drawer/DrawerAboutTab.svelte';

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

  // Achievements Derived
  $: unlockedCount = achievements.filter(a => a.unlocked).length;
  $: totalAchievements = achievements.length;
  $: achievementPercent = totalAchievements > 0 ? Math.round((unlockedCount / totalAchievements) * 100) : 0;
  $: recentUnlocked = achievements
    .filter(a => a.unlocked)
    .sort((a, b) => (b.unlock_time || 0) - (a.unlock_time || 0))
    .slice(0, 5);
  $: spotlightAchievements = recentUnlocked.length > 0
    ? recentUnlocked
    : achievements.slice(0, 5);

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

  function cleanExcerpt(text) {
    if (!text) return '';
    return text
      .replace(/<[^>]*>/g, '')
      .replace(/\[\/?(b|i|u|h1|h2|h3|quote|url|img|list|\*)[^\]]*\]/gi, '')
      .replace(/&quot;/g, '"')
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&#39;/g, "'")
      .replace(/\s+/g, ' ')
      .trim();
  }

  function formatDate(timestamp) {
    if (!timestamp) return '';
    const date = new Date(timestamp > 1e11 ? timestamp : timestamp * 1000);
    return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
  }

  function navigateToCommunity(subTab = 'achievements', newsIndex = 0) {
    dispatch('navigateTab', { tab: 'community', subTab, newsIndex });
  }
</script>

<div class="space-y-6 animate-fade-in text-neutral-200">
  <!-- Game Summary Synopsis & Key Facts -->
  <DrawerAboutTab {game} />

  <!-- Minimal Achievements Spotlight (if title has achievements) -->
  {#if totalAchievements > 0}
    <div class="bg-[#121215] border border-surface-border rounded-2xl p-5 shadow-sm space-y-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-3">
          <div class="w-8 h-8 rounded-xl bg-[#1c1c20] border border-[#303036] flex items-center justify-center text-neutral-200">
            <svg class="w-4 h-4 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
            </svg>
          </div>
          <div>
            <div class="flex items-center space-x-2">
              <h4 class="font-bold text-white text-xs">Achievements</h4>
              <span class="text-3xs font-mono text-neutral-400 bg-[#1c1c20] px-2 py-0.5 rounded-full border border-[#2c2c34]">
                {unlockedCount} / {totalAchievements} ({achievementPercent}%)
              </span>
            </div>
            <p class="text-3xs text-neutral-400 mt-0.5">
              {unlockedCount === totalAchievements ? '100% Completed' : `${totalAchievements - unlockedCount} remaining to unlock`}
            </p>
          </div>
        </div>

        <button
          type="button"
          on:click={() => navigateToCommunity('achievements')}
          class="text-xs font-semibold text-neutral-300 hover:text-white flex items-center space-x-1.5 transition cursor-pointer"
        >
          <span>View All ({totalAchievements})</span>
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      <!-- Completion Progress Bar -->
      <div class="w-full bg-[#18181c] border border-[#222226] rounded-full h-1.5 overflow-hidden">
        <div
          class="h-full bg-neutral-200 rounded-full transition-all duration-300"
          style="width: {achievementPercent}%"
        ></div>
      </div>

      <!-- Spotlight Achievement Badges Strip -->
      <div class="flex items-center gap-3 overflow-x-auto custom-scrollbar pt-1">
        {#each spotlightAchievements as ach (ach.id)}
          <button
            type="button"
            on:click={() => navigateToCommunity('achievements')}
            title="{ach.name}: {ach.description || 'Achievement'}"
            class="group flex items-center space-x-2.5 p-2 rounded-xl bg-[#0e0e11] hover:bg-[#18181c] border border-[#222226] hover:border-surface-borderHighlight transition cursor-pointer flex-shrink-0"
          >
            <div class="w-10 h-10 rounded-lg overflow-hidden bg-black/40 border border-surface-border relative flex-shrink-0">
              {#if ach.unlocked && ach.icon_url}
                <img src={ach.icon_url} alt={ach.name} class="w-full h-full object-cover" />
              {:else if ach.icon_gray_url || ach.icon_url}
                <img src={ach.icon_gray_url || ach.icon_url} alt={ach.name} class="w-full h-full object-cover filter grayscale contrast-75" />
              {:else}
                <div class="w-full h-full flex items-center justify-center text-neutral-600">
                  <svg class="w-4 h-4 opacity-70" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                  </svg>
                </div>
              {/if}
              {#if ach.unlocked}
                <div class="absolute top-0.5 right-0.5 w-3 h-3 bg-white rounded-full flex items-center justify-center text-black">
                  <svg class="w-2 h-2 text-black" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                </div>
              {/if}
            </div>

            <div class="text-left pr-2 max-w-[130px]">
              <p class="text-xs font-bold truncate text-neutral-200 group-hover:text-white transition">
                {ach.name || ach.id}
              </p>
              <p class="text-3xs text-neutral-400 truncate">
                {ach.unlocked ? 'Unlocked' : 'Locked'}
              </p>
            </div>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <!-- 3 Latest Updates & Patch Notes (if available) -->
  {#if newsItems.length > 0}
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <h4 class="text-2xs font-bold uppercase tracking-wider text-neutral-400 flex items-center space-x-2">
          <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
          </svg>
          <span>Latest Updates & Patch Notes</span>
        </h4>

        <button
          type="button"
          on:click={() => navigateToCommunity('news')}
          class="text-xs font-semibold text-neutral-300 hover:text-white flex items-center space-x-1.5 transition cursor-pointer"
        >
          <span>View All Updates</span>
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
        {#each newsItems as item, idx}
          {@const excerpt = cleanExcerpt(item.contents)}
          <button
            type="button"
            on:click={() => navigateToCommunity('news', idx)}
            class="bg-[#121215] hover:bg-[#18181c] border border-surface-border hover:border-surface-borderHighlight p-4 rounded-2xl transition shadow-sm flex flex-col justify-between space-y-3 text-left group cursor-pointer"
          >
            <div class="space-y-1.5">
              <div class="flex items-center justify-between text-3xs font-mono text-neutral-400">
                <span>{formatDate(item.date)}</span>
                {#if item.feedlabel || item.feed_label}
                  <span class="px-1.5 py-0.5 rounded bg-[#1c1c20] text-neutral-300 border border-[#2c2c34]">
                    {item.feedlabel || item.feed_label}
                  </span>
                {/if}
              </div>

              <h5 class="font-bold text-white text-xs group-hover:text-neutral-100 transition line-clamp-2 leading-snug">
                {item.title}
              </h5>

              {#if excerpt}
                <p class="text-neutral-400 text-xs line-clamp-2 leading-relaxed">
                  {excerpt}
                </p>
              {/if}
            </div>

            <div class="pt-2 border-t border-[#222226] flex items-center justify-between text-2xs text-neutral-400 group-hover:text-white transition">
              <span class="font-medium">Read Update</span>
              <svg class="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </button>
        {/each}
      </div>
    </div>
  {/if}
</div>
