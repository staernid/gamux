import { describe, it, expect } from 'vitest';
import {
  normalizePath,
  isSamePath,
  normalizeGame,
  getInitials,
  isSteamApp,
  getPlatformType,
  getPlatformColors,
} from './game.js';

describe('Game Data Model & Normalization Helpers', () => {
  it('normalizes file paths correctly', () => {
    expect(normalizePath('/home/user/games/cyberpunk/')).toBe('/home/user/games/cyberpunk');
    expect(normalizePath('C:\\Games\\Cyberpunk\\')).toBe('C:/Games/Cyberpunk');
    expect(normalizePath('///a///b///')).toBe('/a/b');
    expect(normalizePath('')).toBe('');
    expect(normalizePath(null)).toBe('');
  });

  it('compares paths with case and slash normalization', () => {
    expect(isSamePath('/home/user/games', '/home/user/games/')).toBe(true);
    expect(isSamePath('/home/user/games/a', '/home/user/games/b')).toBe(false);
    expect(isSamePath('/home/user/games/a', '')).toBe(false);
    expect(isSamePath('', null)).toBe(true);
    expect(isSamePath(null, null)).toBe(true);
  });

  it('normalizes raw Go/backend game entity into canonical camelCase model', () => {
    const rawBackendGame = {
      Name: 'Cyberpunk 2077',
      AppID: '1091500',
      GameDir: '/home/user/Games/Cyberpunk 2077/',
      Store: 'Steam',
      Platform: 'win64',
      DiskSizeBytes: 73014444032,
      file_count: 1420,
      dlc_count: 3,
      achievement_count: 44,
      state: 'Portable-Patched',
      LaunchCandidates: [
        { Name: 'Main Game', Executable: 'bin/x64/Cyberpunk2077.exe', Arguments: '-vulkan' },
        { Name: 'Technical Loader', Executable: 'steamclient_loader.exe', Arguments: '' }
      ],
      LaunchProfile: {
        exe_path: 'bin/x64/Cyberpunk2077.exe',
        exe_args: '-vulkan',
        operating_mode: 'standalone',
        runner: 'wine',
        wine_prefix: '~/.wine'
      }
    };

    const normalized = normalizeGame(rawBackendGame);

    // Canonical camelCase properties
    expect(normalized.name).toBe('Cyberpunk 2077');
    expect(normalized.appId).toBe('1091500');
    expect(normalized.gameDir).toBe('/home/user/Games/Cyberpunk 2077');
    expect(normalized.store).toBe('Steam');
    expect(normalized.platform).toBe('win64');
    expect(normalized.exePath).toBe('bin/x64/Cyberpunk2077.exe');
    expect(normalized.exeArgs).toBe('-vulkan');
    expect(normalized.isPatched).toBe(true);
    expect(normalized.isPortable).toBe(true);
    expect(normalized.isReady).toBe(true);
    expect(normalized.diskSizeBytes).toBe(73014444032);
    expect(normalized.fileCount).toBe(1420);
    expect(normalized.dlcCount).toBe(3);
    expect(normalized.achievementCount).toBe(44);
    expect(normalized.operatingMode).toBe('standalone');
    expect(normalized.runner).toBe('wine');
    expect(normalized.winePrefix).toBe('~/.wine');

    // Technical loader should be filtered out from candidates
    expect(normalized.launchCandidates.length).toBe(1);
    expect(normalized.launchCandidates[0].name).toBe('Main Game');

    // Seamless backwards-compatibility aliases
    expect(normalized.GameDir).toBe('/home/user/Games/Cyberpunk 2077');
    expect(normalized.game_dir).toBe('/home/user/Games/Cyberpunk 2077');
    expect(normalized.AppID).toBe('1091500');
    expect(normalized.app_id).toBe('1091500');
    expect(normalized.DiskSizeBytes).toBe(73014444032);
    expect(normalized.disk_size_bytes).toBe(73014444032);
  });

  it('calculates game title initials', () => {
    expect(getInitials('Cyberpunk 2077')).toBe('C2');
    expect(getInitials('Baldur\'s Gate 3')).toBe('BG');
    expect(getInitials('Celeste')).toBe('CE');
    expect(getInitials('')).toBe('GX');
  });

  it('detects steam AppIDs accurately', () => {
    expect(isSteamApp('1091500')).toBe(true);
    expect(isSteamApp(1091500)).toBe(true);
    expect(isSteamApp('0')).toBe(false);
    expect(isSteamApp('Custom')).toBe(false);
    expect(isSteamApp('')).toBe(false);
    expect(isSteamApp(null)).toBe(false);
  });

  it('determines platform colors and types', () => {
    expect(getPlatformType({ store: 'Steam', app_id: '1091500' })).toBe('Steam');
    expect(getPlatformType({ store: 'GOG' })).toBe('GOG');
    expect(getPlatformType({ store: 'Standalone' })).toBe('Standalone');

    const steamColors = getPlatformColors({ store: 'Steam' });
    expect(steamColors.label).toBe('Steam');
    expect(steamColors.border).toBe('#1a9fff');
  });
});
