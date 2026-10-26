<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import { multiseatStore } from '../../../../stores/multiseat.js';

  let caps = $derived($multiseatStore.capabilities);
  let hasBwrap = $derived(Boolean(caps?.hasBwrap));
  let hasGamescope = $derived(Boolean(caps?.hasGamescope));
  let hasCage = $derived(Boolean(caps?.hasCage));
  let hasSunshine = $derived(Boolean(caps?.hasSunshine));
  let hasPipewire = $derived(Boolean(caps?.hasPipewire));
  let hasInputPermissions = $derived(Boolean(caps?.hasInputPermissions));
</script>

<SettingSection title="System Diagnostics & Isolation Tools">
  <div class="space-y-3">
    <p class="text-xs text-neutral-400">
      Host system tools used for sandboxing, screen composition, audio partitioning, and streaming.
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

      <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
        <div class="space-y-0.5">
          <div class="text-xs font-bold text-neutral-200">Input Devices</div>
          <div class="text-3xs text-neutral-400 font-mono">/dev/input</div>
        </div>
        <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasInputPermissions ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-rose-500/20 text-rose-300 border border-rose-500/40'}">
          {hasInputPermissions ? 'Permitted' : 'Denied'}
        </span>
      </div>
    </div>

    {#if !hasBwrap}
      <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl text-xs text-amber-300 flex items-start space-x-2.5">
        <svg class="w-4 h-4 shrink-0 mt-0.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div class="space-y-1">
          <div class="font-bold text-amber-200">Recommended Package Missing</div>
          <div class="text-2xs text-amber-300/90 leading-relaxed">
            Install <code class="font-mono bg-amber-500/20 px-1 py-0.5 rounded">bubblewrap</code> via your package manager for complete filesystem and input device sandboxing between player seats.
          </div>
        </div>
      </div>
    {/if}

    {#if !hasInputPermissions}
      <div class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-xs text-rose-300 flex items-start space-x-2.5">
        <svg class="w-4 h-4 shrink-0 mt-0.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div class="space-y-1">
          <div class="font-bold text-rose-200">Input Device Permissions Denied</div>
          <div class="text-2xs text-rose-300/90 leading-relaxed">
            The active user cannot read evdev input devices. For dedicated controller and keyboard isolation, add your user to the <code class="font-mono bg-rose-500/20 px-1 py-0.5 rounded">input</code> group:
            <div class="mt-1 font-mono text-3xs bg-black/40 px-2 py-1 rounded text-rose-200">sudo usermod -aG input $USER</div>
          </div>
        </div>
      </div>
    {/if}
  </div>
</SettingSection>
