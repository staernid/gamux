import { isSteamApp } from './game.js';

/**
 * Resolves effective Steam App ID from game properties.
 * @param {import('../types/domain').Game | null} [game]
 * @returns {string}
 */
export function getEffectiveAppId(game) {
  if (game?.appId && isSteamApp(game.appId)) {
    return String(game.appId).trim();
  }
  if ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId)) {
    return String(game.metadataId).trim();
  }
  return '';
}

/**
 * Sorts Steam depot branches with active branch first, then 'public', then alphabetical.
 * @param {Array<{ name: string, description?: string, build_id?: string }>} branches
 * @param {string} currentBranch
 * @returns {Array<{ name: string, description?: string, build_id?: string }>}
 */
export function sortBranches(branches = [], currentBranch = 'public') {
  return [...branches].sort((a, b) => {
    if (a.name === currentBranch && b.name !== currentBranch) return -1;
    if (b.name === currentBranch && a.name !== currentBranch) return 1;
    if (a.name === 'public') return -1;
    if (b.name === 'public') return 1;
    return (a.name || '').localeCompare(b.name || '');
  });
}

/**
 * Filters branches by a case-insensitive query against name, description, or build_id.
 * @param {Array<{ name: string, description?: string, build_id?: string }>} branches
 * @param {string} query
 * @returns {Array<{ name: string, description?: string, build_id?: string }>}
 */
export function filterBranches(branches = [], query = '') {
  if (!query || !query.trim()) return branches;
  const q = query.trim().toLowerCase();
  return branches.filter(b => {
    return (b.name && b.name.toLowerCase().includes(q)) ||
      (b.description && b.description.toLowerCase().includes(q)) ||
      (b.build_id && String(b.build_id).includes(q));
  });
}

/**
 * Determines whether the installed files are out of sync with the selected branch build.
 * Absence sentinels ('', '0', 'unknown') are ignored per Core Tenet 4.
 * @param {boolean} isInstalled
 * @param {string|number|null|undefined} installedBuildId
 * @param {string|number|null|undefined} targetBuildId
 * @returns {boolean}
 */
export function isBranchBuildMismatch(isInstalled, installedBuildId, targetBuildId) {
  if (!isInstalled) return false;
  const cleanInstalled = installedBuildId && installedBuildId !== '0' && installedBuildId !== 'unknown'
    ? String(installedBuildId).trim()
    : '';
  if (!cleanInstalled) return false;

  const cleanTarget = targetBuildId && targetBuildId !== '0' && targetBuildId !== 'unknown'
    ? String(targetBuildId).trim()
    : '';
  if (!cleanTarget) return false;

  return cleanInstalled !== cleanTarget;
}
