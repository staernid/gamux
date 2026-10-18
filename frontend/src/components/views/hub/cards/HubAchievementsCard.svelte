<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { formatLastPlayed } from '../../../../utils/game.js';

  /**
   * @typedef {Object} Props
   * @property {string} [gameDir]
   */

  /** @type {Props} */
  let { gameDir = '' } = $props();

  let achievements = $state([]);
  let isLoading = $state(false);
  let error = $state('');
  let loadedGameDir = $state('');
  let filterMode = $state('all'); // 'all' | 'unlocked' | 'locked'
  let sortBy = $state('default'); // 'default' | 'rarest'





  onMount(() => {
    if (gameDir && gameDir !== loadedGameDir) {
      loadAchievements();
    }
  });

  async function loadAchievements() {
    if (!gameDir) return;
    loadedGameDir = gameDir;
    isLoading = true;
    error = '';
    try {
      achievements = await AppAPI.getGameAchievements(gameDir) || [];
    } catch (err) {
      error = err?.message || 'Failed to load achievements';
      achievements = [];
    } finally {
      isLoading = false;
    }
  }
  let unlockedCount = $derived(achievements.filter(a => a.unlocked).length);
  let totalCount = $derived(achievements.length);
  let lockedCount = $derived(totalCount - unlockedCount);
  let percent = $derived(totalCount > 0 ? Math.round((unlockedCount / totalCount) * 100) : 0);
  // Sorting: rarest first or default (unlocked first desc, then locked)
  let sortedAchievements = $derived([...achievements].sort((a, b) => {
    if (sortBy === 'rarest') {
      const pctA = typeof a.global_percent === 'number' && a.global_percent > 0 ? a.global_percent : 100.1;
      const pctB = typeof b.global_percent === 'number' && b.global_percent > 0 ? b.global_percent : 100.1;
      if (pctA !== pctB) {
        return pctA - pctB;
      }
    }
    if (a.unlocked !== b.unlocked) {
      return a.unlocked ? -1 : 1;
    }
    if (a.unlocked) {
      return (b.unlock_time || 0) - (a.unlock_time || 0);
    }
    return 0;
  }));
  let filteredAchievements = $derived(sortedAchievements.filter(a => {
    if (filterMode === 'unlocked') return a.unlocked;
    if (filterMode === 'locked') return !a.unlocked;
    return true;
  }));
  $effect(() => {
    if (gameDir && gameDir !== loadedGameDir) {
      loadAchievements();
    }
  });
</script>

