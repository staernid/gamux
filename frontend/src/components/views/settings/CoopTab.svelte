<script>
  import { onMount } from 'svelte';
  import { multiseatStore, resolveGamepadAssignments } from '../../../stores/multiseat.js';
  import { profilesStore } from '../../../stores/profiles.js';
  import CoopDiagnosticsCard from './cards/CoopDiagnosticsCard.svelte';
  import GamepadDiagnosticsCard from './cards/GamepadDiagnosticsCard.svelte';
  import CoopLayoutCard from './cards/CoopLayoutCard.svelte';
  import CoopSeatCard from './cards/CoopSeatCard.svelte';
  import CoopStreamingCard from './cards/CoopStreamingCard.svelte';

  let { cfgDraft = $bindable({
    multiseat: {
      player2_profile_id: 'p2',
      default_seat1_mode: 'local',
      default_seat2_mode: 'local',
      default_layout: 'dual_monitor',
      seat1_gamepad: 'auto',
      seat2_gamepad: 'auto',
      seat1_keyboard: 'auto',
      seat2_keyboard: 'auto',
      seat1_mouse: 'auto',
      seat2_mouse: 'auto',
      seat1_force_grab_cursor: false,
      seat1_grab_keyboard: false,
      seat1_show_cursor: true,
      seat2_force_grab_cursor: false,
      seat2_grab_keyboard: false,
      seat2_show_cursor: true,
      seat2_compositor: 'gamescope',
      seat1_display_index: 0,
      seat2_display_index: 1,
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    }
  }) } = $props();


  onMount(() => {
    multiseatStore.loadCapabilities();
    multiseatStore.refreshDisplays();
    multiseatStore.refreshGamepads();
    multiseatStore.refreshKeyboards();
    multiseatStore.refreshMice();
    if (cfgDraft?.multiseat) {
      multiseatStore.syncFromConfig(cfgDraft);
    }
  });

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
  // Ensure nested multiseat defaults exist
  $effect(() => {
    if (!cfgDraft.multiseat) {
      cfgDraft.multiseat = {};
    }
    const ms = cfgDraft.multiseat;
    if (ms.player2_profile_id === undefined) ms.player2_profile_id = 'p2';
    if (ms.default_seat1_mode === undefined) ms.default_seat1_mode = 'local';
    if (ms.default_seat2_mode === undefined) ms.default_seat2_mode = 'local';
    if (ms.default_layout === undefined) ms.default_layout = 'dual_monitor';
    if (ms.seat1_gamepad === undefined) ms.seat1_gamepad = 'auto';
    if (ms.seat2_gamepad === undefined) ms.seat2_gamepad = 'auto';
    if (ms.seat1_keyboard === undefined) ms.seat1_keyboard = 'auto';
    if (ms.seat2_keyboard === undefined) ms.seat2_keyboard = 'auto';
    if (ms.seat1_mouse === undefined) ms.seat1_mouse = 'auto';
    if (ms.seat2_mouse === undefined) ms.seat2_mouse = 'auto';
    if (ms.seat1_force_grab_cursor === undefined) ms.seat1_force_grab_cursor = false;
    if (ms.seat1_grab_keyboard === undefined) ms.seat1_grab_keyboard = false;
    if (ms.seat1_show_cursor === undefined) ms.seat1_show_cursor = true;
    if (ms.seat2_force_grab_cursor === undefined) ms.seat2_force_grab_cursor = false;
    if (ms.seat2_grab_keyboard === undefined) ms.seat2_grab_keyboard = false;
    if (ms.seat2_show_cursor === undefined) ms.seat2_show_cursor = true;
    if (ms.seat2_compositor === undefined || ms.seat2_compositor === '') ms.seat2_compositor = 'gamescope';
    if (ms.seat1_display_index === undefined) ms.seat1_display_index = 0;
    if (ms.seat2_display_index === undefined) ms.seat2_display_index = 1;
    if (ms.sunshine_port_p1 === undefined) ms.sunshine_port_p1 = 47989;
    if (ms.sunshine_port_p2 === undefined) ms.sunshine_port_p2 = 48010;
  });
  let seats = $derived($multiseatStore.staging?.seats || [
    $multiseatStore.staging?.seat1 || { seat_id: 'p1', mode: 'local', display_index: 0 },
    $multiseatStore.staging?.seat2 || { seat_id: 'p2', mode: 'local', display_index: (cfgDraft.multiseat?.seat2_display_index ?? 1) }
  ]);
  // Sync seats into cfgDraft.multiseat.seats
  $effect(() => {
    if (cfgDraft?.multiseat && Array.isArray(seats)) {
      cfgDraft.multiseat.seats = seats.map((s, idx) => ({
        seat_id: s.seat_id || `p${idx + 1}`,
        mode: s.mode || (idx === 0 ? (cfgDraft.multiseat.default_seat1_mode || 'local') : (idx === 1 ? (cfgDraft.multiseat.default_seat2_mode || 'local') : 'local')),
        display_index: Number(s.display_index ?? (idx === 0 ? (cfgDraft.multiseat.seat1_display_index ?? 0) : (idx === 1 ? (cfgDraft.multiseat.seat2_display_index ?? 1) : idx))),
        screen_slot: Number(s.screen_slot ?? idx),
        disabled: Boolean(s.disabled),
        gamepad_node: s.gamepad_node || (idx === 0 ? (cfgDraft.multiseat.seat1_gamepad || 'auto') : (idx === 1 ? (cfgDraft.multiseat.seat2_gamepad || 'auto') : '')),
        keyboard_node: s.keyboard_node || (idx === 0 ? (cfgDraft.multiseat.seat1_keyboard || 'auto') : (idx === 1 ? (cfgDraft.multiseat.seat2_keyboard || 'auto') : '')),
        mouse_node: s.mouse_node || (idx === 0 ? (cfgDraft.multiseat.seat1_mouse || 'auto') : (idx === 1 ? (cfgDraft.multiseat.seat2_mouse || 'auto') : '')),
        force_grab_cursor: Boolean(s.force_grab_cursor ?? (idx === 0 ? cfgDraft.multiseat.seat1_force_grab_cursor : false)),
        grab_keyboard: Boolean(s.grab_keyboard ?? (idx === 0 ? cfgDraft.multiseat.seat1_grab_keyboard : false)),
        compositor: 'gamescope',
        nest_cage: false,
        show_cursor: s.show_cursor ?? (idx === 0 ? (cfgDraft.multiseat.seat1_show_cursor ?? true) : (cfgDraft.multiseat.seat2_show_cursor ?? true)),
        profile_id: s.profile_id || (idx === 0 ? 'main' : (idx === 1 ? (cfgDraft.multiseat.player2_profile_id || 'p2') : `p${idx + 1}`)),
        sunshine_port: Number(s.sunshine_port) || (idx === 0 ? (cfgDraft.multiseat.sunshine_port_p1 || 47989) : (idx === 1 ? (cfgDraft.multiseat.sunshine_port_p2 || 48010) : 48010 + (idx - 1) * 10)),
      }));
    }
  });
  let gamepads = $derived($multiseatStore.gamepads);
  let displays = $derived($multiseatStore.displays || []);
  let activeSeats = $derived(seats.filter(s => !s.disabled));
  let fallbackDisplays = $derived([{ name: 'Primary Display', index: 0, width: 1920, height: 1080, refresh_rate: 60, scale: 1.0, is_primary: true }]);
  let effectiveDisplays = $derived(displays.length > 0 ? displays : fallbackDisplays);
  let gamepadAssignments = $derived(resolveGamepadAssignments(
    gamepads,
    cfgDraft.multiseat?.seat1_gamepad || 'auto',
    cfgDraft.multiseat?.seat2_gamepad || 'auto'
  ));
  let gamepadOptions = $derived([
    { value: 'auto', label: 'Auto (Smart Allocation)' },
    { value: 'none', label: 'Keyboard & Mouse Only (No Gamepad)' },
    ...(gamepads || []).map(pad => ({
      value: pad.dev_node,
      label: `${pad.name} (${pad.dev_node})`
    }))
  ]);
  let keyboards = $derived($multiseatStore.keyboards);
  let mice = $derived($multiseatStore.mice);
  function normalizeDeviceNode(val, availableList = []) {
    if (!val || val === 'auto' || val === 'none') return val;
    const match = String(val).match(/\((\/dev\/input\/[^)]+)\)/);
    const candidatePath = match ? match[1] : (val.startsWith('/dev/') ? val : '');
    const candidateName = match ? val.slice(0, match.index).trim() : (!val.startsWith('/dev/') ? val.trim() : '');

    if (candidatePath && availableList.some(item => item.dev_node === candidatePath)) {
      return candidatePath;
    }
    if (candidateName && availableList.length > 0) {
      const found = availableList.find(item => item.name && item.name.toLowerCase() === candidateName.toLowerCase());
      if (found) return found.dev_node;
    }
    if (candidatePath) return candidatePath;
    return val;
  }

  $effect(() => {
    if (cfgDraft?.multiseat) {
      if (cfgDraft.multiseat.seat1_keyboard) {
        const norm = normalizeDeviceNode(cfgDraft.multiseat.seat1_keyboard, keyboards || []);
        if (norm !== cfgDraft.multiseat.seat1_keyboard) cfgDraft.multiseat.seat1_keyboard = norm;
      }
      if (cfgDraft.multiseat.seat2_keyboard) {
        const norm = normalizeDeviceNode(cfgDraft.multiseat.seat2_keyboard, keyboards || []);
        if (norm !== cfgDraft.multiseat.seat2_keyboard) cfgDraft.multiseat.seat2_keyboard = norm;
      }
      if (cfgDraft.multiseat.seat1_mouse) {
        const norm = normalizeDeviceNode(cfgDraft.multiseat.seat1_mouse, mice || []);
        if (norm !== cfgDraft.multiseat.seat1_mouse) cfgDraft.multiseat.seat1_mouse = norm;
      }
      if (cfgDraft.multiseat.seat2_mouse) {
        const norm = normalizeDeviceNode(cfgDraft.multiseat.seat2_mouse, mice || []);
        if (norm !== cfgDraft.multiseat.seat2_mouse) cfgDraft.multiseat.seat2_mouse = norm;
      }
    }
  });

  let keyboardOptions = $derived([
    { value: 'auto', label: 'Auto (Shared / Host Default)' },
    { value: 'none', label: 'None (Mask All Keyboards)' },
    ...(keyboards || []).map(k => ({
      value: k.dev_node,
      label: k.name ? `${k.name} (${k.dev_node})` : k.dev_node
    }))
  ]);
  let mouseOptions = $derived([
    { value: 'auto', label: 'Auto (Shared / Host Default)' },
    { value: 'none', label: 'None (Mask All Mice)' },
    ...(mice || []).map(m => ({
      value: m.dev_node,
      label: m.name ? `${m.name} (${m.dev_node})` : m.dev_node
    }))
  ]);
  let caps = $derived($multiseatStore.capabilities);
  let hasBwrap = $derived(caps ? Boolean(caps?.hasBwrap) : true);
  let profileOptions = $derived((profilesStore.profiles || []).map(p => ({
    value: p.id,
    label: `${p.name} (${p.is_main ? 'Primary Host' : `Seat ${p.index || 2}`})`
  })));
