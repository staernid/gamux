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
  const isPatched = Boolean(g.is_patched ?? (g.state === 'Portable-Patched' || g.state === 'Loader-Configured' || g.settings_dir_exists));
  const isPortable = Boolean(g.is_portable ?? (g.state === 'Portable-Patched'));
  const isReady = Boolean(g.is_ready ?? (isPatched && (g.lutris_registered || g.operating_mode === 'standalone')));
  const hasUpdatesOrIssues = Boolean(g.has_updates_or_issues ?? (store === 'Steam' && (g.has_update || (g.modified_files && g.modified_files.length > 0) || (g.missing_files && g.missing_files.length > 0))));
  const lutrisRegistered = Boolean(g.lutris_registered ?? false);
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
    lutrisRegistered,
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
    lutris_registered: lutrisRegistered,
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
