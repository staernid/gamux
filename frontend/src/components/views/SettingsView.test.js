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
    expect(screen.getByRole('tab', { name: /Advanced/i })).toBeInTheDocument();

    // Default tab is General
    expect(screen.getByText('Display & Interface')).toBeInTheDocument();
    expect(screen.getByText('Interface Scale')).toBeInTheDocument();
    expect(screen.getByText('Network & Downloads')).toBeInTheDocument();
    expect(screen.getByText('Transfer Speed Limit')).toBeInTheDocument();
    expect(screen.getByText('Launch Notifications')).toBeInTheDocument();
  });

  it('switches to Storage tab and renders storage directories, backups, and API credentials', async () => {
    render(SettingsView);

    const storageTab = screen.getByRole('tab', { name: /Storage/i });
    await fireEvent.click(storageTab);

    expect(screen.getByText('Library Directories')).toBeInTheDocument();
    expect(screen.getByText('Automation & Backups')).toBeInTheDocument();
    expect(screen.getByText('Normalize Folders')).toBeInTheDocument();
    expect(screen.getByText('Auto-Backup Saves')).toBeInTheDocument();
    expect(screen.getByText('Advanced Storage Locations')).toBeInTheDocument();

    // Credentials unified in Storage
    expect(screen.getByText('API Keys & Service Credentials')).toBeInTheDocument();
    expect(screen.getByText('Steam Web API Key')).toBeInTheDocument();
    expect(screen.getByText('Hubcap Manifest API Key')).toBeInTheDocument();
    expect(screen.getByText('SteamGridDB API Key')).toBeInTheDocument();
    expect(screen.getByText('RAWG API Key')).toBeInTheDocument();
    expect(screen.getAllByText('Get Key').length).toBeGreaterThan(0);
    expect(screen.getByText('Docs')).toBeInTheDocument();
  });

  it('renders custom speed limit input when custom bandwidth is selected in General tab', async () => {
    render(SettingsView);

    const speedSelect = screen.getByDisplayValue(/Unlimited/i);
    await fireEvent.change(speedSelect, { target: { value: '-1' } });

    expect(screen.getByText(/Custom Limit \(MB\/s\):/i)).toBeInTheDocument();
  });

  it('switches to Compatibility tab and renders System Baseline banner and Global Env editor', async () => {
    render(SettingsView);

    const compatTab = screen.getByRole('tab', { name: /Compatibility/i });
    await fireEvent.click(compatTab);

    expect(screen.getByText('Global Runtime Baseline')).toBeInTheDocument();
    expect(screen.getByText('Runtime & Execution Layer')).toBeInTheDocument();
    expect(screen.getByText('Steam & Goldberg Emulation')).toBeInTheDocument();
    expect(screen.getByText('Process Environment')).toBeInTheDocument();
    expect(screen.getByText('Global Environment Variables')).toBeInTheDocument();
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

  it('switches to Advanced tab and displays Maintenance subtab with toolchain and batch operations', async () => {
    render(SettingsView);

    const advancedTab = screen.getByRole('tab', { name: /Advanced/i });
    await fireEvent.click(advancedTab);

    // Advanced subtabs exist
    const maintenanceSubtab = screen.getByRole('tab', { name: /Maintenance/i });
    expect(maintenanceSubtab).toBeInTheDocument();
    await fireEvent.click(maintenanceSubtab);

    expect(screen.getByText('Helper Tools')).toBeInTheDocument();
    expect(screen.getByText('Batch Maintenance')).toBeInTheDocument();
    expect(screen.getByText('Achievements')).toBeInTheDocument();
    expect(screen.getByText('Artwork')).toBeInTheDocument();
    expect(screen.getByText('Apply Setup')).toBeInTheDocument();
    expect(screen.getByText('Normalize Folders')).toBeInTheDocument();
    expect(screen.getByText('Verify Manifests')).toBeInTheDocument();
  });

  it('switches to Advanced tab, selects Co-Op subtab and persists software cursor toggle', async () => {
    const saveSpy = vi.spyOn(AppAPI, 'saveConfig').mockResolvedValue();
    render(SettingsView);

    // Wait for onMount to complete initial config hydration
    await new Promise(r => setTimeout(r, 50));

    const advancedTab = screen.getByRole('tab', { name: /Advanced/i });
    await fireEvent.click(advancedTab);

    const coopSubtab = screen.getByRole('tab', { name: /Co-Op/i });
    expect(coopSubtab).toBeInTheDocument();
    await fireEvent.click(coopSubtab);

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

  it('switches to Advanced tab, selects All Options schema subtab and renders declarative configuration engine', async () => {
    render(SettingsView);

    const advancedTab = screen.getByRole('tab', { name: /Advanced/i });
    await fireEvent.click(advancedTab);

    const schemaSubtab = screen.getByRole('tab', { name: /All Options/i });
    expect(schemaSubtab).toBeInTheDocument();
    await fireEvent.click(schemaSubtab);

    expect(screen.getByText('Declarative Configuration Engine')).toBeInTheDocument();
  });
});

