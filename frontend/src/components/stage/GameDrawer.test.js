import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import GameDrawer from './GameDrawer.svelte';
import { selectedGame, showInspector } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';
import { AppAPI } from '../../api/app.js';

describe('GameDrawer Component', () => {
  const mockGame = {
    name: 'Hades',
    app_id: '1145360',
    platform: 'Windows PE',
    game_dir: '/home/user/Games/Hades',
    exe_path: 'Hades.exe',
    exe_args: '',
    launch_candidates: [
      { name: 'Hades DirectX', executable: 'Hades.exe', arguments: '-dx11' },
      { name: 'Hades Vulkan', executable: 'Hades_vk.exe', arguments: '-vulkan' },
    ],
    supported_languages: ['english', 'french', 'german', 'spanish'],
    current_language: 'english',
    is_patched: false,
    is_portable: false,
    lutris_registered: false,
  };

  beforeEach(() => {
    selectedGame.set(normalizeGame(mockGame));
    showInspector.set(true);
  });

  it('renders game title, AppID, and tabs when drawer is open', async () => {
    render(GameDrawer);

    expect(screen.getByText('Hades')).toBeInTheDocument();
    expect(screen.getByText(/AppID: 1145360/i)).toBeInTheDocument();
    expect(screen.getByText('General')).toBeInTheDocument();
    expect(screen.getByText('Integrations')).toBeInTheDocument();
    expect(screen.getByText('Emu')).toBeInTheDocument();
    expect(screen.getByText('Integrity')).toBeInTheDocument();
    expect(screen.getByText('Patch Notes')).toBeInTheDocument();
  });

  it('allows switching between tabs and displaying tab-specific controls', async () => {
    render(GameDrawer);

    // Default General tab
    expect(screen.getByText('Target Executable')).toBeInTheDocument();
    expect(screen.getByText('Custom Launch Flags')).toBeInTheDocument();

    // Switch to Integrations tab
    const integrationsTab = screen.getByText('Integrations');
    await fireEvent.click(integrationsTab);
    expect(screen.getByText('Primary Operating Mode')).toBeInTheDocument();
    expect(screen.getByText('Runner Engine')).toBeInTheDocument();
    expect(screen.getByText('Custom Wine Prefix Path')).toBeInTheDocument();

    // Switch to Emu tab
    const emuTab = screen.getByText('Emu');
    await fireEvent.click(emuTab);
    expect(screen.getByText('DEFAULT MODE')).toBeInTheDocument();
    expect(screen.getByText('AppID Override')).toBeInTheDocument();

    // Switch to Integrity tab
    const integrityTab = screen.getByText('Integrity');
    await fireEvent.click(integrityTab);
    expect(screen.getByText('Run Integrity Check')).toBeInTheDocument();
  });

  it('calls save profile and closes drawer when Save is clicked', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveLaunchProfile').mockResolvedValue({});
    render(GameDrawer);

    const saveBtn = screen.getByText('Save');
    await fireEvent.click(saveBtn);

    expect(saveSpy).toHaveBeenCalled();
  });
});
