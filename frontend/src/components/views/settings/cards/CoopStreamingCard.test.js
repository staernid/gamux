import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import CoopStreamingCard from './CoopStreamingCard.svelte';
import { multiseatStore } from '../../../../stores/multiseat.js';

describe('CoopStreamingCard Component', () => {
  const keyboardOptions = [
    { value: 'auto', label: 'Auto-detect' },
    { value: '/dev/input/event3', label: 'Mechanical Keyboard (event3)' }
  ];

  const mouseOptions = [
    { value: 'auto', label: 'Auto-detect' },
    { value: '/dev/input/event4', label: 'Optical Mouse (event4)' }
  ];

  const mockCfgDraft = {
    multiseat: {
      seat1_keyboard: 'auto',
      seat2_keyboard: 'auto',
      seat1_mouse: 'auto',
      seat2_mouse: 'auto',
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
      sunshine_pin: '1234',
      sunshine_auto_start: true,
      capture_cursor_p1: true,
      capture_cursor_p2: true
    }
  };

  it('renders Dual Keyboards & Mice section and options', () => {
    render(CoopStreamingCard, {
      cfgDraft: { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } },
      keyboardOptions,
      mouseOptions
    });

    expect(screen.getByText('Dual Keyboards & Mice (Experimental)')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 1 Keyboard')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 2 Keyboard')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 1 Mouse')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 2 Mouse')).toBeInTheDocument();
  });

  it('triggers refreshKeyboards and refreshMice when clicking Rescan KBM', async () => {
    const kbdSpy = vi.spyOn(multiseatStore, 'refreshKeyboards');
    const mouseSpy = vi.spyOn(multiseatStore, 'refreshMice');

    render(CoopStreamingCard, {
      cfgDraft: { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } },
      keyboardOptions,
      mouseOptions
    });

    const rescanBtn = screen.getByRole('button', { name: /Rescan KBM/i });
    await fireEvent.click(rescanBtn);

    expect(kbdSpy).toHaveBeenCalled();
    expect(mouseSpy).toHaveBeenCalled();
  });

  it('renders Sunshine streaming ports and settings', () => {
    render(CoopStreamingCard, {
      cfgDraft: { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } },
      keyboardOptions,
      mouseOptions
    });

    expect(screen.getByText('Networking & Sunshine Streaming')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 1 Sunshine Port')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 2 Sunshine Port')).toBeInTheDocument();
  });
});
