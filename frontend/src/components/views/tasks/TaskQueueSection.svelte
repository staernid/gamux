<script>
    import { isDownloadTask } from '../../../stores/tasks.js';
  import { formatBytes } from '../../../utils/format.js';
  import PosterArt from '../../common/PosterArt.svelte';

  let { tasks = [] ,
    onresume,
    onpause,
    oncancel,
    onreorder
  } = $props();

  function handleResume(id) {
    onresume?.(id);
  }

  function handlePause(id) {
    onpause?.(id);
  }

  function handleCancel(id) {
    oncancel?.(id);
  }

  function moveUp(index) {
    if (index <= 0) return;
    const ids = tasks.map((t) => t.id);
    const temp = ids[index];
    ids[index] = ids[index - 1];
    ids[index - 1] = temp;
    onreorder?.(ids);
  }

  function moveDown(index) {
    if (index >= tasks.length - 1) return;
    const ids = tasks.map((t) => t.id);
    const temp = ids[index];
    ids[index] = ids[index + 1];
    ids[index + 1] = temp;
    onreorder?.(ids);
  }

  function prioritize(index) {
    if (index <= 0) return;
    const ids = tasks.map((t) => t.id);
    const [item] = ids.splice(index, 1);
    ids.unshift(item);
    onreorder?.(ids);
  }
</script>

<div class="space-y-2.5 pt-2">
  <div class="flex items-center justify-between">
    <h3 class="font-bold text-neutral-400 text-xs tracking-wider uppercase font-mono">
      Queue & Paused ({tasks.length})
    </h3>
  </div>

  <div class="bg-surface-card border border-surface-border rounded-2xl divide-y divide-surface-borderSubtle overflow-hidden">
    {#each tasks as tr, index (tr.id)}
      <div class="p-3.5 flex items-center justify-between hover:bg-surface-cardHover transition gap-3">
        <div class="flex items-center space-x-3 min-w-0">
          <!-- Mini Poster Art: True 2:3 uncropped ratio -->
          <div class="w-14 sm:w-16 aspect-2/3 rounded-xl overflow-hidden bg-surface-base border border-surface-border shrink-0 shadow-md">
            <PosterArt
              appId={tr.appId}
              name={tr.gameName}
              posterUrl={tr.posterUrl}
              className="w-full h-full object-cover"
            />
          </div>

          <div class="min-w-0 space-y-0.5">
            <div class="flex items-center space-x-2">
              <span class="px-1.5 py-0.5 text-4xs font-bold rounded bg-surface-header text-neutral-300 uppercase font-mono border border-surface-border shrink-0">
                {tr.status}
              </span>
              <span class="font-bold text-white text-xs truncate" title={tr.gameName || tr.title}>
                {tr.gameName || tr.title}
              </span>
            </div>
            <div class="flex items-center space-x-2 text-3xs text-neutral-400 font-mono truncate">
              {#if tr.totalBytes && tr.totalBytes > 0}
                <span>{formatBytes(tr.currentBytes || 0)} / {formatBytes(tr.totalBytes)} ({tr.percent?.toFixed(0) || 0}%)</span>
                <span>•</span>
              {/if}
              <span class="truncate">{tr.details || tr.stage || 'Waiting in queue'}</span>
            </div>
          </div>
        </div>

        <div class="flex items-center space-x-1.5 shrink-0">
          <!-- Reorder buttons if multiple in queue -->
          {#if tasks.length > 1}
            <div class="flex items-center space-x-0.5 mr-1 bg-surface-input p-0.5 rounded-lg border border-surface-border">
              <button
                type="button"
                onclick={() => moveUp(index)}
                disabled={index === 0}
                class="p-1 rounded text-neutral-400 hover:text-white disabled:opacity-25 disabled:cursor-not-allowed transition cursor-pointer"
                title="Move Up"
                aria-label="Move Up"
              >
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                </svg>
              </button>
              <button
                type="button"
                onclick={() => moveDown(index)}
                disabled={index === tasks.length - 1}
                class="p-1 rounded text-neutral-400 hover:text-white disabled:opacity-25 disabled:cursor-not-allowed transition cursor-pointer"
                title="Move Down"
                aria-label="Move Down"
              >
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>
              {#if index > 0}
                <button
                  type="button"
                  onclick={() => prioritize(index)}
                  class="px-1.5 py-0.5 rounded text-4xs font-mono font-bold text-neutral-300 hover:text-white hover:bg-surface-card transition cursor-pointer"
                  title="Download Next"
                  aria-label="Download Next"
                >
                  Top
                </button>
              {/if}
            </div>
          {/if}

          {#if tr.status === 'paused'}
            <button
              type="button"
              onclick={() => handleResume(tr.id)}
              class="bg-surface-header hover:bg-surface-cardHover text-neutral-100 border border-surface-border text-xs px-2.5 py-1 rounded-xl font-bold transition cursor-pointer"
            >
              Resume
            </button>
          {:else if isDownloadTask(tr)}
            <button
              type="button"
              onclick={() => handlePause(tr.id)}
              class="bg-surface-header hover:bg-surface-cardHover text-neutral-200 border border-surface-border text-xs px-2.5 py-1 rounded-xl font-bold transition cursor-pointer"
            >
              Pause
            </button>
          {/if}
          <button
            type="button"
            onclick={() => handleCancel(tr.id)}
            class="text-neutral-400 hover:text-rose-400 text-xs px-2 py-1 font-bold transition cursor-pointer"
          >
            Cancel
          </button>
        </div>
      </div>
    {/each}
  </div>
</div>
