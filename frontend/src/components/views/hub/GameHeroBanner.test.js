import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import GameHeroBanner from './GameHeroBanner.svelte';
import { AppAPI } from '../../../api/app.js';
import { normalizeGame } from '../../../utils/game.js';

describe('GameHeroBanner Component', () => {
  const mockSteamGame = normalizeGame({
    name: 'Counter-Strike 2',
    app_id: '730',
    game_dir: '/home/staernid/Games/Counter-Strike 2',
    platform: 'linux',
    runner: 'linux',
    is_installed: true,
  });

  it('renders title and metadata', () => {
    render(GameHeroBanner, {
      props: {
        game: mockSteamGame,
        isGameRunning: false,
        isTaskActive: false,
      },
    });

    expect(screen.getByText('Counter-Strike 2')).toBeInTheDocument();
    expect(screen.getByText('#730')).toBeInTheDocument();
  });

  it('fetches and displays live player count badge for Steam games', async () => {
    vi.spyOn(AppAPI, 'getGameCurrentPlayers').mockResolvedValue(55420);

    render(GameHeroBanner, {
      props: {
        game: mockSteamGame,
        isGameRunning: false,
        isTaskActive: false,
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/55,420 playing/i)).toBeInTheDocument();
    });
  });

  it('displays UPDATE AVAILABLE badge when version check detects update', async () => {
    vi.spyOn(AppAPI, 'checkGameUpdates').mockResolvedValue({
      checked: true,
      is_up_to_date: false,
      message: 'New patch available',
      latest_patch_title: 'Major Patch v1.5',
    });

    render(GameHeroBanner, {
      props: {
        game: mockSteamGame,
        isGameRunning: false,
        isTaskActive: false,
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/UPDATE AVAILABLE/i)).toBeInTheDocument();
    });
  });
});
