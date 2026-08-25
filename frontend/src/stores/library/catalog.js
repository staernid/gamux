import { writable } from 'svelte/store';
import { normalizeGame, isSamePath } from '../../utils/game.js';
import { selectedGame } from './selection.js';

export const games = writable([]);
export const isScanning = writable(false);
export const runningGames = writable(new Set());

export function setGameRunning(gameDir, isRunning) {
  if (!gameDir) return;
  runningGames.update(set => {
    const next = new Set(set);
    if (isRunning) {
      next.add(gameDir);
    } else {
      next.delete(gameDir);
    }
    return next;
  });
}

export function renameGameInStore(oldDir, rawOrNormalizedGame) {
  if (!rawOrNormalizedGame) return null;
  const updated = normalizeGame(rawOrNormalizedGame);
  if (!updated) return null;

  games.update(list => {
    const idx = list.findIndex(g => isSamePath(g.gameDir, oldDir) || isSamePath(g.gameDir, updated.gameDir));
    if (idx >= 0) {
      const copy = [...list];
      copy[idx] = updated;
      return copy;
    }
    return [updated, ...list];
  });

  selectedGame.update(current => {
    if (current && (isSamePath(current.gameDir, oldDir) || isSamePath(current.gameDir, updated.gameDir))) {
      return updated;
    }
    return current;
  });

  return updated;
}

export function updateGameInStore(rawOrNormalizedGame) {
  if (!rawOrNormalizedGame) return null;
  const updated = normalizeGame(rawOrNormalizedGame);
  if (!updated) return null;

  games.update(list => {
    const idx = list.findIndex(g => isSamePath(g.gameDir, updated.gameDir));
    if (idx >= 0) {
      const copy = [...list];
      copy[idx] = updated;
      return copy;
    }
    return [updated, ...list];
  });

  selectedGame.update(current => {
    if (current && isSamePath(current.gameDir, updated.gameDir)) {
      return updated;
    }
    return current;
  });

  return updated;
}
