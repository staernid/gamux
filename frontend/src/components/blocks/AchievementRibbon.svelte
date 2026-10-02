<script>
    import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [achievements]
   * @property {number} [maxBadges]
   * @property {string} [className]
   * @property {any} [onselect]
   * @property {any} [onviewAll]
   */

  /** @type {Props} */
  let {
    achievements = [],
    maxBadges = 5,
    className = '',
    onselect = null,
    onviewAll = null
  } = $props();

  let list = $derived(Array.isArray(achievements) ? achievements : []);
  let total = $derived(list.length);
  let unlocked = $derived(list.filter(a => a.unlocked).length);
  let percent = $derived(total > 0 ? Math.round((unlocked / total) * 100) : 0);

  let recentUnlocked = $derived(list
    .filter(a => a.unlocked)
    .sort((a, b) => (b.unlock_time || 0) - (a.unlock_time || 0)));

  let spotlightBadges = $derived((recentUnlocked.length > 0 ? recentUnlocked : list).slice(0, maxBadges));

  function handleBadgeClick(ach) {
    onselect?.({ achievement: ach });
  }

  function handleViewAll() {
    onviewAll?.();
  }
</script>

{#if total > 0}
  <div class="space-y-2.5 select-none {className}" data-testid="achievement-ribbon">
    <!-- Header Row -->
    <div class="flex items-center justify-between">
      <div class="flex items-center space-x-2">
        <Icon name="trophy" className="w-4 h-4 text-neutral-400" />
        <span class="text-xs font-bold text-neutral-200">Achievements</span>
        <span class="text-3xs font-mono font-bold text-neutral-400 bg-[#18181b] px-2 py-0.5 rounded-md border border-[#27272a]">
          {unlocked} / {total} ({percent}%)
        </span>
      </div>

      <button
        type="button"
        onclick={handleViewAll}
        class="text-3xs font-mono text-neutral-400 hover:text-white flex items-center space-x-1 transition cursor-pointer group"
      >
        <span>View All ({total})</span>
        <Icon name="chevron-right" className="w-3 h-3 text-neutral-400 group-hover:text-white transition-colors" />
      </button>
    </div>

    <!-- Progress Bar -->
    <div class="w-full bg-[#18181b] border border-[#27272a] rounded-full h-1.5 overflow-hidden">
      <div
        class="h-full bg-neutral-200 rounded-full transition-all duration-300"
        style="width: {percent}%"
      ></div>
    </div>

    <!-- Badge Strip -->
    {#if spotlightBadges.length > 0}
      <div class="flex items-center gap-2 overflow-x-auto custom-scrollbar pt-1">
        {#each spotlightBadges as ach (ach.id || ach.name)}
          <button
            type="button"
            onclick={() => handleBadgeClick(ach)}
            title="{ach.name}: {ach.description || 'Achievement'}"
            class="group flex items-center space-x-2 p-1.5 rounded-xl bg-[#121214] hover:bg-[#1c1c20] border border-[#27272a] hover:border-neutral-500 transition cursor-pointer flex-shrink-0"
          >
            <div class="w-8 h-8 rounded-lg overflow-hidden bg-black/40 border border-[#27272a] relative flex-shrink-0">
              {#if ach.unlocked && (ach.icon_url || ach.icon)}
                <img src={ach.icon_url || ach.icon} alt={ach.name} class="w-full h-full object-cover" />
              {:else if ach.icon_gray_url || ach.icon_url || ach.icon}
                <img src={ach.icon_gray_url || ach.icon_url || ach.icon} alt={ach.name} class="w-full h-full object-cover filter grayscale contrast-75" />
              {:else}
                <div class="w-full h-full flex items-center justify-center text-neutral-600">
                  <Icon name="trophy" className="w-3.5 h-3.5 text-neutral-600" />
                </div>
              {/if}

              {#if ach.unlocked}
                <div class="absolute top-0.5 right-0.5 w-2.5 h-2.5 bg-white rounded-full flex items-center justify-center">
                  <span class="w-1.5 h-1.5 rounded-full bg-neutral-950"></span>
                </div>
              {/if}
            </div>

            <div class="text-left pr-1.5 max-w-[110px]">
              <p class="text-3xs font-bold truncate text-neutral-300 group-hover:text-white transition">
                {ach.name || ach.id}
              </p>
              <p class="text-4xs font-mono text-neutral-500 truncate">
                {ach.unlocked ? 'Unlocked' : 'Locked'}
              </p>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
{/if}
