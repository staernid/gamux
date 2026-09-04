<script>
  export let title = '';
  export let description = '';
  export let tooltip = '';
  export let badge = '';
  export let badgeColor = 'cyan'; // 'cyan' | 'emerald' | 'amber' | 'purple' | 'slate'
  export let className = '';

  const badgeColors = {
    neutral: 'bg-neutral-800 text-neutral-300 border-neutral-700',
    emerald: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
    amber: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
    rose: 'bg-rose-500/15 text-rose-300 border-rose-500/30',
    cyan: 'bg-neutral-800 text-neutral-300 border-neutral-700',
    purple: 'bg-neutral-800 text-neutral-300 border-neutral-700',
    slate: 'bg-neutral-800 text-neutral-300 border-neutral-700'
  };

  $: effectiveTooltip = tooltip || description;
</script>

<div class="bg-surface-header border border-surface-border rounded-xl overflow-hidden shadow-sm {className}">
  {#if title || $$slots.icon || $$slots.action || description}
    <div class="flex items-center justify-between gap-3 px-4 py-2.5 bg-surface-card border-b border-surface-border">
      <div class="flex items-center space-x-2 min-w-0">
        {#if $$slots.icon}
          <div class="text-neutral-400 flex-shrink-0 flex items-center justify-center">
            <slot name="icon" />
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

      {#if $$slots.action}
        <div class="flex items-center space-x-2 flex-shrink-0">
          <slot name="action" />
        </div>
      {/if}
    </div>
  {/if}

  <div class="p-3 sm:p-4">
    <slot />
  </div>
</div>

