import { describe, it, expect } from 'vitest';
import {
  normalizePath,
  isSamePath,
  normalizeGame,
  normalizeDlc,
  normalizeMetadataProvider,
  normalizeGameMetadata,
  normalizeRunningInstance,
  getInitials,
  isSteamApp,
  getPlatformType,
  getPlatformColors,
  formatPlaytime,
  formatLastPlayed,
  hasMetadataMatch,
  getMetadataInfo,
  filterLaunchCandidates,
  getExecutablePath,
  countOverrides,
  isPlaceholderTitle,
  getThumbnailUrl,
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
    expect(isSamePath('/home/user/games/How to Fish/How to Fish.exe', 'How to Fish/How to Fish.exe')).toBe(true);
    expect(isSamePath('Binaries/Win64/game.exe', '/games/MyGame/Binaries/Win64/Game.exe')).toBe(true);
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
      installed_at: 1700000000,
      playtime_seconds: 7200,
      last_played: 1720000000,
      summary: 'An open-world RPG set in Night City.',
      release_date: 'Dec 10, 2020',
      developers: ['CD PROJEKT RED'],
      publishers: ['CD PROJEKT RED'],
      genres: ['RPG', 'Action'],
      LaunchCandidates: [
        { Name: 'Main Game', Executable: 'bin/x64/Cyberpunk2077.exe', Arguments: '-vulkan' },
        { Name: 'Technical Loader', Executable: 'steamclient_loader.exe', Arguments: '' }
      ],
      LaunchProfile: {
        exe_path: 'bin/x64/Cyberpunk2077.exe',
        exe_args: '-vulkan',
        operating_mode: 'portable',
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
    expect(normalized.operatingMode).toBe('portable');
    expect(normalized.runner).toBe('wine');
    expect(normalized.winePrefix).toBe('~/.wine');
    expect(normalized.installedAt).toBe(1700000000);
    expect(normalized.playtimeSeconds).toBe(7200);
    expect(normalized.lastPlayed).toBe(1720000000);
    expect(normalized.summary).toBe('An open-world RPG set in Night City.');
    expect(normalized.releaseDate).toBe('Dec 10, 2020');
    expect(normalized.developers).toEqual(['CD PROJEKT RED']);
    expect(normalized.publishers).toEqual(['CD PROJEKT RED']);
    expect(normalized.genres).toEqual(['RPG', 'Action']);

    // Technical loader should be filtered out from candidates
    expect(normalized.launchCandidates.length).toBe(1);
    expect(normalized.launchCandidates[0].name).toBe('Main Game');

    // Verify no legacy alias pollution
    expect(normalized.GameDir).toBeUndefined();
    expect(normalized.game_dir).toBeUndefined();
    expect(normalized.AppID).toBeUndefined();
    expect(normalized.app_id).toBeUndefined();
    expect(normalized.DiskSizeBytes).toBeUndefined();
    expect(normalized.disk_size_bytes).toBeUndefined();
  });

  it('normalizes a clean game without launch profile by leaving overrides empty', () => {
    const cleanGame = {
      name: 'Silksong',
      app_id: '1030300',
      game_dir: '/home/user/Games/Silksong',
      store: 'Steam',
      platform: 'linux',
      exe_path: 'Silksong.x86_64',
      state: 'Original',
    };

    const normalized = normalizeGame(cleanGame);
    expect(normalized.operatingMode).toBe('');
    expect(normalized.runner).toBe('');
    expect(normalized.protonPath).toBe('');
  });

  it('normalizes modTopology, mods, and outdatedModsCount properly', () => {
    const rawGame = {
      name: 'Unity Modded Game',
      app_id: '1234',
      game_dir: '/games/unity_game',
      mod_topology: {
        engine: 'unity',
        mod_dir: 'BepInEx/plugins',
        detected: true,
        supported_formats: ['.dll']
      },
      mods: [
        {
          id: 'ModA.dll',
          name: 'Mod A',
          workshop_id: 111,
          enabled: true,
          path: '/games/unity_game/BepInEx/plugins/ModA.dll',
          has_update: true
        }
      ],
      outdated_mods_count: 1
    };

    const normalized = normalizeGame(rawGame);
    expect(normalized.modTopology).toEqual({
      engine: 'unity',
      modDir: 'BepInEx/plugins',
      absoluteModDir: '',
      detected: true,
      supportedFormats: ['.dll']
    });
    expect(normalized.mods.length).toBe(1);
    expect(normalized.mods[0].id).toBe('ModA.dll');
    expect(normalized.mods[0].hasUpdate).toBe(true);
    expect(normalized.outdatedModsCount).toBe(1);
  });

  it('normalizes running instances from raw backend payloads', () => {
    const raw = {
      instance_id: '/games/hl2:p2:p2',
      game_dir: '/games/hl2/',
      game_name: 'Half-Life 2',
      app_id: '220',
      profile_id: 'p2',
      seat_id: 'p2',
      pid: 1234,
      is_multiseat: true,
      start_time: 1700000000,
      mode: 'local'
    };

    const inst = normalizeRunningInstance(raw);
    expect(inst.instanceId).toBe('/games/hl2:p2:p2');
    expect(inst.gameDir).toBe('/games/hl2');
    expect(inst.gameName).toBe('Half-Life 2');
    expect(inst.appId).toBe('220');
    expect(inst.profileId).toBe('p2');
    expect(inst.seatId).toBe('p2');
    expect(inst.pid).toBe(1234);
    expect(inst.isMultiseat).toBe(true);
    expect(inst.startTime).toBe(1700000000);
    expect(inst.mode).toBe('local');

    // Verify no legacy alias pollution
    expect(inst.instance_id).toBeUndefined();
    expect(inst.game_dir).toBeUndefined();
    expect(inst.game_name).toBeUndefined();
    expect(inst.app_id).toBeUndefined();
    expect(inst.profile_id).toBeUndefined();
    expect(inst.seat_id).toBeUndefined();
    expect(inst.is_multiseat).toBeUndefined();
    expect(inst.start_time).toBeUndefined();
  });

  it('generates canonical instanceId for solo and multiseat instances if missing', () => {
    const solo = normalizeRunningInstance({
      game_dir: '/games/portal',
      app_id: '400',
    });
    expect(solo.instanceId).toBe('400');
    expect(solo.profileId).toBe('main');
    expect(solo.seatId).toBe('');
    expect(solo.isMultiseat).toBe(false);

    const seat = normalizeRunningInstance({
      gameDir: '/games/portal',
      appId: '400',
      profileId: 'p2',
      seatId: 'p2',
    });
    expect(seat.instanceId).toBe('400:p2:p2');
    expect(seat.isMultiseat).toBe(true);
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

  it('detects placeholder titles accurately', () => {
    expect(isPlaceholderTitle('', '1458140')).toBe(true);
    expect(isPlaceholderTitle('   ', '1458140')).toBe(true);
    expect(isPlaceholderTitle('Unknown Game', '1458140')).toBe(true);
    expect(isPlaceholderTitle('unknown', '1458140')).toBe(true);
    expect(isPlaceholderTitle('1458140', '1458140')).toBe(true);
    expect(isPlaceholderTitle('Steam App 1458140', '1458140')).toBe(true);
    expect(isPlaceholderTitle('Steam AppID 1458140', '1458140')).toBe(true);
    expect(isPlaceholderTitle('SteamApp 1458140', '1458140')).toBe(true);
    expect(isPlaceholderTitle('Game 1458140', '1458140')).toBe(true);
    expect(isPlaceholderTitle('Steam App 9999', '')).toBe(true);
    expect(isPlaceholderTitle('Pacific Drive', '1458140')).toBe(false);
    expect(isPlaceholderTitle('Half-Life 2', '220')).toBe(false);
    expect(isPlaceholderTitle('Cyberpunk 2077', '1091500')).toBe(false);
  });

  it('determines platform colors and types', () => {
    expect(getPlatformType({ store: 'Steam', app_id: '1091500' })).toBe('Steam');
    expect(getPlatformType({ store: 'GOG' })).toBe('GOG');
    expect(getPlatformType({ store: 'Standalone' })).toBe('Standalone');

    const steamColors = getPlatformColors({ store: 'Steam' });
    expect(steamColors.label).toBe('Steam');
    expect(steamColors.border).toBe('#1a9fff');
  });

  it('formats playtime and last played timestamps cleanly', () => {
    expect(formatPlaytime(0)).toBe('Never played');
    expect(formatPlaytime(45)).toBe('0 mins');
    expect(formatPlaytime(120)).toBe('2 mins');
    expect(formatPlaytime(3600)).toBe('1 hr');
    expect(formatPlaytime(9000)).toBe('2.5 hrs');

    expect(formatLastPlayed(0)).toBe('');
    expect(formatLastPlayed(Math.floor(Date.now() / 1000) - 10)).toBe('Just now');
  });

  it('correctly resolves metadata info and matches for Steam, GOG, and RAWG', () => {
    const rawgGame = {
      store: 'Standalone',
      source: 'Standalone',
      metadata_provider: 'rawg',
      metadata_id: '34744',
      app_id: '0'
    };

    expect(getMetadataInfo(rawgGame)).toEqual({ provider: 'RAWG', id: '34744' });
    expect(hasMetadataMatch(rawgGame)).toBe(true);

    const gogGame = {
      store: 'Standalone',
      source: 'Standalone',
      metadata_provider: 'gog',
      metadata_id: '12345',
      app_id: '0'
    };
    expect(getMetadataInfo(gogGame)).toEqual({ provider: 'GOG', id: '12345' });
    expect(hasMetadataMatch(gogGame)).toBe(true);

    const standaloneGame = {
      store: 'Standalone',
      source: 'Standalone',
      metadata_provider: 'none',
      metadata_id: '0',
      app_id: '0'
    };
    expect(getMetadataInfo(standaloneGame)).toEqual({ provider: 'None', id: '' });
    expect(hasMetadataMatch(standaloneGame)).toBe(false);
  });

  it('filters launch candidates to exclude technical loader helpers', () => {
    const raw = [
      { name: 'Loader', executable: 'C:/Game/steamclient_loader.exe' },
      { name: 'Cold Loader', executable: 'C:/Game/coldclientloader.exe' },
      { name: 'Main Game', executable: 'C:/Game/game.exe', arguments: '-debug' },
    ];
    const filtered = filterLaunchCandidates(raw);
    expect(filtered.length).toBe(1);
    expect(filtered[0].name).toBe('Main Game');
    expect(filtered[0].executable).toBe('C:/Game/game.exe');
    expect(filtered[0].arguments).toBe('-debug');
  });

  it('extracts executable path avoiding loader binaries', () => {
    const candidates = [{ name: 'Main', executable: '/games/main.exe' }];
    const gameWithLoader = {
      exe_path: '/games/steamclient_loader.exe',
      launchCandidates: candidates,
    };
    expect(getExecutablePath(gameWithLoader, candidates)).toBe('/games/main.exe');

    const gameNormal = {
      exe_path: '/games/custom.exe',
    };
    expect(getExecutablePath(gameNormal, candidates)).toBe('/games/custom.exe');
  });

  it('accurately calculates active overrides against baseline defaults', () => {
    const game = {
      name: 'Cyberpunk 2077',
      platform: 'win64',
      exePath: 'bin/x64/Cyberpunk2077.exe',
      launchCandidates: [{ executable: 'bin/x64/Cyberpunk2077.exe', arguments: '' }],
    };
    const globalCfg = { default_runner: 'umu', mangohud: false, language: 'auto' };

    // Clean baseline
    const cleanDraft = {
      name: '',
      platform: 'win64',
      exePath: 'bin/x64/Cyberpunk2077.exe',
      exeArgs: '',
      operatingMode: '',
      runner: '',
      winePrefix: '',
      protonPath: '',
      dllOverrides: '',
      mangoHud: null,
      customEnv: {},
      language: '',
      sourceOverride: '',
      metadataProvider: '',
    };
    expect(countOverrides(cleanDraft, game, globalCfg)).toBe(0);

    // Clean baseline with default operatingMode, default metadata provider, and false disableOverlay
    const cleanWithDefaults = {
      ...cleanDraft,
      operatingMode: 'loader',
      metadataProvider: 'steam',
      disableOverlay: false,
    };
    expect(countOverrides(cleanWithDefaults, { ...game, store: 'Steam' }, globalCfg)).toBe(0);

    // Add runner, mangohud, env override, and title override
    const modifiedDraft = {
      ...cleanDraft,
      name: 'Cyberpunk Custom Edition',
      runner: 'wine',
      mangoHud: true,
      customEnv: { PROTON_FORCE_NVAPI: '1' },
      dllOverrides: 'dxgi=n,b',
    };
    expect(countOverrides(modifiedDraft, game, globalCfg)).toBe(5);
  });

  it('generates proper thumbnail URLs via getThumbnailUrl', () => {
    // /art/ URL with timestamp
    expect(getThumbnailUrl('/art/portal-2/cover.jpg?t=1700000000', 300)).toBe(
      '/art/thumb?id=portal-2%2Fcover.jpg&w=300&t=1700000000'
    );

    // /art/ URL without query
    expect(getThumbnailUrl('/art/portal-2/cover.jpg', 200)).toBe(
      '/art/thumb?id=portal-2%2Fcover.jpg&w=200'
    );

    // Already a thumbnail URL -> return unchanged
    expect(getThumbnailUrl('/art/thumb?id=abc&w=300', 300)).toBe(
      '/art/thumb?id=abc&w=300'
    );

    // /local-img/ URL
    expect(getThumbnailUrl('/local-img/home/user/games/cover.jpg', 300)).toBe(
      '/art/thumb?id=%2Flocal-img%2Fhome%2Fuser%2Fgames%2Fcover.jpg&w=300'
    );

    // AppID fallback when no URL
    expect(getThumbnailUrl('', 300, '620')).toBe(
      '/art/thumb?id=620&w=300'
    );

    // Invalid inputs return empty string
    expect(getThumbnailUrl('', 300, '')).toBe('');
    expect(getThumbnailUrl(null, 300, '')).toBe('');
  });

  it('normalizes raw Go DLC entity into canonical camelCase DLC model', () => {
    expect(normalizeDlc(null)).toBeNull();
    expect(normalizeDlc(undefined)).toBeNull();

    const raw = {
      app_id: '200715',
      name: 'Torchlight II: Pet Pack',
    };

    const dlc = normalizeDlc(raw);
    expect(dlc).toEqual({
      appId: '200715',
      name: 'Torchlight II: Pet Pack',
    });
  });

  it('normalizes raw Go ProviderDescriptor into canonical camelCase MetadataProvider', () => {
    expect(normalizeMetadataProvider(null)).toBeNull();
    expect(normalizeMetadataProvider(undefined)).toBeNull();

    const raw = {
      id: 'steam',
      display_name: 'Steam',
      description: 'Official Steam Store',
      capabilities: { is_game_store: true },
    };

    const provider = normalizeMetadataProvider(raw);
    expect(provider).toEqual({
      id: 'steam',
      displayName: 'Steam',
      description: 'Official Steam Store',
      capabilities: { is_game_store: true },
    });
  });

  it('normalizes raw Go GameMetadataDetails into canonical camelCase GameMetadata', () => {
    expect(normalizeGameMetadata(null)).toBeNull();
    expect(normalizeGameMetadata(undefined)).toBeNull();

    const raw = {
      name: 'Cyberpunk 2077',
      short_description: 'An open-world action-adventure RPG',
      detailed_description: 'Full story description',
      about_the_game: 'About description',
      header_image: 'https://cdn.example.com/header.jpg',
      developers: ['CD Projekt Red'],
      publishers: ['CD Projekt Red'],
      genres: ['RPG', 'Action'],
      release_date: '2020-12-10',
    };

    const meta = normalizeGameMetadata(raw);
    expect(meta).toEqual({
      name: 'Cyberpunk 2077',
      shortDescription: 'An open-world action-adventure RPG',
      detailedDescription: 'Full story description',
      aboutTheGame: 'About description',
      headerImage: 'https://cdn.example.com/header.jpg',
      developers: ['CD Projekt Red'],
      publishers: ['CD Projekt Red'],
      genres: ['RPG', 'Action'],
      releaseDate: '2020-12-10',
    });
  });
});
