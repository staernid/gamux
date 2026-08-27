// Game data normalization and status helpers

export function normalizePath(p) {
  if (!p || typeof p !== 'string') return '';
  let s = p.trim().replace(/\\/g, '/');
  s = s.replace(/\/+/g, '/');
  if (s.length > 1 && s.endsWith('/')) {
    s = s.slice(0, -1);
  }
  return s;
}

export function isSamePath(pathA, pathB) {
  if (!pathA && !pathB) return true;
  if (!pathA || !pathB) return false;
  return normalizePath(pathA) === normalizePath(pathB);
}

export function normalizeGame(g) {
  if (!g) return null;

  // Filter launch candidates to exclude technical loader helpers
  const rawCandidates = g.launch_candidates || g.LaunchCandidates || [];
  const launchCandidates = rawCandidates
    .filter(c => {
      const exe = (c.executable || c.Executable || '').toLowerCase();
      return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
    })
    .map(c => ({
      name: c.name || c.Name || '',
      executable: normalizePath(c.executable || c.Executable || ''),
      arguments: c.arguments || c.Arguments || '',
      description: c.description || c.Description || '',
    }));

  const launchProfile = g.launch_profile || g.LaunchProfile || null;
  let rawExePath = launchProfile?.exe_path || g.exe_path || g.ExePath || '';
  if (rawExePath && (rawExePath.toLowerCase().includes('steamclient_loader') || rawExePath.toLowerCase().includes('coldclientloader'))) {
    rawExePath = launchCandidates[0]?.executable || '';
  }
  const exePath = normalizePath(rawExePath);

  const name = g.name || g.Name || 'Unknown Game';
  const appId = String(g.app_id || g.AppID || '');
  const gameDir = normalizePath(g.game_dir || g.GameDir || '');
  const store = g.store || g.Store || 'Steam';
  const platform = g.platform || g.Platform || 'win64';
  const exeArgs = launchProfile?.exe_args || g.exe_args || g.ExeArgs || '';
  const isPatched = Boolean(g.is_patched ?? g.isPatched ?? g.IsPatched ?? (g.state === 'Portable-Patched' || g.state === 'Loader-Configured' || g.settings_dir_exists || g.SettingsDirExists));
  const isPortable = Boolean(g.is_portable ?? g.isPortable ?? g.IsPortable ?? (g.state === 'Portable-Patched'));
  const isReady = Boolean(g.is_ready ?? g.isReady ?? g.IsReady ?? (isPatched || !!exePath));
  const hasUpdatesOrIssues = Boolean(g.has_updates_or_issues ?? g.hasUpdatesOrIssues ?? g.HasUpdatesOrIssues ?? (store === 'Steam' && (g.has_update || g.HasUpdate || (g.modified_files && g.modified_files.length > 0) || (g.missing_files && g.missing_files.length > 0))));
  const diskSizeBytes = Number(g.disk_size_bytes ?? g.DiskSizeBytes ?? 0);
  const fileCount = Number(g.file_count ?? g.FileCount ?? 0);
  const officialFileCount = Number(g.official_file_count ?? g.OfficialFileCount ?? 0);
  const dlcCount = Number(g.dlc_count ?? g.DLCCount ?? 0);
  const achievementCount = Number(g.achievement_count ?? g.AchievementCount ?? 0);
  const operatingMode = launchProfile?.operating_mode || g.operating_mode || 'standalone';
  const runnerType = launchProfile?.runner || g.runner || 'umu';
  const winePrefix = normalizePath(launchProfile?.wine_prefix || g.wine_prefix || '');
  const currentLanguage = g.current_language || g.CurrentLanguage || '';
  const supportedLanguages = g.supported_languages || g.SupportedLanguages || [];
  const modifiedFiles = g.modified_files || g.ModifiedFiles || [];
  const missingFiles = g.missing_files || g.MissingFiles || [];
  const untrackedFiles = g.untracked_files || g.UntrackedFiles || [];
  const hasUpdate = Boolean(g.has_update ?? g.HasUpdate ?? false);
  const manifestId = g.manifest_id || g.ManifestID || '';
  const buildId = g.build_id || g.BuildID || '';
  const remoteManifestId = g.remote_manifest_id || g.RemoteManifestID || '';
  const recentPatchNote = g.recent_patch_note || g.RecentPatchNote || '';
  const newsItems = g.news_items || g.NewsItems || [];
  const coverUrl = g.cover_url || g.coverUrl || g.custom_cover_url || '';
  const localCoverUrl = g.local_cover_url || g.localCoverUrl || g.local_cover || '';
  const heroUrl = g.hero_url || g.heroUrl || g.custom_hero_url || '';
  const localHeroUrl = g.local_hero_url || g.localHeroUrl || g.local_hero || '';
  const bannerUrl = g.banner_url || g.bannerUrl || g.custom_banner_url || '';
  const localBannerUrl = g.local_banner_url || g.localBannerUrl || g.local_banner || '';
  const logoUrl = g.logo_url || g.logoUrl || g.custom_logo_url || '';
  const localLogoUrl = g.local_logo_url || g.localLogoUrl || g.local_logo || '';

  return {
    ...g,
    // Canonical camelCase domain entity properties
    name,
    appId,
    gameDir,
    store,
    platform,
    exePath,
    exeArgs,
    launchCandidates,
    launchProfile,
    isPatched,
    isPortable,
    isReady,
    hasUpdatesOrIssues,
    diskSizeBytes,
    fileCount,
    officialFileCount,
    dlcCount,
    achievementCount,
    operatingMode,
    runner: runnerType,
    winePrefix,
    currentLanguage,
    supportedLanguages,
    modifiedFiles,
    missingFiles,
    untrackedFiles,
    hasUpdate,
    manifestId,
    buildId,
    remoteManifestId,
    recentPatchNote,
    newsItems,
    coverUrl,
    localCoverUrl,
    heroUrl,
    localHeroUrl,
    bannerUrl,
    localBannerUrl,
    logoUrl,
    localLogoUrl,

    // Aliases for seamless backwards compatibility
    Name: name,
    AppID: appId,
    app_id: appId,
    GameDir: gameDir,
    game_dir: gameDir,
    Store: store,
    Platform: platform,
    ExePath: exePath,
    exe_path: exePath,
    ExeArgs: exeArgs,
    exe_args: exeArgs,
    LaunchCandidates: launchCandidates,
    launch_candidates: launchCandidates,
    LaunchProfile: launchProfile,
    launch_profile: launchProfile,
    is_patched: isPatched,
    is_portable: isPortable,
    is_ready: isReady,
    has_updates_or_issues: hasUpdatesOrIssues,
    DiskSizeBytes: diskSizeBytes,
    disk_size_bytes: diskSizeBytes,
    FileCount: fileCount,
    file_count: fileCount,
    OfficialFileCount: officialFileCount,
    official_file_count: officialFileCount,
    DLCCount: dlcCount,
    dlc_count: dlcCount,
    AchievementCount: achievementCount,
    achievement_count: achievementCount,
    OperatingMode: operatingMode,
    operating_mode: operatingMode,
    Runner: runnerType,
    WinePrefix: winePrefix,
    wine_prefix: winePrefix,
    CurrentLanguage: currentLanguage,
    current_language: currentLanguage,
    SupportedLanguages: supportedLanguages,
    supported_languages: supportedLanguages,
    ModifiedFiles: modifiedFiles,
    modified_files: modifiedFiles,
    MissingFiles: missingFiles,
    missing_files: missingFiles,
    UntrackedFiles: untrackedFiles,
    untracked_files: untrackedFiles,
    HasUpdate: hasUpdate,
    has_update: hasUpdate,
    ManifestID: manifestId,
    manifest_id: manifestId,
    BuildID: buildId,
    build_id: buildId,
    RemoteManifestID: remoteManifestId,
    remote_manifest_id: remoteManifestId,
    RecentPatchNote: recentPatchNote,
    recent_patch_note: recentPatchNote,
    NewsItems: newsItems,
    news_items: newsItems,
    custom_cover_url: coverUrl,
    custom_hero_url: heroUrl,
    custom_banner_url: bannerUrl,
    custom_logo_url: logoUrl,
    local_cover_url: localCoverUrl,
    local_hero_url: localHeroUrl,
    local_banner_url: localBannerUrl,
    local_logo_url: localLogoUrl,
  };
}

