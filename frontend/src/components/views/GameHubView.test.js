import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import GameHubView from './GameHubView.svelte';
import { selectedGame, activeView } from '../../stores/library.js';
import { tasksStore } from '../../stores/tasks.js';
import { drawerDraft } from '../../stores/drawerDraft.js';
import { normalizeGame } from '../../utils/game.js';
import { AppAPI } from '../../api/app.js';
import { clearOverviewHubCache } from './hub/HubOverviewTab.svelte';
import { get } from 'svelte/store';

describe('GameHubView Component', () => {
  const mockGame = {
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    platform: 'Windows PE',
    game_dir: '/home/user/Games/Cyberpunk 2077',
    exe_path: 'bin/x64/Cyberpunk2077.exe',
    exe_args: '-dx12',
    runner: 'UMU',
    disk_size_bytes: 75161927680,
    is_patched: true,
    is_ready: true,
    is_running: false,
    store: 'steam'
  };

  beforeEach(() => {
    clearOverviewHubCache();
    tasksStore.reset();
    selectedGame.set(normalizeGame(mockGame));
    activeView.set('game-hub');
    vi.restoreAllMocks();
  });

  it('renders hero banner with game title, platform, size, and 7 consolidated primary tabs', async () => {
    render(GameHubView);

    expect(screen.getByRole('heading', { name: 'Cyberpunk 2077', level: 1 })).toBeInTheDocument();
    expect(screen.getAllByText(/#1091500/i).length).toBeGreaterThan(0);
    expect(screen.getByText('Play Now')).toBeInTheDocument();

    // 7 primary tabs
    expect(screen.getByRole('button', { name: /^Overview$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Community$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Resources$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^General$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Environment$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Saves$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Files & Diagnostics$/i })).toBeInTheDocument();
  });

  it('renders minimal overview tab with synopsis, achievements spotlight, and latest updates', async () => {
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue([
      { id: 'ach_1', name: 'The Fool', description: 'Become a mercenary.', unlocked: true, unlock_time: 1720000000 }
    ]);
    vi.spyOn(AppAPI, 'fetchNews').mockResolvedValue([
      { title: 'Patch 2.13 Available Now', contents: 'Adds support for AMD FidelityFX Super Resolution 3.', date: 1726142400, feedlabel: 'Patch Notes' }
    ]);

    render(GameHubView);

    // Synopsis from metadata
    expect(await screen.findByText(/Cyberpunk 2077 is an open-world/i)).toBeInTheDocument();

    // Achievements spotlight in overview
    expect(await screen.findByText('The Fool')).toBeInTheDocument();

    // Latest updates in overview
    expect(await screen.findByText('Patch 2.13 Available Now')).toBeInTheDocument();

    // External resources moved out of overview to Resources tab
    expect(screen.queryByText('Game Resources & Community Hub')).not.toBeInTheDocument();
    expect(screen.queryByText('PCGamingWiki')).not.toBeInTheDocument();
  });

  it('navigates to Community tab from Overview achievements and updates links', async () => {
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue([
      { id: 'ach_1', name: 'The Fool', description: 'Become a mercenary.', unlocked: true, unlock_time: 1720000000 }
    ]);
    vi.spyOn(AppAPI, 'fetchNews').mockResolvedValue([
      { title: 'Patch 2.13 Available Now', contents: 'Adds support for AMD FidelityFX Super Resolution 3.', date: 1726142400, feedlabel: 'Patch Notes' }
    ]);

    render(GameHubView);

    const viewAllAchBtn = await screen.findByRole('button', { name: /View All \(1\)/i });
    await fireEvent.click(viewAllAchBtn);

    // Should switch to Community tab with Achievements active
    expect(screen.getByRole('button', { name: /^Achievements$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /News & Updates/i })).toBeInTheDocument();
  });

  it('preserves all overview news articles when navigating to community and returning', async () => {
    vi.spyOn(AppAPI, 'fetchNews').mockResolvedValue([
      { title: 'Update 1', contents: 'First update notes', date: 1726142400, feedlabel: 'Patch Notes' },
      { title: 'Update 2', contents: 'Second update notes', date: 1726141400, feedlabel: 'Announcement' },
      { title: 'Update 3', contents: 'Third update notes', date: 1726140400, feedlabel: 'News' }
    ]);

    render(GameHubView);

    // Verify all 3 news items render initially on Overview
    expect(await screen.findByText('Update 1')).toBeInTheDocument();
    expect(screen.getByText('Update 2')).toBeInTheDocument();
    expect(screen.getByText('Update 3')).toBeInTheDocument();

    // Click on the second news article card to navigate to Community
    const update2Btn = screen.getByText('Update 2').closest('button');
    expect(update2Btn).not.toBeNull();
    await fireEvent.click(update2Btn);

    // Verify we are now on Community tab with News active
    expect(screen.getByRole('button', { name: /News & Updates/i })).toBeInTheDocument();

    // Now return to the Overview tab
    const overviewTabBtn = screen.getByRole('button', { name: /^Overview$/i });
    await fireEvent.click(overviewTabBtn);

    // Verify all 3 articles remain present on Overview and didn't shrink to 1
    expect(await screen.findByText('Update 1')).toBeInTheDocument();
    expect(screen.getByText('Update 2')).toBeInTheDocument();
    expect(screen.getByText('Update 3')).toBeInTheDocument();
  });

  it('switches between primary tabs cleanly', async () => {
    render(GameHubView);

    // Switch to Resources tab
    const resourcesBtn = screen.getByRole('button', { name: /^Resources$/i });
    await fireEvent.click(resourcesBtn);
    expect(screen.getByText('Game Resources & Community Hub')).toBeInTheDocument();
    expect(screen.getByText('PCGamingWiki')).toBeInTheDocument();
    expect(screen.getByText('ProtonDB')).toBeInTheDocument();
    expect(screen.getByText('SteamDB')).toBeInTheDocument();

    // Switch to General tab
    const generalBtn = screen.getByRole('button', { name: /^General$/i });
    await fireEvent.click(generalBtn);
    expect(screen.getByText('Identity, Source & Artwork')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Match Store|Change Store Match/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Match Metadata|Change Match/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Open Artwork Hub/i })).toBeInTheDocument();
    expect(screen.getByText('Target Executable')).toBeInTheDocument();

    // Switch to Environment tab
    const envBtn = screen.getByRole('button', { name: /^Environment$/i });
    await fireEvent.click(envBtn);
    expect(screen.getByText('Runner & Platform')).toBeInTheDocument();
    expect(screen.getByText('MangoHud Overlay')).toBeInTheDocument();

    // Switch to Community tab and test subtabs
    const communityBtn = screen.getByRole('button', { name: /^Community$/i });
    await fireEvent.click(communityBtn);
    expect(screen.getByRole('button', { name: /^Achievements$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /News & Updates/i })).toBeInTheDocument();

    // Switch to Files & Diagnostics
    const filesBtn = screen.getByRole('button', { name: /^Files & Diagnostics$/i });
    await fireEvent.click(filesBtn);
    expect(screen.getByText('DLC & Depots')).toBeInTheDocument();
    expect(screen.getByText('Integrity & Repair')).toBeInTheDocument();
    expect(screen.getByText('Execution Logs')).toBeInTheDocument();
  });

  it('navigates back to shelf when Shelf back button is clicked', async () => {
    render(GameHubView);

    const backBtn = screen.getByTitle(/Back to Shelf/i);
    await fireEvent.click(backBtn);

    expect(get(activeView)).toBe('library');
  });

  it('launches game when Play Now is clicked', async () => {
    const launchSpy = vi.spyOn(AppAPI, 'launchGame').mockResolvedValue({});
    render(GameHubView);

    const playBtn = screen.getByText('Play Now');
    await fireEvent.click(playBtn);

    expect(launchSpy).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Cyberpunk 2077',
      game_dir: '/home/user/Games/Cyberpunk 2077',
      app_id: '1091500'
    }));
  });

  it('opens directory when Files button is clicked', async () => {
    const openDirSpy = vi.spyOn(AppAPI, 'openDirectory').mockResolvedValue({});
    render(GameHubView);

    const filesBtn = screen.getByTitle(/Open Directory in File Manager/i);
    await fireEvent.click(filesBtn);

    expect(openDirSpy).toHaveBeenCalledWith('/home/user/Games/Cyberpunk 2077');
  });

  it('switches to General tab when X key is pressed', async () => {
    render(GameHubView);

    await fireEvent.keyDown(window, { key: 'x' });

    expect(await screen.findByText('Target Executable')).toBeInTheDocument();
  });

  it('keeps Overview tab accessible and renders unlinked card for standalone titles', async () => {
    const standaloneGame = {
      name: 'Warcraft III',
      app_id: '0',
      gog_id: '',
      store: 'Standalone',
      source: 'Standalone',
      game_dir: '/home/user/Games/Warcraft III',
      exe_path: 'Warcraft III.exe',
      runner: 'Wine',
      is_ready: true,
    };
    selectedGame.set(normalizeGame(standaloneGame));
    render(GameHubView);

    // Overview tab exists and shows local info + Link Metadata CTA
    expect(screen.getByRole('button', { name: /^Overview$/i })).toBeInTheDocument();
    expect(screen.getByText('Link Metadata')).toBeInTheDocument();
    // Resources are in Resources tab, not Overview
    expect(screen.queryByText('Game Resources & Community Hub')).not.toBeInTheDocument();

    // Resources tab is accessible
    const resBtn = screen.getByRole('button', { name: /^Resources$/i });
    await fireEvent.click(resBtn);
    expect(screen.getByText('Game Resources & Community Hub')).toBeInTheDocument();
    expect(screen.getByText('PCGamingWiki')).toBeInTheDocument();
  });

  it('renders live inline task progress HUD and disables primary action button when an operation is running for the active game', async () => {
    tasksStore.handleTaskEvent({
      type: 'task:started',
      id: 'task_test_apply_1',
      task_id: 'task_test_apply_1',
      operation: 'apply',
      game_name: 'Cyberpunk 2077',
      game_dir: '/home/user/Games/Cyberpunk 2077',
      stage: 'GBE Steam Client Patching',
      details: 'Patching steamclient64.dll wrapper...',
      progress: {
        percent: 45,
        current_items: 3,
        total_items: 6
      }
    });

    render(GameHubView);

    expect(screen.getByText('APPLYING SETUP')).toBeInTheDocument();
    expect(screen.getByText('GBE Steam Client Patching')).toBeInTheDocument();
    expect(screen.getByText('Patching steamclient64.dll wrapper...')).toBeInTheDocument();
    expect(screen.getByText(/Step 3 of 6 \(45%\)/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /View Tasks/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Cancel/i })).toBeInTheDocument();

    const busyBtn = screen.getByRole('button', { name: /Task in Progress\.\.\./i });
    expect(busyBtn).toBeInTheDocument();
    expect(busyBtn).toBeDisabled();
  });

  it('toggles inline Source and Metadata setup wizards in General tab', async () => {
    render(GameHubView);

    // Navigate to General tab
    const generalBtn = screen.getByRole('button', { name: /^General$/i });
    await fireEvent.click(generalBtn);

    // Toggle Source Setup Wizard
    const matchStoreBtn = screen.getByRole('button', { name: /Match Store|Change Store Match/i });
    await fireEvent.click(matchStoreBtn);
    expect(screen.getByText('Configure Physical Installation Source & Store')).toBeInTheDocument();
    expect(screen.getByText('Close Matcher')).toBeInTheDocument();

    // Toggle Metadata Wizard (should close Source Wizard)
    const matchMetaBtn = screen.getByRole('button', { name: /Match Metadata|Change Match/i });
    await fireEvent.click(matchMetaBtn);
    expect(screen.getByText('Match Online Store Metadata')).toBeInTheDocument();
    expect(screen.queryByText('Configure Physical Installation Source & Store')).not.toBeInTheDocument();
  });

  it('searches store candidates and sets AppID in Match Store wizard', async () => {
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '620', name: 'Portal 2', store: 'steam', release_date: '2011-04-19' },
      { id: '400', name: 'Portal', store: 'steam', release_date: '2007-10-10' },
    ]);

    render(GameHubView);

    // Navigate to General tab
    const generalBtn = screen.getByRole('button', { name: /^General$/i });
    await fireEvent.click(generalBtn);

    // Open Match Store
    const matchStoreBtn = screen.getByRole('button', { name: /Match Store|Change Store Match/i });
    await fireEvent.click(matchStoreBtn);

    const searchInput = screen.getByPlaceholderText(/Search STEAM catalogue for AppID/i);
    await fireEvent.input(searchInput, { target: { value: 'Portal' } });

    const searchBtn = screen.getByRole('button', { name: /^Search$/i });
    await fireEvent.click(searchBtn);

    expect(await screen.findByText('Portal 2')).toBeInTheDocument();

    const matchBtns = screen.getAllByRole('button', { name: /Match Store/i });
    await fireEvent.click(matchBtns[0]);

    expect(get(drawerDraft).appId).toBe('620');
  });


  it('renders Download and Add to Library buttons for uninstalled virtual titles', async () => {
    const virtualGame = normalizeGame({
      name: 'Hades II',
      app_id: '1145350',
      store: 'Steam',
      game_dir: '',
      is_installed: false,
      is_stub: false,
      is_virtual: true,
      is_ready: false,
      file_count: 0,
      disk_size_bytes: 0,
    });
    selectedGame.set(virtualGame);

    render(GameHubView);

    expect(screen.getByText('Download')).toBeInTheDocument();
    expect(screen.getByText('Add to Library')).toBeInTheDocument();
    expect(screen.getByText('Online Store Title')).toBeInTheDocument();
  });

  it('triggers download transfer and pre-creates stub when clicking Download on uninstalled virtual game', async () => {
    const virtualGame = normalizeGame({
      name: 'Hades II',
      app_id: '1145350',
      store: 'Steam',
      game_dir: '',
      is_installed: false,
      is_stub: false,
      is_virtual: true,
      is_ready: false,
      file_count: 0,
      disk_size_bytes: 0,
    });
    selectedGame.set(virtualGame);

    const stubSpy = vi.spyOn(AppAPI, 'createGameStub').mockResolvedValue({
      name: 'Hades II',
      app_id: '1145350',
      game_dir: '/home/user/Games/Hades II',
      store: 'Steam',
      file_count: 0,
      disk_size_bytes: 0,
      is_installed: false,
      is_stub: true,
      is_ready: false,
    });
    const startSpy = vi.spyOn(tasksStore, 'start').mockResolvedValue('dl_123');

    render(GameHubView);

    const downloadBtn = screen.getByText('Download');
    await fireEvent.click(downloadBtn);

    expect(stubSpy).toHaveBeenCalledWith(expect.any(String), '1145350', 'Hades II', 'Steam');
    expect(startSpy).toHaveBeenCalledWith(expect.objectContaining({
      app_id: 1145350,
      game_name: 'Hades II',
      target_dir: '/home/user/Games/Hades II'
    }));
  });

  it('creates skeleton stub when clicking Add to Library on virtual title', async () => {
    const virtualGame = normalizeGame({
      name: 'Hades II',
      app_id: '1145350',
      store: 'Steam',
      game_dir: '',
      is_installed: false,
      is_stub: false,
      is_virtual: true,
      is_ready: false,
      file_count: 0,
      disk_size_bytes: 0,
    });
    selectedGame.set(virtualGame);

    const stubSpy = vi.spyOn(AppAPI, 'createGameStub').mockResolvedValue({
      name: 'Hades II',
      app_id: '1145350',
      game_dir: '/home/user/Games/Hades II',
      store: 'Steam',
      file_count: 0,
      disk_size_bytes: 0,
      is_installed: false,
      is_stub: true,
      is_ready: false,
    });

    render(GameHubView);

    const addBtn = screen.getByText('Add to Library');
    await fireEvent.click(addBtn);

    expect(stubSpy).toHaveBeenCalledWith(expect.any(String), '1145350', 'Hades II', 'Steam');
  });
});


