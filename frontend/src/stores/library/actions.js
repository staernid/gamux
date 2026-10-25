import { get } from 'svelte/store';
import { AppAPI } from '../../api/app.js';
import { normalizeGame, isSamePath } from '../../utils/game.js';
import { notificationStore } from '../notifications.js';
import { libraryPaths, libraryPath, setOnPathsChangedHook } from './paths.js';
import { games, isScanning, updateGameInStore, removeGameFromStore } from './catalog.js';
import { selectedGame, activeView } from './selection.js';

export async function scanLibrary(pathOverride = '') {
  let targetPaths = [];
  if (pathOverride) {
    targetPaths = [pathOverride];
  } else {
    targetPaths = get(libraryPaths) || [];
    if (targetPaths.length === 0) {
      const single = get(libraryPath);
      if (single) targetPaths = [single];
    }
  }

  if (targetPaths.length === 0) {
    games.set([]);
    return;
  }

  isScanning.set(true);
  try {
    const results = await AppAPI.inspectLibraries(targetPaths);
    const normalized = (results || []).map(normalizeGame).filter(Boolean);
    games.set(normalized);
    selectedGame.update(current => {
      if (current && normalized.some(g => isSamePath(g.gameDir, current.gameDir))) {
        return normalized.find(g => isSamePath(g.gameDir, current.gameDir));
      }
      return normalized.length > 0 ? normalized[0] : null;
    });
  } catch (err) {
    notificationStore.error('Scan Failed', err);
  } finally {
    isScanning.set(false);
  }
}

// Automatically trigger scanLibrary when library paths change
setOnPathsChangedHook(() => {
  scanLibrary();
});

export async function importGame() {
  const currentLib = get(libraryPath) || '';

  const selected = await AppAPI.selectDirectory('Select Game Folder to Import', currentLib);
  if (!selected) return;

  try {
    const raw = await AppAPI.inspectGame(selected, 5);
    const game = updateGameInStore(raw);
    if (game) {
      selectedGame.set(game);
      activeView.set('library');
      notificationStore.success('Game Imported', `Loaded ${game.name}`);
    }
  } catch (err) {
    notificationStore.error('Import Failed', err);
  }
}

export async function refreshGame(gameDir) {
  if (!gameDir) return null;
  const raw = await AppAPI.inspectGame(gameDir, 0);
  return updateGameInStore(raw);
}

export async function saveGameProfile(gameDir, profile) {
  if (!gameDir) return null;
  const raw = await AppAPI.saveLaunchProfile(gameDir, profile);
  return updateGameInStore(raw);
}

export async function resetGameProfile(gameDir) {
  if (!gameDir) return null;
  const raw = await AppAPI.resetLaunchProfile(gameDir);
  return updateGameInStore(raw);
}

export async function rollbackGame(gameDir, dryRun = false) {
  if (!gameDir) return null;
  try {
    await AppAPI.submitRollback(gameDir, dryRun);
    notificationStore.items.add({
      type: 'info',
      title: 'Rollback Enqueued',
      message: 'Reverting patches and configs...'
    });
  } catch (err) {
    notificationStore.error('Rollback Failed', err);
  }
}

export async function queueApplyGame(game, customOpts = null) {
  if (!game) return;

  const gameDir = typeof game === 'string' ? game : (game.gameDir || game.GameDir);
  const gameName = typeof game === 'string' ? '' : (game.name || game.Name || '');
  const gameMode = typeof game === 'object' ? (game.operatingMode || game.operating_mode || game.OperatingMode || 'loader') : 'loader';

  if (!gameDir) return;

  const defaultOpts = {
    path: gameDir,
    operating_mode: gameMode,
    normalize_dir: true,
    no_steamless: false,
    fetch_achievements: true,
  };

  const opts = customOpts ? { ...defaultOpts, ...customOpts, path: gameDir } : defaultOpts;

  try {
    await AppAPI.submitApply(opts);
    notificationStore.items.add({
      type: 'info',
      title: 'Task Enqueued',
      message: `Configuring ${gameName || 'game'}...`,
      gameName: gameName
    });
  } catch (err) {
    notificationStore.error('Apply Failed', err);
  }
}

export function batchApplyCleanGames() {
  const allGames = get(games) || [];
  const cleanGames = allGames.filter(g => !g.isPatched);
  if (cleanGames.length === 0) return;

  cleanGames.forEach(g => {
    queueApplyGame(g);
  });
}

export async function trashGame(gameDir) {
  if (!gameDir) return false;
  try {
    await AppAPI.trashGame(gameDir);
    removeGameFromStore(gameDir);
    return true;
  } catch (err) {
    notificationStore.error('Trash Failed', err);
    throw err;
  }
}
