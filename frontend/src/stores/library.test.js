import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import {
  libraryPaths,
  libraryPath,
  addLibraryPath,
  removeLibraryPath,
  games,
  selectedGame,
  runningGames,
  setGameRunning,
  isGamePathRunning,
  updateGameInStore,
  renameGameInStore,
  patchGameInStore,
  resolvePlaceholderTitle,
  activeFilter,
  searchQuery,
  sortBy,
  filteredGames,
  stats,
  cycleSortOrder,
  getSortLabel,
} from './library.js';

describe('Decoupled Library Stores', () => {
  beforeEach(() => {
    games.set([]);
    selectedGame.set(null);
    runningGames.set(new Set());
    activeFilter.set('all');
    searchQuery.set('');
    sortBy.set('name_asc');
  });

  it('manages library paths correctly', () => {
    libraryPaths.set(['/games/a']);
    addLibraryPath('/games/b');
    expect(get(libraryPaths)).toEqual(['/games/a', '/games/b']);

    // Duplicate path should not be added
    addLibraryPath('/games/b');
    expect(get(libraryPaths)).toEqual(['/games/a', '/games/b']);

    removeLibraryPath('/games/a');
    expect(get(libraryPaths)).toEqual(['/games/b']);
  });

  it('manages running games state', () => {
    setGameRunning('/games/cyberpunk', true);
    expect(get(runningGames).has('/games/cyberpunk')).toBe(true);

    setGameRunning('/games/cyberpunk', false);
    expect(get(runningGames).has('/games/cyberpunk')).toBe(false);
  });

  it('handles path normalization and isGamePathRunning correctly', () => {
    setGameRunning('/games/cyberpunk/', true);
    expect(isGamePathRunning(get(runningGames), '/games/cyberpunk')).toBe(true);
    expect(isGamePathRunning(get(runningGames), '/games/cyberpunk/')).toBe(true);
    expect(isGamePathRunning(['/games/cyberpunk'], '/games/cyberpunk/')).toBe(true);
    expect(isGamePathRunning(new Set(['/games/cyberpunk']), '/games/other')).toBe(false);

    // Stop using non-trailing slash version removes the entry cleanly
    setGameRunning('/games/cyberpunk', false);
    expect(isGamePathRunning(get(runningGames), '/games/cyberpunk')).toBe(false);
    expect(get(runningGames).size).toBe(0);
  });

  it('updates and renames games in the catalog store', () => {
    const rawGame = {
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      game_dir: '/games/cp2077',
      is_patched: true,
      disk_size_bytes: 70000000000,
    };

    updateGameInStore(rawGame);
    expect(get(games).length).toBe(1);
    expect(get(games)[0].name).toBe('Cyberpunk 2077');
    expect(get(games)[0].appId).toBe('1091500');

    // Renaming game folder
    renameGameInStore('/games/cp2077', {
      ...rawGame,
      game_dir: '/games/Cyberpunk 2077',
    });

    expect(get(games).length).toBe(1);
    expect(get(games)[0].gameDir).toBe('/games/Cyberpunk 2077');
  });

  it('filters, searches, and computes statistics in query store', () => {
    const gameA = {
      name: 'Alpha Game',
      app_id: '100',
      game_dir: '/games/alpha',
      is_patched: true,
      disk_size_bytes: 1000,
    };
    const gameB = {
      name: 'Beta Game',
      app_id: '200',
      game_dir: '/games/beta',
      is_patched: false,
      disk_size_bytes: 5000,
    };

    updateGameInStore(gameA);
    updateGameInStore(gameB);

    expect(get(filteredGames).length).toBe(2);

    // Filter by patched
    activeFilter.set('patched');
    expect(get(filteredGames).length).toBe(1);
    expect(get(filteredGames)[0].name).toBe('Alpha Game');

    // Filter by clean
    activeFilter.set('clean');
    expect(get(filteredGames).length).toBe(1);
    expect(get(filteredGames)[0].name).toBe('Beta Game');

    // Search query
    activeFilter.set('all');
    searchQuery.set('alpha');
    expect(get(filteredGames).length).toBe(1);
    expect(get(filteredGames)[0].name).toBe('Alpha Game');

    // Stats computation
    const s = get(stats);
    expect(s.totalCount).toBe(2);
    expect(s.patchedCount).toBe(1);
    expect(s.cleanCount).toBe(1);
    expect(s.totalBytes).toBe(6000);
  });

  it('sorts and cycles sort orders across metadata dimensions', () => {
    const gameZeta = {
      name: 'Zeta Game',
      app_id: '100',
      game_dir: '/games/zeta',
      is_patched: true,
      disk_size_bytes: 100,
      installed_at: 1720000000,
      last_played: 1725000000,
      playtime_seconds: 500,
    };
    const gameAlpha = {
      name: 'Alpha Game',
      app_id: '200',
      game_dir: '/games/alpha',
      is_patched: false,
      disk_size_bytes: 9000,
      installed_at: 1710000000,
      last_played: 1715000000,
      playtime_seconds: 15000,
    };

    updateGameInStore(gameZeta);
    updateGameInStore(gameAlpha);

    // 1. Initial: name_asc (Alpha Game first)
    sortBy.set('name_asc');
    expect(get(sortBy)).toBe('name_asc');
    expect(get(filteredGames)[0].name).toBe('Alpha Game');

    // 2. Sort by last_played_desc (Zeta Game played more recently)
    sortBy.set('last_played_desc');
    expect(get(filteredGames)[0].name).toBe('Zeta Game');
    expect(getSortLabel('last_played_desc')).toBe('Last Played (Recent)');

    // 3. Sort by playtime_desc (Alpha Game has more playtime)
    sortBy.set('playtime_desc');
    expect(get(filteredGames)[0].name).toBe('Alpha Game');
    expect(getSortLabel('playtime_desc')).toBe('Playtime (Most Played)');

    // 4. Sort by install_desc (Zeta Game installed more recently)
    sortBy.set('install_desc');
    expect(get(filteredGames)[0].name).toBe('Zeta Game');
    expect(getSortLabel('install_desc')).toBe('Installed (Newest)');

    // 5. Sort by size_desc (Alpha Game is larger)
    sortBy.set('size_desc');
    expect(get(filteredGames)[0].name).toBe('Alpha Game');
    expect(getSortLabel('size_desc')).toBe('Size (Largest)');

    // 6. Test cycleSortOrder from name_asc
    sortBy.set('name_asc');
    const next = cycleSortOrder();
    expect(next).toBe('last_played_desc');
    expect(get(sortBy)).toBe('last_played_desc');
  });

  it('applies delta updates via patchGameInStore with reference preservation', () => {
    const rawGame = {
      name: 'Elden Ring',
      app_id: '1245620',
      game_dir: '/games/eldenring',
      playtime_seconds: 3600,
      last_played: 1700000000,
    };
    updateGameInStore(rawGame);
    const initialList = get(games);
    selectedGame.set(initialList[0]);

    // 1. Patch with identical values -> should preserve array reference
    const unchangedResult = patchGameInStore('/games/eldenring', {
      playtimeSeconds: 3600,
      lastPlayed: 1700000000,
    });
    expect(unchangedResult).toBe(initialList[0]);
    expect(get(games)).toBe(initialList); // strict reference equality

    // 2. Patch with new delta values
    const patched = patchGameInStore('/games/eldenring', {
      playtimeSeconds: 7200,
      lastPlayed: 1700005000,
    });
    expect(patched).not.toBeNull();
    expect(patched.playtimeSeconds).toBe(7200);
    expect(patched.lastPlayed).toBe(1700005000);

    // Array reference should now be updated
    const updatedList = get(games);
    expect(updatedList).not.toBe(initialList);
    expect(updatedList[0].playtimeSeconds).toBe(7200);

    // Selected game should also be reactively updated
    expect(get(selectedGame).playtimeSeconds).toBe(7200);
    expect(get(selectedGame).lastPlayed).toBe(1700005000);

    // 3. Patch non-existent game -> returns null
    const nonExistent = patchGameInStore('/games/unknown', { playtimeSeconds: 10 });
    expect(nonExistent).toBeNull();
  });

  it('deduplicates concurrent calls in resolvePlaceholderTitle', async () => {
    const rawGame = {
      name: 'Steam App 12345',
      app_id: '12345',
      game_dir: '/games/app12345',
      store: 'Steam',
    };
    updateGameInStore(rawGame);

    let resolveApi;
    const apiPromise = new Promise((resolve) => {
      resolveApi = resolve;
    });

    const metaSpy = vi.spyOn(AppAPI, 'getGameMetadata').mockImplementation(() => apiPromise);

    const game = get(games)[0];
    selectedGame.set(game);

    // Call resolvePlaceholderTitle twice concurrently
    const p1 = resolvePlaceholderTitle(game);
    const p2 = resolvePlaceholderTitle(game);

    // Resolve the network call
    resolveApi({
      name: 'Resolved Real Title',
      header_image: 'https://cdn.steam.com/header.jpg',
    });

    await Promise.all([p1, p2]);

    // Should make only 1 AppAPI call
    expect(metaSpy).toHaveBeenCalledTimes(1);
    expect(metaSpy).toHaveBeenCalledWith('steam', '12345');

    // Should update store and selection
    expect(get(games)[0].name).toBe('Resolved Real Title');
    expect(get(games)[0].coverUrl).toBe('https://cdn.steam.com/header.jpg');
    expect(get(selectedGame).name).toBe('Resolved Real Title');

    metaSpy.mockRestore();
  });
});

