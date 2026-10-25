/**
 * Normalizes a raw tools.ToolStatus Go entity into a canonical camelCase ToolStatus model.
 * @param {any} raw
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
