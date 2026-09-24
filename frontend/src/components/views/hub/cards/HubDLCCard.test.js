import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubDLCCard from './HubDLCCard.svelte';
import { AppAPI } from '../../../../api/app.js';
import { tasksStore } from '../../../../stores/tasks.js';
import { normalizeGame } from '../../../../utils/game.js';

describe('HubDLCCard Component', () => {
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

    render(HubDLCCard, { props: { game: mockGame } });

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

    render(HubDLCCard, { props: { game: mockGame } });

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

  it('renders configured DLCs and allows syncing via AppAPI.fetchDLCs', async () => {
    vi.spyOn(AppAPI, 'getGameDepots').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'getGameDLCs').mockResolvedValue([
      { app_id: '200715', name: 'Torchlight II: Pet Pack' },
    ]);
    const fetchSpy = vi.spyOn(AppAPI, 'fetchDLCs').mockResolvedValue();

    render(HubDLCCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText('Torchlight II: Pet Pack')).toBeInTheDocument();
      expect(screen.getByText(/Sync Steam DLCs/i)).toBeInTheDocument();
    });

    const syncBtn = screen.getByText(/Sync Steam DLCs/i);
    await fireEvent.click(syncBtn);

    expect(fetchSpy).toHaveBeenCalledWith(mockGame.gameDir);
  });

  it('renders language pack category badges and filters by category', async () => {
    const multiDepots = [
      {
        depot_id: 200711,
        name: 'Torchlight II Base Content',
        category: 'common',
        content_category: 'core',
        file_count: 50,
        size: 1400000000,
        installed: true,
      },
      {
        depot_id: 200713,
        name: 'Torchlight II French Audio',
        category: 'common',
        content_category: 'locale:french',
        language: 'french',
        file_count: 30,
        size: 300000000,
        installed: false,
      },
      {
        depot_id: 200714,
        name: 'Torchlight II 4K Textures',
        category: 'common',
        content_category: 'optional_hd',
        file_count: 10,
        size: 800000000,
        installed: false,
      },
    ];

    vi.spyOn(AppAPI, 'getGameDepots').mockResolvedValue(multiDepots);

    render(HubDLCCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/French Voice \/ Audio/i)).toBeInTheDocument();
      expect(screen.getByText(/High-Res \/ Optional Assets/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Languages/i })).toBeInTheDocument();
    });

    // Click on "Languages" filter pill
    const languagesFilterBtn = screen.getByRole('button', { name: /Languages/i });
    await fireEvent.click(languagesFilterBtn);

    await waitFor(() => {
      expect(screen.getByText(/Torchlight II French Audio/i)).toBeInTheDocument();
      expect(screen.queryByText(/Torchlight II 4K Textures/i)).not.toBeInTheDocument();
      expect(screen.queryByText(/Torchlight II Base Content/i)).not.toBeInTheDocument();
    });
  });

  it('installs single depot on demand with included_depots', async () => {
    const multiDepots = [
      {
        depot_id: 200713,
        name: 'Torchlight II French Audio',
        category: 'common',
        content_category: 'locale:french',
        language: 'french',
        file_count: 30,
        size: 300000000,
        installed: false,
      },
    ];

    vi.spyOn(AppAPI, 'getGameDepots').mockResolvedValue(multiDepots);
    const startSpy = vi.spyOn(tasksStore, 'start').mockResolvedValue('task_depot_1');

    render(HubDLCCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/Torchlight II French Audio/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /^Install$/i })).toBeInTheDocument();
    });

    const installBtn = screen.getByRole('button', { name: /^Install$/i });
    await fireEvent.click(installBtn);

    expect(startSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        app_id: 200710,
        included_depots: [200713],
        auto_apply: true,
      })
    );
  });

  it('uninstalls optional depot on demand', async () => {
    const installedOptionalDepot = [
      {
        depot_id: 200714,
        name: 'Torchlight II 4K Textures',
        category: 'common',
        content_category: 'optional_hd',
        file_count: 10,
        size: 800000000,
        installed: true,
      },
    ];

    vi.spyOn(AppAPI, 'getGameDepots').mockResolvedValue(installedOptionalDepot);
    const uninstallSpy = vi.spyOn(AppAPI, 'uninstallDepot').mockResolvedValue();

    render(HubDLCCard, { props: { game: mockGame } });

    await waitFor(() => {
      expect(screen.getByText(/Torchlight II 4K Textures/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Uninstall/i })).toBeInTheDocument();
    });

    const uninstallBtn = screen.getByRole('button', { name: /Uninstall/i });
    await fireEvent.click(uninstallBtn);

    expect(uninstallSpy).toHaveBeenCalledWith(mockGame.gameDir, 200714);
  });

  it('loads remote depots via AppAPI.getAppDepots for uninstalled games', async () => {
    const uninstalledGame = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '',
      is_virtual: true,
      is_installed: false,
      is_stub: false,
    });

    const remoteDepots = [
      {
        depot_id: 1091501,
        name: 'Cyberpunk 2077 Content',
        category: 'common',
        content_category: 'core',
        installed: false,
        size: 50000000000,
      },
      {
        depot_id: 1091504,
        name: 'Cyberpunk 2077 French Audio',
        category: 'common',
        content_category: 'locale:french',
        language: 'french',
        installed: false,
        size: 4500000000,
      },
    ];

    const getAppDepotsSpy = vi.spyOn(AppAPI, 'getAppDepots').mockResolvedValue(remoteDepots);
    const startSpy = vi.spyOn(tasksStore, 'start').mockResolvedValue('task_dl_all');

    render(HubDLCCard, { props: { game: uninstalledGame } });

    await waitFor(() => {
      expect(getAppDepotsSpy).toHaveBeenCalledWith('1091500');
      expect(screen.getByText(/Cyberpunk 2077 Content/i)).toBeInTheDocument();
      expect(screen.getByText(/Cyberpunk 2077 French Audio/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Download Selected/i })).toBeInTheDocument();
    });

    const dlSelectedBtn = screen.getByRole('button', { name: /Download Selected/i });
    await fireEvent.click(dlSelectedBtn);

    expect(startSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        app_id: 1091500,
        auto_apply: true,
      })
    );
  });
});
