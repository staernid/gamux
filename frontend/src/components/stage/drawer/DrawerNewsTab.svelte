<script>
  import { formatNewsHTML } from '../../../utils/format.js';

  export let isLoadingNews = false;
  export let newsError = '';
  export let newsItems = [];
  export let expandedNewsIndex = 0;
  export let onRefreshNews = () => {};
  export let onToggleExpand = (idx) => {};
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between">
    <span class="text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      Steam Community Updates & News
    </span>
    <button
      type="button"
      on:click={onRefreshNews}
      disabled={isLoadingNews}
      class="bg-[#181822] hover:bg-[#202030] text-slate-300 font-bold text-xs px-3 py-1 rounded-lg border border-[#282838] transition-colors disabled:opacity-50"
    >
      {isLoadingNews ? 'Loading...' : 'Refresh'}
    </button>
  </div>

  {#if isLoadingNews}
    <div class="py-8 text-center text-slate-400 font-mono text-xs animate-pulse">Fetching official Steam news...</div>
  {:else if newsError && newsItems.length === 0}
    <div class="p-4 rounded-xl bg-[#1e1014] border border-rose-900/50 text-center space-y-2">
      <p class="text-xs text-rose-300 font-medium">{newsError}</p>
      <button
        type="button"
        on:click={onRefreshNews}
        class="bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs px-4 py-1.5 rounded-lg transition-colors shadow-sm"
      >
        Retry Fetch
      </button>
    </div>
  {:else if newsItems.length > 0}
    {#each newsItems as item, idx}
      {@const isExpanded = expandedNewsIndex === idx}
      <div class="bg-[#0c0c10] border border-[#222230] rounded-xl overflow-hidden shadow-sm transition-colors">
        <div class="p-3 flex items-center justify-between hover:bg-[#14141e]">
          <button
            type="button"
            class="text-left space-y-0.5 flex-1 pr-3 truncate cursor-pointer select-none"
            on:click={() => onToggleExpand(idx)}
          >
            <strong class="text-slate-100 font-bold text-xs block truncate">{item.title}</strong>
            <span class="text-[10px] text-slate-500 font-mono">
              {item.author ? `${item.author} · ` : ''}{new Date(item.date * 1000).toLocaleDateString()}
            </span>
          </button>

          <div class="flex items-center space-x-2 flex-shrink-0">
            {#if item.url}
              <a
                href={item.url}
                target="_blank"
                rel="noopener noreferrer"
                class="p-1 text-slate-400 hover:text-cyan-400 text-xs font-mono"
                title="Open on Steam Web"
              >
                ↗
              </a>
            {/if}
            <button
              type="button"
              class="text-slate-400 font-mono text-xs p-1"
              on:click={() => onToggleExpand(idx)}
            >
              <span class="inline-block transform transition-transform {isExpanded ? 'rotate-180' : ''}">▼</span>
            </button>
          </div>
        </div>

        {#if isExpanded}
          <div class="p-4 border-t border-[#202030] bg-[#101017] text-slate-200 leading-relaxed text-xs space-y-2 max-h-96 overflow-y-auto">
            {@html formatNewsHTML(item.contents)}
          </div>
        {/if}
      </div>
    {/each}
  {:else}
    <div class="py-8 text-center text-slate-500 font-mono text-xs">No patch notes available for this title.</div>
  {/if}
</div>
