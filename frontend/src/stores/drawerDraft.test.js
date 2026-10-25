import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { drawerDraft, createInitialDraft, computeEnvEntries, createAutosave } from './drawerDraft.js';
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
      name: 'Hades',
      app_id: '1145360',
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
      launchProfile: {
        ...mockGame.launchProfile,
        metadata_provider: 'steam',
        metadata_id: '1145360',
      },
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

  it('dynamically preserves and maps arbitrary new profile properties without manual field unpacking', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveLaunchProfile').mockResolvedValue({});
    vi.spyOn(AppAPI, 'inspectGame').mockResolvedValue(mockGame);

    const gameWithFutureProps = {
      ...mockGame,
      launchProfile: {
        ...mockGame.launchProfile,
        gamescope_fps: 144,
        gamescope_width: 2560,
        gamescope_height: 1440,
        working_dir: '/custom/working/dir',
        future_arbitrary_setting: 'experimental_value',
      },
    };

    drawerDraft.init(gameWithFutureProps, mockGlobalConfig);
    const state = get(drawerDraft);

    expect(state.gamescopeFps).toBe(144);
    expect(state.gamescopeWidth).toBe(2560);
    expect(state.gamescopeHeight).toBe(1440);
    expect(state.workingDir).toBe('/custom/working/dir');
    expect(state.futureArbitrarySetting).toBe('experimental_value');

    // Mutate and save
    drawerDraft.setField('futureArbitrarySetting', 'updated_value');
    drawerDraft.setField('gamescopeFps', 165);
    await drawerDraft.save(gameWithFutureProps);

    expect(saveSpy).toHaveBeenLastCalledWith(
      gameWithFutureProps.gameDir,
      expect.objectContaining({
        gamescope_fps: 165,
        gamescope_width: 2560,
        gamescope_height: 1440,
        working_dir: '/custom/working/dir',
        future_arbitrary_setting: 'updated_value',
      })
    );
  });

  it('tracks dirty state accurately across mutations, init, and markSaved', () => {
    drawerDraft.init(mockGame, mockGlobalConfig);
    expect(get(drawerDraft.isDirty)).toBe(false);

    drawerDraft.setField('exeArgs', '-vulkan');
    expect(get(drawerDraft.isDirty)).toBe(true);

    drawerDraft.markSaved();
    expect(get(drawerDraft.isDirty)).toBe(false);

    drawerDraft.selectCandidate(mockGame.launchCandidates[1]);
    expect(get(drawerDraft.isDirty)).toBe(true);

    drawerDraft.init(mockGame, mockGlobalConfig);
    expect(get(drawerDraft.isDirty)).toBe(false);
  });

  it('triggers exactly one save after debounce on field change, and cancels on re-init', async () => {
    vi.useFakeTimers();
    const saveFn = vi.fn().mockImplementation(() => {
      drawerDraft.markSaved();
    });

    drawerDraft.init(mockGame, mockGlobalConfig);
    const autosave = createAutosave(drawerDraft, saveFn, 400);

    expect(get(drawerDraft.isDirty)).toBe(false);
    expect(saveFn).not.toHaveBeenCalled();

    // 1. Changing a field marks it dirty and schedules debounced save
    drawerDraft.setField('exeArgs', '-vulkan');
    expect(get(drawerDraft.isDirty)).toBe(true);
    expect(saveFn).not.toHaveBeenCalled();

    // Advance 300ms (before debounce expires)
    vi.advanceTimersByTime(300);
    expect(saveFn).not.toHaveBeenCalled();

    // Change another field (resets debounce timer)
    drawerDraft.setField('exeArgs', '-vulkan -novid');
    vi.advanceTimersByTime(300);
    expect(saveFn).not.toHaveBeenCalled();

    // Advance past remaining 100ms (total 400ms since last change)
    vi.advanceTimersByTime(100);
    expect(saveFn).toHaveBeenCalledTimes(1);
    expect(get(drawerDraft.isDirty)).toBe(false);

    // 2. Re-initialising for another game does not trigger save
    drawerDraft.setField('exeArgs', '-dx12');
    expect(get(drawerDraft.isDirty)).toBe(true);

    vi.advanceTimersByTime(200);
    // Re-init for another game before debounce fires
    const otherGame = { ...mockGame, gameDir: '/games/Other', name: 'Other' };
    drawerDraft.init(otherGame, mockGlobalConfig);
    expect(get(drawerDraft.isDirty)).toBe(false);

    // Advance past debounce delay
    vi.advanceTimersByTime(500);
    // Still only 1 save from the first change, none from the second
    expect(saveFn).toHaveBeenCalledTimes(1);

    autosave.destroy();
    vi.useRealTimers();
  });
});


