<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import { multiseatStore } from '../../../../stores/multiseat.js';

  export let cfgDraft = { multiseat: {} };
  export let gamepadOptions = [];
  export let gamepadAssignments = {};
  export let gamepads = [];
</script>

<SettingSection title="Host Controllers & Gamepads">
  <div class="space-y-4">
    <div class="flex items-center justify-between pb-2 border-b border-surface-border/50">
      <p class="text-xs text-neutral-400">
        Detected gamepads on host system. In Co-Op mode, controllers are dynamically masked and allocated to ensure zero input bleed between instances.
      </p>
      <button
        type="button"
        on:click={() => multiseatStore.refreshGamepads()}
        class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
        title="Rescan host input devices"
      >
        <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>Rescan</span>
      </button>
    </div>

    <!-- Controller Allocation Selectors -->
    <div class="grid grid-cols-2 gap-3 p-3 bg-surface-card border border-surface-border rounded-xl">
      <div class="space-y-1.5">
        <label for="seat1-gamepad-select" class="block text-2xs font-bold text-neutral-300">Player 1 Controller</label>
        <select
          id="seat1-gamepad-select"
          bind:value={cfgDraft.multiseat.seat1_gamepad}
          on:change={() => { cfgDraft = { ...cfgDraft }; }}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
        >
          {#each gamepadOptions as opt}
            <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
          {/each}
        </select>
        <p class="text-3xs text-neutral-400 leading-relaxed">
          {#if cfgDraft.multiseat.seat1_gamepad === 'none'}
            Forces Keyboard &amp; Mouse. All host gamepads are masked from Player 1.
          {:else if cfgDraft.multiseat.seat1_gamepad === 'auto'}
            Auto-allocates controller if available; falls back to Keyboard &amp; Mouse.
          {:else}
            Assigned to selected controller.
          {/if}
        </p>
      </div>

      <div class="space-y-1.5">
        <label for="seat2-gamepad-select" class="block text-2xs font-bold text-neutral-300">Player 2 Controller</label>
        <select
          id="seat2-gamepad-select"
          bind:value={cfgDraft.multiseat.seat2_gamepad}
          on:change={() => { cfgDraft = { ...cfgDraft }; }}
          class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
        >
          {#each gamepadOptions as opt}
            <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
          {/each}
        </select>
        <p class="text-3xs text-neutral-400 leading-relaxed">
          {#if cfgDraft.multiseat.seat2_gamepad === 'none'}
            Forces Keyboard &amp; Mouse. All host gamepads are masked from Player 2.
          {:else if cfgDraft.multiseat.seat2_gamepad === 'auto'}
            Auto-allocates dedicated controller to Player 2.
          {:else}
            Assigned to selected controller.
          {/if}
        </p>
      </div>
    </div>

    <!-- Detected Gamepads List -->
    {#if gamepads && gamepads.length > 0}
      <div class="space-y-2">
        {#each gamepads as pad}
          {@const isP1 = gamepadAssignments.p1Dev === pad.dev_node}
          {@const isP2 = gamepadAssignments.p2Dev === pad.dev_node}
          <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
            <div class="flex items-center space-x-2.5">
              <svg class="w-4 h-4 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
              </svg>
              <div class="space-y-0.5">
                <div class="text-xs font-bold text-neutral-200">{pad.name}</div>
                <div class="text-3xs text-neutral-400 font-mono">{pad.dev_node} ({pad.bus})</div>
              </div>
            </div>
            <div>
              {#if isP1}
                <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-white text-black">
                  Player 1{cfgDraft.multiseat.seat1_gamepad === 'auto' ? ' (Auto)' : ''}
                </span>
              {:else if isP2}
                <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-white text-black">
                  Player 2{cfgDraft.multiseat.seat2_gamepad === 'auto' ? ' (Auto)' : ''}
                </span>
              {:else}
                <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-400">
                  Unassigned
                </span>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="p-3 bg-surface-card/60 border border-surface-border rounded-xl text-2xs text-neutral-400">
        No physical gamepads detected. Connect USB or Bluetooth controllers before launching co-op sessions to isolate player inputs.
      </div>
    {/if}
  </div>
</SettingSection>
