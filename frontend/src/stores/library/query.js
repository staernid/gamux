import { writable, derived } from 'svelte/store';
import { games } from './catalog.js';

export const sortBy = writable(
  (typeof localStorage !== 'undefined' && localStorage.getItem('gamux_sort_by')) || 'name_asc'
);
export const activeFilter = writable('all');
export const searchQuery = writable('');

if (typeof localStorage !== 'undefined') {
  sortBy.subscribe(val => {
    localStorage.setItem('gamux_sort_by', val);
  });
}

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
      list = list.filter(g => g.lutrisRegistered);
    } else if ($activeFilter === 'updates') {
      list = list.filter(g => g.hasUpdatesOrIssues);
    }

    // Search query
    const q = $searchQuery.toLowerCase().trim();
    if (q) {
      list = list.filter(g =>
        (g.name && g.name.toLowerCase().includes(q)) ||
        (g.appId && g.appId.includes(q)) ||
        (g.platform && g.platform.toLowerCase().includes(q)) ||
        (g.gameDir && g.gameDir.toLowerCase().includes(q))
      );
    }

    // Sort
    list.sort((a, b) => {
      if ($sortBy === 'name_asc') {
        return (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'name_desc') {
        return (b.name || '').localeCompare(a.name || '');
      } else if ($sortBy === 'size_desc') {
        return (b.diskSizeBytes || 0) - (a.diskSizeBytes || 0);
      } else if ($sortBy === 'size_asc') {
        return (a.diskSizeBytes || 0) - (b.diskSizeBytes || 0);
      } else if ($sortBy === 'appid') {
        return (parseInt(a.appId, 10) || 0) - (parseInt(b.appId, 10) || 0);
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
    totalBytes += g.diskSizeBytes || 0;
    if (g.lutrisRegistered) lutrisCount++;
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
