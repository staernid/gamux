// Game data normalization and status helpers

export function normalizeGame(g) {
  if (!g) return null;
  const name = g.name || g.Name || 'Unknown Game';
  const appId = g.app_id || g.AppID || '';
  const store = g.store || g.Store || 'Steam';
  const gameDir = g.game_dir || g.GameDir || '';
  const platform = g.platform || g.Platform || 'win64';
  let exePath = g.exe_path || g.ExePath || '';
  if (exePath && (exePath.toLowerCase().includes('steamclient_loader') || exePath.toLowerCase().includes('coldclientloader'))) {
    exePath = '';
  }

  const exeArgs = g.exe_args || g.ExeArgs || '';
  const launchCandidates = (g.launch_candidates || g.LaunchCandidates || [])
    .filter(c => {
      const exe = (c.executable || c.Executable || '').toLowerCase();
      return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
    })
    .map(c => ({
      name: c.name || c.Name || '',
      executable: c.executable || c.Executable || '',
      arguments: c.arguments || c.Arguments || '',
      description: c.description || c.Description || '',
    }));

  if (!exePath && launchCandidates.length > 0) {
    exePath = launchCandidates[0].executable;
  }
  const stateVal = g.state || g.State || 'Original';
  const originalBackups = g.original_backups || g.OriginalBackups || [];
  const settingsDirExists = Boolean(g.settings_dir_exists ?? g.SettingsDirExists ?? false);
  const steamAppIdTxtFound = Boolean(g.steam_appid_txt_found ?? g.SteamAppIDTxtFound ?? false);
  const manifestId = g.manifest_id || g.ManifestID || '';
  const buildId = g.build_id || g.BuildID || '';
  const diskSizeBytes = Number(g.disk_size_bytes ?? g.DiskSizeBytes ?? 0);
  const fileCount = Number(g.file_count ?? g.FileCount ?? 0);
  const officialFileCount = Number(g.official_file_count ?? g.OfficialFileCount ?? 0);
  const modifiedFiles = g.modified_files || g.ModifiedFiles || [];
  const missingFiles = g.missing_files || g.MissingFiles || [];
  const untrackedFiles = g.untracked_files || g.UntrackedFiles || [];
  const redistModified = g.redist_modified || g.RedistModified || [];
  const redistMissing = g.redist_missing || g.RedistMissing || [];
  const hasUpdate = Boolean(g.has_update ?? g.HasUpdate ?? false);
  const remoteManifestId = g.remote_manifest_id || g.RemoteManifestID || '';
  const dlcCount = Number(g.dlc_count ?? g.DLCCount ?? 0);
  const achievementCount = Number(g.achievement_count ?? g.AchievementCount ?? 0);
  const lutrisRegistered = Boolean(g.lutris_registered ?? g.LutrisRegistered ?? false);
  const recentPatchNote = g.recent_patch_note || g.RecentPatchNote || '';
  const newsItems = (g.news_items || g.NewsItems || []).map(item => ({
    title: item.title || item.Title || 'Steam Update Note',
    url: item.url || item.URL || '',
    author: item.author || item.Author || '',
    contents: item.contents || item.Contents || '',
    feed_label: item.feed_label || item.feedlabel || item.FeedLabel || 'Steam News',
    date: item.date || item.Date || 0,
  }));

  const isPatched = stateVal.toLowerCase().includes('portable') ||
                    stateVal.toLowerCase().includes('loader') ||
                    settingsDirExists;
  const isPortable = stateVal.toLowerCase().includes('portable');
  const hasUpdatesOrIssues = hasUpdate || modifiedFiles.length > 0 || missingFiles.length > 0;

  return {
    ...g,
    name, Name: name,
    app_id: appId, AppID: appId,
    store, Store: store,
    game_dir: gameDir, GameDir: gameDir,
    platform, Platform: platform,
    exe_path: exePath, ExePath: exePath,
    exe_args: exeArgs, ExeArgs: exeArgs,
    launch_candidates: launchCandidates, LaunchCandidates: launchCandidates,
    state: stateVal, State: stateVal,
    isPatched,
    isPortable,
    hasUpdatesOrIssues,
    original_backups: originalBackups, OriginalBackups: originalBackups,
    settings_dir_exists: settingsDirExists, SettingsDirExists: settingsDirExists,
    steam_appid_txt_found: steamAppIdTxtFound, SteamAppIDTxtFound: steamAppIdTxtFound,
    manifest_id: manifestId, ManifestID: manifestId,
    build_id: buildId, BuildID: buildId,
    disk_size_bytes: diskSizeBytes, DiskSizeBytes: diskSizeBytes,
    file_count: fileCount, FileCount: fileCount,
    official_file_count: officialFileCount, OfficialFileCount: officialFileCount,
    modified_files: modifiedFiles, ModifiedFiles: modifiedFiles,
    missing_files: missingFiles, MissingFiles: missingFiles,
    untracked_files: untrackedFiles, UntrackedFiles: untrackedFiles,
    redist_modified: redistModified, RedistModified: redistModified,
    redist_missing: redistMissing, RedistMissing: redistMissing,
    has_update: hasUpdate, HasUpdate: hasUpdate,
    remote_manifest_id: remoteManifestId, RemoteManifestID: remoteManifestId,
    dlc_count: dlcCount, DLCCount: dlcCount,
    achievement_count: achievementCount, AchievementCount: achievementCount,
    lutris_registered: lutrisRegistered, LutrisRegistered: lutrisRegistered,
    recent_patch_note: recentPatchNote, RecentPatchNote: recentPatchNote,
    news_items: newsItems, NewsItems: newsItems,
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

