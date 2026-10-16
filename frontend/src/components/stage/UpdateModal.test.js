import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import UpdateModal from './UpdateModal.svelte';
import { tasksStore } from '../../stores/tasks.js';

describe('UpdateModal Component', () => {
  const mockGame = {
    appId: '1091500',
    name: 'Cyberpunk 2077',
    store: 'Steam',
    gameDir: '/Games/Cyberpunk 2077',
    isInstalled: true,
    platform: 'win64',
    branch: 'public',
    buildId: 100
  };

  const mockVersionStatus = {
    is_up_to_date: false,
    current_build_id: 100,
    latest_build_id: 105,
    latest_patch_title: 'Patch 2.13 Stability Fixes',
    latest_patch_date: 1726142400
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(tasksStore, 'start').mockResolvedValue('task_update_123');
  });

  it('renders game info and build comparison', () => {
    render(UpdateModal, { game: mockGame, versionStatus: mockVersionStatus });

    expect(screen.getByText('Cyberpunk 2077')).toBeInTheDocument();
    expect(screen.getByText('Build #100')).toBeInTheDocument();
    expect(screen.getByText('Build #105')).toBeInTheDocument();
    expect(screen.getByText('Patch 2.13 Stability Fixes')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Update Game Now/i })).toBeInTheDocument();
  });

  it('triggers update callback or tasksStore.start when clicked', async () => {
    const onclose = vi.fn();
    const onupdate = vi.fn();
    render(UpdateModal, { game: mockGame, versionStatus: mockVersionStatus, onclose, onupdate });

    const updateBtn = screen.getByRole('button', { name: /Update Game Now/i });
    await fireEvent.click(updateBtn);

    await waitFor(() => {
      expect(onupdate).toHaveBeenCalledTimes(1);
      expect(onclose).toHaveBeenCalledTimes(1);
    });
  });

  it('falls back to tasksStore.start when onupdate is not provided', async () => {
    const onclose = vi.fn();
    render(UpdateModal, { game: mockGame, versionStatus: mockVersionStatus, onclose });

    const updateBtn = screen.getByRole('button', { name: /Update Game Now/i });
    await fireEvent.click(updateBtn);

    await waitFor(() => {
      expect(tasksStore.start).toHaveBeenCalledWith(
        expect.objectContaining({
          app_id: 1091500,
          target_dir: '/Games/Cyberpunk 2077',
          branch: 'public'
        })
      );
      expect(onclose).toHaveBeenCalledTimes(1);
    });
  });
});
