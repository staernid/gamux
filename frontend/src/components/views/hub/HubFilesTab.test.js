import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import HubFilesTab from './HubFilesTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { normalizeGame } from '../../../utils/game.js';
import { selectedGame } from '../../../stores/library.js';

describe('HubFilesTab Component', () => {
  const mockSteamGame = normalizeGame({
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    store: 'Steam',
    game_dir: '/games/cyberpunk',
    is_installed: true,
    branch: 'beta'
  });

  beforeEach(() => {
    vi.restoreAllMocks();
    selectedGame.set(null);
  });

  it('renders all 4 subtabs including Branch for Steam games with branch badge', async () => {
    render(HubFilesTab, { props: { game: mockSteamGame } });

    expect(screen.getByRole('button', { name: /DLC & Depots/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Branch/i })).toBeInTheDocument();
    expect(screen.getByText('beta')).toBeInTheDocument(); // Badge on Branch pill
    expect(screen.getByRole('button', { name: /Integrity & Repair/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Execution Logs/i })).toBeInTheDocument();
  });

  it('switches to Branch subtab and renders Steam Betas & Branches manager', async () => {
    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue([
      { name: 'public', description: 'Default', build_id: '100', time_updated: 0, pwd_required: false },
      { name: 'beta', description: 'Testing', build_id: '200', time_updated: 0, pwd_required: false }
    ]);

    render(HubFilesTab, { props: { game: mockSteamGame, activeSubTab: 'branch' } });

    expect(await screen.findByText('Steam Betas & Branches')).toBeInTheDocument();
  });

  it('allows uninstalled Steam titles to access Branch subtab while gating disk inspection', async () => {
    const uninstalledSteamGame = normalizeGame({
      name: 'Lossless Scaling',
      app_id: '993090',
      store: 'Steam',
      game_dir: '',
      is_virtual: true,
      is_installed: false,
      is_stub: false,
      branch: 'public'
    });

    vi.spyOn(AppAPI, 'getSteamBranches').mockResolvedValue([
      { name: 'public', description: 'Default', build_id: '100', time_updated: 0, pwd_required: false }
    ]);

    render(HubFilesTab, { props: { game: uninstalledSteamGame, activeSubTab: 'branch' } });

    // Branch manager renders cleanly for uninstalled titles
    expect(await screen.findByText('Steam Betas & Branches')).toBeInTheDocument();

    // Clicking DLC & Depots shows uninstalled placeholder inside files tab
    const depotsBtn = screen.getByRole('button', { name: /DLC & Depots/i });
    await fireEvent.click(depotsBtn);

    expect(await screen.findByText('Game Files Not Installed')).toBeInTheDocument();
  });

  it('hides Branch subtab for non-Steam standalone titles', async () => {
    const standaloneGame = normalizeGame({
      name: 'Freedoom',
      app_id: '0',
      store: 'Standalone',
      game_dir: '/games/freedoom',
      is_installed: true
    });

    render(HubFilesTab, { props: { game: standaloneGame } });

    expect(screen.getByRole('button', { name: /DLC & Depots/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Branch$/i })).not.toBeInTheDocument();
  });
});
