<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tag = '';
  export let value = null; // null (inherit) | true (force on) | false (force off)
  export let inheritedValue = false; // boolean global default
  export let color = 'sky'; // 'sky' | 'emerald' | 'cyan' | 'purple' | 'amber'
  export let disabled = false;

  const dispatch = createEventDispatcher();

  function setValue(newVal) {
    if (disabled) return;
    value = newVal;
    dispatch('change', value);
  }

  const activeColorClasses = {
    sky: 'bg-sky-500/20 border-sky-500/50 text-sky-300 shadow-sm',
    emerald: 'bg-emerald-500/20 border-emerald-500/50 text-emerald-300 shadow-sm',
    cyan: 'bg-cyan-500/20 border-cyan-500/50 text-cyan-300 shadow-sm',
    purple: 'bg-purple-500/20 border-purple-500/50 text-purple-300 shadow-sm',
    amber: 'bg-amber-500/20 border-amber-500/50 text-amber-300 shadow-sm'
  };
</script>

<div
  class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 sm:p-3.5 bg-[#0a0a0f] border border-[#1e1e2c] hover:border-[#2a2a3e] rounded-xl transition-all {disabled ? 'opacity-50 pointer-events-none' : ''}"
  role="group"
  aria-label={label || tag}
>
  <!-- Label & Description -->
  <div class="space-y-0.5 min-w-0 sm:max-w-md">
    <div class="flex items-center space-x-2">
      {#if tag}
        <span class="px-1.5 py-0.2 rounded text-[9px] font-mono font-bold bg-[#161622] border border-[#2c2c3e] text-slate-300">
          {tag}
        </span>
      {/if}
      {#if label}
        <span class="block text-xs font-bold text-slate-200 truncate">
          {label}
        </span>
      {/if}
    </div>
    {#if description}
      <p class="text-[11px] text-slate-400 leading-snug">
        {description}
      </p>
    {/if}
  </div>

  <!-- Segmented 3-Way Control Dock -->
  <div class="flex items-center bg-[#12121a] p-1 rounded-xl border border-[#242436] flex-shrink-0 self-start sm:self-auto">
    <!-- Inherit Pill -->
    <button
      type="button"
      on:click={() => setValue(null)}
      class="px-2.5 py-1 rounded-lg text-[11px] font-mono font-bold transition-all cursor-pointer {value === null ? 'bg-[#222234] text-slate-200 border border-[#383850] shadow-sm' : 'text-slate-500 hover:text-slate-300 border border-transparent'}"
      title="Inherit system-wide setting"
    >
      <span>Inherit</span>
      <span class="text-[9px] ml-1 opacity-75 font-normal">({inheritedValue ? 'On' : 'Off'})</span>
    </button>

    <!-- Force ON Pill -->
    <button
      type="button"
      on:click={() => setValue(true)}
      class="px-2.5 py-1 rounded-lg text-[11px] font-mono font-bold transition-all cursor-pointer {value === true ? (activeColorClasses[color] || activeColorClasses.sky) + ' border' : 'text-slate-500 hover:text-slate-300 border border-transparent'}"
      title="Force enabled for this game"
    >
      On
    </button>

    <!-- Force OFF Pill -->
    <button
      type="button"
      on:click={() => setValue(false)}
      class="px-2.5 py-1 rounded-lg text-[11px] font-mono font-bold transition-all cursor-pointer {value === false ? 'bg-rose-950/40 border border-rose-800/60 text-rose-300 shadow-sm' : 'text-slate-500 hover:text-slate-300 border border-transparent'}"
      title="Force disabled for this game"
    >
      Off
    </button>
  </div>
</div>
