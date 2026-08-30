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
  const rawCandidates = g.launch_candidates || g.LaunchCandidates || g.launchCandidates || [];
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

  const lp = g.launch_profile || g.LaunchProfile || g.launchProfile || null;
  let rawExe = lp?.exe_path || lp?.ExePath || lp?.exePath || g.exe_path || g.ExePath || g.exePath || '';
  if (rawExe && (rawExe.toLowerCase().includes('steamclient_loader') || rawExe.toLowerCase().includes('coldclientloader'))) {
    rawExe = launchCandidates[0]?.executable || '';
  }

  const isPatched = Boolean(
    g.is_patched ?? g.isPatched ?? g.IsPatched ??
    (g.state === 'Portable-Patched' || g.state === 'Loader-Configured' || g.settings_dir_exists || g.SettingsDirExists)
  );

  return {
    name: g.name || g.Name || 'Unknown Game',
    appId: String(g.app_id ?? g.AppID ?? g.appId ?? ''),
    gameDir: normalizePath(g.game_dir ?? g.GameDir ?? g.gameDir ?? ''),
    store: g.store || g.Store || 'Steam',
    source: g.source || g.Source || g.store || g.Store || 'Steam',
    gogId: String(g.gog_id ?? g.GOGID ?? g.gogId ?? lp?.gog_id ?? lp?.GOGID ?? lp?.gogId ?? ''),
    metadataProvider: lp?.metadata_provider || lp?.MetadataProvider || lp?.metadataProvider || g.metadata_provider || g.MetadataProvider || g.metadataProvider || '',
    metadataId: String(lp?.metadata_id ?? lp?.MetadataID ?? lp?.metadataId ?? g.metadata_id ?? g.MetadataID ?? g.metadataId ?? ''),
    platform: g.platform || g.Platform || 'win64',
    exePath: normalizePath(rawExe),
    exeArgs: lp?.exe_args || lp?.ExeArgs || lp?.exeArgs || g.exe_args || g.ExeArgs || g.exeArgs || '',
    launchCandidates,
    launchProfile: lp,
    isPatched,
    isPortable: Boolean(g.is_portable ?? g.isPortable ?? g.IsPortable ?? (g.state === 'Portable-Patched')),
    isReady: Boolean(g.is_ready ?? g.isReady ?? g.IsReady ?? (isPatched || !!rawExe)),
    hasUpdatesOrIssues: Boolean(
      g.has_updates_or_issues ?? g.hasUpdatesOrIssues ?? g.HasUpdatesOrIssues ??
      ((g.store === 'Steam' || g.Store === 'Steam') && (g.has_update || g.HasUpdate || (g.modified_files?.length > 0) || (g.missing_files?.length > 0)))
    ),
    diskSizeBytes: Number(g.disk_size_bytes ?? g.DiskSizeBytes ?? g.diskSizeBytes ?? 0),
    fileCount: Number(g.file_count ?? g.FileCount ?? g.fileCount ?? 0),
    officialFileCount: Number(g.official_file_count ?? g.OfficialFileCount ?? g.officialFileCount ?? 0),
    dlcCount: Number(g.dlc_count ?? g.DLCCount ?? g.dlcCount ?? 0),
    achievementCount: Number(g.achievement_count ?? g.AchievementCount ?? g.achievementCount ?? 0),
    operatingMode: lp?.operating_mode || lp?.OperatingMode || lp?.operatingMode || g.operating_mode || g.OperatingMode || g.operatingMode || '',
    runner: lp?.runner || lp?.Runner || g.runner || g.Runner || '',
    protonPath: normalizePath(lp?.proton_path || lp?.ProtonPath || lp?.protonPath || g.proton_path || g.ProtonPath || g.protonPath || ''),
    winePrefix: normalizePath(lp?.wine_prefix || lp?.WinePrefix || lp?.winePrefix || g.wine_prefix || g.WinePrefix || g.winePrefix || ''),
    currentLanguage: g.current_language || g.CurrentLanguage || g.currentLanguage || '',
    supportedLanguages: g.supported_languages || g.SupportedLanguages || g.supportedLanguages || [],
    modifiedFiles: g.modified_files || g.ModifiedFiles || g.modifiedFiles || [],
    missingFiles: g.missing_files || g.MissingFiles || g.missingFiles || [],
    untrackedFiles: g.untracked_files || g.UntrackedFiles || g.untrackedFiles || [],
    hasUpdate: Boolean(g.has_update ?? g.HasUpdate ?? g.hasUpdate ?? false),
    manifestId: g.manifest_id || g.ManifestID || g.manifestId || '',
    buildId: g.build_id || g.BuildID || g.buildId || '',
    remoteManifestId: g.remote_manifest_id || g.RemoteManifestID || g.remoteManifestId || '',
    recentPatchNote: g.recent_patch_note || g.RecentPatchNote || g.recentPatchNote || '',
    newsItems: g.news_items || g.NewsItems || g.newsItems || [],
    coverUrl: g.cover_url || g.coverUrl || g.custom_cover_url || g.customCoverUrl || '',
    localCoverUrl: g.local_cover_url || g.localCoverUrl || g.local_cover || g.localCover || '',
    heroUrl: g.hero_url || g.heroUrl || g.custom_hero_url || g.customHeroUrl || '',
    localHeroUrl: g.local_hero_url || g.localHeroUrl || g.local_hero || g.localHero || '',
    bannerUrl: g.banner_url || g.bannerUrl || g.custom_banner_url || g.customBannerUrl || '',
    localBannerUrl: g.local_banner_url || g.localBannerUrl || g.local_banner || g.localBanner || '',
    logoUrl: g.logo_url || g.logoUrl || g.custom_logo_url || g.customLogoUrl || '',
    localLogoUrl: g.local_logo_url || g.localLogoUrl || g.local_logo || g.localLogo || '',
    playtimeSeconds: Number(g.playtime_seconds ?? g.PlaytimeSeconds ?? g.playtimeSeconds ?? lp?.playtime_seconds ?? lp?.PlaytimeSeconds ?? lp?.playtimeSeconds ?? 0),
    lastPlayed: Number(g.last_played ?? g.LastPlayed ?? g.lastPlayed ?? lp?.last_played ?? lp?.LastPlayed ?? lp?.lastPlayed ?? 0),
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

export function hasMetadataMatch(game) {
  if (!game) return false;
  const lp = game.launch_profile || game.launchProfile || game.LaunchProfile || null;
  const provider = (game.metadataProvider || game.metadata_provider || game.MetadataProvider || lp?.metadata_provider || lp?.MetadataProvider || '').toLowerCase();
  if (provider === 'none' || provider === 'standalone' || provider === 'custom') {
    return false;
  }
  const metaId = String(game.metadataId || game.metadata_id || game.MetadataID || lp?.metadata_id || lp?.MetadataID || '');
  if (metaId && metaId !== '0' && metaId !== 'undefined' && metaId !== 'null') {
    return true;
  }
  const appId = String(game.appId || game.app_id || '');
  if (isSteamApp(appId)) return true;
  const gogId = String(game.gogId || game.gog_id || game.GOGID || lp?.gog_id || '');
  if (gogId && gogId !== '0' && gogId !== 'undefined' && gogId !== 'null') return true;
  return false;
}

export function getGameSource(game) {
  if (!game) return 'Standalone';
  const src = (game.source || game.Source || '').toLowerCase();
  if (src.includes('steam')) return 'Steam';
  if (src.includes('gog')) return 'GOG';
  if (src.includes('epic')) return 'Epic';
  if (src.includes('itch')) return 'Itch';
  if (src.includes('custom')) return 'Custom';
  if (src.includes('standalone')) return 'Standalone';

  const store = (game.store || game.Store || '').toLowerCase();
  if (store.includes('steam')) return 'Steam';
  if (store.includes('gog')) return 'GOG';
  if (store.includes('epic')) return 'Epic';
  if (store.includes('itch')) return 'Itch';
  return 'Standalone';
}

export function getMetadataInfo(game) {
  if (!game) return { provider: 'None', id: '' };
  const lp = game.launch_profile || game.launchProfile || game.LaunchProfile || null;
  const provider = (game.metadataProvider || game.metadata_provider || game.MetadataProvider || lp?.metadata_provider || lp?.MetadataProvider || '').toLowerCase();
  if (provider === 'none' || provider === 'standalone' || provider === 'custom') {
    return { provider: 'None', id: '' };
  }
  const metaId = String(game.metadataId || game.metadata_id || game.MetadataID || lp?.metadata_id || lp?.MetadataID || '');
  const appId = String(game.appId || game.app_id || game.AppID || '');
  const gogId = String(game.gogId || game.gog_id || game.GOGID || lp?.gog_id || '');

  if (provider === 'steam' || (!provider && isSteamApp(appId))) {
    return { provider: 'Steam', id: metaId || appId };
  }
  if (provider === 'gog' || (!provider && gogId && gogId !== '0')) {
    return { provider: 'GOG', id: metaId || gogId };
  }
  if (provider === 'rawg') {
    return { provider: 'RAWG', id: metaId };
  }
  if (provider) {
    return { provider: provider.toUpperCase(), id: metaId };
  }
  if (isSteamApp(appId)) {
    return { provider: 'Steam', id: appId };
  }
  if (gogId && gogId !== '0') {
    return { provider: 'GOG', id: gogId };
  }
  return { provider: 'None', id: '' };
}

export function getPlatformType(game) {
  return getGameSource(game);
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

/**
 * Returns human-friendly, player-centric readiness status and badge styles
 */
export function getGameReadinessLabel(game, isRunning = false) {
  if (!game) return { label: 'Unknown', type: 'unknown', badgeClass: 'bg-slate-800 text-slate-400' };

  if (isRunning) {
    return {
      label: 'Playing',
      type: 'running',
      badgeClass: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40',
    };
  }

  if (game.hasUpdatesOrIssues || game.hasUpdate) {
    return {
      label: 'Update Available',
      type: 'update',
      badgeClass: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40',
    };
  }

  if (game.isPatched || game.isReady) {
    if (game.isPortable) {
      return {
        label: 'Ready to Play (Standalone)',
        type: 'ready',
        badgeClass: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/40',
      };
    }
    return {
      label: 'Ready to Play',
      type: 'ready',
      badgeClass: 'bg-sky-500/20 text-sky-300 border-sky-500/40',
    };
  }

  if (isSteamApp(game.appId) && !game.isPatched) {
    return {
      label: 'Setup Required',
      type: 'setup_required',
      badgeClass: 'bg-amber-500/20 text-amber-300 border-amber-500/40',
    };
  }

  return {
    label: 'Ready (DRM-Free)',
    type: 'ready',
    badgeClass: 'bg-slate-700/40 text-slate-300 border-slate-600/50',
  };
}

/**
 * Returns clear, non-jargony descriptions for compatibility & emulation modes
 */
export function getOperatingModeDescriptor(mode) {
  const m = (mode || '').toLowerCase();
  switch (m) {
    case 'loader':
      return {
        value: 'loader',
        label: 'Seamless (Recommended)',
        tag: 'Non-Invasive',
        description: 'Leaves game directory files pristine; injects Steam emulation cleanly at launch.',
      };
    case 'portable':
      return {
        value: 'portable',
        label: 'Standalone Portable',
        tag: 'Direct File Injection',
        description: 'Replaces Steam API files directly inside the game folder for standalone portability.',
      };
    case 'none':
    case 'standalone':
      return {
        value: 'none',
        label: 'Direct Launch',
        tag: 'DRM-Free / No Emulation',
        description: 'Runs directly without any Steamworks emulator (ideal for GOG or native Linux titles).',
      };
    default:
      return {
        value: '',
        label: 'Default (Inherited)',
        tag: 'Global Setting',
        description: 'Inherits the global operating mode configured in Settings.',
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

export function formatPlaytime(seconds) {
  if (!seconds || seconds <= 0) return 'Never played';
  const mins = Math.floor(seconds / 60);
  if (mins < 60) return `${mins} min${mins === 1 ? '' : 's'}`;
  const hours = Number((seconds / 3600).toFixed(1));
  return `${hours} hr${hours === 1 ? '' : 's'}`;
}

export function formatLastPlayed(timestamp) {
  if (!timestamp || timestamp <= 0) return '';
  const now = Math.floor(Date.now() / 1000);
  const diff = now - timestamp;
  if (diff < 60) return 'Just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  const date = new Date(timestamp * 1000);
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

