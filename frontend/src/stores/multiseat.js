import { writable, get } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { configStore } from './config.js';
import { profilesStore } from './profiles.js';
import { isSamePath } from '../utils/game.js';

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

const defaultSeats = [
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

function syncStaging(staging) {
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

  let seats;
  if (Array.isArray(mCfg.seats) && mCfg.seats.length > 0) {
    seats = mCfg.seats.map((s, idx) => ({
      ...defaultSeats[Math.min(idx, 1)],
      ...s,
      seat_id: s.seat_id || `p${idx + 1}`,
      mode: s.mode || 'local',
      display_index: Number(s.display_index ?? (idx === 1 ? (mCfg.seat2_display_index ?? 1) : idx)),
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
    seats = [
      {
        ...defaultSeats[0],
        mode: mCfg.default_seat1_mode || 'local',
        display_index: Number(mCfg.seat1_display_index ?? 0),
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
        display_index: Number(mCfg.seat2_display_index ?? 1),
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

function createMultiseatStore() {
  const { subscribe, set, update } = writable({
    capabilities: {
      has_bwrap: true,
      has_gamescope: true,
      has_cage: false,
      has_sunshine: true,
      has_pipewire: true,
    },
    gamepads: [],
    keyboards: [],
    mice: [],
    displays: [],
    isCoopEnabled: false,
    staging: {
      seats: [
        { ...defaultSeats[0] },
        { ...defaultSeats[1] },
      ],
      seat1: { ...defaultSeats[0] },
      seat2: { ...defaultSeats[1] },
      layout: 'dual_monitor', // 'dual_monitor' | 'side_by_side' | 'top_bottom' | 'quad_grid' | 'three_player' | 'windowed'
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    },
    activeSessions: {}, // gameDir -> status
  });

  return {
    subscribe,

    syncFromConfig(cfg) {
      if (!cfg?.multiseat) return;
      const hydrated = hydrateStagingFromConfig(cfg.multiseat);
      if (hydrated) {
        update(state => ({
          ...state,
          staging: {
            ...state.staging,
            ...hydrated,
          },
        }));
      }
    },

    async loadCapabilities() {
      try {
        const caps = await AppAPI.getMultiseatCapabilities();
        if (caps) {
          update(state => ({
            ...state,
            capabilities: {
              has_bwrap: Boolean(caps.has_bwrap ?? caps.HasBwrap ?? caps.hasBwrap),
              has_gamescope: Boolean(caps.has_gamescope ?? caps.HasGamescope ?? caps.hasGamescope),
              has_cage: Boolean(caps.has_cage ?? caps.HasCage ?? caps.hasCage),
              has_sunshine: Boolean(caps.has_sunshine ?? caps.HasSunshine ?? caps.hasSunshine),
              has_pipewire: Boolean(caps.has_pipewire ?? caps.HasPipewire ?? caps.hasPipewire),
            },
          }));
        }
      } catch (err) {
        console.warn('Failed to probe multiseat capabilities:', err);
      }
    },

    async refreshDisplays() {
      try {
        const displays = await AppAPI.getConnectedDisplays();
        if (Array.isArray(displays)) {
          update(state => ({ ...state, displays }));
        }
      } catch (err) {
        console.warn('Failed to probe displays:', err);
      }
    },

    async refreshGamepads() {
      try {
        const pads = await AppAPI.getConnectedGamepads();
        if (Array.isArray(pads)) {
          update(state => ({ ...state, gamepads: pads }));
        }
      } catch (err) {
        console.warn('Failed to refresh connected gamepads:', err);
      }
    },

    async refreshKeyboards() {
      try {
        const kbds = await AppAPI.getConnectedKeyboards();
        if (Array.isArray(kbds)) {
          update(state => ({ ...state, keyboards: kbds }));
        }
      } catch (err) {
        console.warn('Failed to refresh connected keyboards:', err);
      }
    },

    async refreshMice() {
      try {
        const mice = await AppAPI.getConnectedMice();
        if (Array.isArray(mice)) {
          update(state => ({ ...state, mice }));
        }
      } catch (err) {
        console.warn('Failed to refresh connected mice:', err);
      }
    },

    toggleCoop(forceVal) {
      update(state => {
        const enabled = typeof forceVal === 'boolean' ? forceVal : !state.isCoopEnabled;
        return { ...state, isCoopEnabled: enabled };
      });
    },

    addSeat() {
      update(state => {
        const currentSeats = state.staging.seats || [state.staging.seat1, state.staging.seat2];
        if (currentSeats.length >= 4) return state;
        const nextIdx = currentSeats.length + 1;
        const newSeat = {
          seat_id: `p${nextIdx}`,
          mode: 'local',
          display_index: currentSeats.length % 2,
          disabled: false,
          gamepad_node: '',
          keyboard_node: '',
          mouse_node: '',
          physical_sink: '',
          profile_id: `p${nextIdx}`,
          screen_slot: currentSeats.length,
          force_grab_cursor: false,
          grab_keyboard: false,
          exclusive_grab: false,
          compositor: '',
          nest_cage: false,
          show_cursor: true,
        };
        const nextSeats = [...currentSeats, newSeat];
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            seats: nextSeats,
          }),
        };
      });
    },

    toggleSeatActive(seatId) {
      update(state => {
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        const idx = currentSeats.findIndex(s => s.seat_id === seatId || (seatId === 'seat1' && s.seat_id === 'p1') || (seatId === 'seat2' && s.seat_id === 'p2'));
        if (idx < 0) return state;

        const nextDisabled = !currentSeats[idx].disabled;
        currentSeats[idx] = { ...currentSeats[idx], disabled: nextDisabled };

        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            seats: currentSeats,
          }),
        };
      });
    },

    setSeatDisplay(seatId, displayIndex) {
      update(state => {
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        const idx = currentSeats.findIndex(s => s.seat_id === seatId || (seatId === 'seat1' && s.seat_id === 'p1') || (seatId === 'seat2' && s.seat_id === 'p2'));
        if (idx < 0) return state;

        currentSeats[idx] = { ...currentSeats[idx], display_index: Number(displayIndex) };

        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            seats: currentSeats,
          }),
        };
      });
    },

    setSeatSlot(seatId, targetSlot) {
      update(state => {
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        const seatIdx = currentSeats.findIndex(s => s.seat_id === seatId);
        if (seatIdx < 0) return state;

        const oldSlot = currentSeats[seatIdx].screen_slot ?? seatIdx;
        const otherIdx = currentSeats.findIndex((s, idx) => idx !== seatIdx && (s.screen_slot ?? idx) === targetSlot);

        currentSeats[seatIdx] = { ...currentSeats[seatIdx], screen_slot: targetSlot };
        if (otherIdx >= 0) {
          let fallbackSlot = oldSlot;
          if (fallbackSlot === targetSlot) {
            fallbackSlot = targetSlot === 0 ? 1 : 0;
          }
          currentSeats[otherIdx] = { ...currentSeats[otherIdx], screen_slot: fallbackSlot };
        }

        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            seats: currentSeats,
          }),
        };
      });
    },

    swapSeatsSlots(seatId1, seatId2) {
      update(state => {
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        const idx1 = currentSeats.findIndex(s => s.seat_id === seatId1);
        const idx2 = currentSeats.findIndex(s => s.seat_id === seatId2);
        if (idx1 < 0 || idx2 < 0) return state;

        let slot1 = currentSeats[idx1].screen_slot ?? idx1;
        let slot2 = currentSeats[idx2].screen_slot ?? idx2;
        if (slot1 === slot2) {
          slot1 = 0;
          slot2 = 1;
        } else {
          const temp = slot1;
          slot1 = slot2;
          slot2 = temp;
        }

        currentSeats[idx1] = { ...currentSeats[idx1], screen_slot: slot1 };
        currentSeats[idx2] = { ...currentSeats[idx2], screen_slot: slot2 };

        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            seats: currentSeats,
          }),
        };
      });
    },

    removeSeat(index) {
      update(state => {
        const currentSeats = state.staging.seats || [state.staging.seat1, state.staging.seat2];
        if (currentSeats.length <= 2) return state;
        const nextSeats = currentSeats.filter((_, idx) => idx !== index).map((s, idx) => ({
          ...s,
          seat_id: `p${idx + 1}`,
        }));
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            seats: nextSeats,
          }),
        };
      });
    },

    updateSeat(indexOrId, partial) {
      update(state => {
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        const idx = typeof indexOrId === 'number'
          ? indexOrId
          : currentSeats.findIndex(s => s.seat_id === indexOrId || (indexOrId === 'seat1' && s.seat_id === 'p1') || (indexOrId === 'seat2' && s.seat_id === 'p2'));

        if (idx < 0 || idx >= currentSeats.length) return state;
        currentSeats[idx] = { ...currentSeats[idx], ...partial };

        const nextStaging = {
          ...state.staging,
          seats: currentSeats,
        };
        if (idx === 0) nextStaging.seat1 = currentSeats[0];
        if (idx === 1) nextStaging.seat2 = currentSeats[1];

        return {
          ...state,
          staging: syncStaging(nextStaging),
        };
      });
    },

    setSeatMode(seatId, mode) {
      update(state => {
        const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
        const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        if (idx >= 0 && idx < currentSeats.length) {
          currentSeats[idx] = { ...currentSeats[idx], mode };
        }
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            [key]: { ...(state.staging[key] || {}), mode },
            seats: currentSeats,
          }),
        };
      });
    },

    setLayout(layout) {
      update(state => ({
        ...state,
        staging: {
          ...state.staging,
          layout,
        },
      }));
    },

    assignGamepad(seatId, devNode) {
      update(state => {
        const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
        const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        if (idx >= 0 && idx < currentSeats.length) {
          currentSeats[idx] = { ...currentSeats[idx], gamepad_node: devNode };
        }
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            [key]: { ...(state.staging[key] || {}), gamepad_node: devNode },
            seats: currentSeats,
          }),
        };
      });
    },

    assignKeyboard(seatId, devNode) {
      update(state => {
        const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
        const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        if (idx >= 0 && idx < currentSeats.length) {
          currentSeats[idx] = { ...currentSeats[idx], keyboard_node: devNode };
        }
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            [key]: { ...(state.staging[key] || {}), keyboard_node: devNode },
            seats: currentSeats,
          }),
        };
      });
    },

    assignMouse(seatId, devNode) {
      update(state => {
        const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
        const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
        const currentSeats = [...(state.staging.seats || [state.staging.seat1, state.staging.seat2])];
        if (idx >= 0 && idx < currentSeats.length) {
          currentSeats[idx] = { ...currentSeats[idx], mouse_node: devNode };
        }
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            [key]: { ...(state.staging[key] || {}), mouse_node: devNode },
            seats: currentSeats,
          }),
        };
      });
    },

    updateStaging(partial) {
      update(state => {
        let stagingSeats = state.staging.seats ? state.staging.seats.map(s => ({ ...s })) : [];
        if (partial.seats) {
          stagingSeats = partial.seats.map(s => ({ ...s }));
        }
        if (partial.seat1 && stagingSeats.length > 0) {
          stagingSeats[0] = { ...stagingSeats[0], ...partial.seat1 };
        }
        if (partial.seat2 && stagingSeats.length > 1) {
          stagingSeats[1] = { ...stagingSeats[1], ...partial.seat2 };
        }
        return {
          ...state,
          staging: syncStaging({
            ...state.staging,
            ...partial,
            seats: stagingSeats,
          }),
        };
      });
    },

    async launchSession(game, userCfg = {}) {
      const state = get({ subscribe });
      const cfg = get(configStore);
      const mCfg = cfg?.multiseat || {};

      let effectiveUserCfg = userCfg;
      if (!effectiveUserCfg || !effectiveUserCfg.account_name) {
        try {
          effectiveUserCfg = await AppAPI.getGSEUserConfig();
        } catch {
          effectiveUserCfg = {
            account_name: 'GamuxPlayer',
            account_steamid: '76561197960287930',
            language: 'english',
            ip_country: 'US',
          };
        }
      }

      const allProfs = profilesStore.profiles || [];
      const activeProf = profilesStore.activeProfile;

      // Probe gamepads if not yet populated
      let pads = state.gamepads;
      if (!pads || pads.length === 0) {
        try {
          const probed = await AppAPI.getConnectedGamepads();
          if (Array.isArray(probed)) {
            pads = probed;
          }
        } catch {
          pads = [];
        }
      }

      const rawSeats = (state.staging?.seats && state.staging.seats.length > 0)
        ? state.staging.seats
        : [state.staging.seat1, state.staging.seat2];

      const activeRawSeats = rawSeats.filter(s => !s.disabled);
      const effectiveRawSeats = activeRawSeats.length > 0 ? activeRawSeats : [rawSeats[0]];

      const seatGamepadPrefs = effectiveRawSeats.map((s, idx) => {
        if (s.seat_id === 'p1') return s?.gamepad_node || mCfg.seat1_gamepad || 'auto';
        if (s.seat_id === 'p2') return s?.gamepad_node || mCfg.seat2_gamepad || 'auto';
        return s?.gamepad_node || 'auto';
      });

      const resolvedPads = resolveMultiGamepadAssignments(pads, seatGamepadPrefs);

      const isExplicitDeviceNode = (node) => Boolean(node && node !== 'none' && node !== 'auto');

      const extractDevNode = (val) => {
        if (!val || val === 'auto' || val === 'none') return val || '';
        const match = String(val).match(/\((\/dev\/input\/[^)]+)\)/);
        if (match) return match[1];
        return String(val).trim();
      };

      const seatConfigs = effectiveRawSeats.map((s, idx) => {
        const seatId = s.seat_id || `p${idx + 1}`;
        let prof;
        if (s.seat_id === 'p1' || idx === 0) {
          prof = activeProf || { name: 'GamuxPlayer', steam_id: '76561197960287930', language: 'english', ip_country: 'US' };
        } else {
          const profId = s.profile_id || (s.seat_id === 'p2' ? (mCfg.player2_profile_id || 'p2') : s.seat_id);
          prof = allProfs.find(p => p.id === profId) || allProfs.find(p => !p.is_main) || {
            id: profId,
            name: `Player${idx + 1}`,
            steam_id: `7656119899887766${idx + 4}`,
            language: 'english',
            ip_country: 'US',
          };
        }

        const playerProfile = {
          account_name: prof.name || `Player${idx + 1}`,
          account_steamid: prof.steam_id || `7656119899887766${idx + 4}`,
          language: prof.language || 'english',
          ip_country: prof.ip_country || 'US',
        };

        const devPad = resolvedPads[idx] === 'none' ? '' : extractDevNode(resolvedPads[idx]);
        const rawKbd = s.keyboard_node || (s.seat_id === 'p1' ? (mCfg.seat1_keyboard || '') : (s.seat_id === 'p2' ? (mCfg.seat2_keyboard || '') : ''));
        const rawMouse = s.mouse_node || (s.seat_id === 'p1' ? (mCfg.seat1_mouse || '') : (s.seat_id === 'p2' ? (mCfg.seat2_mouse || '') : ''));
        const kbd = extractDevNode(rawKbd);
        const mouse = extractDevNode(rawMouse);

        const mode = s.mode || (s.seat_id === 'p1' ? (mCfg.default_seat1_mode || 'local') : (s.seat_id === 'p2' ? (mCfg.default_seat2_mode || 'local') : 'local'));
        const maxDisp = (state.displays && state.displays.length > 0) ? state.displays.length - 1 : 0;
        let displayIndex = s.display_index ?? (s.seat_id === 'p1' ? (mCfg.seat1_display_index ?? 0) : (s.seat_id === 'p2' ? (mCfg.seat2_display_index ?? 0) : 0));
        if (displayIndex > maxDisp || displayIndex < 0) {
          displayIndex = 0;
        }

        const baseSunshinePort = s.seat_id === 'p1'
          ? (mCfg.sunshine_port_p1 || state.staging?.sunshine_port_p1 || 47989)
          : (s.seat_id === 'p2'
            ? (mCfg.sunshine_port_p2 || state.staging?.sunshine_port_p2 || 48010)
            : 48010 + (idx - 1) * 10);

        return {
          seat_id: seatId,
          profile_id: prof.id || (s.seat_id === 'p1' ? 'main' : seatId),
          player: playerProfile,
          mode,
          display_index: displayIndex,
          screen_slot: s.screen_slot ?? idx,
          gamepad_node: devPad,
          keyboard_node: kbd,
          mouse_node: mouse,
          force_grab_cursor: s.force_grab_cursor ?? (s.seat_id === 'p1' ? (mCfg.seat1_force_grab_cursor ?? false) : false),
          grab_keyboard: s.grab_keyboard ?? (s.seat_id === 'p1' ? (mCfg.seat1_grab_keyboard ?? false) : false),
          exclusive_grab: s.exclusive_grab ?? false,
          compositor: s.compositor || (s.seat_id === 'p2' ? (mCfg.seat2_compositor || 'auto') : 'auto'),
          nest_cage: false,
          show_cursor: s.show_cursor ?? (s.seat_id === 'p1' ? (mCfg.seat1_show_cursor ?? true) : (mCfg.seat2_show_cursor ?? true)),
          physical_sink: s.physical_sink || '',
          wine_prefix: s.wine_prefix || (s.seat_id === 'p1' ? (cfg?.wine_prefix || '') : (s.seat_id === 'p2' ? (mCfg.player2_wine_prefix || '') : '')),
          sunshine_port: baseSunshinePort,
        };
      });

      const rawLayout = state.staging?.layout || mCfg.default_layout || 'dual_monitor';
      let layout = rawLayout;
      if (effectiveRawSeats.length === 2 && (layout === 'three_player' || layout === 'quad_grid')) {
        layout = (mCfg.default_layout === 'top_bottom' || state.staging?.layout === 'top_bottom') ? 'top_bottom' : 'side_by_side';
      } else if (effectiveRawSeats.length === 3 && layout === 'quad_grid') {
        layout = 'three_player';
      }
      const isNative = game.platform === 'linux';

      const sessionConfig = {
        game_dir: game.gameDir,
        name: game.name,
        app_id: game.appId || '',
        platform: game.platform || 'win64',
        exe_path: game.exePath || '',
        exe_args: game.exeArgs || '',
        runner: isNative ? 'linux' : (game.runner || cfg?.default_runner || 'umu'),
        layout,
        sunshine_port_p1: seatConfigs[0]?.sunshine_port || 47989,
        sunshine_port_p2: seatConfigs[1]?.sunshine_port || 48010,
        seats: seatConfigs,
        seat1: seatConfigs[0],
        seat2: seatConfigs[1] || seatConfigs[0],
      };

      await AppAPI.launchMultiseatSession(sessionConfig);

      const activeSeatsMap = {};
      seatConfigs.forEach((sc, idx) => {
        activeSeatsMap[sc.seat_id] = { mode: sc.mode, is_running: true };
        activeSeatsMap[`seat${idx + 1}`] = { mode: sc.mode, is_running: true };
      });

      update(s => ({
        ...s,
        activeSessions: {
          ...s.activeSessions,
          [game.gameDir]: {
            is_running: true,
            ...activeSeatsMap,
            seat1: { mode: seatConfigs[0]?.mode || 'local', is_running: true },
            seat2: { mode: seatConfigs[1]?.mode || 'local', is_running: true },
            seats: seatConfigs.map(sc => ({ seat_id: sc.seat_id, profile_id: sc.profile_id, mode: sc.mode, is_running: true })),
          },
        },
      }));
    },

    async stopSession(gameDir) {
      await AppAPI.stopMultiseatSession(gameDir);
      update(state => {
        const next = {};
        for (const [key, sess] of Object.entries(state.activeSessions)) {
          if (!isSamePath(key, gameDir)) {
            next[key] = sess;
          }
        }
        return { ...state, activeSessions: next };
      });
    },

    async stopSeat(gameDir, seatId) {
      await AppAPI.stopMultiseatSeat(gameDir, seatId);
      update(state => {
        let matchedKey = null;
        for (const key of Object.keys(state.activeSessions)) {
          if (isSamePath(key, gameDir)) {
            matchedKey = key;
            break;
          }
        }
        if (!matchedKey) return state;
        const sess = state.activeSessions[matchedKey];
        const seatKey = (seatId === 'p1' || seatId === 'seat1') ? 'seat1' : ((seatId === 'p2' || seatId === 'seat2') ? 'seat2' : seatId);
        const normId = seatId.startsWith('seat') ? `p${seatId.replace('seat', '')}` : seatId;

        const nextSess = { ...sess };
        if (nextSess[seatKey]) nextSess[seatKey] = { ...nextSess[seatKey], is_running: false };
        if (nextSess[normId]) nextSess[normId] = { ...nextSess[normId], is_running: false };
        if (Array.isArray(nextSess.seats)) {
          nextSess.seats = nextSess.seats.map(s => s.seat_id === normId ? { ...s, is_running: false } : s);
        }

        const anyRunning = (nextSess.seats && nextSess.seats.length > 0)
          ? nextSess.seats.some(s => s.is_running)
          : Boolean(nextSess.seat1?.is_running || nextSess.seat2?.is_running);

        nextSess.is_running = anyRunning;

        return {
          ...state,
          activeSessions: {
            ...state.activeSessions,
            [matchedKey]: nextSess,
          },
        };
      });
    },

    setSessionInactive(gameDir) {
      update(state => {
        let changed = false;
        const next = {};
        for (const [key, sess] of Object.entries(state.activeSessions)) {
          if (isSamePath(key, gameDir)) {
            changed = true;
          } else {
            next[key] = sess;
          }
        }
        if (!changed) return state;
        return { ...state, activeSessions: next };
      });
    },

    async refreshStatus(gameDir) {
      try {
        const st = await AppAPI.getMultiseatSessionStatus(gameDir);
        if (st) {
          update(state => {
            const next = { ...state.activeSessions };
            for (const key of Object.keys(next)) {
              if (isSamePath(key, gameDir) && key !== gameDir) {
                delete next[key];
              }
            }
            next[gameDir] = st;
            return {
              ...state,
              activeSessions: next,
            };
          });
        }
      } catch (err) {
        console.warn('Failed to refresh multiseat status:', err);
      }
    },
  };
}

export const multiseatStore = createMultiseatStore();

if (typeof configStore?.subscribe === 'function') {
  configStore.subscribe(cfg => {
    if (cfg?.multiseat) {
      multiseatStore.syncFromConfig(cfg);
    }
  });
}
