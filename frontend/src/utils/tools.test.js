import { describe, it, expect } from 'vitest';
import { normalizeTool } from './tools.js';

describe('tools utilities', () => {
  it('normalizes raw Go ToolStatus wire shape to canonical camelCase ToolStatus', () => {
    expect(normalizeTool(null)).toBeNull();
    expect(normalizeTool(undefined)).toBeNull();

    const raw = {
      name: 'Goldberg Steam Emulator',
      key: 'gbe',
      installed_path: '/home/user/.local/share/gamux/tools/gbe',
      installed_tag: 'v0.2.6',
      latest_tag: 'v0.2.7',
      release_name: 'GBE v0.2.7',
      release_notes: 'Bug fixes',
      release_url: 'https://github.com/example/gbe',
      published_at: '2026-10-01T00:00:00Z',
      has_update: true,
      is_installed: true,
    };

    const tool = normalizeTool(raw);
    expect(tool).toEqual({
      name: 'Goldberg Steam Emulator',
      key: 'gbe',
      installedPath: '/home/user/.local/share/gamux/tools/gbe',
      installedTag: 'v0.2.6',
      latestTag: 'v0.2.7',
      releaseName: 'GBE v0.2.7',
      releaseNotes: 'Bug fixes',
      releaseUrl: 'https://github.com/example/gbe',
      publishedAt: '2026-10-01T00:00:00Z',
      hasUpdate: true,
      isInstalled: true,
    });
  });
});
