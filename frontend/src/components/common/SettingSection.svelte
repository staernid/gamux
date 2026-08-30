<script>
  export let title = '';
  export let description = '';
  export let tooltip = '';
  export let badge = '';
  export let badgeColor = 'cyan'; // 'cyan' | 'emerald' | 'amber' | 'purple' | 'slate'
  export let className = '';

  const badgeColors = {
    cyan: 'bg-cyan-500/15 text-cyan-300 border-cyan-500/30',
    emerald: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
    amber: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
    purple: 'bg-purple-500/15 text-purple-300 border-purple-500/30',
    slate: 'bg-slate-800 text-slate-300 border-slate-700'
  };

  $: effectiveTooltip = tooltip || description;
</script>

<div class="bg-[#0f0f16] border border-[#1e1e2c] rounded-xl overflow-hidden shadow-sm {className}">
  {#if title || $$slots.icon || $$slots.action || description}
    <div class="flex items-center justify-between gap-3 px-4 py-2.5 bg-[#13131c] border-b border-[#1c1c28]">
      <div class="flex items-center space-x-2 min-w-0">
        {#if $$slots.icon}
          <div class="text-slate-400 flex-shrink-0 flex items-center justify-center">
            <slot name="icon" />
          </div>
        {/if}
        {#if title}
          <h3 class="text-xs font-bold text-slate-200 tracking-wide uppercase truncate">
            {title}
          </h3>
        {/if}
        {#if badge}
          <span class="px-1.5 py-0.2 rounded text-[9px] font-mono font-bold border {badgeColors[badgeColor] || badgeColors.cyan}">
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

