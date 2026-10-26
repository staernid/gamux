import { isLiveStatus, taskMatchesGame } from '../stores/tasks.js';
import { isSteamApp } from './game.js';

export const PRIMARY_HUB_TABS = ['overview', 'config', 'files', 'mods', 'saves'];

/**
 * Cycles to the next or previous primary hub tab.
 * @param {string} currentTab
 * @param {number} dir - +1 or -1
 * @param {string[]} [tabs]
 * @returns {string}
 */
export function cycleHubTab(currentTab, dir, tabs = PRIMARY_HUB_TABS) {
  const idx = tabs.indexOf(currentTab);
  if (idx === -1) return tabs[0];
  let next = idx + dir;
  if (next < 0) next = tabs.length - 1;
  if (next >= tabs.length) next = 0;
  return tabs[next];
}

/**
 * Finds the currently active live task associated with a game.
 * @param {Array<import('../types/domain').Task>} [tasks]
 * @param {import('../types/domain').Game | null} [game]
 * @returns {import('../types/domain').Task | null}
 */
export function findActiveGameTask(tasks = [], game) {
  if (!game || !tasks) return null;
  return tasks.find(t => isLiveStatus(t.status) && taskMatchesGame(t, game)) || null;
}

/**
 * Finds the latest verify task for a given game.
 * @param {Array<import('../types/domain').Task>} [tasks]
 * @param {import('../types/domain').Game | null} [game]
 * @returns {import('../types/domain').Task | null}
 */
export function findVerifyTask(tasks = [], game) {
  if (!game || !tasks) return null;
  for (let i = tasks.length - 1; i >= 0; i--) {
    const t = tasks[i];
    if (t.type === 'verify' && taskMatchesGame(t, game)) {
      return t;
    }
  }
  return null;
}

/**
 * Checks whether a game entity is linked to Steam depot/store metadata.
 * @param {import('../types/domain').Game | null} [game]
 * @returns {boolean}
 */
export function isSteamTitle(game) {
  if (!game) return false;
  return Boolean(
    (game.appId && isSteamApp(game.appId)) ||
    ((game.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game.metadataId))
  );
}
