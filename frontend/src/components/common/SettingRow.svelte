<script>
  export let label = '';
  export let description = '';
  export let tooltip = '';
  export let forId = '';
  export let badge = '';
  export let badgeColor = 'cyan';
  export let inline = true;
  export let className = '';

  const badgeColors = {
    cyan: 'bg-cyan-500/15 text-cyan-300 border-cyan-500/30',
    emerald: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
    amber: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
    purple: 'bg-purple-500/15 text-purple-300 border-purple-500/30',
    slate: 'bg-slate-800 text-slate-300 border-slate-700'
  };

  $: effectiveDescription = description || tooltip;
</script>

<div class="{inline ? 'flex flex-col sm:flex-row sm:items-center justify-between gap-3' : 'space-y-1.5'} p-2 rounded-lg transition-colors {className}">
  {#if label || effectiveDescription}
    <div class="space-y-0.5 min-w-0 {inline ? 'flex-1 sm:max-w-md' : 'w-full'}">
      <div class="flex items-center space-x-1.5 flex-wrap">
        {#if label}
          {#if forId}
            <label for={forId} class="block text-xs font-bold text-slate-200 cursor-pointer truncate">
              {label}
            </label>
          {:else}
            <span class="block text-xs font-bold text-slate-200 truncate">
              {label}
            </span>
          {/if}
        {/if}
        {#if badge}
          <span class="px-1.5 py-0.2 rounded text-[9px] font-mono font-bold border {badgeColors[badgeColor] || badgeColors.cyan}">
            {badge}
          </span>
        {/if}
      </div>
      {#if effectiveDescription}
        <p class="text-[11px] text-slate-400 leading-snug">
          {effectiveDescription}
        </p>
      {/if}
    </div>
  {/if}

  <div class="{inline ? 'flex-shrink-0 w-full sm:w-auto flex items-center justify-start sm:justify-end' : 'w-full'}">
    <slot />
  </div>
</div>

