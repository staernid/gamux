import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import SettingsView from './SettingsView.svelte';

describe('SettingsView Component', () => {
  it('renders primary system settings tabs and persona form fields', async () => {
    render(SettingsView);

    expect(screen.getByText('System Settings')).toBeInTheDocument();
    expect(screen.getByText('Player Persona')).toBeInTheDocument();
    expect(screen.getByText('Library & Storage')).toBeInTheDocument();
    expect(screen.getByText('Compatibility')).toBeInTheDocument();
    expect(screen.getByText('Media & APIs')).toBeInTheDocument();
    expect(screen.getByText('Tools & Maintenance')).toBeInTheDocument();
    expect(screen.getByText('Logs & Activity')).toBeInTheDocument();

    // Default tab is Persona
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByText(/Steam ID/i)).toBeInTheDocument();
  });

  it('switches to Tools & Maintenance tab and displays toolchain and batch operations', async () => {
    render(SettingsView);

    const toolsTab = screen.getByRole('button', { name: /Tools & Maintenance/i });
    await fireEvent.click(toolsTab);

    expect(screen.getByText('Helper Tools')).toBeInTheDocument();
    expect(screen.getByText('Batch Maintenance')).toBeInTheDocument();
    expect(screen.getByText('Achievements')).toBeInTheDocument();
    expect(screen.getByText('Artwork')).toBeInTheDocument();
    expect(screen.getByText('Apply Setup')).toBeInTheDocument();
    expect(screen.getByText('Normalize Folders')).toBeInTheDocument();
    expect(screen.getByText('Verify Manifests')).toBeInTheDocument();
  });
});

