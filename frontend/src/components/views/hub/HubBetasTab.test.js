import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import HubBetasTab from './HubBetasTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { normalizeGame } from '../../../utils/game.js';
import { selectedGame } from '../../../stores/library.js';

describe('HubBetasTab Component', () => {
  const mockBranches = [
    {
      name: 'public',
      description: 'Default public release branch',
      build_id: '1200000',
      time_updated: 1720000000,
      pwd_required: false
    },
    {
      name: 'beta',
      description: 'Public testing branch',
      build_id: '1205000',
      time_updated: 1725000000,
      pwd_required: false
    },
    {
      name: 'private_test',
      description: 'QA internal branch',
      build_id: '1210000',
      time_updated: 1726000000,
      pwd_required: true
    }
  ];

  beforeEach(() => {
    vi.restoreAllMocks();
    selectedGame.set(null);
  });

  it('renders non-Steam warning when game lacks Steam App ID', async () => {
    const nonSteamGame = normalizeGame({
      name: 'Standalone Game',
      app_id: '0',
      store: 'Standalone',
      game_dir: '/games/standalone'
    });

    render(HubBetasTab, { props: { game: nonSteamGame } });

    expect(screen.getByText('Steam AppID Required')).toBeInTheDocument();
  });

  it('fetches and displays branches for a Steam game', async () => {
    const mockSteamGame = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '/games/cyberpunk',
      is_installed: true,
      build_id: '1200000',
      branch: 'public'
    });

    const getSpy = vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue(mockBranches);

    render(HubBetasTab, { props: { game: mockSteamGame } });

    expect(getSpy).toHaveBeenCalledWith(1091500);
    expect(await screen.findByText('None / Default (public)')).toBeInTheDocument();
    expect(screen.getByText('beta')).toBeInTheDocument();
    expect(screen.getByText('private_test')).toBeInTheDocument();
    expect(screen.getByText('Public testing branch')).toBeInTheDocument();
    expect(screen.getByText('Build #1205000')).toBeInTheDocument();
    expect(screen.getByText('Private')).toBeInTheDocument();
  });

  it('allows selecting branch on virtual store title and dispatches branchChange', async () => {
    const virtualGame = normalizeGame({
      name: 'Hades II',
      app_id: '1145350',
      store: 'Steam',
      game_dir: '',
      is_virtual: true,
      is_installed: false,
      is_stub: false,
      branch: 'public'
    });
    selectedGame.set(virtualGame);

    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue(mockBranches);
    const branchChangeHandler = vi.fn();

    render(HubBetasTab, { props: { game: virtualGame, onbranchChange: branchChangeHandler } });

    expect(await screen.findByText('beta')).toBeInTheDocument();

    const switchBtns = screen.getAllByRole('button', { name: /Switch to Branch/i });
    await fireEvent.click(switchBtns[0]); // Click beta branch switch

    expect(branchChangeHandler).toHaveBeenCalledWith(expect.objectContaining({
      branch: 'beta',
      targetBuildId: '1205000'
    }));
  });

  it('switches branch on library stub and updates via setGameBranch API', async () => {
    const stubGame = normalizeGame({
      name: 'Hades II',
      app_id: '1145350',
      store: 'Steam',
      game_dir: '/games/hades2',
      is_virtual: false,
      is_installed: false,
      is_stub: true,
      branch: 'public'
    });
    selectedGame.set(stubGame);

    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue(mockBranches);
    const setSpy = vi.spyOn(AppAPI, 'setGameBranch').mockResolvedValue({
      ...stubGame,
      branch: 'beta'
    });

    render(HubBetasTab, { props: { game: stubGame } });

    expect(await screen.findByText('beta')).toBeInTheDocument();

    const switchBtns = screen.getAllByRole('button', { name: /Switch to Branch/i });
    await fireEvent.click(switchBtns[0]);

    expect(setSpy).toHaveBeenCalledWith('/games/hades2', 'beta', '');
  });

  it('shows file sync warning on installed game when branch build differs from installed build', async () => {
    const installedGame = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '/games/cyberpunk',
      is_virtual: false,
      is_installed: true,
      is_stub: false,
      build_id: '1200000',
      branch: 'beta' // Active branch target is 1205000, installed is 1200000
    });
    selectedGame.set(installedGame);

    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue(mockBranches);

    render(HubBetasTab, { props: { game: installedGame } });

    expect(await screen.findByText('Files Pending Branch Sync')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Switch Files to Branch/i })).toBeInTheDocument();
  });

  it('does NOT show false file sync warning when installed build ID is 0 or empty', async () => {
    const placeholderGame = normalizeGame({
      name: 'Lossless Scaling',
      app_id: '993090',
      store: 'Steam',
      game_dir: '/games/lossless',
      is_virtual: false,
      is_installed: true,
      is_stub: false,
      build_id: '0',
      branch: 'beta'
    });
    selectedGame.set(placeholderGame);

    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue(mockBranches);

    render(HubBetasTab, { props: { game: placeholderGame } });

    expect(await screen.findByText('beta')).toBeInTheDocument();
    expect(screen.queryByText('Files Pending Branch Sync')).not.toBeInTheDocument();
    expect(screen.queryByText(/Installed files: #0/i)).not.toBeInTheDocument();
  });

  it('checks private beta code and updates password-protected branch', async () => {
    const stubGame = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '/games/cyberpunk',
      is_virtual: false,
      is_installed: false,
      is_stub: true,
      branch: 'public'
    });
    selectedGame.set(stubGame);

    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue(mockBranches);
    const setSpy = vi.spyOn(AppAPI, 'setGameBranch').mockResolvedValue({
      ...stubGame,
      branch: 'private_test'
    });

    render(HubBetasTab, { props: { game: stubGame } });

    expect(await screen.findByText('Private Betas')).toBeInTheDocument();

    const input = screen.getByPlaceholderText(/Enter beta access code/i);
    await fireEvent.input(input, { target: { value: 'SECRET_BETA_2026' } });

    const checkBtn = screen.getByRole('button', { name: /Check Code/i });
    await fireEvent.click(checkBtn);

    expect(setSpy).toHaveBeenCalledWith('/games/cyberpunk', 'private_test', 'SECRET_BETA_2026');
    expect(await screen.findByText(/Beta code accepted/i)).toBeInTheDocument();
  });
});
