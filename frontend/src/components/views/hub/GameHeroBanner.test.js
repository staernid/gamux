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

  it('opens launch options popover and toggles 2-player co-op mode', async () => {
    const { fireEvent } = await import('@testing-library/svelte');
    const { multiseatStore } = await import('../../../stores/multiseat.js');

    render(GameHeroBanner, {
      props: {
        game: { ...mockSteamGame, isReady: true, is_ready: true },
        isGameRunning: false,
        isTaskActive: false,
      },
    });

    expect(screen.getByText('Play Now')).toBeInTheDocument();

    // Click the split menu trigger
    const splitBtn = screen.getByLabelText(/Launch Options/i);
    await fireEvent.click(splitBtn);

    // Popover appears
    expect(screen.getByText('Launch Mode')).toBeInTheDocument();
    expect(screen.getByText('Single Player')).toBeInTheDocument();
    const coopOption = screen.getByText('Co-Op (2P)');
    expect(coopOption).toBeInTheDocument();

    // Select Co-Op (2P)
    await fireEvent.click(coopOption);

    // Button updates to Play (2P)
    expect(screen.getByText('Play (2P)')).toBeInTheDocument();

    // Reset store
    multiseatStore.toggleCoop();
  });

  it('renders trash button with no text next to files button and handles click', async () => {
    const { fireEvent } = await import('@testing-library/svelte');
    const trashHandler = vi.fn();
    render(GameHeroBanner, {
      props: {
        game: mockSteamGame,
        isGameRunning: false,
        isTaskActive: false,
        ontrash: trashHandler,
      },
    });

    const trashBtn = screen.getByRole('button', { name: /Move Game to Trash/i });
    expect(trashBtn).toBeInTheDocument();
    // Verify strictly no text inside button
    expect(trashBtn.textContent.trim()).toBe('');

    await fireEvent.click(trashBtn);
    expect(trashHandler).toHaveBeenCalled();
  });

  it('disables trash button when game is currently running', () => {
    render(GameHeroBanner, {
      props: {
        game: mockSteamGame,
        isGameRunning: true,
        isTaskActive: false,
      },
    });

    const trashBtn = screen.getByRole('button', { name: /Move Game to Trash/i });
    expect(trashBtn).toBeDisabled();
  });

  it('renders trash button for uninstalled Stubs that have a game directory in the library', () => {
    const stubGame = normalizeGame({
      name: 'Skeleton Stub Game',
      app_id: '99999',
      game_dir: '/home/staernid/Games/Skeleton Stub Game',
      platform: 'win64',
      is_installed: false,
      is_stub: true,
    });

    render(GameHeroBanner, {
      props: {
        game: stubGame,
        isGameRunning: false,
        isTaskActive: false,
      },
    });

    const trashBtn = screen.getByRole('button', { name: /Move Game to Trash/i });
    expect(trashBtn).toBeInTheDocument();
  });

  it('does NOT render trash button for online catalog previews (virtual titles)', () => {
    const virtualGame = {
      name: 'Online Catalog Preview',
      app_id: '123456',
      game_dir: '',
      gameDir: '',
      isVirtual: true,
      is_virtual: true,
      is_installed: false,
    };

    render(GameHeroBanner, {
      props: {
        game: virtualGame,
        isGameRunning: false,
        isTaskActive: false,
      },
    });

    expect(screen.queryByRole('button', { name: /Move Game to Trash/i })).not.toBeInTheDocument();
  });
});
