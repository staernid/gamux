<script>
  import { createEventDispatcher } from 'svelte';
  import { isDownloadTask } from '../../../stores/tasks.js';

  export let tasks = [];

  const dispatch = createEventDispatcher();

  function handleResume(id) {
    dispatch('resume', id);
  }

  function handlePause(id) {
    dispatch('pause', id);
  }

  function handleCancel(id) {
    dispatch('cancel', id);
  }
</script>

<div class="space-y-2.5 pt-2">
  <h3 class="font-bold text-neutral-400 text-xs tracking-wider uppercase font-mono">
    Queue & Paused ({tasks.length})
  </h3>

  <div class="bg-surface-card border border-surface-border rounded-2xl divide-y divide-surface-borderSubtle overflow-hidden">
    {#each tasks as tr (tr.id)}
      <div class="p-3.5 flex items-center justify-between hover:bg-surface-cardHover transition">
        <div class="flex items-center space-x-3 min-w-0">
          <span class="px-1.5 py-0.5 text-4xs font-bold rounded bg-surface-header text-neutral-300 uppercase font-mono border border-surface-border">
            {tr.status}
          </span>
          <div class="min-w-0">
            <span class="font-bold text-white text-xs truncate block">{tr.gameName || tr.title}</span>
            <p class="text-3xs text-neutral-400 font-mono mt-0.5 truncate">{tr.details || tr.stage || 'Waiting'}</p>
          </div>
        </div>

        <div class="flex items-center space-x-2 flex-shrink-0">
          {#if tr.status === 'paused'}
            <button
              type="button"
              on:click={() => handleResume(tr.id)}
              class="bg-surface-header hover:bg-surface-cardHover text-neutral-100 border border-surface-border text-xs px-3 py-1 rounded-xl font-bold transition cursor-pointer"
            >
              Resume
            </button>
          {:else if isDownloadTask(tr)}
            <button
              type="button"
              on:click={() => handlePause(tr.id)}
              class="bg-surface-header hover:bg-surface-cardHover text-neutral-200 border border-surface-border text-xs px-3 py-1 rounded-xl font-bold transition cursor-pointer"
            >
              Pause
            </button>
          {/if}
          <button
            type="button"
            on:click={() => handleCancel(tr.id)}
            class="text-neutral-400 hover:text-rose-400 text-xs px-2 py-1 font-bold transition cursor-pointer"
          >
            Cancel
          </button>
        </div>
      </div>
    {/each}
  </div>
</div>
