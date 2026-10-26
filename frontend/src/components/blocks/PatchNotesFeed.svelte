<script>
    import { formatDate } from '../../utils/format.js';
  import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../types/domain').NewsItem[]} [newsItems]
   * @property {number} [maxItems]
   * @property {string} [className]
   * @property {((item: import('../../types/domain').NewsItem) => void) | null} [onread]
   * @property {(() => void) | null} [onviewAll]
   */

  /** @type {Props} */
  let {
    newsItems = [],
    maxItems = 3,
    className = '',
    onread = null,
    onviewAll = null
  } = $props();

  let list = $derived(Array.isArray(newsItems) ? newsItems.slice(0, maxItems) : []);

  function cleanExcerpt(raw) {
    if (!raw) return '';
    return raw
      .replace(/<[^>]*>/g, ' ')
      .replace(/\[\/?(b|i|u|url|img|quote|list|code)\]/gi, '')
      .replace(/&quot;/g, '"')
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&#39;/g, "'")
      .replace(/\s+/g, ' ')
      .trim();
  }

  function handleCardClick(item, index) {
    onread?.({ item, index });
  }

  function handleViewAll() {
    onviewAll?.();
  }
</script>

{#if list.length > 0}
  <div class="space-y-3 select-none {className}" data-testid="patch-notes-feed">
    <div class="flex items-center justify-between">
      <div class="flex items-center space-x-2">
        <Icon name="artwork" className="w-4 h-4 text-neutral-400" />
        <span class="text-xs font-bold text-neutral-200">Latest Updates & Notes</span>
      </div>

      <button
        type="button"
        onclick={handleViewAll}
        class="text-3xs font-mono text-neutral-400 hover:text-white flex items-center space-x-1 transition cursor-pointer group"
      >
        <span>View All</span>
        <Icon name="chevron-right" className="w-3 h-3 text-neutral-400 group-hover:text-white transition-colors" />
      </button>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5">
      {#each list as item, idx}
        {@const excerpt = cleanExcerpt(item.contents)}
        <button
          type="button"
          onclick={() => handleCardClick(item, idx)}
          class="bg-[#121214] hover:bg-[#18181b] border border-[#27272a] hover:border-neutral-500 p-3.5 rounded-2xl transition-all shadow-sm flex flex-col justify-between space-y-2.5 text-left group cursor-pointer"
        >
          <div class="space-y-1.5">
            <div class="flex items-center justify-between text-4xs font-mono text-neutral-500">
              <span>{item.date ? formatDate(item.date) : ''}</span>
              {#if item.feedlabel || item.feed_label}
                <span class="px-1.5 py-0.5 rounded bg-[#18181b] text-neutral-400 border border-[#27272a]">
                  {item.feedlabel || item.feed_label}
                </span>
              {/if}
            </div>

            <h5 class="font-bold text-white text-xs group-hover:text-neutral-100 transition line-clamp-2 leading-snug">
              {item.title}
            </h5>

            {#if excerpt}
              <p class="text-neutral-400 text-3xs line-clamp-2 leading-relaxed">
                {excerpt}
              </p>
            {/if}
          </div>

          <div class="pt-2 border-t border-[#27272a] flex items-center justify-between text-3xs font-mono text-neutral-400 group-hover:text-white transition">
            <span>Read Update</span>
            <Icon name="chevron-right" className="w-3 h-3 text-neutral-400 group-hover:text-white group-hover:translate-x-0.5 transition-all" />
          </div>
        </button>
      {/each}
    </div>
  </div>
{/if}
