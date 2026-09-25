import { writable } from 'svelte/store';
import { normalizeGame, isSamePath } from '../../utils/game.js';
import { selectedGame } from './selection.js';

export const games = writable([]);
export const isScanning = writable(false);
export const runningGames = writable(new Set());
export const runningInstances = writable([]);

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

export function setRunningInstances(instances) {
  const list = Array.isArray(instances) ? instances : [];
  runningInstances.set(list);

  runningGames.set(new Set(list.map(i => i.gameDir || i.game_dir).filter(Boolean)));
}

export function setGameRunning(gameDir, isRunning, playtimeSeconds, lastPlayed, extra = {}) {
  if (!gameDir) return;
  const profileId = extra.profileId || extra.profile_id || 'main';
  const seatId = extra.seatId || extra.seat_id || '';
  const appId = extra.appId || extra.app_id || '';
  const gameName = extra.gameName || extra.game_name || '';
  const instanceId = extra.instanceId || extra.instance_id || (seatId ? `${appId || gameDir}:${profileId}:${seatId}` : `${appId || gameDir}:${profileId}:`);

  let nextInstances = [];
  runningInstances.update(current => {
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
    nextInstances = filtered;
    return filtered;
  });

  runningGames.update(set => {
    const next = new Set();
    for (const p of set) {
      if (!isSamePath(p, gameDir)) {
        next.add(p);
      }
    }
    const hasRemaining = nextInstances.some(i => isSamePath(i.gameDir || i.game_dir, gameDir));
    if (isRunning || hasRemaining) {
      next.add(gameDir);
    }
    return next;
  });

  if (!isRunning && (playtimeSeconds > 0 || lastPlayed > 0)) {
    games.update(list => {
      const idx = list.findIndex(g => isSamePath(g.gameDir, gameDir));
      if (idx >= 0) {
        const copy = [...list];
        copy[idx] = {
          ...copy[idx],
          playtimeSeconds: playtimeSeconds || copy[idx].playtimeSeconds,
          lastPlayed: lastPlayed || copy[idx].lastPlayed,
        };
        return copy;
      }
      return list;
    });

    selectedGame.update(curr => {
      if (curr && isSamePath(curr.gameDir, gameDir)) {
        return {
          ...curr,
          playtimeSeconds: playtimeSeconds || curr.playtimeSeconds,
          lastPlayed: lastPlayed || curr.lastPlayed,
        };
      }
      return curr;
    });
  }
}

export function renameGameInStore(oldDir, rawOrNormalizedGame) {
  if (!rawOrNormalizedGame) return null;
  const updated = normalizeGame(rawOrNormalizedGame);
  if (!updated) return null;

  games.update(list => {
    const idx = list.findIndex(g => (oldDir && isSamePath(g.gameDir, oldDir)) || isSamePath(g.gameDir, updated.gameDir));
    if (idx >= 0) {
      const copy = [...list];
      copy[idx] = updated;
      return copy;
    }
    return [updated, ...list];
  });

  selectedGame.update(current => {
    if (current && ((oldDir && isSamePath(current.gameDir, oldDir)) || isSamePath(current.gameDir, updated.gameDir))) {
      return updated;
    }
    return current;
  });

  return updated;
}

export function updateGameInStore(rawOrNormalizedGame) {
  return renameGameInStore(null, rawOrNormalizedGame);
}
