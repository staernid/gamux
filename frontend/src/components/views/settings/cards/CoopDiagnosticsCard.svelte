<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import { multiseatStore } from '../../../../stores/multiseat.js';

  $: caps = $multiseatStore.capabilities;
  $: hasBwrap = Boolean(caps?.has_bwrap ?? caps?.hasBwrap ?? caps?.HasBwrap);
  $: hasPasta = Boolean(caps?.has_pasta ?? caps?.hasPasta ?? caps?.HasPasta);
  $: hasGamescope = Boolean(caps?.has_gamescope ?? caps?.hasGamescope ?? caps?.HasGamescope);
  $: hasCage = Boolean(caps?.has_cage ?? caps?.hasCage ?? caps?.HasCage);
  $: hasSunshine = Boolean(caps?.has_sunshine ?? caps?.hasSunshine ?? caps?.HasSunshine);
  $: hasPipewire = Boolean(caps?.has_pipewire ?? caps?.hasPipewire ?? caps?.HasPipewire);
</script>

<SettingSection title="System Diagnostics & Isolation Tools">
  <div class="space-y-3">
    <p class="text-xs text-neutral-400">
      Host system tools used for sandboxing, screen composition, audio partitioning, and networking.
    </p>

    <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 pt-1">
      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">Bubblewrap</div>
          <div class="text-3xs text-neutral-400 font-mono">bwrap</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasBwrap ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
          {hasBwrap ? 'Available' : 'Missing'}
        </span>
      </div>

      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">Pasta (NetNS)</div>
          <div class="text-3xs text-neutral-400 font-mono">pasta</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasPasta ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
          {hasPasta ? 'Available' : 'Missing'}
        </span>
      </div>

      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">Gamescope</div>
          <div class="text-3xs text-neutral-400 font-mono">gamescope</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasGamescope ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
          {hasGamescope ? 'Available' : 'Missing'}
        </span>
      </div>

      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">Cage (Wayland)</div>
          <div class="text-3xs text-neutral-400 font-mono">cage</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasCage ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
          {hasCage ? 'Available' : 'Optional'}
        </span>
      </div>

      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">Sunshine</div>
          <div class="text-3xs text-neutral-400 font-mono">sunshine</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasSunshine ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
          {hasSunshine ? 'Available' : 'Optional'}
        </span>
      </div>

      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">PipeWire</div>
          <div class="text-3xs text-neutral-400 font-mono">pactl</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasPipewire ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
          {hasPipewire ? 'Available' : 'Missing'}
        </span>
      </div>
    </div>

    {#if !hasBwrap || !hasPasta}
      <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl text-xs text-amber-300 flex items-start space-x-2.5">
        <svg class="w-4 h-4 flex-shrink-0 mt-0.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div class="space-y-1">
          <div class="font-bold text-amber-200">Recommended Packages Missing</div>
          <div class="text-2xs text-amber-300/90 leading-relaxed">
            Install <code class="font-mono bg-amber-500/20 px-1 py-0.5 rounded">{[!hasBwrap && 'bubblewrap', !hasPasta && 'passt'].filter(Boolean).join(' and ')}</code> via your package manager for complete filesystem sandboxing and rootless network isolation between player seats.
          </div>
        </div>
      </div>
    {/if}
  </div>
</SettingSection>
