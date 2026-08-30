import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { drawerDraft, createInitialDraft, computeEnvEntries } from './drawerDraft.js';
import { AppAPI } from '../api/app.js';
import { showInspector } from './library.js';

describe('drawerDraft Store', () => {
  const mockGame = {
    name: 'Hades',
    appId: '1145360',
    platform: 'Windows PE',
    gameDir: '/home/user/Games/Hades',
    exePath: 'Hades.exe',
    exeArgs: '-dx11',
    launchCandidates: [
      { name: 'Hades DirectX', executable: 'Hades.exe', arguments: '-dx11' },
      { name: 'Hades Vulkan', executable: 'Hades_vk.exe', arguments: '-vulkan' },
    ],
    supportedLanguages: ['english', 'french', 'german', 'spanish'],
    currentLanguage: 'english',
    isPatched: true,
    isPortable: false,
    launchProfile: {
      exe_path: 'Hades.exe',
      exe_args: '-dx11',
      operating_mode: 'loader',
      runner: 'umu',
      proton_path: '/custom/proton',
      mangohud: true,
      custom_env: { MY_FLAG: '1' }
    }
  };

  const mockGlobalConfig = {
    operating_mode: 'loader',
    default_runner: 'umu',
    global_env: { GLOBAL_VAR: 'val', MY_FLAG: 'default' },
    normalize: true,
    steamless: true,
    achievements: true,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('initializes draft accurately from game and launch profile', () => {
    drawerDraft.init(mockGame, mockGlobalConfig);
    const state = get(drawerDraft);

    expect(state.name).toBe('Hades');
    expect(state.appId).toBe('1145360');
    expect(state.exePath).toBe('Hades.exe');
    expect(state.exeArgs).toBe('-dx11');
    expect(state.operatingMode).toBe('loader');
    expect(state.runner).toBe('umu');
    expect(state.protonPath).toBe('/custom/proton');
    expect(state.mangoHud).toBe(true);
    expect(state.customEnv).toEqual({ MY_FLAG: '1' });
  });

  it('computes env entries with global inheritance and overrides', () => {
    const custom = { MY_FLAG: '1', GAME_ONLY: '2' };
    const globalEnv = { GLOBAL_VAR: 'gval', MY_FLAG: 'default' };

    const entries = computeEnvEntries(custom, globalEnv);

    // GLOBAL_VAR inherited
    const gVar = entries.find(e => e.key === 'GLOBAL_VAR');
    expect(gVar).toBeDefined();
    expect(gVar.isInherited).toBe(true);
    expect(gVar.isOverriding).toBe(false);

    // MY_FLAG overriding
    const mFlag = entries.find(e => e.key === 'MY_FLAG');
    expect(mFlag).toBeDefined();
    expect(mFlag.isInherited).toBe(false);
    expect(mFlag.isOverriding).toBe(true);
    expect(mFlag.value).toBe('1');
    expect(mFlag.globalValue).toBe('default');

    // GAME_ONLY custom
    const gameOnly = entries.find(e => e.key === 'GAME_ONLY');
    expect(gameOnly).toBeDefined();
    expect(gameOnly.isInherited).toBe(false);
    expect(gameOnly.isOverriding).toBe(false);
    expect(gameOnly.value).toBe('2');
  });

  it('modifies fields, candidates, and environment variables reactively', () => {
    drawerDraft.init(mockGame, mockGlobalConfig);

    drawerDraft.setField('exeArgs', '-vulkan -novid');
    expect(get(drawerDraft).exeArgs).toBe('-vulkan -novid');

    drawerDraft.selectCandidate(mockGame.launchCandidates[1]);
    expect(get(drawerDraft).exePath).toBe('Hades_vk.exe');
    expect(get(drawerDraft).exeArgs).toBe('-vulkan');

    drawerDraft.overrideEnv('GLOBAL_VAR', 'overridden');
    expect(get(drawerDraft).customEnv['GLOBAL_VAR']).toBe('overridden');

    drawerDraft.revertEnv('GLOBAL_VAR');
    expect(get(drawerDraft).customEnv['GLOBAL_VAR']).toBeUndefined();
  });

  it('saves launch profile and closes inspector on success', async () => {
    vi.spyOn(AppAPI, 'saveLaunchProfile').mockResolvedValue({});
    vi.spyOn(AppAPI, 'submitApply').mockResolvedValue('task-1');
    vi.spyOn(AppAPI, 'inspectGame').mockResolvedValue(mockGame);

    showInspector.set(true);
    drawerDraft.init(mockGame, mockGlobalConfig);

    const success = await drawerDraft.save(mockGame, mockGlobalConfig);

    expect(success).toBe(true);
    expect(AppAPI.saveLaunchProfile).toHaveBeenCalled();
    expect(get(showInspector)).toBe(false);
  });

  it('correctly tracks and persists DRM operating mode (loader, portable, none, default)', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveLaunchProfile').mockResolvedValue({});
    vi.spyOn(AppAPI, 'inspectGame').mockResolvedValue(mockGame);

    drawerDraft.init(mockGame, mockGlobalConfig);

    // Initial draft from mockGame with operating_mode 'loader'
    expect(get(drawerDraft).operatingMode).toBe('loader');

    // Switch to 'portable'
    drawerDraft.setField('operatingMode', 'portable');
    await drawerDraft.save(mockGame);
    expect(saveSpy).toHaveBeenLastCalledWith(
      mockGame.gameDir,
      expect.objectContaining({ operating_mode: 'portable' })
    );

    // Switch to 'none'
    drawerDraft.setField('operatingMode', 'none');
    await drawerDraft.save(mockGame);
    expect(saveSpy).toHaveBeenLastCalledWith(
      mockGame.gameDir,
      expect.objectContaining({ operating_mode: 'none' })
    );

    // Switch to default/inherited ('')
    drawerDraft.setField('operatingMode', '');
    await drawerDraft.save(mockGame);
    expect(saveSpy).toHaveBeenLastCalledWith(
      mockGame.gameDir,
      expect.objectContaining({ operating_mode: '' })
    );
  });

  it('correctly tracks and persists runner and proton path inheritance', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveLaunchProfile').mockResolvedValue({});
    vi.spyOn(AppAPI, 'inspectGame').mockResolvedValue(mockGame);

    drawerDraft.init(mockGame, mockGlobalConfig);

    // Explicit overrides
    drawerDraft.setField('runner', 'wine');
    drawerDraft.setField('protonPath', '/opt/proton-ge');
    await drawerDraft.save(mockGame);
    expect(saveSpy).toHaveBeenLastCalledWith(
      mockGame.gameDir,
      expect.objectContaining({ runner: 'wine', proton_path: '/opt/proton-ge' })
    );

    // Reset to default (empty string)
    drawerDraft.setField('runner', '');
    drawerDraft.setField('protonPath', '');
    await drawerDraft.save(mockGame);
    expect(saveSpy).toHaveBeenLastCalledWith(
      mockGame.gameDir,
      expect.objectContaining({ runner: '', proton_path: '' })
    );
  });

  it('preserves playtime and metadata when saving launch profile', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveLaunchProfile').mockResolvedValue({});
    vi.spyOn(AppAPI, 'inspectGame').mockResolvedValue(mockGame);

    const gameWithStats = {
      ...mockGame,
      playtimeSeconds: 5400,
      lastPlayed: 1720000000,
      customCoverUrl: 'https://example.com/art.jpg',
      lockedArtwork: true,
      metadataProvider: 'steam',
      metadataId: '1145360',
    };

    drawerDraft.init(gameWithStats, mockGlobalConfig);
    drawerDraft.setField('runner', 'wine');

    await drawerDraft.save(gameWithStats);

    expect(saveSpy).toHaveBeenLastCalledWith(
      gameWithStats.gameDir,
      expect.objectContaining({
        runner: 'wine',
        playtime_seconds: 5400,
        last_played: 1720000000,
        custom_cover_url: 'https://example.com/art.jpg',
        locked_artwork: true,
        metadata_provider: 'steam',
        metadata_id: '1145360',
      })
    );
  });
});