export function getInitials(name) {
  if (!name) return 'GX';
  const parts = name.trim().split(/[\s_-]+/);
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

export function isSteamApp(appId) {
  return Boolean(appId && appId !== '0' && appId !== 'Custom' && /^\d+$/.test(String(appId).trim()));
}

export function getPlatformType(game) {
  if (!game) return 'Standalone';
  const store = (game.store || game.Store || '').toLowerCase();
  if (store.includes('gog')) return 'GOG';
  if (store.includes('steam') || isSteamApp(game.app_id || game.AppID)) return 'Steam';
  return 'Standalone';
}

export function getPlatformColors(game) {
  const pType = getPlatformType(game);
  switch (pType) {
    case 'Steam':
      return {
        label: 'Steam',
        border: '#1a9fff',
        badgeBg: '#1a9fff',
        badgeText: '#ffffff',
        glow: 'rgba(26, 159, 255, 0.4)',
        ringClass: 'ring-[#1a9fff]',
      };
    case 'GOG':
      return {
        label: 'GOG',
        border: '#862d86',
        badgeBg: '#862d86',
        badgeText: '#ffffff',
        glow: 'rgba(134, 45, 134, 0.4)',
        ringClass: 'ring-[#862d86]',
      };
    case 'Standalone':
    default:
      return {
        label: 'Standalone',
        border: '#f4f4f6',
        badgeBg: '#e2e8f0',
        badgeText: '#09090c',
        glow: 'rgba(244, 244, 246, 0.3)',
        ringClass: 'ring-[#f4f4f6]',
      };
  }
}

// In-memory cache for extracted dominant cover colors
const coverColorCache = new Map();

export function hexToRgb(hex) {
  if (!hex) return { r: 26, g: 159, b: 255 };
  let c = hex.replace('#', '');
  if (c.length === 3) {
    c = c.split('').map(x => x + x).join('');
  }
  const num = parseInt(c, 16);
  if (isNaN(num)) return { r: 26, g: 159, b: 255 };
  return {
    r: (num >> 16) & 255,
    g: (num >> 8) & 255,
    b: num & 255,
  };
}

export function parseColorToRgba(colorStr, alpha = 1) {
  if (!colorStr) return `rgba(26, 159, 255, ${alpha})`;
  if (colorStr.startsWith('#')) {
    const { r, g, b } = hexToRgb(colorStr);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
  }
  if (colorStr.startsWith('rgb')) {
    const match = colorStr.match(/\d+/g);
    if (match && match.length >= 3) {
      return `rgba(${match[0]}, ${match[1]}, ${match[2]}, ${alpha})`;
    }
  }
  return `rgba(26, 159, 255, ${alpha})`;
}

const coverPaletteCache = new Map();

/**
 * Extracts a palette of 1-3 dominant vibrant colors from an image URL.
 * Results are cached in memory for instant O(1) lookup during carousel browsing.
 */
export function extractCoverColorPalette(url) {
  if (!url || typeof url !== 'string' || typeof document === 'undefined') {
    return Promise.resolve([]);
  }
  if (coverPaletteCache.has(url)) {
    return Promise.resolve(coverPaletteCache.get(url));
  }

  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = 'Anonymous';
    img.onload = () => {
      try {
        const canvas = document.createElement('canvas');
        const size = 12;
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext('2d', { willReadFrequently: true });
        if (!ctx) {
          resolve([]);
          return;
        }
        ctx.drawImage(img, 0, 0, size, size);
        const data = ctx.getImageData(0, 0, size, size).data;

        // Sample pixels, score by saturation and vibrancy
        const samples = [];
        for (let i = 0; i < data.length; i += 4) {
          const r = data[i];
          const g = data[i + 1];
          const b = data[i + 2];
          const a = data[i + 3];
          if (a < 128) continue;

          const max = Math.max(r, g, b);
          const min = Math.min(r, g, b);
          const brightness = (r + g + b) / 3;
          const saturation = max === 0 ? 0 : (max - min) / max;

          // Skip completely dark or blown out pixels
          if (brightness < 18 || brightness > 245) continue;

          samples.push({ r, g, b, saturation, brightness, score: saturation * 1.5 + (brightness / 255) });
        }

        if (samples.length === 0) {
          const fallback = ['rgb(26, 159, 255)'];
          coverPaletteCache.set(url, fallback);
          resolve(fallback);
          return;
        }

        // Sort by vibrant score descending
        samples.sort((a, b) => b.score - a.score);

        // Select up to 3 distinct color clusters
        const palette = [];
        for (const s of samples) {
          let { r, g, b } = s;
          const maxVal = Math.max(r, g, b);
          if (maxVal > 0 && maxVal < 130) {
            const boost = Math.min(2.2, 140 / maxVal);
            r = Math.min(255, Math.round(r * boost));
            g = Math.min(255, Math.round(g * boost));
            b = Math.min(255, Math.round(b * boost));
          }

          // Check color difference with already selected palette items (Euclidean RGB distance > 65)
          const isDistinct = palette.every(p => {
            const dr = p.r - r;
            const dg = p.g - g;
            const db = p.b - b;
            return Math.sqrt(dr * dr + dg * dg + db * db) > 65;
          });

          if (isDistinct) {
            palette.push({ r, g, b, str: `rgb(${r}, ${g}, ${b})` });
            if (palette.length >= 3) break;
          }
        }

        const result = palette.length > 0 ? palette.map(p => p.str) : [`rgb(${samples[0].r}, ${samples[0].g}, ${samples[0].b})`];
        coverPaletteCache.set(url, result);
        resolve(result);
      } catch {
        resolve([]);
      }
    };
    img.onerror = () => resolve([]);
    img.src = url;
  });
}

export function getCachedCoverPalette(url) {
  if (!url || typeof url !== 'string') return null;
  return coverPaletteCache.get(url) || null;
}

export function extractCoverColor(url) {
  return extractCoverColorPalette(url).then(palette => (palette && palette.length > 0 ? palette[0] : null));
}

export function getCachedCoverColor(url) {
  const pal = getCachedCoverPalette(url);
  return pal && pal.length > 0 ? pal[0] : null;
}

