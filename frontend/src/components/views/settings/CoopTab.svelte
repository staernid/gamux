<script>
  import { onMount } from 'svelte';
  import SettingSection from '../../common/SettingSection.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import { multiseatStore, resolveGamepadAssignments } from '../../../stores/multiseat.js';
  import { profilesStore } from '../../../stores/profiles.js';
  import CoopDiagnosticsCard from './cards/CoopDiagnosticsCard.svelte';
  import GamepadDiagnosticsCard from './cards/GamepadDiagnosticsCard.svelte';

  export let cfgDraft = {
    multiseat: {
      player2_profile_id: 'p2',
      default_seat1_mode: 'local',
      default_seat2_mode: 'local',
      default_layout: 'dual_monitor',
      broadcast_relay_enabled: true,
      seat1_gamepad: 'auto',
      seat2_gamepad: 'auto',
      seat1_keyboard: 'auto',
      seat2_keyboard: 'auto',
      seat1_mouse: 'auto',
      seat2_mouse: 'auto',
      seat2_force_grab_cursor: false,
      seat2_grab_keyboard: false,
      seat2_show_cursor: true,
      seat2_compositor: 'auto',
      seat1_display_index: 0,
      seat2_display_index: 1,
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    }
  };

  // Ensure nested multiseat defaults exist
  $: if (!cfgDraft.multiseat) {
    cfgDraft.multiseat = {};
  }
  $: if (cfgDraft.multiseat.player2_profile_id === undefined) cfgDraft.multiseat.player2_profile_id = 'p2';
  $: if (cfgDraft.multiseat.default_seat1_mode === undefined) cfgDraft.multiseat.default_seat1_mode = 'local';
  $: if (cfgDraft.multiseat.default_seat2_mode === undefined) cfgDraft.multiseat.default_seat2_mode = 'local';
  $: if (cfgDraft.multiseat.default_layout === undefined) cfgDraft.multiseat.default_layout = 'dual_monitor';
  $: if (cfgDraft.multiseat.broadcast_relay_enabled === undefined) cfgDraft.multiseat.broadcast_relay_enabled = true;
  $: if (cfgDraft.multiseat.seat1_gamepad === undefined) cfgDraft.multiseat.seat1_gamepad = 'auto';
  $: if (cfgDraft.multiseat.seat2_gamepad === undefined) cfgDraft.multiseat.seat2_gamepad = 'auto';
  $: if (cfgDraft.multiseat.seat1_keyboard === undefined) cfgDraft.multiseat.seat1_keyboard = 'auto';
  $: if (cfgDraft.multiseat.seat2_keyboard === undefined) cfgDraft.multiseat.seat2_keyboard = 'auto';
  $: if (cfgDraft.multiseat.seat1_mouse === undefined) cfgDraft.multiseat.seat1_mouse = 'auto';
  $: if (cfgDraft.multiseat.seat2_mouse === undefined) cfgDraft.multiseat.seat2_mouse = 'auto';
  $: if (cfgDraft.multiseat.seat2_force_grab_cursor === undefined) cfgDraft.multiseat.seat2_force_grab_cursor = false;
  $: if (cfgDraft.multiseat.seat2_grab_keyboard === undefined) cfgDraft.multiseat.seat2_grab_keyboard = false;
  $: if (cfgDraft.multiseat.seat2_show_cursor === undefined) cfgDraft.multiseat.seat2_show_cursor = true;
  $: if (cfgDraft.multiseat.seat2_compositor === undefined || cfgDraft.multiseat.seat2_compositor === '') cfgDraft.multiseat.seat2_compositor = 'auto';
  $: if (cfgDraft.multiseat.seat1_display_index === undefined) cfgDraft.multiseat.seat1_display_index = 0;
  $: if (cfgDraft.multiseat.seat2_display_index === undefined) cfgDraft.multiseat.seat2_display_index = 0;
  $: if (cfgDraft.multiseat.sunshine_port_p1 === undefined) cfgDraft.multiseat.sunshine_port_p1 = 47989;
  $: if (cfgDraft.multiseat.sunshine_port_p2 === undefined) cfgDraft.multiseat.sunshine_port_p2 = 48010;

  onMount(() => {
    multiseatStore.loadCapabilities();
    multiseatStore.refreshDisplays();
    multiseatStore.refreshGamepads();
    multiseatStore.refreshKeyboards();
    multiseatStore.refreshMice();
    if (cfgDraft.multiseat?.default_layout) {
      let layout = cfgDraft.multiseat.default_layout;
      if (seats.length === 2 && (layout === 'three_player' || layout === 'quad_grid')) {
        layout = 'side_by_side';
        cfgDraft.multiseat.default_layout = layout;
      } else if (seats.length === 3 && layout === 'quad_grid') {
        layout = 'three_player';
        cfgDraft.multiseat.default_layout = layout;
      }
      multiseatStore.setLayout(layout);
    }
  });

  $: if (seats.length === 2 && (cfgDraft.multiseat?.default_layout === 'three_player' || cfgDraft.multiseat?.default_layout === 'quad_grid')) {
    cfgDraft.multiseat.default_layout = 'side_by_side';
    multiseatStore.setLayout('side_by_side');
  } else if (seats.length === 3 && cfgDraft.multiseat?.default_layout === 'quad_grid') {
    cfgDraft.multiseat.default_layout = 'three_player';
    multiseatStore.setLayout('three_player');
  }

  $: gamepads = $multiseatStore.gamepads;
  $: displays = $multiseatStore.displays || [];
  $: seats = $multiseatStore.staging?.seats || [
    $multiseatStore.staging?.seat1 || { seat_id: 'p1', mode: 'local', display_index: 0 },
    $multiseatStore.staging?.seat2 || { seat_id: 'p2', mode: 'local', display_index: (cfgDraft.multiseat?.seat2_display_index ?? 0) }
  ];
  $: activeSeats = seats.filter(s => !s.disabled);
  $: benchedSeats = seats.filter(s => s.disabled);
  $: fallbackDisplays = [{ name: 'Primary Display', index: 0, width: 1920, height: 1080, refresh_rate: 60, scale: 1.0, is_primary: true }];
  $: effectiveDisplays = displays.length > 0 ? displays : fallbackDisplays;

  // Auto-clamp seat display index if host has fewer screens than assigned
  $: if (effectiveDisplays.length > 0 && seats.length > 0) {
    seats.forEach((s) => {
      if ((s.display_index ?? 0) >= effectiveDisplays.length) {
        multiseatStore.setSeatDisplay(s.seat_id, 0);
      }
    });
  }

  $: gamepadAssignments = resolveGamepadAssignments(
    gamepads,
    cfgDraft.multiseat?.seat1_gamepad || 'auto',
    cfgDraft.multiseat?.seat2_gamepad || 'auto'
  );

  $: gamepadOptions = [
    { value: 'auto', label: 'Auto (Smart Allocation)' },
    { value: 'none', label: 'Keyboard & Mouse Only (No Gamepad)' },
    ...(gamepads || []).map(pad => ({
      value: pad.dev_node,
      label: `${pad.name} (${pad.dev_node})`
    }))
  ];

  $: keyboards = $multiseatStore.keyboards;
  $: mice = $multiseatStore.mice;

  $: keyboardOptions = [
    { value: 'auto', label: 'Auto (Shared / Host Default)' },
    { value: 'none', label: 'None (Mask All Keyboards)' },
    ...(keyboards || []).map(k => ({
      value: k.name ? `${k.name} (${k.dev_node})` : k.dev_node,
      label: k.name ? `${k.name} (${k.dev_node})` : k.dev_node
    }))
  ];

  $: mouseOptions = [
    { value: 'auto', label: 'Auto (Shared / Host Default)' },
    { value: 'none', label: 'None (Mask All Mice)' },
    ...(mice || []).map(m => ({
      value: m.name ? `${m.name} (${m.dev_node})` : m.dev_node,
      label: m.name ? `${m.name} (${m.dev_node})` : m.dev_node
    }))
  ];

  $: profileOptions = ($profilesStore.profiles || []).map(p => ({
    value: p.id,
    label: `${p.name} (${p.is_main ? 'Primary Host' : `~/Games/umu-p${p.index}`})`
  }));

  const visualLayouts = [
    {
      id: 'dual_monitor',
      name: 'Multi-Monitor',
      desc: 'Separate physical displays per seat with zero screen partitioning',
      type: 'monitors',
      badge: '1 per Display',
    },
    {
      id: 'side_by_side',
      name: 'Side-by-Side',
      desc: 'Vertical split columns across a single monitor',
      type: 'split-v',
      badge: '2-4 Players',
    },
    {
      id: 'top_bottom',
      name: 'Top / Bottom',
      desc: 'Horizontal split rows across a single monitor',
      type: 'split-h',
      badge: '2-4 Players',
    },
    {
      id: 'quad_grid',
      name: '2x2 Quad Grid',
      desc: 'Four equal quadrant viewports (automatically sets 4 seats)',
      type: 'quad',
      badge: '4 Players',
    },
    {
      id: 'three_player',
      name: '3-Player Split',
      desc: 'Wide primary half with two secondary halves (automatically sets 3 seats)',
      type: 'three',
      badge: '3 Players',
    },
    {
      id: 'windowed',
      name: 'Windowed',
      desc: 'Floating desktop windows for window managers',
      type: 'window',
      badge: 'Any',
    },
  ];

  function toggleSeat(seatId) {
    const target = seats.find(s => s.seat_id === seatId);
    if (!target) return;
    if (!target.disabled && activeSeats.length <= 1) {
      return; // Keep at least one active seat
    }
    multiseatStore.toggleSeatActive(seatId);
  }

  function getSeatForSlot(slotIdx, dispIdx = 0) {
    const dispSeats = activeSeats.filter(s => (s.display_index ?? 0) === dispIdx);
    const s = dispSeats.find((st, idx) => (st.screen_slot ?? idx) === slotIdx);
    return s ? s.seat_id.toUpperCase() : `P${slotIdx + 1}`;
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
    cfgDraft.multiseat.default_layout = layoutId;
    cfgDraft = { ...cfgDraft };
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

  function handleRemoveSeat(idx) {
    multiseatStore.removeSeat(idx);
    const nextCount = seats.length - 1;
    if (cfgDraft.multiseat.default_layout === 'quad_grid' && nextCount < 4) {
      const fallback = nextCount === 3 ? 'three_player' : 'side_by_side';
      cfgDraft.multiseat.default_layout = fallback;
      multiseatStore.setLayout(fallback);
      cfgDraft = { ...cfgDraft };
    } else if (cfgDraft.multiseat.default_layout === 'three_player' && nextCount < 3) {
      cfgDraft.multiseat.default_layout = 'side_by_side';
      multiseatStore.setLayout('side_by_side');
      cfgDraft = { ...cfgDraft };
    }
  }

  function swap2PSlots(dispIdx = 0) {
    const dispSeats = activeSeats.filter(s => (s.display_index ?? 0) === dispIdx);
    if (dispSeats.length >= 2) {
      const s0 = dispSeats[0];
      const s1 = dispSeats[1];
      const slot0 = s0.screen_slot ?? 0;
      const slot1 = s1.screen_slot ?? 1;
      multiseatStore.setSeatSlot(s0.seat_id, slot1);
      multiseatStore.setSeatSlot(s1.seat_id, slot0);
    }
  }

  function setSeatOutputMode(seatIdx, mode) {
    if (seatIdx === 0) {
      cfgDraft.multiseat.default_seat1_mode = mode;
      multiseatStore.setSeatMode('p1', mode);
    } else if (seatIdx === 1) {
      cfgDraft.multiseat.default_seat2_mode = mode;
      multiseatStore.setSeatMode('p2', mode);
    } else {
      multiseatStore.updateSeat(seatIdx, { mode });
    }
    cfgDraft = { ...cfgDraft };
  }
</script>

<div class="space-y-6">
  <!-- 1. System Diagnostics & Isolation Tools -->
  <CoopDiagnosticsCard />

  <!-- 2. Display Topology & Screen Layout -->
  <SettingSection title="Display Topology & Screen Layout">
    <div class="space-y-5">
      <div class="flex items-center justify-between pb-1 border-b border-surface-border/50">
        <p class="text-xs text-neutral-400 leading-relaxed">
          Arrange active player viewports across connected physical monitors. Participating screens are automatically locked to 100% scale to prevent 1px seams and blur.
        </p>
        <button
          type="button"
          on:click={() => multiseatStore.refreshDisplays()}
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
              on:click={() => selectLayout(opt.id)}
              class="p-2.5 rounded-xl border text-left transition flex flex-col justify-between space-y-1.5 cursor-pointer {cfgDraft.multiseat.default_layout === opt.id ? 'bg-surface-header border-neutral-500 shadow-sm' : 'bg-surface-card border-surface-border hover:border-neutral-600'}"
            >
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold {cfgDraft.multiseat.default_layout === opt.id ? 'text-white' : 'text-neutral-300'}">{opt.name}</span>
                <span class="text-4xs font-mono px-1.5 py-0.5 rounded {cfgDraft.multiseat.default_layout === opt.id ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-surface-input text-neutral-400 border border-surface-border'}">{opt.badge}</span>
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
          {#if activeSeats.length >= 2 && (cfgDraft.multiseat.default_layout === 'side_by_side' || cfgDraft.multiseat.default_layout === 'top_bottom')}
            <button
              type="button"
              on:click={() => swap2PSlots(0)}
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
                  {#if d.is_primary}
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
                    {:else if cfgDraft.multiseat.default_layout === 'top_bottom'}
                      <!-- Horizontal Split Rows -->
                      <div class="h-36 grid grid-rows-2 gap-1.5">
                        {#each [0, 1] as slotIdx}
                          {@const sObj = getSeatObjForSlot(slotIdx, dIdx) || dispSeats[slotIdx]}
                          <div class="rounded-lg bg-surface-card border border-neutral-700/80 px-4 py-2 flex items-center justify-between">
                            <div class="flex items-center space-x-2">
                              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-input text-neutral-300 border border-surface-border">
                                {sObj ? sObj.seat_id.toUpperCase() : `SLOT ${slotIdx + 1}`}
                              </span>
                              <span class="text-xs font-bold text-neutral-200">
                                {slotIdx === 0 ? 'Top Half' : 'Bottom Half'} • {sObj?.player?.account_name || sObj?.profile_id || 'Seat'}
                              </span>
                            </div>
                            <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                              {getSlotDimensions(d, 'top_bottom', slotIdx, dispSeats.length)}
                            </span>
                          </div>
                        {/each}
                      </div>
                    {:else if cfgDraft.multiseat.default_layout === 'quad_grid' && dispSeats.length >= 4}
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
                    {:else if cfgDraft.multiseat.default_layout === 'three_player' && dispSeats.length >= 3}
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
                          {@const isFirst = sIdx === 0}
                          <div class="flex-1 rounded-lg bg-surface-card border border-neutral-700/80 p-2.5 flex flex-col items-center justify-center space-y-1">
                            <div class="flex items-center space-x-2">
                              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {isFirst ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-surface-input text-neutral-300 border border-surface-border'}">
                                {sSeat.seat_id.toUpperCase()}
                              </span>
                              <span class="text-xs font-bold text-white truncate max-w-[140px]">
                                {sSeat.player?.account_name || sSeat.profile_id || `Player ${sIdx + 1}`}
                              </span>
                            </div>
                            <span class="text-3xs font-mono text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-500/30 font-bold">
                              {getSlotDimensions(d, 'side_by_side', slotIdx, dispSeats.length)}
                            </span>
                          </div>
                        {/each}
                      </div>
                    {/if}
                  </div>

                  <!-- Viewport Position Controls (when 2+ players share this screen) -->
                  {#if dispSeats.length >= 2}
                    <div class="flex items-center justify-end pt-1">
                      <button
                        type="button"
                        on:click={() => swap2PSlots(dIdx)}
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

  <!-- 3. Player Seats Configuration (N-Player Support) -->
  <SettingSection title="Player Seats & Profiles">
    <div class="space-y-4">
      <div class="flex items-center justify-between pb-1 border-b border-surface-border/50">
        <p class="text-xs text-neutral-400 leading-relaxed">
          Configure player seats, profile associations, output targets, and controller bindings. Supports up to 4 concurrent players with independent lifecycle tracking.
        </p>
        {#if seats.length < 4}
          <button
            type="button"
            on:click={() => multiseatStore.addSeat()}
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
                  on:click={() => toggleSeat(seat.seat_id)}
                  class="px-2.5 py-0.5 rounded-full text-4xs font-mono font-bold transition flex items-center space-x-1.5 cursor-pointer {isBenched ? 'bg-zinc-800/80 hover:bg-zinc-700 text-zinc-400 border border-zinc-700/60' : 'bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/40'}"
                  title={isBenched ? 'Activate this seat' : 'Bench this seat (settings saved)'}
                >
                  <span class="w-1.5 h-1.5 rounded-full {isBenched ? 'bg-neutral-500' : 'bg-emerald-400'}"></span>
                  <span>{isBenched ? 'Benched' : 'Active'}</span>
                </button>

                {#if idx >= 2}
                  <button
                    type="button"
                    on:click={() => multiseatStore.removeSeat(idx)}
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
                <button type="button" on:click={() => toggleSeat(seat.seat_id)} class="text-emerald-400 hover:text-emerald-300 font-bold underline cursor-pointer ml-2">Activate</button>
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
                    <span>{$profilesStore.activeProfile?.name || 'Primary Host'}</span>
                    <span class="text-3xs font-mono text-neutral-500">Active Profile</span>
                  </div>
                {:else if isP2}
                  <select
                    id="seat-2-prof"
                    bind:value={cfgDraft.multiseat.player2_profile_id}
                    on:change={() => { multiseatStore.updateSeat('p2', { profile_id: cfgDraft.multiseat.player2_profile_id }); cfgDraft = { ...cfgDraft }; }}
                    class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
                  >
                    {#each (profileOptions.length > 0 ? profileOptions : [{ value: 'p2', label: 'PlayerTwo (~/Games/umu-p2)' }]) as opt}
                      <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
                    {/each}
                  </select>
                {:else}
                  <select
                    id={`seat-${idx}-prof`}
                    bind:value={seat.profile_id}
                    on:change={() => multiseatStore.updateSeat(idx, { profile_id: seat.profile_id })}
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
                    on:click={() => setSeatOutputMode(idx, 'local')}
                    class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {currentMode === 'local' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
                  >
                    Local Screen
                  </button>
                  <button
                    type="button"
                    on:click={() => setSeatOutputMode(idx, 'stream')}
                    class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {currentMode === 'stream' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
                  >
                    Sunshine Stream
                  </button>
                </div>
              </div>

              <!-- Column 3: Screen Portion / Display Assignment -->
              {#if currentMode === 'local'}
                {#if cfgDraft.multiseat.default_layout === 'multi_monitor' || cfgDraft.multiseat.default_layout === 'dual_monitor'}
                  <div class="space-y-1.5">
                    <label for={`seat-${idx}-disp-idx`} class="block text-2xs font-bold text-neutral-300">Physical Monitor</label>
                    {#if displays.length > 0}
                      <select
                        id={`seat-${idx}-disp-idx`}
                        value={isP1 ? (cfgDraft.multiseat.seat1_display_index ?? 0) : (isP2 ? (cfgDraft.multiseat.seat2_display_index ?? 1) : (seat.display_index ?? idx))}
                        on:change={(e) => {
                          const val = Number(e.target.value);
                          if (isP1) { cfgDraft.multiseat.seat1_display_index = val; multiseatStore.updateSeat(0, { display_index: val }); cfgDraft = { ...cfgDraft }; }
                          else if (isP2) { cfgDraft.multiseat.seat2_display_index = val; multiseatStore.updateSeat(1, { display_index: val }); cfgDraft = { ...cfgDraft }; }
                          else { multiseatStore.updateSeat(idx, { display_index: val }); }
                        }}
                        class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
                      >
                        {#each displays as d, dIdx}
                          <option value={dIdx} class="bg-surface-base text-neutral-200">Display {dIdx}: {d.name} ({d.width}x{d.height})</option>
                        {/each}
                      </select>
                    {:else}
                      <input
                        id={`seat-${idx}-disp-idx`}
                        type="number"
                        min="0"
                        max="10"
                        value={isP1 ? (cfgDraft.multiseat.seat1_display_index ?? 0) : (isP2 ? (cfgDraft.multiseat.seat2_display_index ?? 1) : (seat.display_index ?? idx))}
                        on:change={(e) => {
                          const val = Number(e.target.value);
                          if (isP1) { cfgDraft.multiseat.seat1_display_index = val; multiseatStore.updateSeat(0, { display_index: val }); cfgDraft = { ...cfgDraft }; }
                          else if (isP2) { cfgDraft.multiseat.seat2_display_index = val; multiseatStore.updateSeat(1, { display_index: val }); cfgDraft = { ...cfgDraft }; }
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
                      on:change={(e) => multiseatStore.setSeatSlot(seat.seat_id || seatId, Number(e.target.value))}
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

  <!-- 4. Controller & Gamepad Diagnostics -->
  <GamepadDiagnosticsCard bind:cfgDraft {gamepadOptions} {gamepadAssignments} {gamepads} />

  <!-- 5. Dual Keyboards & Mice (Games Without Gamepad Support) -->
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

  <!-- 6. Networking & Remote Streaming -->
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

  <!-- 7. Audio Architecture -->
  <SettingSection title="Audio Routing & PipeWire">
    <div class="space-y-2">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Co-Op mode allocates dedicated PipeWire null sinks (<code class="font-mono text-neutral-300 bg-surface-input px-1.5 py-0.5 rounded border border-surface-border">gamux-p1</code>, <code class="font-mono text-neutral-300 bg-surface-input px-1.5 py-0.5 rounded border border-surface-border">gamux-p2</code>, etc.). Games receive assigned pulse sinks via sandbox environment variables to isolate audio streams between player headsets or remote Moonlight streams.
      </p>
    </div>
  </SettingSection>
</div>
