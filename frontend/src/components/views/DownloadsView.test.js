import { render, fireEvent, screen, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import DownloadsView from './DownloadsView.svelte';
import { tasksStore } from '../../stores/tasks.js';
import { libraryPath } from '../../stores/library.js';
import { AppAPI } from '../../api/app.js';

describe('DownloadsView Component', () => {
  beforeEach(() => {
    tasksStore.reset();
    libraryPath.set('/home/staernid/Games');
  });

  it('renders header, telemetry and empty state by default', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([]);
    render(DownloadsView);

    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(/Downloads/i);
    expect(screen.getByText(/0 active/i)).toBeInTheDocument();
    expect(screen.getByText(/No downloads active/i)).toBeInTheDocument();
    expect(screen.getByText(/Browse Catalog/i)).toBeInTheDocument();
  });

  it('renders active download with formatted byte progress, speed, and ETA', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([
      {
        id: 'dl_half_life',
        game_id: '70',
        game_name: 'Half-Life',
        target_dir: '/home/staernid/Games/Half-Life',
        store: 'steam',
        bytes_downloaded: 262144000, // 250 MiB
        total_bytes: 838860800,      // 800 MiB
        speed_bps: 15728640,         // 15 MiB/s
        eta_seconds: 38,
        percent: 31.25,
        stage: 'Downloading Depots',
        details: 'valve/maps/c1a1b.bsp',
        status: 'downloading',
      }
    ]);

    render(DownloadsView);

    expect((await screen.findAllByText('Half-Life')).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/250.*MiB \/ 800.*MiB \(31%\)/i)).toBeInTheDocument();
    expect(screen.getAllByText(/15.*MiB\/s/i).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/ETA: 38s/i)).toBeInTheDocument();
  });

  it('handles pause and cancel button clicks on active downloads', async () => {
    const pauseSpy = vi.spyOn(AppAPI, 'pauseTask').mockResolvedValue(true);
    const cancelSpy = vi.spyOn(AppAPI, 'cancelTask').mockResolvedValue(true);
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([
      {
        id: 'dl_test_1',
        game_id: '123',
        game_name: 'Test Game',
        target_dir: '/tmp/test',
        status: 'downloading',
        stage: 'Downloading',
        details: 'chunk1.bin',
        bytes_downloaded: 1000,
        total_bytes: 5000,
        percent: 20,
      }
    ]);

    render(DownloadsView);

    const pauseBtn = await screen.findByText('Pause');
    await fireEvent.click(pauseBtn);
    expect(pauseSpy).toHaveBeenCalledWith('dl_test_1');

    const cancelBtn = screen.getByText('Cancel');
    await fireEvent.click(cancelBtn);
    expect(cancelSpy).toHaveBeenCalledWith('dl_test_1');
  });

  it('moves completed download to finished history section and navigates to gamehub on click', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([
      {
        id: 'dl_completed_1',
        game_id: '70',
        game_name: 'Half-Life',
        target_dir: '/home/staernid/Games/Half-Life',
        status: 'completed',
        details: 'Download finished',
        percent: 100,
      }
    ]);

    const { games, activeView } = await import('../../stores/library.js');
    games.set([{ gameDir: '/home/staernid/Games/Half-Life', name: 'Half-Life', appId: '70' }]);

    render(DownloadsView);

    expect(await screen.findByText(/Completed History/i)).toBeInTheDocument();
    expect((await screen.findAllByText('Half-Life')).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/View in Game Hub/i)).toBeInTheDocument();
    expect(screen.getByText('Dismiss')).toBeInTheDocument();

    const historyCard = screen.getByTitle(/Open Half-Life in Game Hub/i);
    await fireEvent.click(historyCard);

    let current = '';
    activeView.subscribe(v => current = v)();
    expect(current).toBe('game-hub');
  });

  it('renders drive capacity gauge and allows expanding multi-depot breakdown', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([
      {
        id: 'dl_witcher',
        game_id: '292030',
        game_name: 'The Witcher 3: Wild Hunt',
        target_dir: '/home/staernid/Games/The Witcher 3',
        status: 'downloading',
        bytes_downloaded: 35000000000,
        total_bytes: 50000000000,
        percent: 70,
        current_items: 3500,
        total_items: 5000,
        depots: [
          { id: '292031', name: 'Base Content', downloaded_bytes: 28000000000, total_bytes: 35000000000, percent: 80 },
          { id: '292032', name: 'Executables', downloaded_bytes: 1000000000, total_bytes: 1000000000, percent: 100 }
        ]
      }
    ]);
    vi.spyOn(AppAPI, 'getDiskSpace').mockResolvedValue({
      path: '/home/staernid/Games',
      total_bytes: 1099511627776,
      free_bytes: 492970344448,
      used_bytes: 606541283328
    });

    render(DownloadsView);

    expect(await screen.findByText(/459.*GiB/i)).toBeInTheDocument();
    expect(screen.getByText(/Free/i)).toBeInTheDocument();

    const depotToggle = screen.getByText(/Depots & Chunk Breakdown/i);
    expect(depotToggle).toBeInTheDocument();
    await fireEvent.click(depotToggle);

    expect(await screen.findByText('Base Content')).toBeInTheDocument();
    expect(screen.getByText('Executables')).toBeInTheDocument();
  });

  it('strictly excludes non-download system operations from downloads view', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([
      {
        id: 'op_verify_1',
        type: 'verify',
        title: 'Verify Integrity (Cyberpunk 2077)',
        game_name: 'Cyberpunk 2077',
        target_dir: '/home/staernid/Games/Cyberpunk 2077',
        status: 'running',
        stage: 'Verifying Checksums',
        details: 'Scanning archive hashes (650/1420 files)',
        percent: 45,
        current_items: 650,
        total_items: 1420,
      },
      {
        id: 'dl_half_life',
        type: 'download',
        game_name: 'Half-Life',
        target_dir: '/home/staernid/Games/Half-Life',
        status: 'downloading',
        percent: 60,
        bytes_downloaded: 600,
        total_bytes: 1000,
      }
    ]);

    render(DownloadsView);

    // Download should be visible
    expect((await screen.findAllByText('Half-Life')).length).toBeGreaterThanOrEqual(1);

    // Maintenance operation must NOT be in downloads view
    expect(screen.queryByText('Cyberpunk 2077')).not.toBeInTheDocument();
    expect(screen.queryByText(/INTEGRITY/i)).not.toBeInTheDocument();
  });

  it('supports pause all and resume all master buttons', async () => {
    const pauseSpy = vi.spyOn(AppAPI, 'pauseTask').mockResolvedValue(true);
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([
      {
        id: 'dl_game_1',
        game_id: '1',
        game_name: 'Game One',
        status: 'downloading',
        percent: 30,
      }
    ]);

    render(DownloadsView);

    const pauseAllBtn = await screen.findByRole('button', { name: /Pause All/i });
    expect(pauseAllBtn).toBeInTheDocument();
    await fireEvent.click(pauseAllBtn);
    expect(pauseSpy).toHaveBeenCalledWith('dl_game_1');
  });

  it('supports configuring download speed limit from header dropdown', async () => {
    const limitSpy = vi.spyOn(AppAPI, 'setTransferSpeedLimit').mockResolvedValue(true);
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([]);

    render(DownloadsView);

    const limitTrigger = screen.getByTitle(/Bandwidth Limit/i);
    expect(limitTrigger).toBeInTheDocument();
    await fireEvent.click(limitTrigger);

    const tenMbOpt = await screen.findByRole('button', { name: /10 MB\/s/i });
    await fireEvent.click(tenMbOpt);

    expect(limitSpy).toHaveBeenCalledWith(10 * 1024 * 1024);
  });
});

