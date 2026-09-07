import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubSavesCard from './HubSavesCard.svelte';
import { AppAPI } from '../../../../api/app.js';
import { normalizeGame } from '../../../../utils/game.js';

describe('HubSavesCard Component', () => {
  const mockGame = normalizeGame({
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    game_dir: '/home/staernid/Games/Cyberpunk 2077',
    platform: 'win64',
    runner: 'UMU',
  });

  it('loads and renders discovered save locations and metrics', async () => {
    render(HubSavesCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/Total Save Size/i)).toBeInTheDocument();
    });

    expect(screen.getAllByText(/43.13 MiB/i).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/Discovered Save & Config Directories/i)).toBeInTheDocument();
    expect(screen.getByText(/Save Snapshots & Backups/i)).toBeInTheDocument();
  });

  it('triggers folder open on click', async () => {
    const openSpy = vi.spyOn(AppAPI, 'openDirectory').mockResolvedValue();
    render(HubSavesCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getAllByText(/^Open$/i).length).toBeGreaterThan(0);
    });

    const openBtns = screen.getAllByText(/^Open$/i);
    await fireEvent.click(openBtns[0]);

    expect(openSpy).toHaveBeenCalled();
  });

  it('allows toggling backup input and creating a backup', async () => {
    const createSpy = vi.spyOn(AppAPI, 'createSaveBackup').mockResolvedValue({
      id: '2026-08-27_12-00-00',
      size_bytes: 1024,
    });

    render(HubSavesCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/\+ Create Backup/i)).toBeInTheDocument();
    });

    const createBtn = screen.getByText(/\+ Create Backup/i);
    await fireEvent.click(createBtn);

    const input = screen.getByPlaceholderText(/e\.g\. Before Final Boss/i);
    expect(input).toBeInTheDocument();

    await fireEvent.input(input, { target: { value: 'Before final mission' } });

    const takeSnapshotBtn = screen.getByText(/Take Snapshot/i);
    await fireEvent.click(takeSnapshotBtn);

    expect(createSpy).toHaveBeenCalledWith(mockGame.gameDir, 'Before final mission');
  });
});
