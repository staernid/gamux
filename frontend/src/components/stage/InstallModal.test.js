import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import InstallModal from './InstallModal.svelte';
import { tasksStore } from '../../stores/tasks.js';
import { configStore } from '../../stores/config.js';
import { AppAPI } from '../../api/app.js';

describe('InstallModal Component', () => {
  const mockGame = {
    appId: '1091500',
    name: 'Cyberpunk 2077',
    store: 'Steam',
    isInstalled: false,
    platform: 'win64',
    branch: 'public'
  };

  const mockDepots = [
    { depot_id: 1091501, name: 'Base Game Content', content_category: 'core', size: 50000000000, installed: false },
    { depot_id: 1091502, name: 'Windows Binaries', content_category: 'core', size: 50000000, installed: false },
    { depot_id: 1091503, name: 'Phantom Liberty Expansion', content_category: 'dlc', is_dlc: true, size: 25000000000, installed: false },
    { depot_id: 1091504, name: 'French Audio Pack', content_category: 'locale:french', language: 'french', size: 4000000000, installed: false },
    { depot_id: 1091505, name: 'English Audio Pack', content_category: 'locale:english', language: 'english', size: 4000000000, installed: false }
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(AppAPI, 'getAppDepots').mockResolvedValue(mockDepots);
    vi.spyOn(AppAPI, 'createGameStub').mockResolvedValue({ game_dir: '/Games/Cyberpunk 2077' });
    vi.spyOn(tasksStore, 'start').mockResolvedValue('task_test_123');
    configStore.setSkipInstallConfirm(false);
  });

  it('renders game title, library target, and preselected DLCs', async () => {
    render(InstallModal, { game: mockGame });

    expect(screen.getByText(/Install Cyberpunk 2077/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText(/1 of 1 DLCs Included/i)).toBeInTheDocument();
    });

    expect(screen.getByRole('button', { name: /Install Game/i })).toBeInTheDocument();
  });

  it('allows customizing DLCs and starting install', async () => {
    const onclose = vi.fn();
    render(InstallModal, { game: mockGame, onclose });

    await waitFor(() => {
      expect(screen.getByText(/1 of 1 DLCs Included/i)).toBeInTheDocument();
    });

    // Expand DLC customize list
    const customizeBtn = screen.getByRole('button', { name: /Customize DLCs/i });
    await fireEvent.click(customizeBtn);

    expect(screen.getByText('Phantom Liberty Expansion')).toBeInTheDocument();

    // Click Install Game
    const installBtn = screen.getByRole('button', { name: /Install Game/i });
    await fireEvent.click(installBtn);

    await waitFor(() => {
      expect(tasksStore.start).toHaveBeenCalledWith(expect.objectContaining({
        app_id: 1091500,
        game_name: 'Cyberpunk 2077',
        platform: 'win64'
      }));
      expect(onclose).toHaveBeenCalled();
    });
  });

  it('persists skipInstallConfirm when checkbox is toggled', async () => {
    const onclose = vi.fn();
    render(InstallModal, { game: mockGame, onclose });

    await waitFor(() => {
      expect(screen.getByText(/Install Cyberpunk 2077/i)).toBeInTheDocument();
    });

    const skipCheckbox = screen.getByLabelText(/Don't ask again/i);
    await fireEvent.click(skipCheckbox);

    const installBtn = screen.getByRole('button', { name: /Install Game/i });
    await fireEvent.click(installBtn);

    await waitFor(() => {
      expect(configStore.skipInstallConfirm).toBe(true);
    });
  });
});
