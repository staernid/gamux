import { writable, get } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { configStore } from './config.js';

export function resolveGamepadAssignments(gamepads = [], seat1Pref = 'auto', seat2Pref = 'auto') {
  const pads = Array.isArray(gamepads) ? gamepads : [];
  let p1Dev = 'none';
  let p2Dev = 'none';

  const s1 = seat1Pref || 'auto';
  const s2 = seat2Pref || 'auto';

  if (s1 === 'auto' && s2 === 'auto') {
    if (pads.length >= 2) {
      p1Dev = pads[0].dev_node;
      p2Dev = pads[1].dev_node;
    } else if (pads.length === 1) {
      // Single gamepad: dedicate controller to Player 2; Player 1 uses host keyboard/mouse
      p1Dev = 'none';
      p2Dev = pads[0].dev_node;
    } else {
      p1Dev = 'none';
      p2Dev = 'none';
    }
  } else {
    // Explicit preferences
    if (s1 !== 'auto') {
      p1Dev = s1;
    }
    if (s2 !== 'auto') {
      p2Dev = s2;
    }

    // Resolve auto for Player 1
    if (s1 === 'auto') {
      const unused = pads.find(p => p.dev_node !== p2Dev);
      if (unused && (p2Dev !== 'none' || pads.length > 1)) {
        p1Dev = unused.dev_node;
      } else {
        p1Dev = 'none';
      }
    }

    // Resolve auto for Player 2
    if (s2 === 'auto') {
      const unused = pads.find(p => p.dev_node !== p1Dev);
      p2Dev = unused ? unused.dev_node : 'none';
    }
  }

  return { p1Dev, p2Dev };
}

