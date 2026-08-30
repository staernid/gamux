import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import SeamlessSwitch from './SeamlessSwitch.svelte';

describe('SeamlessSwitch Component', () => {
  it('renders in inherited state with ghost display matching global default', async () => {
    render(SeamlessSwitch, {
      props: {
        label: 'MangoHud Overlay',
        description: 'Display FPS HUD',
        value: null,
        inheritedValue: true,
        tag: 'HUD'
      }
    });

    expect(screen.getByText('MangoHud Overlay')).toBeInTheDocument();
    expect(screen.getByText('Display FPS HUD')).toBeInTheDocument();
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Reset to global default')).not.toBeInTheDocument();
  });

  it('transitions from inherited null to explicit override on click', async () => {
    render(SeamlessSwitch, {
      props: {
        label: 'GameMode',
        value: null,
        inheritedValue: false
      }
    });

    const toggle = screen.getByRole('switch', { name: /GameMode/i });
    expect(toggle.getAttribute('aria-checked')).toBe('false');

    await fireEvent.click(toggle);

    // Clicking flips to override state
    expect(screen.getByText('(Override)')).toBeInTheDocument();
    expect(toggle.getAttribute('aria-checked')).toBe('true');
  });

  it('displays override indicator and allows reverting to inherit state', async () => {
    render(SeamlessSwitch, {
      props: {
        label: 'AMD FSR Upscaling',
        value: true,
        inheritedValue: false
      }
    });

    expect(screen.getByText('(Override)')).toBeInTheDocument();
    const revertBtn = screen.getByLabelText('Reset to global default');
    expect(revertBtn).toBeInTheDocument();

    await fireEvent.click(revertBtn);
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
  });

  it('works as a standard boolean toggle when no inheritedValue is provided', async () => {
    render(SeamlessSwitch, {
      props: {
        label: 'Normalize Folders',
        description: 'Standardize folder names',
        value: false
      }
    });

    expect(screen.getByText('Normalize Folders')).toBeInTheDocument();
    expect(screen.getByText('Standardize folder names')).toBeInTheDocument();
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Reset to global default')).not.toBeInTheDocument();

    const toggle = screen.getByRole('switch', { name: /Normalize Folders/i });
    expect(toggle.getAttribute('aria-checked')).toBe('false');

    await fireEvent.click(toggle);
    expect(toggle.getAttribute('aria-checked')).toBe('true');
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
  });
});
