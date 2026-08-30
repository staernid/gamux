import { render, fireEvent, screen, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import SettingsView from './SettingsView.svelte';
import { AppAPI } from '../../api/app.js';
import { notificationStore } from '../../stores/notifications.js';

describe('SettingsView Component', () => {
  beforeEach(() => {
    notificationStore.items.clearAll();
  });

  it('renders settings sections and tabs properly', async () => {
    render(SettingsView);

    expect(screen.getByText(/System Settings/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'System' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'DRM' })).toBeInTheDocument();
  });

  it('queues GBE update from DRM tab and notifies user with a clean notification', async () => {
    vi.spyOn(AppAPI, 'updateTool').mockResolvedValue('dl_tool_abc123');
    vi.spyOn(AppAPI, 'getToolsStatus').mockResolvedValue([{ key: 'gbe', has_update: true, latest_tag: 'v0.2.7' }]);
    const toastSpy = vi.spyOn(notificationStore.items, 'add');

    render(SettingsView);

    // Switch to DRM tab
    const drmTab = screen.getByRole('button', { name: 'DRM' });
    await fireEvent.click(drmTab);

    expect(screen.getByText('gbe_fork')).toBeInTheDocument();

    const updateBtn = screen.getByText(/Update to Latest|Reinstall/i);
    await fireEvent.click(updateBtn);

    expect(AppAPI.updateTool).toHaveBeenCalledWith('gbe');
    expect(toastSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'info',
        title: 'Download Queued',
        message: expect.stringContaining('Goldberg Steam Emulator update queued in Downloads'),
      })
    );
  });

  it('triggers batch achievements download from Tools tab', async () => {
    vi.spyOn(AppAPI, 'submitBatchLibraryOp').mockResolvedValue('task_achievements_1');
    const toastSpy = vi.spyOn(notificationStore.items, 'add');

    render(SettingsView);

    const toolsTab = screen.getByRole('button', { name: 'Tools' });
    await fireEvent.click(toolsTab);

    const downloadAchBtn = screen.getByRole('button', { name: /Download Achievements/i });
    expect(downloadAchBtn).toBeInTheDocument();
    await fireEvent.click(downloadAchBtn);

    expect(AppAPI.submitBatchLibraryOp).toHaveBeenCalledWith('achievements');
    expect(toastSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'info',
        title: 'Batch Achievements Task Queued',
      })
    );
  });
});
