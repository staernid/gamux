<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tag = '';
  export let triState = false;
  export let value = false; // boolean or null (tri-state)
  export let inheritedValue = false; // for tri-state tooltips
  export let color = 'sky'; // 'sky' | 'emerald' | 'cyan' | 'purple' | 'amber' | 'rose' | 'default'
  export let invertLabel = false; // e.g. for disable_overlay where true = OFF
  export let disabled = false;

  const dispatch = createEventDispatcher();

  // Color theme maps
  const activeColorClasses = {
    sky: 'bg-sky-950/70 border-sky-500/70 text-sky-300 shadow-sm',
    emerald: 'bg-emerald-950/70 border-emerald-500/70 text-emerald-300 shadow-sm',
    cyan: 'bg-cyan-950/70 border-cyan-500/70 text-cyan-300 shadow-sm',
    purple: 'bg-purple-950/70 border-purple-500/70 text-purple-300 shadow-sm',
    amber: 'bg-amber-950/70 border-amber-500/70 text-amber-300 shadow-sm',
    rose: 'bg-rose-950/70 border-rose-500/70 text-rose-300 shadow-sm',
    default: 'bg-sky-950/70 border-sky-500/70 text-sky-300 shadow-sm'
  };

  const offColorClass = 'bg-rose-950/50 border-rose-800/60 text-rose-300';
  const inactiveColorClass = 'bg-[#181824] border-[#2c2c3e] text-slate-500';
  const autoColorClass = 'bg-[#181824] border-[#2c2c3e] text-slate-400';

  $: isOn = triState ? value === true : (invertLabel ? !value : !!value);
  $: isOff = triState ? value === false : (invertLabel ? !value : !value);
  $: isAuto = triState && value === null;

  $: statusText = isAuto ? 'AUTO' : (isOn ? 'ON' : 'OFF');

  $: buttonClass = isAuto
    ? autoColorClass
    : isOn
    ? (activeColorClasses[color] || activeColorClasses.default)
    : (triState ? offColorClass : inactiveColorClass);

  $: tooltipText = (() => {
    if (triState) {
      if (value === true) return `${label || tag}: Forced ON`;
      if (value === false) return `${label || tag}: Forced OFF`;
      return `${label || tag}: Inherited (${inheritedValue ? 'On by default' : 'Off by default'})`;
    }
    return `${label || tag}: ${isOn ? 'Enabled' : 'Disabled'}`;
  })();

  function toggle() {
    if (disabled) return;
    if (triState) {
      if (value === true) value = false;
      else if (value === false) value = null;
      else value = true;
    } else {
      value = !value;
    }
    dispatch('change', value);
  }

  function handleKeydown(e) {
    if (disabled) return;
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      toggle();
    }
  }
</script>

<div
  class="flex items-center justify-between p-3 bg-[#0d0d12] hover:bg-[#13131a] focus-within:bg-[#161622] border border-[#222230] focus-within:border-sky-500/70 rounded-xl transition-all select-none group {disabled ? 'opacity-50 pointer-events-none' : ''}"
  role="group"
  aria-label={label || tag}
>
  <!-- Label & Description -->
  {#if label || description}
    <div class="space-y-0.5 pr-3 flex-1 min-w-0">
      {#if label}
        <span class="block font-bold text-slate-200 text-xs group-hover:text-white transition-colors truncate">
          {label}
        </span>
      {/if}
      {#if description}
        <p class="text-[11px] text-slate-400 leading-tight line-clamp-2">
          {description}
        </p>
      {/if}
    </div>
  {/if}

  <!-- Switcher Button Pill -->
  <button
    type="button"
    on:click={toggle}
    on:keydown={handleKeydown}
    title={tooltipText}
    class="py-1.5 px-3 rounded-xl border text-[11px] font-bold font-mono transition-all flex items-center space-x-1.5 cursor-pointer flex-shrink-0 outline-none focus:ring-2 focus:ring-sky-500/40 {buttonClass}"
    aria-pressed={isOn}
  >
    {#if tag}
      <span>{tag}</span>
    {/if}
    <span class="text-[9px] uppercase px-1.5 py-0.5 rounded bg-black/30 tracking-wider">
      {statusText}
    </span>
  </button>
</div>
