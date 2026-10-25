<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import { multiseatStore } from '../../../../stores/multiseat.js';

  let {
    multiseatDraft = $bindable(),
    cfgDraft = $bindable(),
    visualLayouts = [],
    effectiveDisplays = [],
    activeSeats = [],
    seats = []
  } = $props();

  let ms = $derived(multiseatDraft || cfgDraft?.multiseat || {});

  function selectLayout(layoutId) {
    if (layoutId === 'quad_grid') {
      const needed = 4 - seats.length;
      for (let i = 0; i < needed; i++) {
        multiseatStore.addSeat();
      }
      seats.forEach(s => {
        if (s.disabled) multiseatStore.toggleSeatActive(s.seat_id);
      });
    } else if (layoutId === 'three_player') {
      const needed = 3 - seats.length;
      for (let i = 0; i < needed; i++) {
        multiseatStore.addSeat();
      }
      seats.slice(0, 3).forEach(s => {
        if (s.disabled) multiseatStore.toggleSeatActive(s.seat_id);
      });
    }
    if (multiseatDraft) multiseatDraft.default_layout = layoutId;
    if (cfgDraft?.multiseat) cfgDraft.multiseat.default_layout = layoutId;
    multiseatStore.setLayout(layoutId);

    // Synchronize display assignments based on selected layout
    if (layoutId === 'dual_monitor' && effectiveDisplays.length >= 2) {
      seats.forEach((s, idx) => {
        multiseatStore.setSeatDisplay(s.seat_id, Math.min(idx, effectiveDisplays.length - 1));
      });
    } else if (layoutId !== 'dual_monitor' && layoutId !== 'windowed') {
      const targetDisp = activeSeats[0]?.display_index ?? 0;
      seats.forEach(s => {
        if ((s.display_index ?? 0) !== targetDisp) {
          multiseatStore.setSeatDisplay(s.seat_id, targetDisp);
        }
      });
    }
  }

  function getSeatObjForSlot(slotIdx, dispIdx = 0) {
    const dispSeats = activeSeats.filter(s => (s.display_index ?? 0) === dispIdx);
    return dispSeats.find((st, idx) => (st.screen_slot ?? idx) === slotIdx) || null;
  }

  function getSlotDimensions(display, layout, slotIdx, totalSeatsOnDisp) {
    const w = display?.width || 1920;
    const h = display?.height || 1080;
    if (totalSeatsOnDisp <= 1 || layout === 'dual_monitor' || layout === 'multi_monitor') {
      return `${w} × ${h}`;
    }
    if (layout === 'top_bottom') {
      const rowH = Math.floor(h / totalSeatsOnDisp);
      return `${w} × ${rowH}`;
    }
    if (layout === 'quad_grid') {
      const halfW = Math.floor(w / 2);
      const halfH = Math.floor(h / 2);
      return `${halfW} × ${halfH}`;
    }
    if (layout === 'three_player') {
      if (slotIdx === 0) {
        return `${w} × ${Math.floor(h / 2)}`;
      }
      return `${Math.floor(w / 2)} × ${Math.floor(h / 2)}`;
    }
    // Default side_by_side
    const colW = Math.floor(w / totalSeatsOnDisp);
    return `${colW} × ${h}`;
  }

  function swap2PSlots(dispIdx = 0) {
    const dispSeats = activeSeats.filter(s => (s.display_index ?? 0) === dispIdx);
    if (dispSeats.length >= 2) {
      multiseatStore.swapSeatsSlots(dispSeats[0].seat_id, dispSeats[1].seat_id);
    }
  }
</script>

