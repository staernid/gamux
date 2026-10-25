import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubExtendedInfoCard from './HubExtendedInfoCard.svelte';
import { AppAPI } from '../../../../api/app.js';
import { normalizeGame } from '../../../../utils/game.js';

describe('HubExtendedInfoCard Component', () => {
  const mockSteamGame = normalizeGame({
    name: 'Counter-Strike 2',
    app_id: '730',
    game_dir: '/home/staernid/Games/Counter-Strike 2',
    platform: 'linux',
    runner: 'linux',
    is_installed: true,
    playtime_seconds: 7200,
    last_played: 1672531199,
    disk_size_bytes: 45000000000,
    installed_at: 1670000000,
  });

  it('renders extended info stats: playtime, disk size, and app ID', () => {
    render(HubExtendedInfoCard, {
      props: {
        game: mockSteamGame,
      },
    });

    expect(screen.getByText('Extended Info')).toBeInTheDocument();
    expect(screen.getByText('#730')).toBeInTheDocument();
    expect(screen.getByText('2 hrs')).toBeInTheDocument();
    expect(screen.getByText('41.91 GiB')).toBeInTheDocument();
    expect(screen.getByText(/Counter-Strike 2/)).toBeInTheDocument();
  });

  it('fetches and displays live player count for Steam games', async () => {
    vi.spyOn(AppAPI, 'getGameCurrentPlayers').mockResolvedValue(45120);

    render(HubExtendedInfoCard, {
      props: {
        game: mockSteamGame,
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/45,120 playing now/i)).toBeInTheDocument();
    });
  });
});
