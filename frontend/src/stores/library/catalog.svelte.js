import { normalizeGame, isSamePath } from '../../utils/game.js';
import { selectedGame } from './selection.js';

export function isGamePathRunning(runningSetOrArray, path) {
  if (!path || !runningSetOrArray) return false;
  if (runningSetOrArray instanceof Set) {
    for (const p of runningSetOrArray) {
      if (isSamePath(p, path)) return true;
    }
    return false;
  }
  if (Array.isArray(runningSetOrArray)) {
    return runningSetOrArray.some(p => isSamePath(p, path));
  }
  return false;
}

export class CatalogState {
  games = $state([]);
  isScanning = $state(false);
  runningGames = $state(new Set());
  runningInstances = $state([]);

  #gameSubscribers = new Set();
  #scanSubscribers = new Set();
  #runningGamesSubscribers = new Set();
  #runningInstancesSubscribers = new Set();

  isGamePathRunning(path) {
    return isGamePathRunning(this.runningGames, path);
  }

  setRunningInstances(instances) {
    const list = Array.isArray(instances) ? instances : [];
    this.runningInstances = list;
    this.runningGames = new Set(list.map(i => i.gameDir || i.game_dir).filter(Boolean));
    this.#notifyInstances();
    this.#notifyRunning();
  }

  setGameRunning(gameDir, isRunning, playtimeSeconds, lastPlayed, extra = {}) {
    if (!gameDir) return;
    const profileId = extra.profileId || extra.profile_id || 'main';
    const seatId = extra.seatId || extra.seat_id || '';
    const appId = extra.appId || extra.app_id || '';
    const gameName = extra.gameName || extra.game_name || '';
    const instanceId = extra.instanceId || extra.instance_id || (seatId ? `${appId || gameDir}:${profileId}:${seatId}` : `${appId || gameDir}:${profileId}:`);

    const current = this.runningInstances;
    const filtered = current.filter(i => {
      const samePath = isSamePath(i.gameDir || i.game_dir, gameDir);
      if (!samePath) return true;

      const iId = i.instanceId || i.instance_id;
      if (iId && instanceId && iId === instanceId) return false;

      const iSeat = i.seatId || i.seat_id || '';

      // If updating a seat instance, remove prior entry for the same seat
      if (seatId && iSeat === seatId) return false;

      // If updating a multiseat seat instance, purge any generic orphan non-seat entry
      if (seatId && !iSeat) return false;

      // If updating a solo instance, remove any prior solo instance for this game
      if (!seatId && !iSeat) return false;

      // If stopping the game completely (and no seat specified), remove all instances for this game
      if (!isRunning && !seatId) return false;

      return true;
    });

    if (isRunning) {
      filtered.push({
        instance_id: instanceId,
        instanceId: instanceId,
        game_dir: gameDir,
        gameDir: gameDir,
        game_name: gameName,
        gameName: gameName,
        app_id: appId,
        appId: appId,
        profile_id: profileId,
        profileId: profileId,
        seat_id: seatId,
        seatId: seatId,
        pid: extra.pid || 0,
        is_multiseat: Boolean(seatId),
        isMultiseat: Boolean(seatId),
        start_time: Math.floor(Date.now() / 1000)
      });
    }
    this.runningInstances = filtered;
    this.#notifyInstances();

    const nextSet = new Set();
    for (const p of this.runningGames) {
      if (!isSamePath(p, gameDir)) {
        nextSet.add(p);
      }
    }
    const hasRemaining = filtered.some(i => isSamePath(i.gameDir || i.game_dir, gameDir));
    if (isRunning || hasRemaining) {
      nextSet.add(gameDir);
    }

    let identical = (this.runningGames.size === nextSet.size);
    if (identical) {
      for (const p of nextSet) {
        if (!this.runningGames.has(p)) {
          identical = false;
          break;
        }
      }
    }
    if (!identical) {
      this.runningGames = nextSet;
      this.#notifyRunning();
    }

    if (!isRunning && (playtimeSeconds > 0 || lastPlayed > 0)) {
      const delta = {};
      if (playtimeSeconds > 0) delta.playtimeSeconds = playtimeSeconds;
      if (lastPlayed > 0) delta.lastPlayed = lastPlayed;
      this.patchGameInStore(gameDir, delta);
    }
  }

  patchGameInStore(gameDir, delta) {
    if (!gameDir || !delta || typeof delta !== 'object') return null;

    const normalizedDelta = { ...delta };
    if (delta.playtime_seconds !== undefined && delta.playtimeSeconds === undefined) {
      normalizedDelta.playtimeSeconds = delta.playtime_seconds;
    }
    if (delta.last_played !== undefined && delta.lastPlayed === undefined) {
      normalizedDelta.lastPlayed = delta.last_played;
    }
    if (delta.is_patched !== undefined && delta.isPatched === undefined) {
      normalizedDelta.isPatched = delta.is_patched;
    }
    if (delta.game_dir !== undefined && delta.gameDir === undefined) {
      normalizedDelta.gameDir = delta.game_dir;
    }
    if (delta.app_id !== undefined && delta.appId === undefined) {
      normalizedDelta.appId = String(delta.app_id);
    }

    const idx = this.games.findIndex(g => isSamePath(g.gameDir, gameDir));
    if (idx < 0) return null;

    const current = this.games[idx];
    let hasChanges = false;
    for (const [key, value] of Object.entries(normalizedDelta)) {
      if (value !== undefined && current[key] !== value) {
        hasChanges = true;
        break;
      }
    }

    if (!hasChanges) {
      return current;
    }

    const merged = { ...current };
    for (const [key, value] of Object.entries(normalizedDelta)) {
      if (value !== undefined) {
        merged[key] = value;
      }
    }

    const copy = [...this.games];
    copy[idx] = merged;
    this.games = copy;
    this.#notifyGames();

    selectedGame.update(curr => {
      if (curr && isSamePath(curr.gameDir, gameDir)) {
        let hasSelectedChanges = false;
        for (const [key, value] of Object.entries(normalizedDelta)) {
          if (value !== undefined && curr[key] !== value) {
            hasSelectedChanges = true;
            break;
          }
        }
        if (!hasSelectedChanges) return curr;
        return { ...curr, ...normalizedDelta };
      }
      return curr;
    });

    return merged;
  }

