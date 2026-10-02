<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../../common/SeamlessSwitch.svelte';
  import { multiseatStore } from '../../../../stores/multiseat.js';

  export let cfgDraft;
  export let keyboardOptions = [];
  export let mouseOptions = [];
</script>

<!-- 1. Dual Keyboards & Mice (Games Without Gamepad Support) -->
<SettingSection title="Dual Keyboards & Mice (Experimental)">
  <div class="space-y-4">
    <div class="flex items-center justify-between pb-2 border-b border-surface-border/50">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Enables local co-op for games lacking controller support. Player 2's devices are grabbed exclusively via evdev (<code class="font-mono text-neutral-300">EVIOCGRAB</code>) and injected directly into Gamescope via libei (<code class="font-mono text-neutral-300">gamescope-*-ei</code>). Gamescope with EIS redirection is auto-selected when dedicated KBM is configured.
      </p>
      <button
        type="button"
        on:click={() => { multiseatStore.refreshKeyboards(); multiseatStore.refreshMice(); }}
        class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
        title="Rescan connected keyboards and mice"
      >
        <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>Rescan KBM</span>
      </button>
    </div>

    <!-- Keyboard & Mouse Allocation Selectors -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-surface-card border border-surface-border rounded-xl">
      <div class="space-y-1.5">
        <label for="seat1-kbd-select" class="block text-2xs font-bold text-neutral-300">Player 1 Keyboard</label>
        <select
          id="seat1-kbd-select"
          bind:value={cfgDraft.multiseat.seat1_keyboard}
          on:change={() => { cfgDraft = { ...cfgDraft }; }}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
        >
          {#each keyboardOptions as opt}
            <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
          {/each}
        </select>
      </div>

      <div class="space-y-1.5">
        <label for="seat2-kbd-select" class="block text-2xs font-bold text-neutral-300">Player 2 Keyboard</label>
        <select
          id="seat2-kbd-select"
          bind:value={cfgDraft.multiseat.seat2_keyboard}
          on:change={() => { cfgDraft = { ...cfgDraft }; }}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
        >
          {#each keyboardOptions as opt}
            <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
          {/each}
        </select>
      </div>

      <div class="space-y-1.5 pt-1">
        <label for="seat1-mouse-select" class="block text-2xs font-bold text-neutral-300">Player 1 Mouse</label>
        <select
          id="seat1-mouse-select"
          bind:value={cfgDraft.multiseat.seat1_mouse}
          on:change={() => { cfgDraft = { ...cfgDraft }; }}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
        >
          {#each mouseOptions as opt}
            <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
          {/each}
        </select>
      </div>

      <div class="space-y-1.5 pt-1">
        <label for="seat2-mouse-select" class="block text-2xs font-bold text-neutral-300">Player 2 Mouse</label>
        <select
          id="seat2-mouse-select"
          bind:value={cfgDraft.multiseat.seat2_mouse}
          on:change={() => { cfgDraft = { ...cfgDraft }; }}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
        >
          {#each mouseOptions as opt}
            <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
          {/each}
        </select>
      </div>
    </div>

    <!-- Player 2 Cursor & Gamescope Options -->
    <div class="p-3 bg-surface-card border border-surface-border rounded-xl space-y-3">
      <div class="text-xs font-bold text-neutral-200">Player 2 Pointer &amp; Compositor Options</div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
          <div class="space-y-0.5 max-w-[200px]">
            <div class="text-2xs font-bold text-neutral-200">Force Grab Cursor</div>
            <div class="text-3xs text-neutral-400">Relative mouse mode for 3D/FPS camera control (--force-grab-cursor)</div>
          </div>
          <SeamlessSwitch
            bind:value={cfgDraft.multiseat.seat2_force_grab_cursor}
            tooltip="Force Grab Cursor"
            on:change={(e) => { cfgDraft.multiseat.seat2_force_grab_cursor = e.detail; cfgDraft = { ...cfgDraft }; }}
          />
        </div>

        <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
          <div class="space-y-0.5 max-w-[200px]">
            <div class="text-2xs font-bold text-neutral-200">Grab Keyboard (-g)</div>
            <div class="text-3xs text-neutral-400">Captures all keypresses inside the window to prevent desktop shortcuts</div>
          </div>
          <SeamlessSwitch
            bind:value={cfgDraft.multiseat.seat2_grab_keyboard}
            tooltip="Grab Keyboard"
            on:change={(e) => { cfgDraft.multiseat.seat2_grab_keyboard = e.detail; cfgDraft = { ...cfgDraft }; }}
          />
        </div>

        <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
          <div class="space-y-0.5 max-w-[200px]">
            <div class="text-2xs font-bold text-neutral-200">Software Cursor</div>
            <div class="text-3xs text-neutral-400">Enforces visible pointer via WLR_NO_HARDWARE_CURSORS=1</div>
          </div>
          <SeamlessSwitch
            bind:value={cfgDraft.multiseat.seat2_show_cursor}
            tooltip="Software Cursor"
            on:change={(e) => { cfgDraft.multiseat.seat2_show_cursor = e.detail; cfgDraft = { ...cfgDraft }; }}
          />
        </div>

        <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
          <div class="space-y-0.5 max-w-[170px]">
            <div class="text-2xs font-bold text-neutral-200">Compositor Backend</div>
            <div class="text-3xs text-neutral-400 leading-relaxed">Gamescope EIS auto-selected for dual KBM virtual input</div>
          </div>
          <select
            bind:value={cfgDraft.multiseat.seat2_compositor}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="bg-surface-base border border-surface-border rounded-lg px-2 py-1 text-2xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            <option value="auto">Auto (Gamescope EIS, Cage fallback)</option>
            <option value="gamescope">Gamescope (Direct EIS)</option>
            <option value="cage">Cage (Standalone Kiosk)</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</SettingSection>

<!-- 2. Networking & Remote Streaming -->
<SettingSection title="Networking & Sunshine Streaming">
  <div class="space-y-3">
    <div class="flex items-center justify-between p-3 bg-surface-card border border-surface-border rounded-xl">
      <div class="space-y-0.5 max-w-md">
        <div class="text-xs font-bold text-neutral-200">UDP Broadcast Relay (Port 47584)</div>
        <div class="text-2xs text-neutral-400 leading-relaxed">
          Forwards LAN discovery packets between player instances for Goldberg Steamworks lobby auto-detection.
        </div>
      </div>
      <SeamlessSwitch
        bind:value={cfgDraft.multiseat.broadcast_relay_enabled}
        tooltip="UDP Broadcast Relay"
        on:change={(e) => { cfgDraft.multiseat.broadcast_relay_enabled = e.detail; cfgDraft = { ...cfgDraft }; }}
      />
    </div>

    <div class="grid grid-cols-2 gap-3 pt-1">
      <div>
        <label for="sunshine-port-p1-cfg" class="block text-2xs font-bold text-neutral-300 mb-1">Player 1 Sunshine Port</label>
        <input
          id="sunshine-port-p1-cfg"
          type="number"
          bind:value={cfgDraft.multiseat.sunshine_port_p1}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
        />
      </div>
      <div>
        <label for="sunshine-port-p2-cfg" class="block text-2xs font-bold text-neutral-300 mb-1">Player 2 Sunshine Port</label>
        <input
          id="sunshine-port-p2-cfg"
          type="number"
          bind:value={cfgDraft.multiseat.sunshine_port_p2}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
        />
      </div>
    </div>
  </div>
</SettingSection>

<!-- 3. Audio Architecture -->
<SettingSection title="Audio Routing & PipeWire">
  <div class="space-y-2">
    <p class="text-xs text-neutral-400 leading-relaxed">
      Co-Op mode allocates dedicated PipeWire null sinks (<code class="font-mono text-neutral-300 bg-surface-input px-1.5 py-0.5 rounded border border-surface-border">gamux-p1</code>, <code class="font-mono text-neutral-300 bg-surface-input px-1.5 py-0.5 rounded border border-surface-border">gamux-p2</code>, etc.). Games receive assigned pulse sinks via sandbox environment variables to isolate audio streams between player headsets or remote Moonlight streams.
    </p>
  </div>
</SettingSection>
