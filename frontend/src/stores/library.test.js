import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import {
  libraryPaths,
  libraryPath,
  addLibraryPath,
  removeLibraryPath,
  games,
  selectedGame,
  runningGames,
  setGameRunning,
  updateGameInStore,
  renameGameInStore,
  activeFilter,
  searchQuery,
  sortBy,
  filteredGames,
  stats,
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
      lutris_registered: true,
      disk_size_bytes: 1000,
    };
    const gameB = {
      name: 'Beta Game',
      app_id: '200',
      game_dir: '/games/beta',
      is_patched: false,
      lutris_registered: false,
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
    expect(s.lutrisCount).toBe(1);
    expect(s.totalBytes).toBe(6000);
  });
});