  renameGameInStore(oldDir, rawOrNormalizedGame) {
    if (!rawOrNormalizedGame) return null;
    const updated = normalizeGame(rawOrNormalizedGame);
    if (!updated) return null;

    const idx = this.games.findIndex(g => (oldDir && isSamePath(g.gameDir, oldDir)) || isSamePath(g.gameDir, updated.gameDir));
    const copy = [...this.games];
    if (idx >= 0) {
      copy[idx] = updated;
    } else {
      copy.unshift(updated);
    }
    this.games = copy;
    this.#notifyGames();

    selectedGame.update(current => {
      if (current && ((oldDir && isSamePath(current.gameDir, oldDir)) || isSamePath(current.gameDir, updated.gameDir))) {
        return updated;
      }
      return current;
    });

    return updated;
  }

  updateGameInStore(rawOrNormalizedGame) {
    return this.renameGameInStore(null, rawOrNormalizedGame);
  }

  removeGameFromStore(gameDir) {
    if (!gameDir) return;
    this.games = this.games.filter(g => !isSamePath(g.gameDir, gameDir));
    this.#notifyGames();

    selectedGame.update(current => {
      if (current && isSamePath(current.gameDir, gameDir)) {
        return null;
      }
      return current;
    });
  }

  #notifyGames() {
    for (const fn of this.#gameSubscribers) {
      try { fn(this.games); } catch {}
    }
  }

  #notifyRunning() {
    for (const fn of this.#runningGamesSubscribers) {
      try { fn(this.runningGames); } catch {}
    }
  }

  #notifyInstances() {
    for (const fn of this.#runningInstancesSubscribers) {
      try { fn(this.runningInstances); } catch {}
    }
  }

  #notifyScan() {
    for (const fn of this.#scanSubscribers) {
      try { fn(this.isScanning); } catch {}
    }
  }

  // Bridges for Svelte store contracts
  get _gameStoreBridge() {
    return {
      subscribe: (fn) => {
        this.#gameSubscribers.add(fn);
        fn(this.games);
        return () => this.#gameSubscribers.delete(fn);
      },
      set: (val) => {
        this.games = Array.isArray(val) ? val : [];
        this.#notifyGames();
      },
      update: (fn) => {
        this.games = fn(this.games);
        this.#notifyGames();
      }
    };
  }

  get _scanStoreBridge() {
    return {
      subscribe: (fn) => {
        this.#scanSubscribers.add(fn);
        fn(this.isScanning);
        return () => this.#scanSubscribers.delete(fn);
      },
      set: (val) => {
        this.isScanning = Boolean(val);
        this.#notifyScan();
      },
      update: (fn) => {
        this.isScanning = Boolean(fn(this.isScanning));
        this.#notifyScan();
      }
    };
  }

  get _runningGamesBridge() {
    return {
      subscribe: (fn) => {
        this.#runningGamesSubscribers.add(fn);
        fn(this.runningGames);
        return () => this.#runningGamesSubscribers.delete(fn);
      },
      set: (val) => {
        this.runningGames = val instanceof Set ? val : new Set(val);
        this.#notifyRunning();
      },
      update: (fn) => {
        this.runningGames = fn(this.runningGames);
        this.#notifyRunning();
      }
    };
  }

  get _runningInstancesBridge() {
    return {
      subscribe: (fn) => {
        this.#runningInstancesSubscribers.add(fn);
        fn(this.runningInstances);
        return () => this.#runningInstancesSubscribers.delete(fn);
      },
      set: (val) => {
        this.runningInstances = Array.isArray(val) ? val : [];
        this.#notifyInstances();
      },
      update: (fn) => {
        this.runningInstances = fn(this.runningInstances);
        this.#notifyInstances();
      }
    };
  }
}

export const catalogStore = new CatalogState();
export const catalogState = catalogStore;

// Store contract compatibility exports
export const games = catalogStore._gameStoreBridge;
export const isScanning = catalogStore._scanStoreBridge;
export const runningGames = catalogStore._runningGamesBridge;
export const runningInstances = catalogStore._runningInstancesBridge;

export function setRunningInstances(instances) {
  return catalogStore.setRunningInstances(instances);
}

export function setGameRunning(gameDir, isRunning, playtimeSeconds, lastPlayed, extra = {}) {
  return catalogStore.setGameRunning(gameDir, isRunning, playtimeSeconds, lastPlayed, extra);
}

export function patchGameInStore(gameDir, delta) {
  return catalogStore.patchGameInStore(gameDir, delta);
}

export function renameGameInStore(oldDir, rawOrNormalizedGame) {
  return catalogStore.renameGameInStore(oldDir, rawOrNormalizedGame);
}

export function updateGameInStore(rawOrNormalizedGame) {
  return catalogStore.updateGameInStore(rawOrNormalizedGame);
}

export function removeGameFromStore(gameDir) {
  return catalogStore.removeGameFromStore(gameDir);
}
