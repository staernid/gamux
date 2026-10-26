<script>
  import { formatBytes } from '../../../utils/format.js';
  import PosterArt from '../../common/PosterArt.svelte';

  let { tasks = [] ,
    onopenGame,
    onopenDirectory,
    ondismiss,
    onclearAll
  } = $props();

  function handleOpenGame(task) {
    onopenGame?.(task);
  }

  function handleOpenDirectory(dir) {
    onopenDirectory?.(dir);
  }

  function handleDismiss(id) {
    ondismiss?.(id);
  }

  function handleClearAll() {
    onclearAll?.();
  }
</script>

<div class="space-y-2.5 pt-2">
  <div class="flex items-center justify-between">
    <h3 class="font-bold text-neutral-400 text-xs tracking-wider uppercase font-mono">
      Completed History ({tasks.length})
    </h3>
    <button
      type="button"
      onclick={handleClearAll}
      class="text-2xs font-bold text-neutral-400 hover:text-white transition cursor-pointer"
    >
      Clear All
    </button>
  </div>

  <div class="bg-surface-card border border-surface-border rounded-2xl divide-y divide-surface-borderSubtle overflow-hidden">
    {#each tasks as tr (tr.id)}
      <div
        role="button"
        tabindex="0"
        onclick={() => handleOpenGame(tr)}
        onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && handleOpenGame(tr)}
        class="p-4 flex items-center justify-between hover:bg-surface-cardHover hover:border-surface-borderHighlight transition gap-4 cursor-pointer group"
        title="Open {tr.gameName || tr.title} in Game Hub"
      >
        <div class="flex items-center space-x-3.5 min-w-0 flex-1">
          <!-- Game Cover Artwork: True 2:3 uncropped ratio -->
          <div class="w-16 sm:w-20 aspect-2/3 rounded-xl overflow-hidden bg-surface-base border border-surface-border group-hover:border-surface-borderHighlight shrink-0 shadow-md transition">
            <PosterArt
              appId={tr.appId}
              name={tr.gameName}
              posterUrl={tr.posterUrl}
              className="w-full h-full object-cover transition group-hover:scale-105 duration-300"
            />
          </div>

          <div class="min-w-0 space-y-1">
            <div class="flex items-center space-x-2">
              <span class="w-2 h-2 rounded-full shrink-0 {tr.status === 'completed' ? 'bg-emerald-400' : 'bg-rose-500'}"></span>
              <span class="font-bold text-white text-sm sm:text-base truncate group-hover:text-white transition" title={tr.gameName || tr.title}>
                {tr.gameName || tr.title}
              </span>
            </div>
            <div class="flex items-center space-x-2 text-2xs text-neutral-400 font-mono truncate">
              {#if tr.totalBytes && tr.totalBytes > 0}
                <span>{formatBytes(tr.totalBytes)}</span>
                <span>•</span>
              {/if}
              <span class="truncate">{tr.details || (tr.status === 'completed' ? 'Ready to play' : 'Failed')}</span>
            </div>
            <span class="text-3xs text-neutral-500 font-mono flex items-center space-x-1 group-hover:text-neutral-300 transition">
              <span>View in Game Hub</span>
              <svg class="w-2.5 h-2.5 transform transition-transform group-hover:translate-x-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </span>
          </div>
        </div>

        <div class="flex items-center space-x-2 shrink-0">
          {#if tr.gameDir}
            <button
              type="button"
              onclick={(e) => { e.stopPropagation(); handleOpenDirectory(tr.gameDir); }}
              class="text-neutral-300 hover:text-white text-xs px-2.5 py-1.5 rounded-xl bg-surface-header hover:bg-surface-input border border-surface-border font-medium transition cursor-pointer"
              title="Open Installation Folder"
            >
              Folder
            </button>
          {/if}

          <button
            type="button"
            onclick={(e) => { e.stopPropagation(); handleDismiss(tr.id); }}
            class="text-neutral-400 hover:text-white text-xs px-2.5 py-1.5 rounded-xl hover:bg-surface-input cursor-pointer transition"
            title="Dismiss from history"
          >
            Dismiss
          </button>
        </div>
      </div>
    {/each}
  </div>
</div>
