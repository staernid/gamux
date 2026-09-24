import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import SettingsView from './SettingsView.svelte';
import { AppAPI } from '../../api/app.js';

describe('SettingsView Component', () => {
  it('renders primary system settings tabs and general form fields by default', async () => {
    render(SettingsView);

    expect(screen.getByText('System Settings')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /General/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Storage/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Compatibility/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Profiles/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Credentials/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Maintenance/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Logs/i })).toBeInTheDocument();

    // Default tab is General
    expect(screen.getByText('Display & Interface')).toBeInTheDocument();
    expect(screen.getByText('Interface Scale')).toBeInTheDocument();
    expect(screen.getByText('Network & Downloads')).toBeInTheDocument();
    expect(screen.getByText('Transfer Speed Limit')).toBeInTheDocument();
    expect(screen.getByText('Launch Notifications')).toBeInTheDocument();
  });

  it('switches to Storage tab and renders storage directories and backups', async () => {
    render(SettingsView);

    const storageTab = screen.getByRole('tab', { name: /Storage/i });
    await fireEvent.click(storageTab);

    expect(screen.getByText('Library Directories')).toBeInTheDocument();
    expect(screen.getByText('Automation & Backups')).toBeInTheDocument();
    expect(screen.getByText('Normalize Folders')).toBeInTheDocument();
    expect(screen.getByText('Auto-Backup Saves')).toBeInTheDocument();
    expect(screen.getByText('Advanced Storage Locations')).toBeInTheDocument();
  });

  it('renders custom speed limit input when custom bandwidth is selected in General tab', async () => {
    render(SettingsView);

    const speedSelect = screen.getByDisplayValue(/Unlimited/i);
    await fireEvent.change(speedSelect, { target: { value: '-1' } });

    expect(screen.getByText(/Custom Limit \(MB\/s\):/i)).toBeInTheDocument();
  });

  it('switches to Profiles tab and renders persona settings with custom ISO support', async () => {
    render(SettingsView);

    const profilesTab = screen.getByRole('tab', { name: /Profiles/i });
    await fireEvent.click(profilesTab);

    expect(screen.getByText('Active Profile')).toBeInTheDocument();
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByText(/Steam ID/i)).toBeInTheDocument();
    expect(screen.getByText(/Region Code/i)).toBeInTheDocument();
  });

  it('switches to Credentials tab and renders API key settings with Get Key / Docs actions', async () => {
    render(SettingsView);

    const credsTab = screen.getByRole('tab', { name: /Credentials/i });
    await fireEvent.click(credsTab);

    expect(screen.getByText('API Keys & Service Credentials')).toBeInTheDocument();
    expect(screen.getByText('Steam Web API Key')).toBeInTheDocument();
    expect(screen.getByText('Hubcap Manifest API Key')).toBeInTheDocument();
    expect(screen.getByText('SteamGridDB API Key')).toBeInTheDocument();
    expect(screen.getByText('RAWG API Key')).toBeInTheDocument();
    expect(screen.getAllByText('Get Key').length).toBeGreaterThan(0);
    expect(screen.getByText('Docs')).toBeInTheDocument();
  });

  it('switches to Maintenance tab and displays toolchain and batch operations', async () => {
    render(SettingsView);

    const toolsTab = screen.getByRole('tab', { name: /Maintenance/i });
    await fireEvent.click(toolsTab);

    expect(screen.getByText('Helper Tools')).toBeInTheDocument();
    expect(screen.getByText('Batch Maintenance')).toBeInTheDocument();
    expect(screen.getByText('Achievements')).toBeInTheDocument();
    expect(screen.getByText('Artwork')).toBeInTheDocument();
    expect(screen.getByText('Apply Setup')).toBeInTheDocument();
    expect(screen.getByText('Normalize Folders')).toBeInTheDocument();
    expect(screen.getByText('Verify Manifests')).toBeInTheDocument();
  });

  it('switches to Co-Op tab and persists software cursor and pointer toggles', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveConfig').mockResolvedValue();
    render(SettingsView);

    // Wait for onMount to complete initial config hydration
    await new Promise(r => setTimeout(r, 50));

    const coopTab = screen.getByRole('tab', { name: /Co-Op/i });
    await fireEvent.click(coopTab);

    expect(screen.getByText('Player 2 Pointer & Compositor Options')).toBeInTheDocument();
    const softwareCursorText = screen.getByText('Software Cursor');
    const softwareCursorContainer = softwareCursorText.closest('div.flex');
    const softwareCursorSwitch = softwareCursorContainer.querySelector('[role="switch"]');

    expect(softwareCursorSwitch).toBeInTheDocument();
    expect(softwareCursorSwitch.getAttribute('aria-checked')).toBe('true');

    // Click toggle
    await fireEvent.click(softwareCursorSwitch);
    expect(softwareCursorSwitch.getAttribute('aria-checked')).toBe('false');

    // Changes are auto-saved with debounce
    await waitFor(() => {
      expect(saveSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          multiseat: expect.objectContaining({
            seat2_show_cursor: false,
            seat2_compositor: 'auto'
          })
        })
      );
    }, { timeout: 1500 });
  });

  it('switches to All Options schema tab and renders dynamic declarative configuration engine', async () => {
    render(SettingsView);

    const schemaTab = screen.getByRole('tab', { name: /All Options/i });
    expect(schemaTab).toBeInTheDocument();
    await fireEvent.click(schemaTab);

    expect(screen.getByText('Declarative Configuration Engine')).toBeInTheDocument();
  });
});

