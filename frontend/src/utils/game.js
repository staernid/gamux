import { formatTimeAgo } from './format.js';

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
  const a = normalizePath(pathA);
  const b = normalizePath(pathB);
  if (a === b) return true;
  const aLower = a.toLowerCase();
  const bLower = b.toLowerCase();
  if (aLower === bLower) return true;
  if (aLower.endsWith('/' + bLower) || bLower.endsWith('/' + aLower)) return true;
  return false;
}

export function filterLaunchCandidates(candidates = []) {
  return (candidates || [])
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
}

const LAUNCH_PROFILE_SNAKE_TO_CAMEL = {
  game_dir: 'gameDir',
  exe_path: 'exePath',
  exe_args: 'exeArgs',
  app_id: 'appId',
  operating_mode: 'operatingMode',
  wine_prefix: 'winePrefix',
  proton_path: 'protonPath',
  dll_overrides: 'dllOverrides',
  mangohud: 'mangoHud',
  mango_hud: 'mangoHud',
  disable_overlay: 'disableOverlay',
  offline_mode: 'offlineMode',
  custom_env: 'customEnv',
  source_override: 'sourceOverride',
  metadata_provider: 'metadataProvider',
  metadata_id: 'metadataId',
  store_app_id: 'storeAppId',
  gog_id: 'gogId',
  gamescope_width: 'gamescopeWidth',
  gamescope_height: 'gamescopeHeight',
  gamescope_fps: 'gamescopeFps',
  working_dir: 'workingDir',
  profile_id: 'profileId',
  force_emulation: 'forceEmulation',
  playtime_seconds: 'playtimeSeconds',
  last_played: 'lastPlayed',
  installed_at: 'installedAt',
  custom_cover_url: 'customCoverUrl',
  custom_banner_url: 'customBannerUrl',
  custom_hero_url: 'customHeroUrl',
  custom_logo_url: 'customLogoUrl',
  custom_icon_url: 'customIconUrl',
  locked_artwork: 'lockedArtwork',
  branch_password: 'branchPassword',
  inhibit_sleep: 'inhibitSleep',
  prefer_sdl: 'preferSdl',
  enable_wayland: 'enableWayland',
};

const LAUNCH_PROFILE_CAMEL_TO_SNAKE = {
  gameDir: 'game_dir',
  exePath: 'exe_path',
  exeArgs: 'exe_args',
  appId: 'app_id',
  operatingMode: 'operating_mode',
  winePrefix: 'wine_prefix',
  protonPath: 'proton_path',
  dllOverrides: 'dll_overrides',
  mangoHud: 'mangohud',
  disableOverlay: 'disable_overlay',
  offlineMode: 'offline_mode',
  customEnv: 'custom_env',
  sourceOverride: 'source_override',
  metadataProvider: 'metadata_provider',
  metadataId: 'metadata_id',
  storeAppId: 'store_app_id',
  gogId: 'gog_id',
  gamescopeWidth: 'gamescope_width',
  gamescopeHeight: 'gamescope_height',
  gamescopeFps: 'gamescope_fps',
  workingDir: 'working_dir',
  profileId: 'profile_id',
  forceEmulation: 'force_emulation',
  playtimeSeconds: 'playtime_seconds',
  lastPlayed: 'last_played',
  installedAt: 'installed_at',
  customCoverUrl: 'custom_cover_url',
  customBannerUrl: 'custom_banner_url',
  customHeroUrl: 'custom_hero_url',
  customLogoUrl: 'custom_logo_url',
  customIconUrl: 'custom_icon_url',
  lockedArtwork: 'locked_artwork',
  branchPassword: 'branch_password',
  inhibitSleep: 'inhibit_sleep',
  preferSdl: 'prefer_sdl',
  enableWayland: 'enable_wayland',
};

export function snakeToCamelKey(key) {
  if (LAUNCH_PROFILE_SNAKE_TO_CAMEL[key]) return LAUNCH_PROFILE_SNAKE_TO_CAMEL[key];
  return key.replace(/_([a-z0-9])/g, (_, letter) => letter.toUpperCase());
}

