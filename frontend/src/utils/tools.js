import { isPendingOrActiveStatus } from '../stores/tasks.js';

/**
 * Normalizes a raw tools.ToolStatus Go entity into a canonical camelCase ToolStatus model.
 * @param {Record<string, unknown> | null | undefined} raw
 * @returns {import('../types/domain.d.ts').ToolStatus | null}
 */
export function normalizeTool(raw) {
  if (!raw) return null;
  return {
    name: raw.name || '',
    key: raw.key || '',
    installedPath: raw.installed_path || '',
    installedTag: raw.installed_tag || '',
    latestTag: raw.latest_tag || '',
    releaseName: raw.release_name || '',
    releaseNotes: raw.release_notes || '',
    releaseUrl: raw.release_url || '',
    publishedAt: raw.published_at || '',
    hasUpdate: Boolean(raw.has_update),
    isInstalled: Boolean(raw.is_installed),
  };
}

/**
 * Finds an active batch task matching a given kind.
 * @param {'achievements' | 'artwork' | 'apply' | 'normalize' | 'verify'} kind
 * @param {Array<any>} items
 * @returns {any | null}
 */
export function findActiveBatchTask(kind, items) {
  if (!Array.isArray(items)) return null;
  return items.find((t) => {
    if (!isPendingOrActiveStatus(t.status)) {
      return false;
    }
    const title = (t.title || '').toLowerCase();
    const type = (t.type || '').toLowerCase();
    const batchKind = (t.metadata?.batch_kind || '').toLowerCase();

    if (kind === 'achievements') {
      return batchKind === 'achievements' || title.includes('achievement');
    }
    if (kind === 'artwork') {
      return batchKind === 'artwork' || title.includes('artwork');
    }
    if (kind === 'apply') {
      return batchKind === 'apply' || (type === 'apply' && title.includes('batch')) || title.includes('apply setup') || title.includes('apply drm');
    }
    if (kind === 'normalize') {
      return batchKind === 'normalize' || (type === 'normalize' && title.includes('batch')) || title.includes('normalize');
    }
    if (kind === 'verify') {
      return batchKind === 'verify' || (type === 'verify' && title.includes('batch')) || title.includes('verify');
    }
    return false;
  }) || null;
}

