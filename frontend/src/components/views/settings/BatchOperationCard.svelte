<script>
  /**
   * @typedef {Object} Props
   * @property {string} title
   * @property {string} description
   * @property {string} [actionLabel]
   * @property {string} [countBadge]
   * @property {any} [task]
   * @property {boolean} [disabled]
   * @property {string} [className]
   * @property {import('svelte').Snippet} [icon]
   * @property {() => void} [onrun]
   * @property {() => void} [onviewTasks]
   */

  /** @type {Props} */
  let {
    title,
    description,
    actionLabel = 'Run',
    countBadge = '',
    task = null,
    disabled = false,
    className = '',
    icon = undefined,
    onrun = undefined,
    onviewTasks = undefined,
  } = $props();
</script>

<div class="bg-surface-header border border-surface-border hover:border-surface-borderHighlight p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors {className}">
  <div class="space-y-1">
    <div class="flex items-center justify-between gap-1.5">
      <span class="font-bold text-neutral-100 text-xs flex items-center space-x-1.5 truncate">
        {#if icon}
          {@render icon()}
        {/if}
        <span class="truncate">{title}</span>
      </span>
      {#if countBadge}
        <span class="text-3xs font-mono bg-neutral-800 text-neutral-300 border border-neutral-700 px-1.5 py-0.5 rounded">
          {countBadge}
        </span>
      {/if}
    </div>
    <p class="text-xs text-neutral-400 leading-relaxed">
      {description}
    </p>
    {#if task}
      <div class="mt-2 pt-2 border-t border-surface-border space-y-1.5 animate-fade-in">
        <div class="flex items-center justify-between text-2xs">
          <span class="text-neutral-200 font-semibold truncate flex items-center space-x-1.5">
            <svg class="animate-spin w-3 h-3 text-white shrink-0" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span class="truncate">{task.stage || task.details || 'Processing...'}</span>
          </span>
          <span class="font-mono text-neutral-300 font-bold ml-2 shrink-0">{Math.round(task.percent || 0)}%</span>
        </div>
        <div class="w-full bg-surface-input rounded-full h-1.5 overflow-hidden border border-surface-border">
          <div class="bg-white h-1.5 rounded-full transition-all duration-300" style="width: {Math.max(2, Math.min(100, task.percent || 0))}%"></div>
        </div>
        <div class="flex items-center justify-between text-3xs">
          <span class="truncate text-neutral-500">{task.details || ''}</span>
          <button
            type="button"
            onclick={() => onviewTasks?.()}
            class="text-neutral-300 hover:text-white font-bold flex items-center space-x-0.5 ml-2 shrink-0 cursor-pointer"
          >
            <span>View in Tasks</span>
            <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
          </button>
        </div>
      </div>
    {/if}
  </div>
  <button
    type="button"
    onclick={() => onrun?.()}
    disabled={Boolean(task) || disabled}
    class="w-full bg-surface-card hover:bg-surface-cardHover disabled:opacity-50 text-neutral-200 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-surface-border transition-all cursor-pointer flex items-center justify-center space-x-1.5"
  >
    {#if task}
      <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
      </svg>
      <span>Running ({Math.round(task.percent || 0)}%)...</span>
    {:else}
      <span>{actionLabel}</span>
    {/if}
  </button>
</div>
