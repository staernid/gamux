import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import GameHubView from './GameHubView.svelte';
import { selectedGame, activeView } from '../../stores/library.js';
import { tasksStore } from '../../stores/tasks.js';
import { modalStore } from '../../stores/modal.js';
import { configStore } from '../../stores/config.js';
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

  it('renders hero banner with game title, platform, size, and 4 consolidated primary tabs', async () => {
    render(GameHubView);

    expect(screen.getByRole('heading', { name: 'Cyberpunk 2077', level: 1 })).toBeInTheDocument();
    expect(screen.getAllByText(/#1091500/i).length).toBeGreaterThan(0);
    expect(screen.getByText('Play Now')).toBeInTheDocument();

    // 4 consolidated primary tabs
    expect(screen.getByRole('button', { name: /^Overview$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Configuration$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Files & Depots$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Saves$/i })).toBeInTheDocument();
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

    // Quick external resources embedded directly in Overview
    expect(await screen.findByRole('button', { name: /PCGamingWiki/i })).toBeInTheDocument();
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

    // Switch to Configuration tab
    const configBtn = screen.getByRole('button', { name: /^Configuration$/i });
    await fireEvent.click(configBtn);
    expect(screen.getByText('Executable & Arguments')).toBeInTheDocument();
    expect(screen.getByText('Runtime & Compatibility')).toBeInTheDocument();
    expect(screen.getByText(/Effective Launch Command/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Match Store|Change Store Match/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Match Metadata|Change Match/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Open Artwork Hub/i })).toBeInTheDocument();

    // Switch to Files & Depots
    const filesBtn = screen.getByRole('button', { name: /^Files & Depots$/i });
    await fireEvent.click(filesBtn);
    expect(screen.getByText('DLC & Depots')).toBeInTheDocument();
    expect(screen.getByText('Integrity & Repair')).toBeInTheDocument();
    expect(screen.getByText('Execution Logs')).toBeInTheDocument();

    // Switch to Saves
    const savesBtn = screen.getByRole('button', { name: /^Saves$/i });
    await fireEvent.click(savesBtn);
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

  it('opens confirmation modal and trashes game when trash button is clicked', async () => {
    const { modalStore } = await import('../../stores/modal.js');
    const modalSpy = vi.spyOn(modalStore, 'open');
    const trashSpy = vi.spyOn(AppAPI, 'trashGame').mockResolvedValue({});

    render(GameHubView);

    const trashBtn = screen.getByRole('button', { name: /Move Game to Trash/i });
    expect(trashBtn).toBeInTheDocument();
    expect(trashBtn.textContent.trim()).toBe('');

    await fireEvent.click(trashBtn);

    expect(modalSpy).toHaveBeenCalledWith('confirm', expect.objectContaining({
      title: expect.stringContaining('Move "Cyberpunk 2077" to Trash?'),
      danger: true,
      confirmLabel: 'Move to Trash',
      cancelLabel: 'Keep Game',
    }));

    // Trigger onConfirm from modal config
    const modalConfig = modalSpy.mock.calls[0][1];
    await modalConfig.onConfirm();
    modalStore.close();

    expect(trashSpy).toHaveBeenCalledWith('/home/user/Games/Cyberpunk 2077');
    expect(get(activeView)).toBe('library');
  });

  it('switches to Configuration tab when X key is pressed', async () => {
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
    expect(await screen.findByRole('button', { name: /PCGamingWiki/i })).toBeInTheDocument();
  });

  it('renders live task progress bar on hero button and disables primary action when an operation is running', async () => {
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

    // Verify task status pill is in hero banner
    expect(screen.getByText('TASK ACTIVE')).toBeInTheDocument();

    // Verify hero action button shows Task in Progress... and is disabled
    const busyBtn = screen.getByRole('button', { name: /Task in Progress\.\.\./i });
    expect(busyBtn).toBeInTheDocument();
    expect(busyBtn).toBeDisabled();

    // Verify hero progress bar background is rendered with correct percentage
    const progressBar = screen.getByTestId('hero-progress-bar');
    expect(progressBar).toBeInTheDocument();
    expect(progressBar.style.width).toBe('45%');
  });

  it('renders paused download state with frozen grayed-out progress bar and allows resuming download rather than displaying Play Now', async () => {
    tasksStore.handleTaskEvent({
      type: 'transfer:started',
      id: 'download_cp2077_1',
      task_id: 'download_cp2077_1',
      operation: 'download',
      game_name: 'Cyberpunk 2077',
      game_dir: '/home/user/Games/Cyberpunk 2077',
      app_id: '1091500',
      stage: 'Downloading Depots',
      details: 'depot 1091501 chunk 42',
      progress: {
        percent: 62,
        current_bytes: 62000000,
        total_bytes: 100000000
      }
    });

    tasksStore.handleTaskEvent({
      type: 'transfer:paused',
      id: 'download_cp2077_1',
      task_id: 'download_cp2077_1',
      operation: 'download'
    });

    render(GameHubView);

    // Verify paused badge and telemetry in hero banner
    expect(screen.getByText('DOWNLOAD PAUSED')).toBeInTheDocument();
    expect(screen.getByText('Download paused')).toBeInTheDocument();

    // Verify hero banner displays Resume button instead of Play Now
    expect(screen.queryByRole('button', { name: /Play Now/i })).toBeNull();
    const resumeBtn = screen.getByRole('button', { name: /Resume \(62%\)/i });
    expect(resumeBtn).toBeInTheDocument();
    expect(resumeBtn).not.toBeDisabled();

    // Verify frozen progress bar background is rendered with 62% width
    const progressBar = screen.getByTestId('hero-progress-bar');
    expect(progressBar).toBeInTheDocument();
    expect(progressBar.style.width).toBe('62%');
    expect(progressBar.className).toContain('bg-neutral-700');

    // Verify cancel button is accessible
    expect(screen.getByRole('button', { name: /Cancel Task/i })).toBeInTheDocument();
  });

  it('toggles inline Source and Metadata setup wizards in Configuration tab', async () => {
    render(GameHubView);

    // Navigate to Configuration tab
    const configBtn = screen.getByRole('button', { name: /^Configuration$/i });
    await fireEvent.click(configBtn);

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

    // Navigate to Configuration tab
    const configBtn = screen.getByRole('button', { name: /^Configuration$/i });
    await fireEvent.click(configBtn);

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

  it('opens install modal when clicking Download on uninstalled virtual game by default', async () => {
    configStore.setSkipInstallConfirm(false);
    const modalSpy = vi.spyOn(modalStore, 'open');

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

    const downloadBtn = screen.getByText('Download');
    await fireEvent.click(downloadBtn);

    expect(modalSpy).toHaveBeenCalledWith('install', expect.objectContaining({ game: expect.anything() }));
  });

  it('triggers download transfer and pre-creates stub when skipInstallConfirm is true', async () => {
    configStore.setSkipInstallConfirm(true);
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

    expect(stubSpy).toHaveBeenCalledWith(expect.any(String), '1145350', 'Hades II', 'Steam', 'public');
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

    expect(stubSpy).toHaveBeenCalledWith(expect.any(String), '1145350', 'Hades II', 'Steam', 'public');
  });
});


