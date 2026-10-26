<script>
  /**
   * @typedef {Object} Props
   * @property {string} [title]
   * @property {string} [description]
   * @property {string} [tooltip]
   * @property {string} [badge]
   * @property {string} [badgeColor]
   * @property {string} [className]
   * @property {import('svelte').Snippet} [icon]
   * @property {import('svelte').Snippet} [action]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    title = '',
    description = '',
    tooltip = '',
    badge = '',
    badgeColor = 'cyan',
    className = '',
    icon,
    action,
    children
  } = $props();

  const badgeColors = {
    neutral: 'bg-neutral-800 text-neutral-300 border-neutral-700',
    emerald: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
    amber: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
    rose: 'bg-rose-500/15 text-rose-300 border-rose-500/30',
    cyan: 'bg-neutral-800 text-neutral-300 border-neutral-700',
    purple: 'bg-neutral-800 text-neutral-300 border-neutral-700',
    slate: 'bg-neutral-800 text-neutral-300 border-neutral-700'
  };

  let effectiveTooltip = $derived(tooltip || description);
</script>

<div class="bg-surface-header border border-surface-border rounded-xl overflow-hidden shadow-sm {className}">
  {#if title || icon || action || description}
    <div class="flex items-center justify-between gap-3 px-4 py-2.5 bg-surface-card border-b border-surface-border">
      <div class="flex items-center space-x-2 min-w-0">
        {#if icon}
          <div class="text-neutral-400 shrink-0 flex items-center justify-center">
            {@render icon?.()}
          </div>
        {/if}
        {#if title}
          <h3 class="text-xs font-bold text-neutral-200 tracking-wide uppercase truncate">
            {title}
          </h3>
        {/if}
        {#if badge}
          <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold border {badgeColors[badgeColor] || badgeColors.neutral}">
            {badge}
          </span>
        {/if}
      </div>

      {#if action}
        <div class="flex items-center space-x-2 shrink-0">
          {@render action?.()}
        </div>
      {/if}
    </div>
  {/if}

  <div class="p-3 sm:p-4">
    {@render children?.()}
  </div>
</div>

