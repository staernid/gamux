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

    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(/Tasks/i);
    expect(screen.getByText(/0 active/i)).toBeInTheDocument();
    expect(screen.getByText(/No active tasks running/i)).toBeInTheDocument();
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

    expect(await screen.findByText(/Target Drive Capacity/i)).toBeInTheDocument();
    expect(screen.getByText(/459.*GiB Free/i)).toBeInTheDocument();

    const depotToggle = screen.getByText(/Depots & Chunk Breakdown/i);
    expect(depotToggle).toBeInTheDocument();
    await fireEvent.click(depotToggle);

    expect(await screen.findByText('Base Content')).toBeInTheDocument();
    expect(screen.getByText('Executables')).toBeInTheDocument();
  });

  it('renders active system operations and supports filter tab switching', async () => {
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

    const cpMatches = await screen.findAllByText('Cyberpunk 2077');
    expect(cpMatches.length).toBeGreaterThanOrEqual(1);
    expect((await screen.findAllByText('Half-Life')).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/INTEGRITY/i)).toBeInTheDocument();

    // Switch to Operations tab
    const opsTab = screen.getByRole('button', { name: /Operations/i });
    await fireEvent.click(opsTab);

    expect(screen.getAllByText('Cyberpunk 2077').length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText('Half-Life')).not.toBeInTheDocument();

    // Switch to Transfers tab
    const transfersTab = screen.getByRole('button', { name: /Transfers/i });
    await fireEvent.click(transfersTab);

    expect(screen.getAllByText('Half-Life').length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText('Cyberpunk 2077')).not.toBeInTheDocument();
  });

  it('filters non-Steam search candidates and supports macOS platform target', async () => {
    vi.spyOn(AppAPI, 'listTasks').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1091500', name: 'Cyberpunk 2077', store: 'steam' },
      { id: 'gog_123', name: 'GOG Only Game', store: 'gog' },
      { id: 'rawg_456', name: 'RAWG Database Game', store: 'rawg' },
    ]);
    const startTransferSpy = vi.spyOn(AppAPI, 'startDownloadTransfer').mockResolvedValue('dl_cyberpunk');

    render(DownloadsView);

    await fireEvent.click(screen.getByText('New Download'));

    const searchInput = screen.getByLabelText(/Search Game Title or Steam AppID/i);
    await fireEvent.input(searchInput, { target: { value: 'Cyberpunk' } });
    await fireEvent.click(screen.getByText('Search'));

    expect(await screen.findByText('Cyberpunk 2077')).toBeInTheDocument();
    expect(screen.queryByText('GOG Only Game')).not.toBeInTheDocument();
    expect(screen.queryByText('RAWG Database Game')).not.toBeInTheDocument();

    await fireEvent.click(screen.getByText('Cyberpunk 2077'));

    const platformSelect = screen.getByLabelText(/Platform Target/i);
    expect(platformSelect).toHaveTextContent(/macOS Native/i);
    await fireEvent.change(platformSelect, { target: { value: 'macos' } });

    await fireEvent.click(screen.getByText('Start Download'));

    expect(startTransferSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        app_id: 1091500,
        platform: 'macos',
        auto_apply: true,
      })
    );
  });
});

