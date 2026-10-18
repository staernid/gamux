<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import { multiseatStore } from '../../../../stores/multiseat.js';
  import { profilesStore } from '../../../../stores/profiles.js';

  let {
    multiseatDraft = $bindable(),
    cfgDraft = $bindable(),
    seats = [],
    activeSeats = [],
    displays = [],
    profileOptions = []
  } = $props();

  let ms = $derived(multiseatDraft || cfgDraft?.multiseat || {});

  function toggleSeat(seatId) {
    const target = seats.find(s => s.seat_id === seatId);
    if (!target) return;
    if (!target.disabled && activeSeats.length <= 1) {
      return; // Keep at least one active seat
    }
    multiseatStore.toggleSeatActive(seatId);
  }

  function setSeatOutputMode(seatIdx, mode) {
    if (seatIdx === 0) {
      if (multiseatDraft) multiseatDraft.default_seat1_mode = mode;
      if (cfgDraft?.multiseat) cfgDraft.multiseat.default_seat1_mode = mode;
      multiseatStore.setSeatMode('p1', mode);
    } else if (seatIdx === 1) {
      if (multiseatDraft) multiseatDraft.default_seat2_mode = mode;
      if (cfgDraft?.multiseat) cfgDraft.multiseat.default_seat2_mode = mode;
      multiseatStore.setSeatMode('p2', mode);
    } else {
      multiseatStore.updateSeat(seatIdx, { mode });
    }
  }

  function getPortionOptions(layout, seatsCount) {
    if (layout === 'top_bottom') {
      if (seatsCount === 2) {
        return [
          { slot: 0, label: 'Top Half' },
          { slot: 1, label: 'Bottom Half' },
        ];
      }
      if (seatsCount === 3) {
        return [
          { slot: 0, label: 'Top Row' },
          { slot: 1, label: 'Middle Row' },
          { slot: 2, label: 'Bottom Row' },
        ];
      }
      return [
        { slot: 0, label: 'Row 1 (Top)' },
        { slot: 1, label: 'Row 2' },
        { slot: 2, label: 'Row 3' },
        { slot: 3, label: 'Row 4 (Bottom)' },
      ];
    }
    if (layout === 'quad_grid') {
      return [
        { slot: 0, label: 'Top-Left' },
        { slot: 1, label: 'Top-Right' },
        { slot: 2, label: 'Bottom-Left' },
        { slot: 3, label: 'Bottom-Right' },
      ];
    }
    if (layout === 'three_player') {
      return [
        { slot: 0, label: 'Top (Wide)' },
        { slot: 1, label: 'Bottom-Left' },
        { slot: 2, label: 'Bottom-Right' },
      ];
    }
    // Default side_by_side
    if (seatsCount === 2) {
      return [
        { slot: 0, label: 'Left Half' },
        { slot: 1, label: 'Right Half' },
      ];
    }
    if (seatsCount === 3) {
      return [
        { slot: 0, label: 'Left Column' },
        { slot: 1, label: 'Center Column' },
        { slot: 2, label: 'Right Column' },
      ];
    }
    return [
      { slot: 0, label: 'Column 1 (Left)' },
      { slot: 1, label: 'Column 2' },
      { slot: 2, label: 'Column 3' },
      { slot: 3, label: 'Column 4 (Right)' },
    ];
  }
</script>

