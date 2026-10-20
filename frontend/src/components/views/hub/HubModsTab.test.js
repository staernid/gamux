import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import HubModsTab from './HubModsTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { normalizeGame } from '../../../utils/game.js';

describe('HubModsTab Component', () => {
  const mockGame = normalizeGame({
    name: 'Risk of Rain 2',
    app_id: '632360',
    store: 'Steam',
    game_dir: '/games/ror2',
  });

  const mockModStatus = {
    topology: {
      engine: 'unity',
      mod_dir: 'BepInEx/plugins',
      absolute_mod_dir: '/games/ror2/BepInEx/plugins',
      detected: true,
      supported_formats: ['.dll', '.zip']
    },
    mods: [
      {
        id: 'ItemStatsMod.dll',
        name: 'ItemStatsMod',
        workshop_id: 315783921,
        enabled: true,
        path: '/games/ror2/BepInEx/plugins/ItemStatsMod.dll',
        relative_path: 'BepInEx/plugins/ItemStatsMod.dll',
        time_updated: 1720000000,
        latest_time_updated: 1729000000,
        has_update: true,
        description: 'Displays detailed item stats.',
        size: 1048576,
        files: ['ItemStatsMod.dll']
      },
      {
        id: 'ProperSave.dll',
        name: 'ProperSave',
        workshop_id: 315783922,
        enabled: false,
        path: '/games/ror2/BepInEx/plugins/ProperSave.dll.disabled',
        relative_path: 'BepInEx/plugins/ProperSave.dll.disabled',
        time_updated: 1715000000,
        latest_time_updated: 1715000000,
        has_update: false,
        description: 'Saves run progress.',
        size: 524288,
        files: ['ProperSave.dll']
      }
    ],
    outdated_count: 1
  };

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders engine topology and detected badge', async () => {
    vi.spyOn(AppAPI, 'getGameMods').mockResolvedValue(mockModStatus);

    render(HubModsTab, { props: { game: mockGame } });

    expect(await screen.findByText('unity')).toBeInTheDocument();
    expect(screen.getByText('BepInEx/plugins')).toBeInTheDocument();
  });

  it('renders installed mods list and update alert banner', async () => {
    vi.spyOn(AppAPI, 'getGameMods').mockResolvedValue(mockModStatus);

    render(HubModsTab, { props: { game: mockGame } });

    expect(await screen.findByText('ItemStatsMod')).toBeInTheDocument();
    expect(screen.getByText('ProperSave')).toBeInTheDocument();
    expect(screen.getByText('Update Available')).toBeInTheDocument();
    expect(screen.getByText(/Workshop Item.*updates/i)).toBeInTheDocument();
  });

  it('renders empty state when no mods installed', async () => {
    vi.spyOn(AppAPI, 'getGameMods').mockResolvedValue({
      topology: {
        engine: 'generic',
        mod_dir: 'mods',
        detected: false,
      },
      mods: [],
      outdated_count: 0
    });

    render(HubModsTab, { props: { game: mockGame } });

    expect(await screen.findByText('No mods installed')).toBeInTheDocument();
  });

  it('submits workshop installation request', async () => {
    vi.spyOn(AppAPI, 'getGameMods').mockResolvedValue(mockModStatus);
    const installSpy = vi.spyOn(AppAPI, 'installWorkshopMod').mockResolvedValue('task-123');

    render(HubModsTab, { props: { game: mockGame } });

    const input = await screen.findByPlaceholderText(/Enter Steam Workshop URL or ID/i);
    await fireEvent.input(input, { target: { value: '315783921' } });

    const downloadBtn = screen.getByRole('button', { name: /Download & Install/i });
    await fireEvent.click(downloadBtn);

    expect(installSpy).toHaveBeenCalledWith('/games/ror2', '315783921');
  });
});
