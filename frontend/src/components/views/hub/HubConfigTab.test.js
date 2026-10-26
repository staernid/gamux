import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import HubConfigTab from './HubConfigTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { modalStore } from '../../../stores/modal.js';
import { drawerDraft } from '../../../stores/drawerDraft.js';

describe('HubConfigTab Component', () => {
  const mockGame = {
    appId: '1086940',
    name: "Baldur's Gate 3",
    gameDir: '/games/bg3',
    store: 'steam',
    platform: 'win64',
    exePath: 'bin/bg3.exe',
    isInstalled: true,
    fileCount: 1500,
    launchCandidates: [
      { name: 'Vulkan', executable: 'bin/bg3.exe' },
      { name: 'DirectX 11', executable: 'bin/bg3_dx11.exe' }
    ]
  };

  const mockGlobalCfg = {
    default_runner: 'umu',
    proton_path: 'GE-Proton',
    mangohud: false
  };

  beforeEach(() => {
    vi.restoreAllMocks();
    drawerDraft.set({
      platform: 'win64',
      runner: 'umu',
      appId: '1086940',
      customEnv: {},
      exePath: 'bin/bg3.exe',
      exeArgs: ''
    });
  });

  it('renders configuration sections and launch candidate info', () => {
    render(HubConfigTab, {
      game: mockGame,
      globalCfg: mockGlobalCfg,
      overrideCount: 2
    });

    expect(screen.getByText('Executable & Arguments')).toBeInTheDocument();
    expect(screen.getByText('Environment Variables')).toBeInTheDocument();
    expect(screen.getByText('Store Identity & Artwork')).toBeInTheDocument();
  });

  it('toggles dry run command preview and loads command', async () => {
    vi.spyOn(AppAPI, 'previewGameLaunch').mockResolvedValue({
      command: 'WINEPREFIX=/pfx umu-run /games/bg3/bin/bg3.exe'
    });

    render(HubConfigTab, {
      game: mockGame,
      globalCfg: mockGlobalCfg
    });

    const previewToggle = screen.getByRole('button', { name: /Effective Launch Command/i });
    expect(previewToggle).toBeInTheDocument();
    await fireEvent.click(previewToggle);

    const commandText = await screen.findByText(/umu-run \/games\/bg3\/bin\/bg3.exe/);
    expect(commandText).toBeInTheDocument();
  });

  it('toggles filter overrides mode when clicking Filter Overrides', async () => {
    render(HubConfigTab, {
      game: mockGame,
      globalCfg: mockGlobalCfg,
      overrideCount: 1
    });

    const filterBtn = screen.getByRole('button', { name: /Filter Overrides/i });
    expect(filterBtn).toBeInTheDocument();
    await fireEvent.click(filterBtn);

    expect(screen.getByText('Showing Overrides')).toBeInTheDocument();
  });

  it('opens metadata-link modal when clicking Change Match', async () => {
    const modalSpy = vi.spyOn(modalStore, 'open');

    render(HubConfigTab, {
      game: mockGame,
      globalCfg: mockGlobalCfg
    });

    const matchBtn = screen.getByRole('button', { name: /Change Match/i });
    await fireEvent.click(matchBtn);

    expect(modalSpy).toHaveBeenCalledWith('metadata-link', { game: mockGame });
  });
});