export function camelToSnakeKey(key) {
  if (LAUNCH_PROFILE_CAMEL_TO_SNAKE[key]) return LAUNCH_PROFILE_CAMEL_TO_SNAKE[key];
  return key.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
}

export function normalizeLaunchProfile(raw) {
  if (!raw || typeof raw !== 'object') return null;
  const out = {};
  for (const [k, v] of Object.entries(raw)) {
    if (v === undefined) continue;
    const camel = snakeToCamelKey(k);
    if ((camel === 'gameDir' || camel === 'exePath' || camel === 'winePrefix' || camel === 'protonPath' || camel === 'workingDir') && typeof v === 'string') {
      out[camel] = normalizePath(v);
    } else if (camel === 'customEnv' && v && typeof v === 'object') {
      out[camel] = { ...v };
    } else {
      out[camel] = v;
    }
  }
  return out;
}

export function getExecutablePath(game, candidates = []) {
  if (!game) return '';
  const lp = normalizeLaunchProfile(game.launch_profile || game.launchProfile || game.LaunchProfile || null);
  let rawExe = lp?.exePath || game.exe_path || game.exePath || game.ExePath || '';
  if (rawExe && (rawExe.toLowerCase().includes('steamclient_loader') || rawExe.toLowerCase().includes('coldclientloader'))) {
    rawExe = candidates[0]?.executable || '';
  }
  if (!rawExe && candidates.length > 0) {
    rawExe = candidates[0]?.executable || '';
  }
  return normalizePath(rawExe);
}

export function normalizeRunningInstance(i) {
  if (!i) return null;
  const gameDir = normalizePath(i.gameDir ?? i.game_dir ?? '');
  const profileId = i.profileId ?? i.profile_id ?? 'main';
  const seatId = i.seatId ?? i.seat_id ?? '';
  const appId = String(i.appId ?? i.app_id ?? '');
  const gameName = i.gameName ?? i.game_name ?? '';
  const pid = Number(i.pid ?? 0);
  const isMultiseat = Boolean(i.isMultiseat ?? i.is_multiseat ?? Boolean(seatId));
  const startTime = Number(i.startTime ?? i.start_time ?? Math.floor(Date.now() / 1000));
  const mode = i.mode ?? '';

  let instanceId = i.instanceId ?? i.instance_id ?? '';
  if (!instanceId) {
    if (seatId) {
      instanceId = `${appId || gameDir}:${profileId}:${seatId}`;
    } else if (profileId && profileId !== 'main') {
      instanceId = `${appId || gameDir}:${profileId}`;
    } else {
      instanceId = appId || gameDir;
    }
  }

  return {
    instanceId,
    gameDir,
    gameName,
    appId,
    profileId,
    seatId,
    pid,
    isMultiseat,
    startTime,
    mode
  };
}

