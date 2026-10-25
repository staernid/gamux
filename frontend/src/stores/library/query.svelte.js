import { games } from './catalog.js';
import { notificationStore } from '../notifications.js';

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

function computeFilteredGames(gamesList, filter, query, sort) {
  let list = [...gamesList];

  // Filter
  if (filter === 'patched') {
    list = list.filter(g => g.isPatched);
  } else if (filter === 'clean') {
    list = list.filter(g => !g.isPatched);
  } else if (filter === 'updates') {
    list = list.filter(g => g.hasUpdatesOrIssues);
  }

  // Search query
  const q = (query || '').toLowerCase().trim();
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
    if (sort === 'name_asc') {
      return (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'name_desc') {
      return (b.name || '').localeCompare(a.name || '');
    } else if (sort === 'last_played_desc') {
      const diff = (b.lastPlayed || 0) - (a.lastPlayed || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'last_played_asc') {
      const diff = (a.lastPlayed || 0) - (b.lastPlayed || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'playtime_desc') {
      const diff = (b.playtimeSeconds || 0) - (a.playtimeSeconds || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'playtime_asc') {
      const diff = (a.playtimeSeconds || 0) - (b.playtimeSeconds || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'install_desc') {
      const diff = (b.installedAt || 0) - (a.installedAt || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'install_asc') {
      const diff = (a.installedAt || 0) - (b.installedAt || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'size_desc') {
      const diff = (b.diskSizeBytes || 0) - (a.diskSizeBytes || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'size_asc') {
      const diff = (a.diskSizeBytes || 0) - (b.diskSizeBytes || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'appid') {
      const diff = (parseInt(a.appId, 10) || 0) - (parseInt(b.appId, 10) || 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    } else if (sort === 'status') {
      const diff = (b.isPatched ? 1 : 0) - (a.isPatched ? 1 : 0);
      return diff !== 0 ? diff : (a.name || '').localeCompare(b.name || '');
    }
    return 0;
  });

  return list;
}

function computeStats(gamesList) {
  let patchedCount = 0;
  let cleanCount = 0;
  let updatesCount = 0;
  let totalBytes = 0;

  for (const g of gamesList) {
    totalBytes += g.diskSizeBytes || 0;
    if (g.hasUpdatesOrIssues) updatesCount++;
    if (g.isPatched) patchedCount++;
    else cleanCount++;
  }

  return {
    totalCount: gamesList.length,
    patchedCount,
    cleanCount,
    updatesCount,
    totalBytes,
  };
}

export class QueryState {
  sortBy = $state(
    (typeof localStorage !== 'undefined' && localStorage.getItem('gamux_sort_by')) || 'name_asc'
  );
  activeFilter = $state('all');
  searchQuery = $state('');

  #sortBySubscribers = new Set();
  #activeFilterSubscribers = new Set();
  #searchQuerySubscribers = new Set();
  #filteredSubscribers = new Set();
  #statsSubscribers = new Set();

  #currentGames = [];

  constructor() {
    games.subscribe(list => {
      this.#currentGames = Array.isArray(list) ? list : [];
      this.#notifyFiltered();
      this.#notifyStats();
    });
  }

  setSortBy(val) {
    this.sortBy = val;
    if (typeof localStorage !== 'undefined') {
      try {
        localStorage.setItem('gamux_sort_by', val);
      } catch {}
    }
    this.#notifySortBy();
    this.#notifyFiltered();
  }

  setActiveFilter(val) {
    this.activeFilter = val;
    this.#notifyActiveFilter();
    this.#notifyFiltered();
  }

  setSearchQuery(val) {
    this.searchQuery = val;
    this.#notifySearchQuery();
    this.#notifyFiltered();
  }

  get filteredGames() {
    return computeFilteredGames(this.#currentGames, this.activeFilter, this.searchQuery, this.sortBy);
  }

  get stats() {
    return computeStats(this.#currentGames);
  }

  #notifySortBy() {
    for (const fn of this.#sortBySubscribers) {
      try { fn(this.sortBy); } catch {}
    }
  }

  #notifyActiveFilter() {
    for (const fn of this.#activeFilterSubscribers) {
      try { fn(this.activeFilter); } catch {}
    }
  }

  #notifySearchQuery() {
    for (const fn of this.#searchQuerySubscribers) {
      try { fn(this.searchQuery); } catch {}
    }
  }

  #notifyFiltered() {
    const list = this.filteredGames;
    for (const fn of this.#filteredSubscribers) {
      try { fn(list); } catch {}
    }
  }

  #notifyStats() {
    const st = this.stats;
    for (const fn of this.#statsSubscribers) {
      try { fn(st); } catch {}
    }
  }

  get _sortByBridge() {
    return {
      subscribe: (fn) => {
        this.#sortBySubscribers.add(fn);
        fn(this.sortBy);
        return () => this.#sortBySubscribers.delete(fn);
      },
      set: (val) => this.setSortBy(val),
      update: (fn) => this.setSortBy(fn(this.sortBy))
    };
  }

  get _activeFilterBridge() {
    return {
      subscribe: (fn) => {
        this.#activeFilterSubscribers.add(fn);
        fn(this.activeFilter);
        return () => this.#activeFilterSubscribers.delete(fn);
      },
      set: (val) => this.setActiveFilter(val),
      update: (fn) => this.setActiveFilter(fn(this.activeFilter))
    };
  }

  get _searchQueryBridge() {
    return {
      subscribe: (fn) => {
        this.#searchQuerySubscribers.add(fn);
        fn(this.searchQuery);
        return () => this.#searchQuerySubscribers.delete(fn);
      },
      set: (val) => this.setSearchQuery(val),
      update: (fn) => this.setSearchQuery(fn(this.searchQuery))
    };
  }

  get _filteredGamesBridge() {
    return {
      subscribe: (fn) => {
        this.#filteredSubscribers.add(fn);
        fn(this.filteredGames);
        return () => this.#filteredSubscribers.delete(fn);
      }
    };
  }

  get _statsBridge() {
    return {
      subscribe: (fn) => {
        this.#statsSubscribers.add(fn);
        fn(this.stats);
        return () => this.#statsSubscribers.delete(fn);
      }
    };
  }
}

export const queryStore = new QueryState();
export const sortBy = queryStore._sortByBridge;
export const activeFilter = queryStore._activeFilterBridge;
export const searchQuery = queryStore._searchQueryBridge;
export const filteredGames = queryStore._filteredGamesBridge;
export const stats = queryStore._statsBridge;

export function cycleSortOrder() {
  const current = queryStore.sortBy;
  const idx = SORT_MODES.indexOf(current);
  const next = (idx >= 0 && idx < SORT_MODES.length - 1) ? SORT_MODES[idx + 1] : SORT_MODES[0];
  queryStore.setSortBy(next);
  notificationStore.osd(`Sort: ${getSortLabel(next)}`);
  return next;
}