function createMultiseatStore() {
  const { subscribe, set, update } = writable({
    capabilities: {
      has_bwrap: true,
      has_pasta: true,
      has_gamescope: true,
      has_cage: false,
      has_sunshine: true,
      has_pipewire: true,
    },
    gamepads: [],
    keyboards: [],
    mice: [],
    isCoopEnabled: false,
    staging: {
      seat1: {
        mode: 'local', // 'local' | 'stream'
        display_index: 0,
        gamepad_node: '',
        keyboard_node: '',
        mouse_node: '',
        physical_sink: '',
      },
      seat2: {
        mode: 'local', // 'local' | 'stream'
        display_index: 1,
        gamepad_node: '',
        keyboard_node: '',
        mouse_node: '',
        force_grab_cursor: false,
        grab_keyboard: false,
        exclusive_grab: false,
        compositor: 'auto',
        nest_cage: false,
        show_cursor: true,
        physical_sink: '',
      },
      layout: 'dual_monitor', // 'dual_monitor' | 'side_by_side' | 'top_bottom' | 'windowed'
      enable_broadcast_relay: true,
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    },
    activeSessions: {}, // gameDir -> status
  });

  return {
    subscribe,

    async loadCapabilities() {
      try {
        const caps = await AppAPI.getMultiseatCapabilities();
        if (caps) {
          update(state => ({
            ...state,
            capabilities: {
              has_bwrap: Boolean(caps.has_bwrap ?? caps.HasBwrap ?? caps.hasBwrap),
              has_pasta: Boolean(caps.has_pasta ?? caps.HasPasta ?? caps.hasPasta),
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

    setSeatMode(seatId, mode) {
      update(state => ({
        ...state,
        staging: {
          ...state.staging,
          [seatId]: {
            ...state.staging[seatId],
            mode,
          },
        },
      }));
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
      update(state => ({
        ...state,
        staging: {
          ...state.staging,
          [seatId]: {
            ...state.staging[seatId],
            gamepad_node: devNode,
          },
        },
      }));
    },

    assignKeyboard(seatId, devNode) {
      update(state => ({
        ...state,
        staging: {
          ...state.staging,
          [seatId]: {
            ...state.staging[seatId],
            keyboard_node: devNode,
          },
        },
      }));
    },

    assignMouse(seatId, devNode) {
      update(state => ({
        ...state,
        staging: {
          ...state.staging,
          [seatId]: {
            ...state.staging[seatId],
            mouse_node: devNode,
          },
        },
      }));
    },

    updateStaging(partial) {
      update(state => ({
        ...state,
        staging: {
          ...state.staging,
          ...partial,
        },
      }));
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

      const player1Profile = {
        account_name: effectiveUserCfg?.account_name || 'GamuxPlayer',
        account_steamid: effectiveUserCfg?.account_steamid || '76561197960287930',
        language: effectiveUserCfg?.language || 'english',
        ip_country: effectiveUserCfg?.ip_country || 'US',
      };

      const player2Profile = {
        account_name: mCfg.player2_account_name || 'PlayerTwo',
        account_steamid: mCfg.player2_account_steamid || '76561198998877665',
        language: mCfg.player2_language || 'english',
        ip_country: mCfg.player2_ip_country || 'US',
      };

      // Check connected gamepads for intelligent assignment if not assigned explicitly
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

      // Resolve gamepad assignments respecting config preferences and staging overrides
      const seat1GamepadPref = state.staging?.seat1?.gamepad_node || mCfg.seat1_gamepad || 'auto';
      const seat2GamepadPref = state.staging?.seat2?.gamepad_node || mCfg.seat2_gamepad || 'auto';
      const { p1Dev, p2Dev } = resolveGamepadAssignments(pads, seat1GamepadPref, seat2GamepadPref);

      const p1Gamepad = p1Dev === 'none' ? '' : p1Dev;
      const p2Gamepad = p2Dev === 'none' ? '' : p2Dev;

      const layout = mCfg.default_layout || state.staging?.layout || 'dual_monitor';
      const seat1Mode = mCfg.default_seat1_mode || state.staging?.seat1?.mode || 'local';
      const seat2Mode = mCfg.default_seat2_mode || state.staging?.seat2?.mode || 'local';
      const seat1Display = mCfg.seat1_display_index ?? state.staging?.seat1?.display_index ?? 0;
      const seat2Display = mCfg.seat2_display_index ?? state.staging?.seat2?.display_index ?? 1;
      const enableRelay = mCfg.broadcast_relay_enabled ?? state.staging?.enable_broadcast_relay ?? true;
      const sunPortP1 = mCfg.sunshine_port_p1 || state.staging?.sunshine_port_p1 || 47989;
      const sunPortP2 = mCfg.sunshine_port_p2 || state.staging?.sunshine_port_p2 || 48010;

      const seat1Kbd = state.staging?.seat1?.keyboard_node || mCfg.seat1_keyboard || '';
      const seat2Kbd = state.staging?.seat2?.keyboard_node || mCfg.seat2_keyboard || '';
      const seat1Mouse = state.staging?.seat1?.mouse_node || mCfg.seat1_mouse || '';
      const seat2Mouse = state.staging?.seat2?.mouse_node || mCfg.seat2_mouse || '';
      const seat2ForceGrab = state.staging?.seat2?.force_grab_cursor ?? mCfg.seat2_force_grab_cursor ?? false;
      const seat2GrabKbd = state.staging?.seat2?.grab_keyboard ?? mCfg.seat2_grab_keyboard ?? false;
      const seat2ExclGrab = state.staging?.seat2?.exclusive_grab ?? false;
      const seat2Compositor = state.staging?.seat2?.compositor || mCfg.seat2_compositor || 'auto';
      const seat2NestCage = state.staging?.seat2?.nest_cage ?? mCfg.seat2_nest_cage ?? false;
      const seat2ShowCursor = state.staging?.seat2?.show_cursor ?? mCfg.seat2_show_cursor ?? true;

      const sessionConfig = {
        game_dir: game.gameDir,
        name: game.name,
        app_id: game.appId || '',
        exe_path: game.exePath || '',
        exe_args: game.exeArgs || '',
        runner: game.runner || cfg?.default_runner || 'umu',
        layout,
        enable_broadcast_relay: enableRelay,
        sunshine_port_p1: sunPortP1,
        sunshine_port_p2: sunPortP2,
        seat1: {
          seat_id: 'p1',
          player: player1Profile,
          mode: seat1Mode,
          display_index: seat1Display,
          gamepad_node: p1Gamepad,
          keyboard_node: seat1Kbd,
          mouse_node: seat1Mouse,
          physical_sink: state.staging?.seat1?.physical_sink || '',
          wine_prefix: state.staging?.seat1?.wine_prefix || cfg?.wine_prefix || '',
        },
        seat2: {
          seat_id: 'p2',
          player: player2Profile,
          mode: seat2Mode,
          display_index: seat2Display,
          gamepad_node: p2Gamepad,
          keyboard_node: seat2Kbd,
          mouse_node: seat2Mouse,
          force_grab_cursor: seat2ForceGrab,
          grab_keyboard: seat2GrabKbd,
          exclusive_grab: seat2ExclGrab,
          compositor: seat2Compositor,
          nest_cage: seat2NestCage,
          show_cursor: seat2ShowCursor,
          physical_sink: state.staging?.seat2?.physical_sink || '',
          wine_prefix: state.staging?.seat2?.wine_prefix || mCfg.player2_wine_prefix || '',
        },
      };

      await AppAPI.launchMultiseatSession(sessionConfig);

      update(s => ({
        ...s,
        activeSessions: {
          ...s.activeSessions,
          [game.gameDir]: {
            is_running: true,
            seat1: { mode: seat1Mode, is_running: true },
            seat2: { mode: seat2Mode, is_running: true },
          },
        },
      }));
    },

    async stopSession(gameDir) {
      await AppAPI.stopMultiseatSession(gameDir);
      update(state => {
        const next = { ...state.activeSessions };
        delete next[gameDir];
        return { ...state, activeSessions: next };
      });
    },

    async stopSeat(gameDir, seatId) {
      await AppAPI.stopMultiseatSeat(gameDir, seatId);
      update(state => {
        const sess = state.activeSessions[gameDir];
        if (!sess) return state;
        const nextSeat = { ...sess[seatId], is_running: false };
        const isStillRunning = (seatId === 'p1' ? sess.seat2?.is_running : sess.seat1?.is_running) ?? false;
        return {
          ...state,
          activeSessions: {
            ...state.activeSessions,
            [gameDir]: {
              ...sess,
              [seatId]: nextSeat,
              is_running: isStillRunning,
            },
          },
        };
      });
    },

    setSessionInactive(gameDir) {
      update(state => {
        if (!state.activeSessions[gameDir]) return state;
        const next = { ...state.activeSessions };
        delete next[gameDir];
        return { ...state, activeSessions: next };
      });
    },

    async refreshStatus(gameDir) {
      try {
        const st = await AppAPI.getMultiseatSessionStatus(gameDir);
        if (st) {
          update(state => ({
            ...state,
            activeSessions: {
              ...state.activeSessions,
              [gameDir]: st,
            },
          }));
        }
      } catch (err) {
        console.warn('Failed to refresh multiseat status:', err);
      }
    },
  };
}

export const multiseatStore = createMultiseatStore();