export function normalizeGame(g) {
  if (!g) return null;

  // Filter launch candidates to exclude technical loader helpers
  const rawCandidates = g.launch_candidates || g.launchCandidates || g.LaunchCandidates || [];
  const launchCandidates = filterLaunchCandidates(rawCandidates);

  const lp = normalizeLaunchProfile(g.launch_profile || g.launchProfile || g.LaunchProfile || null);
  const rawExe = getExecutablePath(g, launchCandidates);

  const isPatched = Boolean(
    g.is_patched ?? g.isPatched ?? g.IsPatched ??
    (g.state === 'Portable-Patched' || g.state === 'Loader-Configured' || g.settings_dir_exists)
  );

  const fileCount = Number(g.file_count ?? g.fileCount ?? g.FileCount ?? 0);
  const isInstalled = Boolean(
    g.is_installed ?? g.isInstalled ?? g.IsInstalled ??
    (Boolean(rawExe) || fileCount > 0)
  );
  const isStub = Boolean(
    g.is_stub ?? g.isStub ?? g.IsStub ??
    (!isInstalled && Boolean(g.game_dir ?? g.gameDir ?? g.GameDir ?? ''))
  );
  const isVirtual = Boolean(g.is_virtual ?? g.isVirtual ?? (!isInstalled && !isStub && !g.game_dir && !g.gameDir));

  return {
    name: g.name || g.Name || 'Unknown Game',
    appId: String(g.app_id ?? g.appId ?? g.AppID ?? ''),
    gameDir: normalizePath(g.game_dir ?? g.gameDir ?? g.GameDir ?? ''),
    store: g.store || g.Store || 'Steam',
    source: g.source || g.Source || g.store || g.Store || 'Steam',
    gogId: String(g.gog_id ?? g.gogId ?? g.GOGID ?? lp?.gogId ?? ''),
    metadataProvider: lp?.metadataProvider || g.metadata_provider || g.metadataProvider || '',
    metadataId: String(lp?.metadataId ?? g.metadata_id ?? g.metadataId ?? ''),
    platform: g.platform || g.Platform || 'win64',
    exePath: normalizePath(rawExe),
    exeArgs: lp?.exeArgs || g.exe_args || g.exeArgs || g.ExeArgs || '',
    launchCandidates,
    launchProfile: lp,
    branch: g.branch || g.Branch || lp?.branch || 'public',
    branchPassword: lp?.branchPassword || '',
    isInstalled,
    isStub,
    isVirtual,
    isPatched,
    isPortable: Boolean(g.is_portable ?? g.isPortable ?? g.IsPortable ?? (g.state === 'Portable-Patched')),
    isReady: isInstalled && Boolean(g.is_ready ?? g.isReady ?? g.IsReady ?? (isPatched || !!rawExe)),
    hasUpdatesOrIssues: Boolean(
      g.has_updates_or_issues ?? g.hasUpdatesOrIssues ?? g.HasUpdatesOrIssues ??
      (g.store === 'Steam' && (g.has_update || (g.modified_files?.length > 0) || (g.missing_files?.length > 0)))
    ),
    diskSizeBytes: Number(g.disk_size_bytes ?? g.diskSizeBytes ?? g.DiskSizeBytes ?? 0),
    fileCount: Number(g.file_count ?? g.fileCount ?? g.FileCount ?? 0),
    officialFileCount: Number(g.official_file_count ?? g.officialFileCount ?? g.OfficialFileCount ?? 0),
    dlcCount: Number(g.dlc_count ?? g.dlcCount ?? g.DLCCount ?? 0),
    achievementCount: Number(g.achievement_count ?? g.achievementCount ?? g.AchievementCount ?? 0),
    operatingMode: lp?.operatingMode || g.operating_mode || g.operatingMode || '',
    runner: lp?.runner || g.runner || g.Runner || '',
    protonPath: normalizePath(lp?.protonPath || g.proton_path || g.protonPath || g.ProtonPath || ''),
    winePrefix: normalizePath(lp?.winePrefix || g.wine_prefix || g.winePrefix || g.WinePrefix || ''),
    currentLanguage: g.current_language || g.currentLanguage || '',
    supportedLanguages: g.supported_languages || g.supportedLanguages || [],
    modifiedFiles: g.modified_files || g.modifiedFiles || [],
    missingFiles: g.missing_files || g.missingFiles || [],
    untrackedFiles: g.untracked_files || g.untrackedFiles || [],
    hasUpdate: Boolean(g.has_update ?? g.hasUpdate ?? false),
    versionStatus: g.version_status || g.versionStatus || null,
    manifestId: g.manifest_id || g.manifestId || '',
    buildId: g.build_id || g.buildId || '',
    remoteManifestId: g.remote_manifest_id || g.remoteManifestId || '',
    recentPatchNote: g.recent_patch_note || g.recentPatchNote || '',
    newsItems: g.news_items || g.newsItems || [],
    coverUrl: g.cover_url || g.coverUrl || g.custom_cover_url || g.customCoverUrl || lp?.customCoverUrl || '',
    localCoverUrl: g.local_cover_url || g.localCoverUrl || g.local_cover || g.localCover || '',
    heroUrl: g.hero_url || g.heroUrl || g.custom_hero_url || g.customHeroUrl || lp?.customHeroUrl || '',
    localHeroUrl: g.local_hero_url || g.localHeroUrl || g.local_hero || g.localHero || '',
    bannerUrl: g.banner_url || g.bannerUrl || g.custom_banner_url || g.customBannerUrl || lp?.customBannerUrl || '',
    localBannerUrl: g.local_banner_url || g.localBannerUrl || g.local_banner || g.localBanner || '',
    logoUrl: g.logo_url || g.logoUrl || g.custom_logo_url || g.customLogoUrl || lp?.customLogoUrl || '',
    localLogoUrl: g.local_logo_url || g.localLogoUrl || g.local_logo || g.localLogo || '',
    customCoverUrl: g.custom_cover_url || g.customCoverUrl || lp?.customCoverUrl || '',
    customBannerUrl: g.custom_banner_url || g.customBannerUrl || lp?.customBannerUrl || '',
    customHeroUrl: g.custom_hero_url || g.customHeroUrl || lp?.customHeroUrl || '',
    customLogoUrl: g.custom_logo_url || g.customLogoUrl || lp?.customLogoUrl || '',
    customIconUrl: g.custom_icon_url || g.customIconUrl || lp?.customIconUrl || '',
    lockedArtwork: Boolean(g.locked_artwork ?? g.lockedArtwork ?? lp?.lockedArtwork ?? false),
    installedAt: Number(g.installed_at ?? g.installedAt ?? lp?.installedAt ?? 0),
    playtimeSeconds: Number(g.playtime_seconds ?? g.playtimeSeconds ?? lp?.playtimeSeconds ?? 0),
    lastPlayed: Number(g.last_played ?? g.lastPlayed ?? lp?.lastPlayed ?? 0),
    summary: g.summary || g.Summary || lp?.summary || '',
    releaseDate: g.release_date || g.releaseDate || '',
    developers: g.developers || g.Developers || [],
    publishers: g.publishers || g.Publishers || [],
    genres: g.genres || g.Genres || [],
    modTopology: (g.mod_topology || g.modTopology || g.ModTopology) ? {
      engine: (g.mod_topology || g.modTopology || g.ModTopology).engine || (g.mod_topology || g.modTopology || g.ModTopology).Engine || 'generic',
      modDir: (g.mod_topology || g.modTopology || g.ModTopology).mod_dir || (g.mod_topology || g.modTopology || g.ModTopology).modDir || (g.mod_topology || g.modTopology || g.ModTopology).ModDir || 'mods',
      absoluteModDir: (g.mod_topology || g.modTopology || g.ModTopology).absolute_mod_dir || (g.mod_topology || g.modTopology || g.ModTopology).absoluteModDir || (g.mod_topology || g.modTopology || g.ModTopology).AbsoluteModDir || '',
      detected: Boolean((g.mod_topology || g.modTopology || g.ModTopology).detected ?? (g.mod_topology || g.modTopology || g.ModTopology).Detected ?? false),
      supportedFormats: (g.mod_topology || g.modTopology || g.ModTopology).supported_formats || (g.mod_topology || g.modTopology || g.ModTopology).supportedFormats || (g.mod_topology || g.modTopology || g.ModTopology).SupportedFormats || [],
    } : null,
    mods: Array.isArray(g.mods || g.Mods) ? (g.mods || g.Mods).map(m => ({
      id: String(m.id || m.ID || ''),
      name: m.name || m.Name || '',
      workshopId: Number(m.workshop_id ?? m.workshopId ?? m.WorkshopID ?? 0),
      enabled: Boolean(m.enabled ?? m.Enabled ?? true),
      path: normalizePath(m.path || m.Path || ''),
      relativePath: normalizePath(m.relative_path || m.relativePath || m.RelativePath || ''),
      timeUpdated: Number(m.time_updated ?? m.timeUpdated ?? m.TimeUpdated ?? 0),
      latestTimeUpdated: Number(m.latest_time_updated ?? m.latestTimeUpdated ?? m.LatestTimeUpdated ?? 0),
      hasUpdate: Boolean(m.has_update ?? m.hasUpdate ?? m.HasUpdate ?? false),
      description: m.description || m.Description || '',
      previewUrl: m.preview_url || m.previewUrl || m.PreviewURL || '',
      size: Number(m.size ?? m.Size ?? 0),
      files: m.files || m.Files || [],
    })) : [],
    outdatedModsCount: Number(
      g.outdated_mods_count ?? g.outdatedModsCount ?? g.OutdatedModsCount ??
      (Array.isArray(g.mods || g.Mods) ? (g.mods || g.Mods).filter(m => Boolean(m.has_update ?? m.hasUpdate ?? m.HasUpdate)).length : 0)
    ),
  };
}

