<script>
  export let title = '';
  export let description = '';
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
</script>

<div class="bg-[#101016] border border-[#222232] rounded-2xl p-4 sm:p-5 space-y-4 shadow-sm {className}">
  {#if title || $$slots.icon || $$slots.action || description}
    <div class="flex items-start justify-between gap-3 border-b border-[#1c1c28] pb-3">
      <div class="space-y-1 min-w-0">
        <div class="flex items-center space-x-2.5">
          {#if $$slots.icon}
            <div class="text-slate-400 flex-shrink-0 flex items-center justify-center">
              <slot name="icon" />
            </div>
          {/if}
          {#if title}
            <h3 class="text-xs sm:text-sm font-bold text-white tracking-wide truncate">
              {title}
            </h3>
          {/if}
          {#if badge}
            <span class="px-2 py-0.5 rounded-md text-[10px] font-mono font-bold border {badgeColors[badgeColor] || badgeColors.cyan}">
              {badge}
            </span>
          {/if}
        </div>
        {#if description}
          <p class="text-[11px] text-slate-400 leading-relaxed max-w-2xl">
            {description}
          </p>
        {/if}
      </div>

      {#if $$slots.action}
        <div class="flex items-center space-x-2 flex-shrink-0">
          <slot name="action" />
        </div>
      {/if}
    </div>
  {/if}

  <div class="space-y-3">
    <slot />
  </div>
</div>