<SettingSection title="Display Topology & Screen Layout">
  <div class="space-y-5">
    <div class="flex items-center justify-between pb-1 border-b border-surface-border/50">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Arrange active player viewports across connected physical monitors. Participating screens are automatically locked to 100% scale to prevent 1px seams and blur.
      </p>
      <button
        type="button"
        onclick={() => multiseatStore.refreshDisplays()}
        class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
        title="Rescan connected monitors"
      >
        <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>Rescan Displays</span>
      </button>
    </div>

    <!-- Topology & Partition Presets -->
    <div class="space-y-2">
      <span class="text-2xs font-bold text-neutral-400 uppercase tracking-wider">Topology & Screen Partition Mode</span>
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-2">
        {#each visualLayouts.filter(l => l.id !== 'windowed') as opt}
          <button
            type="button"
            onclick={() => selectLayout(opt.id)}
            class="p-2.5 rounded-xl border text-left transition flex flex-col justify-between space-y-1.5 cursor-pointer {ms.default_layout === opt.id ? 'bg-surface-header border-neutral-500 shadow-sm' : 'bg-surface-card border-surface-border hover:border-neutral-600'}"
          >
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold {ms.default_layout === opt.id ? 'text-white' : 'text-neutral-300'}">{opt.name}</span>
              <span class="text-4xs font-mono px-1.5 py-0.5 rounded {ms.default_layout === opt.id ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-surface-input text-neutral-400 border border-surface-border'}">{opt.badge}</span>
            </div>
            <p class="text-3xs text-neutral-400 leading-tight">{opt.desc}</p>
          </button>
        {/each}
      </div>
    </div>

    <!-- Physical Displays & Viewport Partition Canvas -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <span class="text-2xs font-bold text-neutral-400 uppercase tracking-wider">Connected Physical Displays & Viewports</span>
        {#if activeSeats.length >= 2 && (ms.default_layout === 'side_by_side' || ms.default_layout === 'top_bottom')}
          <button
            type="button"
            onclick={() => swap2PSlots(0)}
            class="px-2.5 py-1 bg-surface-card hover:bg-surface-header border border-surface-border hover:border-neutral-500 text-neutral-300 hover:text-white rounded-lg text-3xs font-mono font-bold transition flex items-center space-x-1.5 cursor-pointer"
            title="Swap screen portions between Player 1 and Player 2"
          >
            <span>Swap Halves ⇄</span>
          </button>
        {/if}
      </div>

      <div class="space-y-4">
        {#each effectiveDisplays as d, dIdx}
          {@const dispSeats = activeSeats.filter(s => (s.display_index ?? 0) === dIdx).sort((a, b) => (a.screen_slot ?? 0) - (b.screen_slot ?? 0))}
          {@const hasSeats = dispSeats.length > 0}
          {@const needsScaleLock = d.scale && Math.abs(d.scale - 1.0) > 0.001}

          <div class="p-4 bg-surface-card border border-surface-border rounded-xl space-y-3.5">
            <!-- Display Header -->
            <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-surface-border/40">
              <div class="flex items-center space-x-2.5">
                <span class="text-xs font-bold text-white font-mono">{d.name}</span>
                {#if d.primary || d.is_primary}
                  <span class="px-1.5 py-0.2 rounded text-4xs font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Primary</span>
                {/if}
                <span class="text-3xs text-neutral-400 font-mono">
                  {d.width}×{d.height} @ {Math.round(d.refresh_rate)}Hz
                </span>
              </div>

              <!-- Scale Status Badge with Tooltip -->
              <div class="flex items-center space-x-2">
                {#if needsScaleLock}
                  <div
                    class="flex items-center space-x-1.5 px-2.5 py-1 rounded-lg text-3xs font-mono font-bold bg-amber-500/15 text-amber-300 border border-amber-500/30"
                    title="Gamux ScaleGuard will temporarily lock this monitor to 100% scale while games run to eliminate 1px seams and blur, then restore {d.scale}x on exit."
                  >
                    <svg class="w-3 h-3 text-amber-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z" clip-rule="evenodd" />
                    </svg>
                    <span>🔒 {d.scale}x ➔ 1.0x Scale Lock (Auto in game)</span>
                  </div>
                {:else}
                  <div class="flex items-center space-x-1 px-2 py-0.5 rounded text-3xs font-mono font-bold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                    <span>✓ 100% Native (Pixel-Perfect)</span>
                  </div>
                {/if}
                <span class="text-3xs font-mono font-bold text-neutral-300 bg-surface-input px-2 py-1 rounded-lg border border-surface-border">
                  Display {dIdx}
                </span>
              </div>
            </div>

            <!-- Viewport Schematic Canvas & Partition Mode -->
            <div class="space-y-2.5">
              {#if hasSeats}
                <!-- Visual Screen Frame -->
                <div class="w-full bg-surface-input/90 rounded-xl p-3 border border-surface-border">
                  {#if dispSeats.length === 1}
                    <!-- Fullscreen single seat -->
                    <div class="h-28 rounded-lg bg-surface-card border border-neutral-700/80 flex flex-col items-center justify-center space-y-1">
                      <div class="flex items-center space-x-2">
                        <span class="px-2 py-0.5 rounded text-2xs font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">
                          {dispSeats[0].seat_id.toUpperCase()}
                        </span>
                        <span class="text-xs font-bold text-white">
                          {dispSeats[0].player?.account_name || dispSeats[0].profile_id || 'Host'}
                        </span>
                      </div>
                      <div class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                        {d.width} × {d.height} (Fullscreen 1:1)
                      </div>
                    </div>
                  {:else if ms.default_layout === 'top_bottom'}
                    <!-- Horizontal Split Rows -->
                    <div class="h-36 grid grid-rows-2 gap-1.5">
                      {#each [0, 1] as slotIdx}
                        {@const sObj = getSeatObjForSlot(slotIdx, dIdx) || dispSeats[slotIdx]}
                        {@const isP1 = sObj?.seat_id === 'p1'}
                        {@const isStream = sObj?.mode === 'stream'}
                        {#if dispSeats.length >= 2}
                          <button
                            type="button"
                            onclick={() => swap2PSlots(dIdx)}
                            class="w-full text-left rounded-lg bg-surface-card border px-4 py-2 flex items-center justify-between transition select-none group hover:border-neutral-500 hover:bg-surface-header/70 cursor-pointer active:scale-[0.99] {isStream ? 'border-indigo-500/40 bg-indigo-950/10' : 'border-neutral-700/80'}"
                            title="Click to swap top/bottom positions"
                          >
                            <div class="flex items-center space-x-2">
                              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {isP1 ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : (isStream ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40' : 'bg-surface-input text-neutral-300 border border-surface-border')}">
                                {sObj ? sObj.seat_id.toUpperCase() : `SLOT ${slotIdx + 1}`}
                              </span>
                              <span class="text-xs font-bold text-neutral-200">
                                {slotIdx === 0 ? 'Top Half' : 'Bottom Half'} • {sObj?.player?.account_name || sObj?.profile_id || 'Seat'}
                              </span>
                              <span class="opacity-0 group-hover:opacity-100 transition-opacity text-4xs font-mono text-neutral-400">
                                (Click to swap ⇄)
                              </span>
                            </div>
                            {#if isStream}
                              <span class="text-4xs font-mono text-indigo-300 bg-indigo-950/60 px-2 py-0.5 rounded border border-indigo-500/30 font-bold">
                                📡 Sunshine Stream
                              </span>
                            {:else}
                              <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                                {getSlotDimensions(d, 'top_bottom', slotIdx, dispSeats.length)}
                              </span>
                            {/if}
                          </button>
                        {:else}
                          <div class="rounded-lg bg-surface-card border px-4 py-2 flex items-center justify-between {isStream ? 'border-indigo-500/40 bg-indigo-950/10' : 'border-neutral-700/80'}">
                            <div class="flex items-center space-x-2">
                              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {isP1 ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : (isStream ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40' : 'bg-surface-input text-neutral-300 border border-surface-border')}">
                                {sObj ? sObj.seat_id.toUpperCase() : `SLOT ${slotIdx + 1}`}
                              </span>
                              <span class="text-xs font-bold text-neutral-200">
                                {slotIdx === 0 ? 'Top Half' : 'Bottom Half'} • {sObj?.player?.account_name || sObj?.profile_id || 'Seat'}
                              </span>
                            </div>
                            {#if isStream}
                              <span class="text-4xs font-mono text-indigo-300 bg-indigo-950/60 px-2 py-0.5 rounded border border-indigo-500/30 font-bold">
                                📡 Sunshine Stream
                              </span>
                            {:else}
                              <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                                {getSlotDimensions(d, 'top_bottom', slotIdx, dispSeats.length)}
                              </span>
                            {/if}
                          </div>
                        {/if}
                      {/each}
                    </div>
                  {:else if ms.default_layout === 'quad_grid' && dispSeats.length >= 4}
                    <!-- 2x2 Quadrant Grid -->
                    <div class="h-36 grid grid-cols-2 grid-rows-2 gap-1.5">
                      {#each [0, 1, 2, 3] as slotIdx}
                        {@const sObj = getSeatObjForSlot(slotIdx, dIdx) || dispSeats[slotIdx]}
                        <div class="rounded-lg bg-surface-card border border-neutral-700/80 p-2 flex flex-col items-center justify-center space-y-1">
                          <div class="flex items-center space-x-1.5">
                            <span class="px-1.5 py-0.5 rounded text-4xs font-mono font-bold bg-surface-input text-neutral-300 border border-surface-border">
                              {sObj ? sObj.seat_id.toUpperCase() : `SLOT ${slotIdx + 1}`}
                            </span>
                            <span class="text-2xs font-bold text-neutral-200 truncate max-w-[120px]">
                              {sObj?.player?.account_name || sObj?.profile_id || 'Seat'}
                            </span>
                          </div>
                          <span class="text-4xs font-mono text-emerald-400 bg-emerald-950/40 px-1.5 py-0.2 rounded border border-emerald-500/30 font-bold">
                            {getSlotDimensions(d, 'quad_grid', slotIdx, dispSeats.length)}
                          </span>
                        </div>
                      {/each}
                    </div>
                  {:else if ms.default_layout === 'three_player' && dispSeats.length >= 3}
                    {@const s0 = getSeatObjForSlot(0, dIdx) || dispSeats[0]}
                    <!-- 3-Player Split -->
                    <div class="h-36 flex flex-col gap-1.5">
                      <div class="flex-1 rounded-lg bg-surface-card border border-neutral-700/80 px-4 py-1.5 flex items-center justify-between">
                        <div class="flex items-center space-x-2">
                          <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold bg-surface-input text-neutral-300 border border-surface-border">
                            {s0 ? s0.seat_id.toUpperCase() : 'P1'}
                          </span>
                          <span class="text-xs font-bold text-neutral-200">Top Wide Half • {s0?.player?.account_name || s0?.profile_id || 'Seat'}</span>
                        </div>
                        <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                          {getSlotDimensions(d, 'three_player', 0, dispSeats.length)}
                        </span>
                      </div>
                      <div class="flex-1 grid grid-cols-2 gap-1.5">
                        {#each [1, 2] as slotIdx}
                          {@const sObj = getSeatObjForSlot(slotIdx, dIdx) || dispSeats[slotIdx]}
                          <div class="rounded-lg bg-surface-card border border-neutral-700/80 px-3 py-1 flex items-center justify-between">
                            <div class="flex items-center space-x-1.5">
                              <span class="px-1.5 py-0.5 rounded text-4xs font-mono font-bold bg-surface-input text-neutral-300 border border-surface-border">
                                {sObj ? sObj.seat_id.toUpperCase() : `P${slotIdx + 1}`}
                              </span>
                              <span class="text-2xs font-bold text-neutral-200 truncate max-w-[100px]">{sObj?.player?.account_name || 'Seat'}</span>
                            </div>
                            <span class="text-4xs font-mono text-emerald-400 bg-emerald-950/40 px-1.5 py-0.2 rounded border border-emerald-500/30 font-bold">
                              {getSlotDimensions(d, 'three_player', slotIdx, dispSeats.length)}
                            </span>
                          </div>
                        {/each}
                      </div>
                    </div>
                  {:else}
                    <!-- Default Side-by-Side (Vertical Split Columns) -->
                    <div class="h-28 flex gap-1.5">
                      {#each dispSeats as sSeat, sIdx}
                        {@const slotIdx = sSeat.screen_slot ?? sIdx}
                        {@const isP1 = sSeat.seat_id === 'p1'}
                        {@const isStream = sSeat.mode === 'stream'}
                        {#if dispSeats.length >= 2}
                          <button
                            type="button"
                            onclick={() => swap2PSlots(dIdx)}
                            class="flex-1 rounded-lg bg-surface-card border p-2.5 flex flex-col items-center justify-center space-y-1 transition select-none group relative hover:border-neutral-500 hover:bg-surface-header/70 cursor-pointer active:scale-[0.99] {isStream ? 'border-indigo-500/40 bg-indigo-950/10' : 'border-neutral-700/80'}"
                            title="Click to swap viewport positions between players"
                          >
                            <div class="flex items-center space-x-2">
                              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {isP1 ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : (isStream ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40' : 'bg-surface-input text-neutral-300 border border-surface-border')}">
                                {sSeat.seat_id.toUpperCase()}
                              </span>
                              <span class="text-xs font-bold text-white truncate max-w-[140px]">
                                {sSeat.player?.account_name || sSeat.profile_id || `Player ${sIdx + 1}`}
                              </span>
                              <span class="text-4xs font-mono text-neutral-400">
                                ({slotIdx === 0 ? 'Left' : (slotIdx === 1 ? 'Right' : `Col ${slotIdx + 1}`)})
                              </span>
                            </div>
                            {#if isStream}
                              <span class="text-4xs font-mono text-indigo-300 bg-indigo-950/60 px-2 py-0.5 rounded border border-indigo-500/30 font-bold">
                                📡 Sunshine Stream
                              </span>
                            {:else}
                              <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                                {getSlotDimensions(d, 'side_by_side', slotIdx, dispSeats.length)}
                              </span>
                            {/if}
                            <span class="opacity-0 group-hover:opacity-100 transition-opacity text-4xs font-mono text-neutral-400 flex items-center space-x-1">
                              <span>Click to swap</span>
                              <span class="text-emerald-400 font-bold">⇄</span>
                            </span>
                          </button>
                        {:else}
                          <div
                            class="flex-1 rounded-lg bg-surface-card border p-2.5 flex flex-col items-center justify-center space-y-1 {isStream ? 'border-indigo-500/40 bg-indigo-950/10' : 'border-neutral-700/80'}"
                          >
                            <div class="flex items-center space-x-2">
                              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {isP1 ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : (isStream ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40' : 'bg-surface-input text-neutral-300 border border-surface-border')}">
                                {sSeat.seat_id.toUpperCase()}
                              </span>
                              <span class="text-xs font-bold text-white truncate max-w-[140px]">
                                {sSeat.player?.account_name || sSeat.profile_id || `Player ${sIdx + 1}`}
                              </span>
                              <span class="text-4xs font-mono text-neutral-400">
                                ({slotIdx === 0 ? 'Left' : (slotIdx === 1 ? 'Right' : `Col ${slotIdx + 1}`)})
                              </span>
                            </div>
                            {#if isStream}
                              <span class="text-4xs font-mono text-indigo-300 bg-indigo-950/60 px-2 py-0.5 rounded border border-indigo-500/30 font-bold">
                                📡 Sunshine Stream
                              </span>
                            {:else}
                              <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                                {getSlotDimensions(d, 'side_by_side', slotIdx, dispSeats.length)}
                              </span>
                            {/if}
                          </div>
                        {/if}
                      {/each}
                    </div>
                  {/if}
                </div>

                <!-- Viewport Position Controls (when 2+ players share this screen) -->
                {#if dispSeats.length >= 2}
                  <div class="flex items-center justify-end pt-1">
                    <button
                      type="button"
                      onclick={() => swap2PSlots(dIdx)}
                      class="px-2.5 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-3xs font-mono font-bold transition flex items-center space-x-1 cursor-pointer"
                      title="Swap viewport slot positions between players"
                    >
                      <span>Swap Positions ⇄</span>
                    </button>
                  </div>
                {/if}
              {:else}
                <!-- No seats mapped to this monitor -->
                <div class="h-20 rounded-xl border border-dashed border-surface-border/80 flex items-center justify-center text-neutral-500 text-xs">
                  <span>No active seats assigned to Display {dIdx}. Desktop / system apps remain undisturbed.</span>
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>
</SettingSection>