</script>

<div class="space-y-6">
  <!-- Missing Sandbox Tool Alert Banner (conditionally displayed if bwrap is missing) -->
  {#if !hasBwrap}
    <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl text-xs text-amber-300 flex items-start space-x-2.5">
      <svg class="w-4 h-4 flex-shrink-0 mt-0.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
      <div class="space-y-1">
        <div class="font-bold text-amber-200">Recommended Sandbox Tool Missing: Bubblewrap</div>
        <div class="text-2xs text-amber-300/90 leading-relaxed">
          Install <code class="font-mono bg-amber-500/20 px-1 py-0.5 rounded">bubblewrap</code> via your package manager for complete filesystem and input isolation between player seats.
        </div>
      </div>
    </div>
  {/if}

  <!-- 1. Display Topology & Screen Layout -->
  <CoopLayoutCard
    bind:cfgDraft
    {visualLayouts}
    {effectiveDisplays}
    {activeSeats}
    {seats}
  />

  <!-- 2. Player Seats Configuration (N-Player Support) -->
  <CoopSeatCard
    bind:cfgDraft
    {seats}
    {activeSeats}
    displays={effectiveDisplays}
    {profileOptions}
  />

  <!-- 3. Controller & Gamepad Diagnostics -->
  <GamepadDiagnosticsCard
    bind:cfgDraft
    {gamepadOptions}
    {gamepadAssignments}
    {gamepads}
  />

  <!-- 4. Dual Keyboards & Mice (Experimental) + Networking & Streaming + Audio -->
  <CoopStreamingCard
    bind:cfgDraft
    {keyboardOptions}
    {mouseOptions}
  />

  <!-- 5. System Diagnostics & Isolation Tools -->
  <CoopDiagnosticsCard />
</div>
