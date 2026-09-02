import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import ToolsMaintenanceTab from './ToolsMaintenanceTab.svelte';
import { tasksStore } from '../../../stores/tasks.js';
import { activeView } from '../../../stores/library.js';

describe('ToolsMaintenanceTab Component', () => {
  const mockTools = [
    {
      name: 'Goldberg Steam Emulator',
      key: 'gbe',
      is_installed: false,
      installed_tag: '',
      latest_tag: 'v0.2.7',
      has_update: true
    },
    {
      name: 'Steamless DRM Unpacker',
      key: 'steamless',
      is_installed: true,
      installed_tag: 'v0.1.0',
      latest_tag: 'v0.1.2',
      has_update: true
    },
    {
      name: 'UMU Proton Launcher',
      key: 'umu',
      is_installed: true,
      installed_tag: 'v1.1.4',
      latest_tag: 'v1.1.4',
      has_update: false
    }
  ];

  it('correctly labels uninstalled tools with "Install" and "Not Installed" badge', () => {
    render(ToolsMaintenanceTab, {
      toolStatuses: mockTools,
      isCheckingTools: false,
      updatingTools: {},
      isSubmittingBatch: false,
      gamesCount: 5,
      unpatchedGamesCount: 2
    });

    // Uninstalled tool (GBE)
    expect(screen.getByText('Missing')).toBeInTheDocument();
    expect(screen.getByText('Not Installed')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Install$/i })).toBeInTheDocument();

    // Outdated installed tool (Steamless)
    expect(screen.getAllByText('Update').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByRole('button', { name: /^Update$/i })).toBeInTheDocument();

    // Up-to-date installed tool (UMU)
    expect(screen.getByRole('button', { name: /^Reinstall$/i })).toBeInTheDocument();
  });

  it('displays "Installing..." when an uninstalled tool is being installed', () => {
    render(ToolsMaintenanceTab, {
      toolStatuses: mockTools,
      isCheckingTools: false,
      updatingTools: { gbe: true },
      isSubmittingBatch: false,
      gamesCount: 5,
      unpatchedGamesCount: 2
    });

    expect(screen.getByText('Installing...')).toBeInTheDocument();
  });

  it('displays "Updating..." when an outdated tool is being updated', () => {
    render(ToolsMaintenanceTab, {
      toolStatuses: mockTools,
      isCheckingTools: false,
      updatingTools: { steamless: true },
      isSubmittingBatch: false,
      gamesCount: 5,
      unpatchedGamesCount: 2
    });

    expect(screen.getByText('Updating...')).toBeInTheDocument();
  });

  it('renders inline batch progress and "View in Tasks" link when a batch task is active', () => {
    tasksStore.items.set([
      {
        id: 'task_batch_achievements_1',
        title: 'Batch Download Missing Achievements',
        type: 'custom',
        status: 'running',
        stage: '[2/5] Cyberpunk 2077',
        details: 'Downloading schema...',
        percent: 40,
        metadata: { batch_kind: 'achievements' }
      }
    ]);

    render(ToolsMaintenanceTab, {
      toolStatuses: mockTools,
      isCheckingTools: false,
      updatingTools: {},
      isSubmittingBatch: false,
      gamesCount: 5,
      unpatchedGamesCount: 2
    });

    // Achievements card should show inline progress
    expect(screen.getByText(/\[2\/5\] Cyberpunk 2077/i)).toBeInTheDocument();
    expect(screen.getByText('40%')).toBeInTheDocument();
    expect(screen.getByText(/Running \(40%\)\.\.\./i)).toBeInTheDocument();

    // Should have inline "View in Tasks" link
    const viewLinks = screen.getAllByRole('button', { name: /View in Tasks/i });
    expect(viewLinks.length).toBeGreaterThan(0);
  });

  it('navigates to tasks view when clicking View in Tasks', async () => {
    activeView.set('settings');

    render(ToolsMaintenanceTab, {
      toolStatuses: mockTools,
      isCheckingTools: false,
      updatingTools: {},
      isSubmittingBatch: false,
      gamesCount: 5,
      unpatchedGamesCount: 2
    });

    const viewBtn = screen.getByTitle('View active background tasks');
    await fireEvent.click(viewBtn);

    let currentView;
    activeView.subscribe(v => { currentView = v; })();
    expect(currentView).toBe('tasks');
  });
});
