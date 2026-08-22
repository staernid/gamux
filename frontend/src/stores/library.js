import { writable, derived } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { normalizeGame } from '../utils/game.js';
import { notificationStore } from './notifications.js';
import { taskQueue } from './queue.js';

export const libraryPath = writable(localStorage.getItem('gamux_library_path') || '');
export const sortBy = writable(localStorage.getItem('gamux_sort_by') || 'name_asc');
export const activeFilter = writable('all');
export const searchQuery = writable('');
export const isScanning = writable(false);
export const games = writable([]);
export const selectedGame = writable(null);

// Active View State ('library' | 'downloader' | 'activity' | 'settings')
export const activeView = writable('library');
export const showInspector = writable(false);
export const pathPopover = writable(false);

libraryPath.subscribe(val => {
  if (val) localStorage.setItem('gamux_library_path', val);
});

sortBy.subscribe(val => {
  localStorage.setItem('gamux_sort_by', val);
});

export const filteredGames = derived(
  [games, activeFilter, searchQuery, sortBy],
  ([$games, $activeFilter, $searchQuery, $sortBy]) => {
    let list = [...$games];

    // Filter
    if ($activeFilter === 'patched') {
      list = list.filter(g => g.isPatched);
    } else if ($activeFilter === 'clean') {
      list = list.filter(g => !g.isPatched);
    } else if ($activeFilter === 'lutris') {
      list = list.filter(g => g.lutris_registered);
    } else if ($activeFilter === 'updates') {
      list = list.filter(g => g.hasUpdatesOrIssues);
    }

    // Search query
    const q = $searchQuery.toLowerCase().trim();
    if (q) {
      list = list.filter(g =>
        (g.Name && g.Name.toLowerCase().includes(q)) ||
        (g.AppID && g.AppID.includes(q)) ||
        (g.Platform && g.Platform.toLowerCase().includes(q)) ||
        (g.GameDir && g.GameDir.toLowerCase().includes(q))
      );
    }

    // Sort
    list.sort((a, b) => {
      if ($sortBy === 'name_asc') {
        return (a.Name || '').localeCompare(b.Name || '');
      } else if ($sortBy === 'name_desc') {
        return (b.Name || '').localeCompare(a.Name || '');
      } else if ($sortBy === 'size_desc') {
        return (b.DiskSizeBytes || 0) - (a.DiskSizeBytes || 0);
      } else if ($sortBy === 'size_asc') {
        return (a.DiskSizeBytes || 0) - (b.DiskSizeBytes || 0);
      } else if ($sortBy === 'appid') {
        return (parseInt(a.AppID, 10) || 0) - (parseInt(b.AppID, 10) || 0);
      } else if ($sortBy === 'status') {
        return (b.isPatched ? 1 : 0) - (a.isPatched ? 1 : 0);
      }
      return 0;
    });

    return list;
  }
);

export const stats = derived(games, $games => {
  let patchedCount = 0;
  let cleanCount = 0;
  let lutrisCount = 0;
  let updatesCount = 0;
  let totalBytes = 0;

  for (const g of $games) {
    totalBytes += g.DiskSizeBytes || 0;
    if (g.lutris_registered) lutrisCount++;
    if (g.hasUpdatesOrIssues) updatesCount++;
    if (g.isPatched) patchedCount++;
    else cleanCount++;
  }

  return {
    totalCount: $games.length,
    patchedCount,
    cleanCount,
    lutrisCount,
    updatesCount,
    totalBytes,
  };
});

export async function scanLibrary(pathOverride = '') {
  let targetPath = pathOverride;
  if (!targetPath) {
    libraryPath.subscribe(v => { targetPath = v; })();
  }

  if (!targetPath) {
    notificationStore.items.add({
      type: 'error',
      title: 'Library Path Required',
      message: 'Please set or browse to a game library folder.'
    });
    return;
  }

  isScanning.set(true);
  try {
    const results = await AppAPI.batchInspect(targetPath);
    const normalized = (results || []).map(normalizeGame).filter(Boolean);
    games.set(normalized);
    selectedGame.update(current => {
      if (current && normalized.some(g => g.GameDir === current.GameDir)) {
        return normalized.find(g => g.GameDir === current.GameDir);
      }
      return normalized.length > 0 ? normalized[0] : null;
    });
    notificationStore.items.add({
      type: 'success',
      title: 'Scan Complete',
      message: `Found ${normalized.length} games in ${targetPath}`
    });
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Scan Failed',
      message: err?.message || String(err)
    });
  } finally {
    isScanning.set(false);
  }
}

export async function importGame() {
  let currentLib = '';
  libraryPath.subscribe(v => { currentLib = v; })();

  const selected = await AppAPI.selectDirectory('Select Game Folder to Import', currentLib);
  if (!selected) return;

  try {
    const raw = await AppAPI.inspectGameEx(selected, 5);
    const game = normalizeGame(raw);
    if (game) {
      games.update(list => {
        const idx = list.findIndex(g => g.GameDir === game.GameDir);
        if (idx >= 0) {
          const updated = [...list];
          updated[idx] = game;
          return updated;
        }
        return [game, ...list];
      });

      selectedGame.set(game);
      activeView.set('library');
      notificationStore.items.add({
        type: 'success',
        title: 'Game Imported',
        message: `Loaded ${game.Name}`
      });
    }
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Import Failed',
      message: err?.message || String(err)
    });
  }
}

export function queueApplyGame(game, customOpts = null) {
  if (!game) return;

  const defaultOpts = {
    Path: game.GameDir,
    ApplyGBE: true,
    AddLutris: true,
    Portable: false,
    NormalizeDir: true,
    NoSteamless: false,
    FetchAchievements: true,
  };

  const opts = customOpts ? { ...defaultOpts, ...customOpts, Path: game.GameDir } : defaultOpts;

  taskQueue.enqueue({
    type: 'apply',
    title: `Apply: ${game.Name}`,
    gameName: game.Name,
    gameDir: game.GameDir,
    run: async () => {
      await AppAPI.applyGame(opts);
      return `Configured Goldberg Emulator & Lutris registration for ${game.Name}`;
    },
    onSuccess: async () => {
      const raw = await AppAPI.inspectGameEx(game.GameDir, 0);
      const updated = normalizeGame(raw);
      if (updated) {
        games.update(list => {
          const idx = list.findIndex(g => g.GameDir === game.GameDir);
          if (idx >= 0) {
            const copy = [...list];
            copy[idx] = updated;
            return copy;
          }
          return list;
        });
        if (selectedGame) {
          selectedGame.update(sg => (sg?.GameDir === updated.GameDir ? updated : sg));
        }
      }
    }
  });
}

export function batchApplyCleanGames() {
  let allGames = [];
  games.subscribe(v => { allGames = v; })();

  const cleanGames = allGames.filter(g => !g.isPatched);
  if (cleanGames.length === 0) return;

  cleanGames.forEach(g => {
    queueApplyGame(g);
  });
}
