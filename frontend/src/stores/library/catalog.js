import { writable } from 'svelte/store';
import { normalizeGame, isSamePath } from '../../utils/game.js';
import { selectedGame } from './selection.js';

export const games = writable([]);
export const isScanning = writable(false);
export const runningGames = writable(new Set());

export function setGameRunning(gameDir, isRunning, playtimeSeconds, lastPlayed) {
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