<SettingSection title="Player Seats & Profiles">
  <div class="space-y-4">
    <div class="flex items-center justify-between pb-1 border-b border-surface-border/50">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Configure player seats, profile associations, output targets, and controller bindings. Supports up to 4 concurrent players with independent lifecycle tracking.
      </p>
      {#if seats.length < 4}
        <button
          type="button"
          onclick={() => multiseatStore.addSeat()}
          class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
          title="Add another player seat (up to 4)"
        >
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <span>Add Player Seat</span>
        </button>
      {/if}
    </div>

    <!-- Player Seat Cards List -->
    <div class="space-y-3">
      {#each seats as seat, idx}
        {@const isP1 = idx === 0}
        {@const isP2 = idx === 1}
        {@const seatId = `p${idx + 1}`}
        {@const seatLabel = isP1 ? 'Player 1 (Host)' : (isP2 ? 'Seat 2 Profile (Co-op Guest)' : `Player ${idx + 1} (Co-op Guest)`)}
        {@const currentMode = isP1 ? (cfgDraft.multiseat.default_seat1_mode || 'local') : (isP2 ? (cfgDraft.multiseat.default_seat2_mode || 'local') : (seat.mode || 'local'))}
        {@const isBenched = Boolean(seat.disabled)}

        <div class="p-3.5 bg-surface-card border border-surface-border rounded-xl space-y-3 {isBenched ? 'opacity-70' : ''}">
          <!-- Seat Card Header -->
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-2">
              <span class="px-2 py-0.5 rounded text-2xs font-mono font-bold {isP1 ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-surface-input text-neutral-300 border border-surface-border'}">
                {seatId.toUpperCase()}
              </span>
              <span class="text-xs font-bold text-neutral-200">{seatLabel}</span>
            </div>

            <div class="flex items-center space-x-2">
              <button
                type="button"
                onclick={() => toggleSeat(seat.seat_id)}
                class="px-2.5 py-0.5 rounded-full text-4xs font-mono font-bold transition flex items-center space-x-1.5 cursor-pointer {isBenched ? 'bg-zinc-800/80 hover:bg-zinc-700 text-zinc-400 border border-zinc-700/60' : 'bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/40'}"
                title={isBenched ? 'Activate this seat' : 'Bench this seat (settings saved)'}
              >
                <span class="w-1.5 h-1.5 rounded-full {isBenched ? 'bg-neutral-500' : 'bg-emerald-400'}"></span>
                <span>{isBenched ? 'Benched' : 'Active'}</span>
              </button>

              {#if idx >= 2}
                <button
                  type="button"
                  onclick={() => multiseatStore.removeSeat(idx)}
                  class="px-2 py-0.5 text-3xs font-bold text-rose-400 hover:text-rose-300 bg-rose-950/40 hover:bg-rose-950/80 border border-rose-800/50 rounded transition cursor-pointer"
                  title="Remove this player seat"
                >
                  Remove Seat
                </button>
              {/if}
            </div>
          </div>

          {#if isBenched}
            <div class="px-3 py-1.5 rounded-lg bg-surface-input/50 border border-surface-border text-3xs text-neutral-400 flex items-center justify-between">
              <span>Seat {seatId.toUpperCase()} is currently benched. Its gamepad bindings and profile are saved, but it will not launch.</span>
              <button type="button" onclick={() => toggleSeat(seat.seat_id)} class="text-emerald-400 hover:text-emerald-300 font-bold underline cursor-pointer ml-2">Activate</button>
            </div>
          {/if}

          <!-- Seat Configuration Form -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-3 pt-1 {isBenched ? 'opacity-40 pointer-events-none' : ''}">
            <!-- Column 1: Profile Assignment -->
            <div class="space-y-1.5">
              <label for={`seat-${idx}-prof`} class="block text-2xs font-bold text-neutral-300">
                {isP2 ? 'Seat 2 Player Profile' : 'Player Profile'}
              </label>
              {#if isP1}
                <div class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-neutral-300 flex items-center justify-between">
                  <span>{profilesStore.activeProfile?.name || 'Primary Host'}</span>
                  <span class="text-3xs font-mono text-neutral-500">Active Profile</span>
                </div>
              {:else if isP2}
                <select
                  id="seat-2-prof"
                  bind:value={cfgDraft.multiseat.player2_profile_id}
                  onchange={() => multiseatStore.updateSeat('p2', { profile_id: cfgDraft.multiseat.player2_profile_id })}
                  class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
                >
                  {#each (profileOptions.length > 0 ? profileOptions : [{ value: 'p2', label: 'PlayerTwo (Seat 2)' }]) as opt}
                    <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
                  {/each}
                </select>
              {:else}
                <select
                  id={`seat-${idx}-prof`}
                  bind:value={seat.profile_id}
                  onchange={() => multiseatStore.updateSeat(idx, { profile_id: seat.profile_id })}
                  class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
                >
                  {#each profileOptions as opt}
                    <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
                  {/each}
                </select>
              {/if}
            </div>

            <!-- Column 2: Output Target Mode -->
            <div class="space-y-1.5">
              <span class="block text-2xs font-bold text-neutral-300">Output Target</span>
              <div class="grid grid-cols-2 gap-1.5 bg-surface-input p-1 rounded-xl border border-surface-border">
                <button
                  type="button"
                  onclick={() => setSeatOutputMode(idx, 'local')}
                  class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {currentMode === 'local' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
                >
                  Local Screen
                </button>
                <button
                  type="button"
                  onclick={() => setSeatOutputMode(idx, 'stream')}
                  class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {currentMode === 'stream' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
                >
                  Sunshine Stream
                </button>
              </div>
            </div>

            <!-- Column 3: Screen Portion / Display Assignment -->
            {#if currentMode === 'local'}
              {#if cfgDraft.multiseat.default_layout === 'multi_monitor' || cfgDraft.multiseat.default_layout === 'dual_monitor'}
                {@const curDispVal = isP1 ? (cfgDraft.multiseat.seat1_display_index ?? 0) : (isP2 ? (cfgDraft.multiseat.seat2_display_index ?? 1) : (seat.display_index ?? idx))}
                <div class="space-y-1.5">
                  <label for={`seat-${idx}-disp-idx`} class="block text-2xs font-bold text-neutral-300">Physical Monitor</label>
                  {#if displays.length > 0}
                    <select
                      id={`seat-${idx}-disp-idx`}
                      value={curDispVal}
                      onchange={(e) => {
                        const val = Number(e.target.value);
                        if (isP1) { cfgDraft.multiseat.seat1_display_index = val; multiseatStore.updateSeat(0, { display_index: val }); }
                        else if (isP2) { cfgDraft.multiseat.seat2_display_index = val; multiseatStore.updateSeat(1, { display_index: val }); }
                        else { multiseatStore.updateSeat(idx, { display_index: val }); }
                      }}
                      class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
                    >
                      {#each displays as d, dIdx}
                        <option value={dIdx} class="bg-surface-base text-neutral-200">Display {dIdx}: {d.name} ({d.width}x{d.height})</option>
                      {/each}
                      {#if curDispVal >= displays.length}
                        <option value={curDispVal} class="bg-surface-base text-neutral-400">Display {curDispVal} (Offline / Disconnected)</option>
                      {/if}
                    </select>
                  {:else}
                    <input
                      id={`seat-${idx}-disp-idx`}
                      type="number"
                      min="0"
                      max="10"
                      value={curDispVal}
                      onchange={(e) => {
                        const val = Number(e.target.value);
                        if (isP1) { cfgDraft.multiseat.seat1_display_index = val; multiseatStore.updateSeat(0, { display_index: val }); }
                        else if (isP2) { cfgDraft.multiseat.seat2_display_index = val; multiseatStore.updateSeat(1, { display_index: val }); }
                        else { multiseatStore.updateSeat(idx, { display_index: val }); }
                      }}
                      class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
                    />
                  {/if}
                </div>
              {:else}
                <div class="space-y-1.5">
                  <label for={`seat-${idx}-portion`} class="block text-2xs font-bold text-neutral-300">Screen Portion</label>
                  <select
                    id={`seat-${idx}-portion`}
                    value={seat.screen_slot ?? idx}
                    onchange={(e) => {
                      multiseatStore.setSeatSlot(seat.seat_id || seatId, Number(e.target.value));
                    }}
                    class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
                  >
                    {#each getPortionOptions(cfgDraft.multiseat.default_layout, seats.length) as opt}
                      <option value={opt.slot} class="bg-surface-base text-neutral-200">{opt.label}</option>
                    {/each}
                  </select>
                </div>
              {/if}
            {:else}
              <div class="space-y-1.5">
                <span class="block text-2xs font-bold text-neutral-300">Sunshine Stream Port</span>
                <div class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-neutral-400 font-mono">
                  Port: {isP1 ? (cfgDraft.multiseat.sunshine_port_p1 || 47989) : (cfgDraft.multiseat.sunshine_port_p2 || 48010)}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>
</SettingSection>
