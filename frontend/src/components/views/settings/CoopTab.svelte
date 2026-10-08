<script>
  import { run } from 'svelte/legacy';

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
      seat2_force_grab_cursor: false,
      seat2_grab_keyboard: false,
      seat2_show_cursor: true,
      seat2_compositor: 'auto',
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
  run(() => {
    if (!cfgDraft.multiseat) {
      cfgDraft.multiseat = {};
    }
  });
  run(() => {
    if (cfgDraft.multiseat.player2_profile_id === undefined) cfgDraft.multiseat.player2_profile_id = 'p2';
  });
  run(() => {
    if (cfgDraft.multiseat.default_seat1_mode === undefined) cfgDraft.multiseat.default_seat1_mode = 'local';
  });
  run(() => {
    if (cfgDraft.multiseat.default_seat2_mode === undefined) cfgDraft.multiseat.default_seat2_mode = 'local';
  });
  run(() => {
    if (cfgDraft.multiseat.default_layout === undefined) cfgDraft.multiseat.default_layout = 'dual_monitor';
  });
  run(() => {
    if (cfgDraft.multiseat.seat1_gamepad === undefined) cfgDraft.multiseat.seat1_gamepad = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_gamepad === undefined) cfgDraft.multiseat.seat2_gamepad = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat1_keyboard === undefined) cfgDraft.multiseat.seat1_keyboard = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_keyboard === undefined) cfgDraft.multiseat.seat2_keyboard = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat1_mouse === undefined) cfgDraft.multiseat.seat1_mouse = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_mouse === undefined) cfgDraft.multiseat.seat2_mouse = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_force_grab_cursor === undefined) cfgDraft.multiseat.seat2_force_grab_cursor = false;
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_grab_keyboard === undefined) cfgDraft.multiseat.seat2_grab_keyboard = false;
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_show_cursor === undefined) cfgDraft.multiseat.seat2_show_cursor = true;
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_compositor === undefined || cfgDraft.multiseat.seat2_compositor === '') cfgDraft.multiseat.seat2_compositor = 'auto';
  });
  run(() => {
    if (cfgDraft.multiseat.seat1_display_index === undefined) cfgDraft.multiseat.seat1_display_index = 0;
  });
  run(() => {
    if (cfgDraft.multiseat.seat2_display_index === undefined) cfgDraft.multiseat.seat2_display_index = 0;
  });
  run(() => {
    if (cfgDraft.multiseat.sunshine_port_p1 === undefined) cfgDraft.multiseat.sunshine_port_p1 = 47989;
  });
  run(() => {
    if (cfgDraft.multiseat.sunshine_port_p2 === undefined) cfgDraft.multiseat.sunshine_port_p2 = 48010;
  });
  let seats = $derived($multiseatStore.staging?.seats || [
    $multiseatStore.staging?.seat1 || { seat_id: 'p1', mode: 'local', display_index: 0 },
    $multiseatStore.staging?.seat2 || { seat_id: 'p2', mode: 'local', display_index: (cfgDraft.multiseat?.seat2_display_index ?? 0) }
  ]);
  run(() => {
    if (seats.length === 2 && (cfgDraft.multiseat?.default_layout === 'three_player' || cfgDraft.multiseat?.default_layout === 'quad_grid')) {
      cfgDraft.multiseat.default_layout = 'side_by_side';
      multiseatStore.setLayout('side_by_side');
    } else if (seats.length === 3 && cfgDraft.multiseat?.default_layout === 'quad_grid') {
      cfgDraft.multiseat.default_layout = 'three_player';
      multiseatStore.setLayout('three_player');
    }
  });
  let gamepads = $derived($multiseatStore.gamepads);
  let displays = $derived($multiseatStore.displays || []);
  let activeSeats = $derived(seats.filter(s => !s.disabled));
  let fallbackDisplays = $derived([{ name: 'Primary Display', index: 0, width: 1920, height: 1080, refresh_rate: 60, scale: 1.0, is_primary: true }]);
  let effectiveDisplays = $derived(displays.length > 0 ? displays : fallbackDisplays);
  // Auto-clamp seat display index if host has fewer screens than assigned
  run(() => {
    if (effectiveDisplays.length > 0 && seats.length > 0) {
      seats.forEach((s) => {
        if ((s.display_index ?? 0) >= effectiveDisplays.length) {
          multiseatStore.setSeatDisplay(s.seat_id, 0);
        }
      });
    }
  });
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

  run(() => {
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
  let profileOptions = $derived((profilesStore.profiles || []).map(p => ({
    value: p.id,
    label: `${p.name} (${p.is_main ? 'Primary Host' : `Seat ${p.index || 2}`})`
  })));
</script>

<div class="space-y-6">
  <!-- 1. System Diagnostics & Isolation Tools -->
  <CoopDiagnosticsCard />

  <!-- 2. Display Topology & Screen Layout -->
  <CoopLayoutCard
    bind:cfgDraft
    {visualLayouts}
    {effectiveDisplays}
    {activeSeats}
    {seats}
  />

  <!-- 3. Player Seats Configuration (N-Player Support) -->
  <CoopSeatCard
    bind:cfgDraft
    {seats}
    {activeSeats}
    displays={effectiveDisplays}
    {profileOptions}
  />

  <!-- 4. Controller & Gamepad Diagnostics -->
  <GamepadDiagnosticsCard
    bind:cfgDraft
    {gamepadOptions}
    {gamepadAssignments}
    {gamepads}
  />

  <!-- 5. Dual Keyboards & Mice (Experimental) + Networking & Streaming + Audio -->
  <CoopStreamingCard
    bind:cfgDraft
    {keyboardOptions}
    {mouseOptions}
  />
</div>