export function normalizeDlc(raw) {
  if (!raw) return null;
  return {
    appId: String(raw.app_id ?? ''),
    name: raw.name || '',
  };
}

export function normalizeMetadataProvider(raw) {
  if (!raw) return null;
  return {
    id: raw.id || '',
    displayName: raw.display_name || '',
    description: raw.description || '',
    capabilities: raw.capabilities || {},
  };
}

export function normalizeGameMetadata(raw) {
  if (!raw) return null;
  return {
    name: raw.name || '',
    shortDescription: raw.short_description || '',
    detailedDescription: raw.detailed_description || '',
    aboutTheGame: raw.about_the_game || '',
    headerImage: raw.header_image || '',
    developers: Array.isArray(raw.developers) ? raw.developers : [],
    publishers: Array.isArray(raw.publishers) ? raw.publishers : [],
    genres: Array.isArray(raw.genres) ? raw.genres : [],
    releaseDate: raw.release_date || '',
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

export function getThumbnailUrl(url, width = 300, appId = '') {
  const w = Number(width) > 0 ? Number(width) : 300;
  if (url && typeof url === 'string') {
    const trimmed = url.trim();
    if (trimmed.startsWith('/art/thumb')) {
      return trimmed;
    }
    if (trimmed.startsWith('/art/')) {
      const qIdx = trimmed.indexOf('?');
      let pathPart = qIdx !== -1 ? trimmed.slice(5, qIdx) : trimmed.slice(5);
      const queryPart = qIdx !== -1 ? trimmed.slice(qIdx + 1) : '';
      pathPart = pathPart.replace(/^\/+/, '');
      if (pathPart) {
        let result = `/art/thumb?id=${encodeURIComponent(pathPart)}&w=${w}`;
        if (queryPart) {
          result += `&${queryPart}`;
        }
        return result;
      }
    }
    if (trimmed.startsWith('/local-img/')) {
      return `/art/thumb?id=${encodeURIComponent(trimmed)}&w=${w}`;
    }
  }
  if (appId && appId !== '0' && appId !== 'Custom' && /^\d+$/.test(String(appId).trim())) {
    return `/art/thumb?id=${encodeURIComponent(String(appId).trim())}&w=${w}`;
  }
  return '';
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
  const lp = normalizeLaunchProfile(game.launchProfile || game.launch_profile || null);
  const provider = (game.metadataProvider || game.metadata_provider || lp?.metadataProvider || '').toLowerCase();
  if (provider === 'none' || provider === 'standalone' || provider === 'custom') {
    return { provider: 'None', id: '' };
  }
  const metaId = String(game.metadataId || game.metadata_id || lp?.metadataId || '');
  const appId = String(game.appId || game.app_id || '');
  const gogId = String(game.gogId || game.gog_id || lp?.gogId || '');

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

export function hasMetadataMatch(game) {
  const info = getMetadataInfo(game);
  return info.provider !== 'None' && Boolean(info.id && info.id !== '0' && info.id !== 'undefined' && info.id !== 'null');
}

export function countOverrides(draft, game = null, globalCfg = {}) {
  if (!draft) return 0;
  let count = 0;

  const defaultExe = game?.launchCandidates?.[0]?.executable || game?.exePath || '';
  const defaultArgs = game?.launchCandidates?.[0]?.arguments || '';

  const defaultOperatingMode = globalCfg?.operating_mode || 'loader';
  const defaultRunner = globalCfg?.default_runner || 'umu';
  const defaultLanguage = globalCfg?.language || 'auto';
  const defaultMetadataProvider = (game?.store || game?.source || (isSteamApp(game?.appId) ? 'steam' : 'none')).toLowerCase();

  const fieldRules = [
    // 1. Title Override
    () => Boolean(draft.name?.trim() && draft.name.trim() !== (game?.name || '').trim()),
    // 2. Target Platform Override
    () => Boolean(draft.platform && game?.platform && draft.platform !== game.platform),
    // 3. Executable Path Override
    () => Boolean(draft.exePath && defaultExe && normalizePath(draft.exePath) !== normalizePath(defaultExe)),
    // 4. Executable Arguments Override
    () => Boolean(draft.exeArgs?.trim() && draft.exeArgs.trim() !== defaultArgs.trim()),
    // 5. Operating Mode Override
    () => Boolean(draft.operatingMode?.trim() && draft.operatingMode !== defaultOperatingMode),
    // 6. Runner Layer Override
    () => Boolean(draft.runner?.trim() && draft.runner !== defaultRunner),
    // 7. Wine Prefix Override
    () => Boolean(draft.winePrefix?.trim()),
    // 8. Proton Path Override
    () => Boolean(draft.protonPath?.trim()),
    // 9. DLL Overrides
    () => Boolean(draft.dllOverrides?.trim()),
    // 10. MangoHud Overlay Override
    () => draft.mangoHud !== null && draft.mangoHud !== undefined && draft.mangoHud !== Boolean(globalCfg?.mangohud),
    // 11. Disable Overlay Override
    () => draft.disableOverlay !== null && draft.disableOverlay !== undefined && draft.disableOverlay !== Boolean(globalCfg?.disable_overlay),
    // 12. Offline Mode Override
    () => draft.offlineMode !== null && draft.offlineMode !== undefined && draft.offlineMode !== Boolean(globalCfg?.offline_mode),
    // 13. Game Language Override
    () => Boolean(draft.language?.trim() && draft.language !== 'auto' && draft.language !== defaultLanguage),
    // 14. Physical Installation Source Override
    () => Boolean(draft.sourceOverride?.trim()),
    // 15. Store Metadata Provider Override
    () => Boolean(draft.metadataProvider?.trim() && draft.metadataProvider.toLowerCase() !== 'none' && draft.metadataProvider.toLowerCase() !== defaultMetadataProvider),
    // 16. Gamescope & Display Overrides
    () => draft.gamescope !== null && draft.gamescope !== undefined && draft.gamescope !== Boolean(globalCfg?.gamescope),
    () => Boolean(draft.gamescopeWidth && draft.gamescopeWidth !== (globalCfg?.gamescope_width || 0)),
    () => Boolean(draft.gamescopeHeight && draft.gamescopeHeight !== (globalCfg?.gamescope_height || 0)),
    () => Boolean(draft.gamescopeFps && draft.gamescopeFps !== (globalCfg?.gamescope_fps || 0)),
    // 17. Working Directory Override
    () => Boolean(draft.workingDir?.trim()),
  ];

  for (const rule of fieldRules) {
    if (rule()) count++;
  }

  // 18. Custom / Overridden Environment Variables
  if (draft.customEnv && typeof draft.customEnv === 'object') {
    count += Object.keys(draft.customEnv).length;
  }

  return count;
}

export const getPlatformType = getGameSource;

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
    img.onerror = () => {
      coverPaletteCache.set(url, []);
      resolve([]);
    };
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
  return formatTimeAgo(timestamp);
}

export function formatInstallDate(timestamp) {
  return formatTimeAgo(timestamp);
}

export function isPlaceholderTitle(name, appId) {
  if (!name || typeof name !== 'string') return true;
  const trimmed = name.trim();
  if (!trimmed || /^unknown(\s*game)?$/i.test(trimmed)) return true;
  const cleanAppId = appId ? String(appId).trim() : '';
  if (cleanAppId && (trimmed === cleanAppId || new RegExp(`^(steam\\s*app(id)?|game)\\s*${cleanAppId}$`, 'i').test(trimmed))) {
    return true;
  }
  if (/^(steam\s*app(id)?|game)\s*\d+$/i.test(trimmed)) {
    return true;
  }
  if (!cleanAppId && /^\d+$/.test(trimmed)) {
    return true;
  }
  return false;
}

