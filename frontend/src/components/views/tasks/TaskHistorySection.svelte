<script>
  import { createEventDispatcher } from 'svelte';

  export let tasks = [];

  const dispatch = createEventDispatcher();

  function handleOpenDirectory(dir) {
    dispatch('openDirectory', dir);
  }

  function handleDismiss(id) {
    dispatch('dismiss', id);
  }

  function handleClearAll() {
    dispatch('clearAll');
  }
</script>

<div class="space-y-2.5 pt-2">
  <div class="flex items-center justify-between">
    <h3 class="font-bold text-neutral-400 text-xs tracking-wider uppercase font-mono">
      Completed History ({tasks.length})
    </h3>
    <button
      type="button"
      on:click={handleClearAll}
      class="text-2xs font-bold text-neutral-400 hover:text-white transition cursor-pointer"
    >
      Clear All
    </button>
  </div>

  <div class="bg-surface-card border border-surface-border rounded-2xl divide-y divide-surface-borderSubtle overflow-hidden">
    {#each tasks as tr (tr.id)}
      <div class="p-3.5 flex items-center justify-between hover:bg-surface-cardHover transition gap-3">
        <div class="flex items-center space-x-3 min-w-0">
          <span class="w-2 h-2 rounded-full flex-shrink-0 {tr.status === 'completed' ? 'bg-emerald-400' : 'bg-rose-500'}"></span>
          <div class="min-w-0">
            <span class="font-bold text-white text-xs truncate block">{tr.gameName || tr.title}</span>
            <p class="text-3xs text-neutral-400 font-mono mt-0.5 truncate">{tr.details || tr.status}</p>
          </div>
        </div>

        <div class="flex items-center space-x-2 flex-shrink-0">
          {#if tr.gameDir}
            <button
              type="button"
              on:click={() => handleOpenDirectory(tr.gameDir)}
              class="text-neutral-300 hover:text-white text-xs px-2.5 py-1 rounded-lg bg-surface-header border border-surface-border font-medium cursor-pointer"
            >
              Folder
            </button>
          {/if}
          <button
            type="button"
            on:click={() => handleDismiss(tr.id)}
            class="text-neutral-400 hover:text-white text-xs px-2 py-1 cursor-pointer"
          >
            Dismiss
          </button>
        </div>
      </div>
    {/each}
  </div>
</div>
