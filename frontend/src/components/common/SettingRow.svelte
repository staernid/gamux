<script>
  let {
    label = '',
    description = '',
    tooltip = '',
    forId = '',
    badge = '',
    badgeColor = 'cyan',
    inline = true,
    className = '',
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

  let effectiveDescription = $derived(description || tooltip);
</script>

<div class="{inline ? 'flex flex-col sm:flex-row sm:items-center justify-between gap-3' : 'space-y-1.5'} p-2 rounded-lg transition-colors {className}">
  {#if label || effectiveDescription}
    <div class="space-y-1 min-w-0 {inline ? 'flex-1 sm:max-w-md' : 'w-full'}">
      <div class="flex items-center space-x-1.5 flex-wrap">
        {#if label}
          {#if forId}
            <label for={forId} class="block text-xs font-bold text-neutral-200 cursor-pointer">
              {label}
            </label>
          {:else}
            <span class="block text-xs font-bold text-neutral-200">
              {label}
            </span>
          {/if}
        {/if}
        {#if badge}
          <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold border {badgeColors[badgeColor] || badgeColors.neutral}">
            {badge}
          </span>
        {/if}
      </div>
      {#if effectiveDescription}
        <p class="text-xs text-neutral-400 leading-relaxed">
          {effectiveDescription}
        </p>
      {/if}
    </div>
  {/if}

  <div class="{inline ? 'shrink-0 w-full sm:w-auto flex items-center justify-start sm:justify-end' : 'w-full'}">
    {@render children?.()}
  </div>
</div>

