import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import GameHubView from './GameHubView.svelte';
import { selectedGame, activeView } from '../../stores/library.js';
import { tasksStore } from '../../stores/tasks.js';
import { normalizeGame } from '../../utils/game.js';
import { AppAPI } from '../../api/app.js';
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
    tasksStore.reset();
    selectedGame.set(normalizeGame(mockGame));
    activeView.set('game-hub');
    vi.restoreAllMocks();
  });

  it('renders hero banner with game title, platform, size, and 6 consolidated primary tabs', async () => {
    render(GameHubView);

    expect(screen.getByRole('heading', { name: 'Cyberpunk 2077', level: 1 })).toBeInTheDocument();
    expect(screen.getAllByText(/#1091500/i).length).toBeGreaterThan(0);
    expect(screen.getByText('Play Now')).toBeInTheDocument();

    // 6 primary tabs
    expect(screen.getByRole('button', { name: /^Overview$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Community$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^General$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Environment$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Saves$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Files & Diagnostics$/i })).toBeInTheDocument();
  });

  it('renders overview tab with clean synopsis and external resources', async () => {
    render(GameHubView);

    // Synopsis from metadata
    expect(await screen.findByText(/Cyberpunk 2077 is an open-world/i)).toBeInTheDocument();

    // External resources in overview
    expect(screen.getByText('Game Resources & Community Hub')).toBeInTheDocument();
    expect(screen.getByText('PCGamingWiki')).toBeInTheDocument();
    expect(screen.getByText('ProtonDB')).toBeInTheDocument();
    expect(screen.getByText('SteamDB')).toBeInTheDocument();
  });

  it('switches between primary tabs cleanly', async () => {
    render(GameHubView);

    // Switch to General tab
    const generalBtn = screen.getByRole('button', { name: /^General$/i });
    await fireEvent.click(generalBtn);
    expect(screen.getByText('Identity, Source & Artwork')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Configure Source/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Match Metadata|Change Match/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Open Artwork Hub/i })).toBeInTheDocument();
    expect(screen.getByText('Target Executable')).toBeInTheDocument();

    // Switch to Environment tab
    const envBtn = screen.getByRole('button', { name: /^Environment$/i });
    await fireEvent.click(envBtn);
    expect(screen.getByText('Runner & Prefix')).toBeInTheDocument();
    expect(screen.getByText('MangoHud Overlay')).toBeInTheDocument();

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

    // Overview tab exists and shows local info + Link Metadata CTA + external search resources
    expect(screen.getByRole('button', { name: /^Overview$/i })).toBeInTheDocument();
    expect(screen.getByText('Link Metadata')).toBeInTheDocument();
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
    const configSourceBtn = screen.getByRole('button', { name: /Configure Source/i });
    await fireEvent.click(configSourceBtn);
    expect(screen.getByText('Configure Physical Installation Source')).toBeInTheDocument();
    expect(screen.getByText('Close Source Setup')).toBeInTheDocument();

    // Toggle Metadata Wizard (should close Source Wizard)
    const matchMetaBtn = screen.getByRole('button', { name: /Match Metadata|Change Match/i });
    await fireEvent.click(matchMetaBtn);
    expect(screen.getByText('Match Online Store Metadata')).toBeInTheDocument();
    expect(screen.queryByText('Configure Physical Installation Source')).not.toBeInTheDocument();
  });
});
