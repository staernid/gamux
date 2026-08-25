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
    expect(screen.getByText(/No active downloads running/i)).toBeInTheDocument();
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

    expect(await screen.findByText('Half-Life')).toBeInTheDocument();
    expect(screen.getByText(/250.*MiB \/ 800.*MiB \(31%\)/i)).toBeInTheDocument();
    expect(screen.getAllByText(/15.*MiB\/s/i).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/ETA: 38s/i)).toBeInTheDocument();
  });

  it('allows toggling add panel, searching game, and initiating download', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1091500', name: 'Cyberpunk 2077', store: 'steam' }
    ]);
    const startTransferSpy = vi.spyOn(AppAPI, 'startDownloadTransfer').mockResolvedValue('dl_cyberpunk');

    render(DownloadsView);

    const newBtn = screen.getByText('New Download');
    await fireEvent.click(newBtn);

    expect(screen.getByText(/Download Title from Store Provider/i)).toBeInTheDocument();

    const searchInput = screen.getByLabelText(/Search Game Title or Steam AppID/i);
    await fireEvent.input(searchInput, { target: { value: 'Cyberpunk' } });

    const searchBtn = screen.getByText('Search');
    await fireEvent.click(searchBtn);

    const candidate = await screen.findByText('Cyberpunk 2077');
    expect(candidate).toBeInTheDocument();
    await fireEvent.click(candidate);

    const startBtn = screen.getByText('Start Download');
    await fireEvent.click(startBtn);

    expect(startTransferSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        app_id: 1091500,
        game_name: 'Cyberpunk 2077',
      })
    );
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

  it('moves completed download to finished history section', async () => {
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

    render(DownloadsView);

    expect(await screen.findByText(/Completed History/i)).toBeInTheDocument();
    expect(screen.getByText('Half-Life')).toBeInTheDocument();
    expect(screen.getByText('Dismiss')).toBeInTheDocument();
  });
});
