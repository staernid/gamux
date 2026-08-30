import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import GhostInput from './GhostInput.svelte';

describe('GhostInput Component', () => {
  it('renders standard text input without inheritance', () => {
    render(GhostInput, {
      props: {
        label: 'Display Name',
        value: 'GamuxPlayer',
        placeholder: 'GamuxPlayer'
      }
    });

    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('GamuxPlayer')).toBeInTheDocument();
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Reset to global default')).not.toBeInTheDocument();
  });

  it('renders with watermark global placeholder in inherit mode', () => {
    render(GhostInput, {
      props: {
        label: 'Wine Prefix',
        value: '',
        globalPlaceholder: 'Isolated: steam_settings/pfx'
      }
    });

    expect(screen.getByText('Wine Prefix')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Isolated: steam_settings/pfx')).toBeInTheDocument();
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Reset to global default')).not.toBeInTheDocument();
  });

  it('shows override badge and reset button when value is typed in inherit mode', async () => {
    render(GhostInput, {
      props: {
        label: 'Wine Prefix',
        value: '/custom/prefix',
        globalPlaceholder: 'Isolated: steam_settings/pfx'
      }
    });

    expect(screen.getByText('Wine Prefix')).toBeInTheDocument();
    expect(screen.getByText('(Override)')).toBeInTheDocument();
    const resetBtn = screen.getByLabelText('Reset to global default');
    expect(resetBtn).toBeInTheDocument();

    await fireEvent.click(resetBtn);
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
  });

  it('renders status badge when provided (e.g. for API keys)', () => {
    render(GhostInput, {
      props: {
        label: 'Steam Web API Key',
        value: 'ABC123XYZ',
        type: 'password',
        status: 'SET'
      }
    });

    expect(screen.getByText('Steam Web API Key')).toBeInTheDocument();
    expect(screen.getByText('SET')).toBeInTheDocument();
  });
});
