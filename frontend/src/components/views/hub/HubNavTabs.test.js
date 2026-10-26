import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubNavTabs from './HubNavTabs.svelte';

describe('HubNavTabs Component', () => {
  it('renders all primary tabs and reflects active state', () => {
    render(HubNavTabs, {
      props: {
        activeTab: 'overview',
        overrideCount: 2,
        isGameRunning: true,
        outdatedModsCount: 3
      }
    });

    expect(screen.getByRole('button', { name: /^Overview$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Configuration/i })).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Files & Depots/i })).toBeInTheDocument();
    expect(screen.getByTitle('Execution log active')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Mods/i })).toBeInTheDocument();
    expect(screen.getByTitle('3 updates available')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Saves$/i })).toBeInTheDocument();
  });

  it('calls onselectTab when clicking a tab button', async () => {
    const selectSpy = vi.fn();
    render(HubNavTabs, {
      props: {
        activeTab: 'overview',
        onselectTab: selectSpy
      }
    });

    const configBtn = screen.getByRole('button', { name: /Configuration/i });
    await fireEvent.click(configBtn);

    expect(selectSpy).toHaveBeenCalledWith('config');
  });
});
