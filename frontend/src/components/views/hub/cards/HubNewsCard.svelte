<script>
  import { formatNewsHTML } from '../../../../utils/format.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [isLoadingNews]
   * @property {string} [newsError]
   * @property {import('../../../../types/domain').NewsItem[]} [newsItems]
   * @property {number} [expandedNewsIndex]
   * @property {(() => void | Promise<void>) | null} [onRefreshNews]
   * @property {((index: number) => void) | null} [onToggleExpand]
   */

  /** @type {Props} */
  let {
    isLoadingNews = false,
    newsError = '',
    newsItems = [],
    expandedNewsIndex = 0,
    onRefreshNews = () => {},
    onToggleExpand = (idx) => {}
  } = $props();
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between">
    <span class="text-neutral-400 font-bold uppercase tracking-wider text-3xs">
      Steam Community Updates & News
    </span>
    <button
      type="button"
      onclick={onRefreshNews}
      disabled={isLoadingNews}
      class="bg-surface-card hover:bg-surface-cardHover text-neutral-200 hover:text-white font-bold text-xs px-3 py-1 rounded-lg border border-surface-border transition-colors disabled:opacity-50 cursor-pointer"
    >
      {isLoadingNews ? 'Loading...' : 'Refresh'}
    </button>
  </div>

  {#if isLoadingNews}
    <div class="py-8 text-center text-neutral-400 font-mono text-xs animate-pulse">Fetching official Steam news...</div>
  {:else if newsError && newsItems.length === 0}
    <div class="p-4 rounded-xl bg-[#181214] border border-rose-900/50 text-center space-y-2">
      <p class="text-xs text-rose-300 font-medium">{newsError}</p>
      <button
        type="button"
        onclick={onRefreshNews}
        class="bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs px-4 py-1.5 rounded-lg transition-colors shadow-sm cursor-pointer"
      >
        Retry Fetch
      </button>
    </div>
  {:else if newsItems.length > 0}
    {#each newsItems as item, idx}
      {@const isExpanded = expandedNewsIndex === idx}
      <div class="bg-surface-card border border-surface-border rounded-xl overflow-hidden shadow-sm transition-colors">
        <div class="p-3 flex items-center justify-between hover:bg-surface-cardHover">
          <button
            type="button"
            class="text-left space-y-0.5 flex-1 pr-3 truncate cursor-pointer select-none"
            onclick={() => onToggleExpand(idx)}
          >
            <div class="flex items-center space-x-2">
              <strong class="text-neutral-100 font-bold text-xs block truncate">{item.title}</strong>
              {#if item.feedlabel || item.feed_label}
                <span class="text-3xs font-mono text-neutral-300 bg-black/40 px-1.5 py-0.5 rounded border border-surface-border">
                  {item.feedlabel || item.feed_label}
                </span>
              {/if}
            </div>
            <span class="text-3xs text-neutral-400 font-mono">
              {item.author ? `${item.author} · ` : ''}{item.date ? new Date(item.date > 1e11 ? item.date : item.date * 1000).toLocaleDateString() : ''}
            </span>
          </button>

          <div class="flex items-center space-x-2 flex-shrink-0">
            {#if item.url}
              <a
                href={item.url}
                target="_blank"
                rel="noopener noreferrer"
                class="p-1 text-neutral-400 hover:text-white transition-colors flex items-center justify-center cursor-pointer"
                title="Open on Steam Web"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              </a>
            {/if}
            <button
              type="button"
              class="text-neutral-400 hover:text-white p-1 transition-colors flex items-center justify-center cursor-pointer"
              onclick={() => onToggleExpand(idx)}
              aria-label={isExpanded ? 'Collapse article' : 'Expand article'}
            >
              <svg class="w-3.5 h-3.5 transform transition-transform {isExpanded ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          </div>
        </div>

        {#if isExpanded}
          <div class="p-4 border-t border-surface-border bg-surface-header text-neutral-200 leading-relaxed text-xs space-y-2 max-h-96 overflow-y-auto custom-scrollbar">
            {@html formatNewsHTML(item.contents)}
          </div>
        {/if}
      </div>
    {/each}
  {:else}
    <div class="py-8 text-center text-neutral-400 font-mono text-xs">No patch notes available for this title.</div>
  {/if}
</div>