<div class="space-y-6">
  {#if isLoading}
    <div class="flex flex-col items-center justify-center py-12 text-neutral-400">
      <div class="w-7 h-7 border-2 border-white/20 border-t-white rounded-full animate-spin mb-3"></div>
      <p class="text-xs font-mono">Loading achievements...</p>
    </div>
  {:else if error}
    <div class="p-4 rounded-xl bg-rose-950/30 border border-rose-800/40 text-rose-300 text-xs">
      <p class="font-bold">Failed to load achievements</p>
      <p class="mt-1 text-neutral-400 font-mono text-2xs">{error}</p>
    </div>
  {:else if achievements.length === 0}
    <div class="text-center py-12 text-neutral-400">
      <p class="text-sm font-bold text-neutral-300">No Achievements Found</p>
      <p class="text-xs text-neutral-400 mt-1 max-w-sm mx-auto">This title either has no Steam achievements or Goldberg achievement data hasn't been generated yet.</p>
    </div>
  {:else}
    <!-- Progress & Filter Header -->
    <div class="p-4 rounded-2xl bg-surface-card border border-surface-border flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h4 class="text-sm font-black text-white flex items-center gap-2">
          <span>Achievements Progress</span>
          <span class="text-xs text-neutral-200 font-mono">({unlockedCount} / {totalCount})</span>
        </h4>
        <p class="text-xs text-neutral-400 mt-0.5">{percent}% completed</p>
      </div>

      <!-- Filter & Sort Pills -->
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex items-center space-x-1.5 bg-surface-header p-1 rounded-xl border border-surface-border">
          <button
            type="button"
            onclick={() => { filterMode = 'all'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-medium transition-all {filterMode === 'all' ? 'bg-surface-cardHover text-white shadow-sm font-bold border border-surface-borderHighlight' : 'text-neutral-400 hover:text-white'}"
          >
            All <span class="font-mono text-2xs opacity-75">({totalCount})</span>
          </button>
          <button
            type="button"
            onclick={() => { filterMode = 'unlocked'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-medium transition-all {filterMode === 'unlocked' ? 'bg-emerald-950/70 text-emerald-300 border border-emerald-700/50 shadow-sm font-bold' : 'text-neutral-400 hover:text-emerald-300'}"
          >
            Unlocked <span class="font-mono text-2xs opacity-75">({unlockedCount})</span>
          </button>
          <button
            type="button"
            onclick={() => { filterMode = 'locked'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-medium transition-all {filterMode === 'locked' ? 'bg-surface-cardHover text-white shadow-sm font-bold border border-surface-borderHighlight' : 'text-neutral-400 hover:text-white'}"
          >
            Locked <span class="font-mono text-2xs opacity-75">({lockedCount})</span>
          </button>
        </div>

        <div class="flex items-center space-x-1.5 bg-surface-header p-1 rounded-xl border border-surface-border">
          <button
            type="button"
            onclick={() => { sortBy = sortBy === 'default' ? 'rarest' : 'default'; }}
            class="px-2.5 py-1 rounded-lg text-xs font-medium transition-all flex items-center space-x-1.5 {sortBy === 'rarest' ? 'bg-surface-cardHover text-white shadow-sm font-bold border border-surface-borderHighlight' : 'text-neutral-400 hover:text-white'}"
            title="Toggle sorting between default and rarest first"
          >
            <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
            </svg>
            <span>{sortBy === 'rarest' ? 'Rarest First' : 'Default Order'}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Achievements Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      {#each filteredAchievements as ach (ach.id)}
        <div class="flex items-start space-x-3.5 p-3 rounded-xl border transition-all {ach.unlocked ? 'bg-emerald-950/15 border-emerald-800/30 hover:border-emerald-700/50' : 'bg-surface-card border-surface-border opacity-70'}">
          <!-- Icon -->
          <div class="w-12 h-12 rounded-lg overflow-hidden bg-black/40 border border-surface-border flex-shrink-0 relative">
            {#if ach.unlocked && ach.icon_url}
              <img src={ach.icon_url} alt={ach.name} class="w-full h-full object-cover" />
            {:else if ach.icon_gray_url || ach.icon_url}
              <img src={ach.icon_gray_url || ach.icon_url} alt={ach.name} class="w-full h-full object-cover filter grayscale contrast-75" />
            {:else}
              <div class="w-full h-full flex items-center justify-center text-neutral-500">
                <svg class="w-4 h-4 opacity-70" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                </svg>
              </div>
            {/if}
            {#if ach.unlocked}
              <div class="absolute top-0.5 right-0.5 w-3.5 h-3.5 bg-emerald-500 rounded-full border border-black flex items-center justify-center text-black shadow">
                <svg class="w-2.5 h-2.5 text-black" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              </div>
            {/if}
          </div>

          <!-- Metadata -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-1.5 min-w-0">
                <h5 class="text-xs font-bold truncate {ach.unlocked ? 'text-emerald-300' : 'text-neutral-200'}">{ach.name || ach.id}</h5>
                {#if typeof ach.global_percent === 'number' && ach.global_percent > 0}
                  <span
                    class="text-4xs font-mono font-bold px-1.5 py-0.5 rounded border tracking-wider uppercase flex-shrink-0 {
                      ach.rarity === 'ultra_rare'
                        ? 'bg-neutral-800 text-amber-200 border-amber-400/40'
                        : ach.rarity === 'rare'
                        ? 'bg-neutral-800 text-neutral-200 border-neutral-600'
                        : 'bg-surface-header text-neutral-400 border-surface-border'
                    }"
                    title="{ach.rarity ? ach.rarity.replace('_', ' ') : 'rarity'}"
                  >
                    {ach.global_percent.toFixed(1)}%
                  </span>
                {/if}
              </div>
              {#if ach.unlocked && ach.unlock_time > 0}
                <span class="text-3xs font-mono text-emerald-400/80 flex-shrink-0">{formatLastPlayed(ach.unlock_time)}</span>
              {/if}
            </div>
            <p class="text-2xs text-neutral-400 mt-0.5 line-clamp-2 leading-relaxed">{ach.description || (ach.hidden ? 'Hidden achievement' : 'No description available')}</p>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
