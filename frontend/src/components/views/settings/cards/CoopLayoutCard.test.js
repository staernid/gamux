import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import CoopLayoutCard from './CoopLayoutCard.svelte';
import { multiseatStore } from '../../../../stores/multiseat.js';

describe('CoopLayoutCard Component', () => {
  const visualLayouts = [
    { id: 'dual_monitor', name: 'Dual Monitor', desc: 'Each player receives a dedicated fullscreen monitor', badge: '1 per Screen' },
    { id: 'side_by_side', name: 'Vertical Split', desc: 'Single display split vertically into left and right halves', badge: '2P Split' },
    { id: 'top_bottom', name: 'Horizontal Split', desc: 'Single display split horizontally into top and bottom halves', badge: '2P Split' },
    { id: 'quad_grid', name: 'Quad Grid', desc: 'Single display split into 4 quadrant viewports', badge: '4P Grid' }
  ];

  const effectiveDisplays = [
    { index: 0, name: 'DP-1', width: 2560, height: 1440, refresh_rate: 144, primary: true },
    { index: 1, name: 'HDMI-1', width: 1920, height: 1080, refresh_rate: 60, primary: false }
  ];

  const seats = [
    { seat_id: 'seat_1', profile_id: 'main', display_index: 0, screen_slot: 0, disabled: false },
    { seat_id: 'seat_2', profile_id: 'p2', display_index: 1, screen_slot: 1, disabled: false }
  ];

  beforeEach(() => {
    multiseatStore.setLayout('dual_monitor');
  });

  it('renders layout choices with labels and descriptions', () => {
    render(CoopLayoutCard, {
      multiseatDraft: { default_layout: 'dual_monitor' },
      cfgDraft: { multiseat: { default_layout: 'dual_monitor' } },
      visualLayouts,
      effectiveDisplays,
      activeSeats: seats,
      seats
    });

    expect(screen.getByText('Dual Monitor')).toBeInTheDocument();
    expect(screen.getByText('Vertical Split')).toBeInTheDocument();
    expect(screen.getByText('Horizontal Split')).toBeInTheDocument();
    expect(screen.getByText('Quad Grid')).toBeInTheDocument();
  });

  it('selects a different layout when clicked', async () => {
    const setLayoutSpy = vi.spyOn(multiseatStore, 'setLayout');
    const multiseatDraft = { default_layout: 'dual_monitor' };

    render(CoopLayoutCard, {
      multiseatDraft,
      cfgDraft: { multiseat: multiseatDraft },
      visualLayouts,
      effectiveDisplays,
      activeSeats: seats,
      seats
    });

    const splitBtn = screen.getByText('Vertical Split').closest('button');
    expect(splitBtn).toBeInTheDocument();
    await fireEvent.click(splitBtn);

    expect(setLayoutSpy).toHaveBeenCalledWith('side_by_side');
    expect(multiseatDraft.default_layout).toBe('side_by_side');
  });

  it('displays connected display info and rescan button', async () => {
    const refreshSpy = vi.spyOn(multiseatStore, 'refreshDisplays');

    render(CoopLayoutCard, {
      multiseatDraft: { default_layout: 'dual_monitor' },
      cfgDraft: { multiseat: { default_layout: 'dual_monitor' } },
      visualLayouts,
      effectiveDisplays,
      activeSeats: seats,
      seats
    });

    expect(screen.getByText(/DP-1/)).toBeInTheDocument();
    expect(screen.getByText(/HDMI-1/)).toBeInTheDocument();

    const rescanBtn = screen.getByRole('button', { name: /Rescan Displays/i });
    expect(rescanBtn).toBeInTheDocument();
    await fireEvent.click(rescanBtn);

    expect(refreshSpy).toHaveBeenCalled();
  });
});
