<script>
  import { createEventDispatcher } from 'svelte';
  import { formatBytes } from '../../utils/format.js';
  import Icon from '../common/Icon.svelte';

  export let activeTasks = [];
  export let totalSpeedBps = 0;
  export let isExpanded = false;
  export let className = '';
  export let ontoggle = null;

  const dispatch = createEventDispatcher();

  $: list = Array.isArray(activeTasks) ? activeTasks : [];
  $: currentTask = list[0] || null;
  $: count = list.length;

  function handleToggle() {
    dispatch('toggle');
    if (typeof ontoggle === 'function') ontoggle();
  }
</script>

{#if count > 0 && currentTask}
  <div
    class="w-full h-9 bg-[#0e0e11]/95 border-t border-[#27272a] px-4 flex items-center justify-between text-xs font-mono select-none backdrop-blur-md transition-all {className}"
    data-testid="task-status-bar"
  >
    <!-- Left: Activity Ring & Active Task Description -->
    <div class="flex items-center space-x-3 min-w-0 flex-1 pr-4">
      <Icon name="spinner" className="w-3.5 h-3.5 text-neutral-300 flex-shrink-0" />

      <span class="text-neutral-400 font-bold uppercase tracking-wider text-4xs">
        {currentTask.operation || currentTask.type || 'TASK'}:
      </span>

      <span class="text-neutral-200 font-medium truncate max-w-sm" title={currentTask.gameName || currentTask.title || ''}>
        {currentTask.gameName || currentTask.title || 'Processing pipeline...'}
      </span>

      {#if currentTask.progress !== undefined && currentTask.progress > 0}
        <span class="text-neutral-400 text-3xs">
          ({Math.round(currentTask.progress)}%)
        </span>
      {/if}
    </div>

    <!-- Center/Right: Speed & Expand Toggle -->
    <div class="flex items-center space-x-4 flex-shrink-0">
      {#if totalSpeedBps > 0}
        <span class="text-3xs text-neutral-300 font-bold">
          {formatBytes(totalSpeedBps)}/s
        </span>
      {/if}

      {#if count > 1}
        <span class="text-4xs text-neutral-500 bg-[#18181b] px-2 py-0.5 rounded-md border border-[#27272a]">
          +{count - 1} queued
        </span>
      {/if}

      <button
        type="button"
        on:click={handleToggle}
        class="text-3xs text-neutral-400 hover:text-white px-2 py-1 rounded-lg hover:bg-neutral-800 transition cursor-pointer flex items-center space-x-1.5 group"
      >
        <span>{isExpanded ? 'Hide Queue' : 'View Queue'}</span>
        <Icon name="chevron-up" className="w-3.5 h-3.5 text-neutral-400 group-hover:text-white transform transition-transform duration-200 {isExpanded ? 'rotate-180' : ''}" />
      </button>
    </div>
  </div>
{/if}
