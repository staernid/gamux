import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import DrawerDLCTab from './DrawerDLCTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { tasksStore } from '../../../stores/tasks.js';
import { normalizeGame } from '../../../utils/game.js';

describe('DrawerDLCTab Component', () => {
  const mockGame = normalizeGame({
    name: 'Torchlight II',
    app_id: '200710',
    game_dir: '/home/staernid/Games/Torchlight II',
    platform: 'win64',
    runner: 'wine',
  });

  const mockDepots = [
    {
      depot_id: 200711,
      name: 'Torchlight II Base Content',
      category: 'common',
      file_count: 50,
      size: 1400000000,
      installed: true,
    },
    {
      depot_id: 200712,
      name: 'Torchlight II Soundtrack',
      category: 'soundtrack',
      file_count: 28,
      size: 250000000,
      installed: false,
    },
  ];

  it('renders depot list and soundtrack download button', async () => {
    vi.spyOn(AppAPI, 'getGameDepots').mockResolvedValue(mockDepots);

    render(DrawerDLCTab, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/DLC & Content Depots/i)).toBeInTheDocument();
      expect(screen.getByText(/Torchlight II Base Content/i)).toBeInTheDocument();
      expect(screen.getByText(/Torchlight II Soundtrack/i)).toBeInTheDocument();
      expect(screen.getByText(/🎵 Save to Music/i)).toBeInTheDocument();
    });
  });

  it('submits soundtrack download via tasksStore.start', async () => {
    vi.spyOn(AppAPI, 'getGameDepots').mockResolvedValue(mockDepots);
    const startSpy = vi.spyOn(tasksStore, 'start').mockResolvedValue('task_soundtrack_1');

    render(DrawerDLCTab, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/🎵 Save to Music/i)).toBeInTheDocument();
    });

    const soundtrackBtn = screen.getByText(/🎵 Save to Music/i);
    await fireEvent.click(soundtrackBtn);

    expect(startSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        app_id: 200710,
        platform: 'soundtrack',
        game_name: 'Torchlight II Soundtrack',
        auto_apply: false,
      })
    );
  });
});
