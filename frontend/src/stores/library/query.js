import { writable, derived, get } from 'svelte/store';
import { games } from './catalog.js';
import { notificationStore } from '../notifications.js';

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

export const SORT_MODES = [
  'name_asc',
  'last_played_desc',
  'playtime_desc',
  'install_desc',
  'size_desc',
  'name_desc',
  'status',
  'appid'
];

export function getSortLabel(sortKey) {
  switch (sortKey) {
    case 'last_played_desc':
      return 'Last Played (Recent)';
    case 'last_played_asc':
      return 'Last Played (Oldest)';
    case 'playtime_desc':
      return 'Playtime (Most Played)';
    case 'playtime_asc':
      return 'Playtime (Least Played)';
    case 'install_desc':
      return 'Installed (Newest)';
    case 'install_asc':
      return 'Installed (Oldest)';
    case 'size_desc':
      return 'Size (Largest)';
    case 'size_asc':
      return 'Size (Smallest)';
    case 'name_desc':
      return 'Name (Z-A)';
    case 'status':
      return 'Status (Ready First)';
    case 'appid':
      return 'App ID';
    case 'name_asc':
    default:
      return 'Name (A-Z)';
  }
}

export function cycleSortOrder() {
  const current = get(sortBy);
  const idx = SORT_MODES.indexOf(current);
  const next = (idx >= 0 && idx < SORT_MODES.length - 1) ? SORT_MODES[idx + 1] : SORT_MODES[0];
  sortBy.set(next);
  notificationStore.osd(`Sort: ${getSortLabel(next)}`);
  return next;
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
      } else if ($sortBy === 'last_played_desc') {
        const diff = (b.lastPlayed || 0) - (a.lastPlayed || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'last_played_asc') {
        const diff = (a.lastPlayed || 0) - (b.lastPlayed || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'playtime_desc') {
        const diff = (b.playtimeSeconds || 0) - (a.playtimeSeconds || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'playtime_asc') {
        const diff = (a.playtimeSeconds || 0) - (b.playtimeSeconds || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'install_desc') {
        const diff = (b.installedAt || 0) - (a.installedAt || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'install_asc') {
        const diff = (a.installedAt || 0) - (b.installedAt || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'size_desc') {
        const diff = (b.diskSizeBytes || 0) - (a.diskSizeBytes || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'size_asc') {
        const diff = (a.diskSizeBytes || 0) - (b.diskSizeBytes || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'appid') {
        const diff = (parseInt(a.appId, 10) || 0) - (parseInt(b.appId, 10) || 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      } else if ($sortBy === 'status') {
        const diff = (b.isPatched ? 1 : 0) - (a.isPatched ? 1 : 0);
        return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
      }
      return 0;
    });

    return list;
  }
);

export const stats = derived(games, $games => {
  let patchedCount = 0;
  let cleanCount = 0;
  let updatesCount = 0;
  let totalBytes = 0;

  for (const g of $games) {
    totalBytes += g.diskSizeBytes || 0;
    if (g.hasUpdatesOrIssues) updatesCount++;
    if (g.isPatched) patchedCount++;
    else cleanCount++;
  }

  return {
    totalCount: $games.length,
    patchedCount,
    cleanCount,
    updatesCount,
    totalBytes,
  };
});
