import { isSamePath } from '../../utils/game.js';

export function getActiveSession(sessions, gameDir) {
  if (!sessions || !gameDir) return null;
  if (sessions[gameDir]) return sessions[gameDir];
  for (const [key, sess] of Object.entries(sessions)) {
    if (isSamePath(key, gameDir)) return sess;
  }
  return null;
}

export function resolveMultiGamepadAssignments(gamepads = [], seatPrefs = ['auto', 'auto']) {
  const pads = Array.isArray(gamepads) ? gamepads : [];
  const assigned = new Array(seatPrefs.length).fill('none');
  const usedPads = new Set();

  // Step 1: Assign explicit (non-auto) preferences
  seatPrefs.forEach((pref, idx) => {
    if (pref && pref !== 'auto' && pref !== 'none') {
      assigned[idx] = pref;
      usedPads.add(pref);
    } else if (pref === 'none') {
      assigned[idx] = 'none';
    }
  });

  // Step 2: Handle special 2-player convention when only 1 pad is connected and both are 'auto':
  // Dedicate the sole pad to Player 2, Player 1 uses host keyboard/mouse ('none').
  if (seatPrefs.length === 2 && seatPrefs[0] === 'auto' && seatPrefs[1] === 'auto' && pads.length === 1 && !usedPads.has(pads[0].dev_node)) {
    assigned[0] = 'none';
    assigned[1] = pads[0].dev_node;
    return assigned;
  }

  // Step 3: For remaining 'auto' seats, assign available pads sequentially
  seatPrefs.forEach((pref, idx) => {
    if (pref === 'auto' && assigned[idx] === 'none') {
      const unused = pads.find(p => !usedPads.has(p.dev_node));
      if (unused) {
        assigned[idx] = unused.dev_node;
        usedPads.add(unused.dev_node);
      }
    }
  });

  return assigned;
}

export function resolveGamepadAssignments(gamepads = [], seat1Pref = 'auto', seat2Pref = 'auto') {
  const [p1Dev, p2Dev] = resolveMultiGamepadAssignments(gamepads, [seat1Pref, seat2Pref]);
  return { p1Dev, p2Dev };
}

export const defaultSeats = [
  {
    seat_id: 'p1',
    mode: 'local',
    display_index: 0,
    screen_slot: 0,
    disabled: false,
    gamepad_node: '',
    keyboard_node: '',
    mouse_node: '',
    physical_sink: '',
    profile_id: 'main',
  },
  {
    seat_id: 'p2',
    mode: 'local',
    display_index: 0,
    screen_slot: 1,
    disabled: false,
    gamepad_node: '',
    keyboard_node: '',
    mouse_node: '',
    force_grab_cursor: false,
    grab_keyboard: false,
    exclusive_grab: false,
    compositor: '',
    nest_cage: false,
    show_cursor: true,
    physical_sink: '',
    profile_id: 'p2',
  },
];

export function syncStaging(staging) {
  let seats;
  if (Array.isArray(staging.seats) && staging.seats.length > 0) {
    seats = staging.seats.map((s, idx) => ({
      ...s,
      seat_id: s.seat_id || `p${idx + 1}`,
      disabled: Boolean(s.disabled),
    }));
  } else {
    seats = [
      { ...defaultSeats[0], ...(staging.seat1 || {}) },
      { ...defaultSeats[1], ...(staging.seat2 || {}) },
    ];
  }

  return {
    ...staging,
    seats,
    seat1: seats[0],
    seat2: seats[1] || seats[0],
  };
}

