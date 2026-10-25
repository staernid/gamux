import { AppAPI } from '../../api/app.js';
import { configStore } from '../config.js';
import { profilesStore } from '../profiles.js';
import { isSamePath } from '../../utils/game.js';
import { resolveMultiGamepadAssignments } from './assignments.js';

export class SessionState {
  activeSessions = $state({});

  async launchSession(game, userCfg = {}, devicesState = null, stagingState = null) {
    const cfg = configStore.current;
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
    let pads = devicesState?.gamepads || [];
    if (!pads || pads.length === 0) {
      try {
        const probed = await AppAPI.getConnectedGamepads();
        if (Array.isArray(probed)) {
          pads = probed;
          if (devicesState) devicesState.gamepads = probed;
        }
      } catch {
        pads = [];
      }
    }

    const staging = stagingState?.staging || {};
    const rawSeats = (staging.seats && staging.seats.length > 0)
      ? staging.seats
      : [staging.seat1, staging.seat2].filter(Boolean);

    const activeRawSeats = rawSeats.filter(s => !s.disabled);
    const effectiveRawSeats = activeRawSeats.length > 0 ? activeRawSeats : [rawSeats[0]];

    const seatGamepadPrefs = effectiveRawSeats.map((s) => {
      if (s.seat_id === 'p1') return s?.gamepad_node || mCfg.seat1_gamepad || 'auto';
      if (s.seat_id === 'p2') return s?.gamepad_node || mCfg.seat2_gamepad || 'auto';
      return s?.gamepad_node || 'auto';
    });

    const resolvedPads = resolveMultiGamepadAssignments(pads, seatGamepadPrefs);

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
        prof = activeProf || { name: 'GamuxPlayer', steamId: '76561197960287930', language: 'english', ipCountry: 'US' };
      } else {
        const profId = s.profile_id || (s.seat_id === 'p2' ? (mCfg.player2_profile_id || 'p2') : s.seat_id);
        prof = allProfs.find(p => p.id === profId) || allProfs.find(p => !p.isMain) || {
          id: profId,
          name: `Player${idx + 1}`,
          steamId: `7656119899887766${idx + 4}`,
          language: 'english',
          ipCountry: 'US',
        };
      }

      const playerProfile = {
        account_name: prof.name || `Player${idx + 1}`,
        account_steamid: prof.steamId || `7656119899887766${idx + 4}`,
        language: prof.language || 'english',
        ip_country: prof.ipCountry || 'US',
      };

      const devPad = resolvedPads[idx] === 'none' ? '' : extractDevNode(resolvedPads[idx]);
      const rawKbd = s.keyboard_node || (s.seat_id === 'p1' ? (mCfg.seat1_keyboard || '') : (s.seat_id === 'p2' ? (mCfg.seat2_keyboard || '') : ''));
      const rawMouse = s.mouse_node || (s.seat_id === 'p1' ? (mCfg.seat1_mouse || '') : (s.seat_id === 'p2' ? (mCfg.seat2_mouse || '') : ''));
      const kbd = extractDevNode(rawKbd);
      const mouse = extractDevNode(rawMouse);

      const mode = s.mode || (s.seat_id === 'p1' ? (mCfg.default_seat1_mode || 'local') : (s.seat_id === 'p2' ? (mCfg.default_seat2_mode || 'local') : 'local'));
      const maxDisp = (devicesState?.displays && devicesState.displays.length > 0) ? devicesState.displays.length - 1 : 0;
      let displayIndex = s.display_index ?? (s.seat_id === 'p1' ? (mCfg.seat1_display_index ?? 0) : (s.seat_id === 'p2' ? (mCfg.seat2_display_index ?? 0) : 0));
      if (displayIndex > maxDisp || displayIndex < 0) {
        displayIndex = 0;
      }

      const baseSunshinePort = s.seat_id === 'p1'
        ? (mCfg.sunshine_port_p1 || staging.sunshine_port_p1 || 47989)
        : (s.seat_id === 'p2'
          ? (mCfg.sunshine_port_p2 || staging.sunshine_port_p2 || 48010)
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

    const rawLayout = staging.layout || mCfg.default_layout || 'dual_monitor';
    let layout = rawLayout;
    if (effectiveRawSeats.length === 2 && (layout === 'three_player' || layout === 'quad_grid')) {
      layout = (mCfg.default_layout === 'top_bottom' || staging.layout === 'top_bottom') ? 'top_bottom' : 'side_by_side';
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

    this.activeSessions = {
      ...this.activeSessions,
      [game.gameDir]: {
        is_running: true,
        ...activeSeatsMap,
        seat1: { mode: seatConfigs[0]?.mode || 'local', is_running: true },
        seat2: { mode: seatConfigs[1]?.mode || 'local', is_running: true },
        seats: seatConfigs.map(sc => ({ seat_id: sc.seat_id, profile_id: sc.profile_id, mode: sc.mode, is_running: true })),
      },
    };
  }

  async stopSession(gameDir) {
    await AppAPI.stopMultiseatSession(gameDir);
    const next = {};
    for (const [key, sess] of Object.entries(this.activeSessions)) {
      if (!isSamePath(key, gameDir)) {
        next[key] = sess;
      }
    }
    this.activeSessions = next;
  }

  async stopSeat(gameDir, seatId) {
    await AppAPI.stopMultiseatSeat(gameDir, seatId);
    let matchedKey = null;
    for (const key of Object.keys(this.activeSessions)) {
      if (isSamePath(key, gameDir)) {
        matchedKey = key;
        break;
      }
    }
    if (!matchedKey) return;
    const sess = this.activeSessions[matchedKey];
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

    this.activeSessions = {
      ...this.activeSessions,
      [matchedKey]: nextSess,
    };
  }

  setSessionInactive(gameDir) {
    let changed = false;
    const next = {};
    for (const [key, sess] of Object.entries(this.activeSessions)) {
      if (isSamePath(key, gameDir)) {
        changed = true;
      } else {
        next[key] = sess;
      }
    }
    if (!changed) return;
    this.activeSessions = next;
  }

  async refreshStatus(gameDir) {
    try {
      const st = await AppAPI.getMultiseatSessionStatus(gameDir);
      if (st) {
        const next = { ...this.activeSessions };
        for (const key of Object.keys(next)) {
          if (isSamePath(key, gameDir) && key !== gameDir) {
            delete next[key];
          }
        }
        next[gameDir] = st;
        this.activeSessions = next;
      }
    } catch (err) {
      console.warn('Failed to refresh multiseat status:', err);
    }
  }
}