export function hydrateStagingFromConfig(mCfg) {
  if (!mCfg) return null;
  const layout = mCfg.default_layout || 'dual_monitor';
  const p1Port = Number(mCfg.sunshine_port_p1) || 47989;
  const p2Port = Number(mCfg.sunshine_port_p2) || 48010;
  const isSplitLayout = layout !== 'dual_monitor' && layout !== 'windowed';
  const p1Disp = Number(mCfg.seat1_display_index ?? 0);

  let seats;
  if (Array.isArray(mCfg.seats) && mCfg.seats.length > 0) {
    const splitDisp = Number(mCfg.seats[0]?.display_index ?? p1Disp);
    seats = mCfg.seats.map((s, idx) => ({
      ...defaultSeats[Math.min(idx, 1)],
      ...s,
      seat_id: s.seat_id || `p${idx + 1}`,
      mode: s.mode || 'local',
      display_index: isSplitLayout ? splitDisp : Number(s.display_index ?? (idx === 1 ? (mCfg.seat2_display_index ?? 1) : idx)),
      screen_slot: Number(s.screen_slot ?? idx),
      disabled: Boolean(s.disabled),
      gamepad_node: s.gamepad_node ?? (idx === 0 ? (mCfg.seat1_gamepad || '') : (idx === 1 ? (mCfg.seat2_gamepad || '') : '')),
      keyboard_node: s.keyboard_node ?? (idx === 0 ? (mCfg.seat1_keyboard || '') : (idx === 1 ? (mCfg.seat2_keyboard || '') : '')),
      mouse_node: s.mouse_node ?? (idx === 0 ? (mCfg.seat1_mouse || '') : (idx === 1 ? (mCfg.seat2_mouse || '') : '')),
      profile_id: s.profile_id || (idx === 0 ? 'main' : (idx === 1 ? (mCfg.player2_profile_id || 'p2') : `p${idx + 1}`)),
      sunshine_port: Number(s.sunshine_port) || (idx === 0 ? p1Port : (idx === 1 ? p2Port : 48010 + (idx - 1) * 10)),
      force_grab_cursor: Boolean(s.force_grab_cursor ?? (idx === 0 ? (mCfg.seat1_force_grab_cursor ?? false) : false)),
      grab_keyboard: Boolean(s.grab_keyboard ?? (idx === 0 ? (mCfg.seat1_grab_keyboard ?? false) : false)),
      compositor: s.compositor || (idx === 1 ? (mCfg.seat2_compositor || 'auto') : 'auto'),
      nest_cage: false,
      show_cursor: s.show_cursor ?? (idx === 0 ? (mCfg.seat1_show_cursor ?? true) : (mCfg.seat2_show_cursor ?? true)),
    }));
    if (seats.length >= 2 && seats[0].display_index === seats[1].display_index && seats[0].screen_slot === seats[1].screen_slot) {
      if (seats[0].screen_slot === 1) {
        seats[1].screen_slot = 0;
      } else {
        seats[1].screen_slot = 1;
      }
    }
  } else {
    const p2Disp = isSplitLayout ? p1Disp : Number(mCfg.seat2_display_index ?? 1);
    seats = [
      {
        ...defaultSeats[0],
        mode: mCfg.default_seat1_mode || 'local',
        display_index: p1Disp,
        screen_slot: 0,
        disabled: false,
        gamepad_node: mCfg.seat1_gamepad || '',
        keyboard_node: mCfg.seat1_keyboard || '',
        mouse_node: mCfg.seat1_mouse || '',
        force_grab_cursor: Boolean(mCfg.seat1_force_grab_cursor),
        grab_keyboard: Boolean(mCfg.seat1_grab_keyboard),
        show_cursor: mCfg.seat1_show_cursor ?? true,
        compositor: 'gamescope',
        profile_id: 'main',
        sunshine_port: p1Port,
      },
      {
        ...defaultSeats[1],
        mode: mCfg.default_seat2_mode || 'local',
        display_index: p2Disp,
        screen_slot: 1,
        disabled: false,
        gamepad_node: mCfg.seat2_gamepad || '',
        keyboard_node: mCfg.seat2_keyboard || '',
        mouse_node: mCfg.seat2_mouse || '',
        force_grab_cursor: false,
        grab_keyboard: false,
        compositor: 'gamescope',
        nest_cage: false,
        show_cursor: mCfg.seat2_show_cursor ?? true,
        profile_id: mCfg.player2_profile_id || 'p2',
        sunshine_port: p2Port,
      },
    ];
  }

  return syncStaging({
    seats,
    layout,
    sunshine_port_p1: p1Port,
    sunshine_port_p2: p2Port,
  });
}
