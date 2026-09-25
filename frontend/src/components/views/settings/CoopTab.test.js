import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import CoopTab from './CoopTab.svelte';
import { multiseatStore } from '../../../stores/multiseat.js';
import { profilesStore } from '../../../stores/profiles.js';

describe('CoopTab Component', () => {
  const mockCfgDraft = {
    multiseat: {
      player2_profile_id: 'p2',
      default_seat1_mode: 'local',
      default_seat2_mode: 'local',
      default_layout: 'dual_monitor',
      broadcast_relay_enabled: true,
      seat1_display_index: 0,
      seat2_display_index: 1,
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    }
  };

  const mockUserCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  const languagesList = [
    { value: 'english', label: 'English' },
    { value: 'french', label: 'French' },
    { value: 'german', label: 'German' }
  ];

  const countriesList = [
    { value: 'US', label: 'United States (US)' },
    { value: 'DE', label: 'Germany (DE)' }
  ];

  beforeEach(() => {
    multiseatStore.setLayout('dual_monitor');
    const cur = get(multiseatStore);
    if (cur.staging?.seats?.length > 2) {
      for (let i = cur.staging.seats.length - 1; i >= 2; i--) {
        multiseatStore.removeSeat(i);
      }
    }
    profilesStore.setProfiles([
      { id: 'main', index: 1, name: 'Alice', is_main: true },
      { id: 'p2', index: 2, name: 'PlayerTwo', is_main: false },
    ]);
  });

  it('renders system diagnostics and capabilities badges', () => {
    render(CoopTab, {
      cfgDraft: { ...mockCfgDraft },
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    expect(screen.getByText('System Diagnostics & Isolation Tools')).toBeInTheDocument();
    expect(screen.getByText('Bubblewrap')).toBeInTheDocument();
    expect(screen.getByText('Pasta (NetNS)')).toBeInTheDocument();
    expect(screen.getByText('Gamescope')).toBeInTheDocument();
  });

  it('renders Seat 2 Profile selector', () => {
    render(CoopTab, {
      cfgDraft: { ...mockCfgDraft },
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    expect(screen.getByText('Seat 2 Profile (Co-op Guest)')).toBeInTheDocument();
    expect(screen.getByText(/Seat 2 Player Profile/i)).toBeInTheDocument();
  });

  it('switches seat output modes between local screen and Sunshine stream', async () => {
    const draft = { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } };
    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    const sunshineBtns = screen.getAllByRole('button', { name: /^Sunshine Stream$/i });
    expect(sunshineBtns.length).toBe(2);

    // Switch Player 1 to Sunshine Stream
    await fireEvent.click(sunshineBtns[0]);
    expect(draft.multiseat.default_seat1_mode).toBe('stream');

    // Switch Player 2 to Sunshine Stream
    await fireEvent.click(sunshineBtns[1]);
    expect(draft.multiseat.default_seat2_mode).toBe('stream');
  });

  it('renders controller allocation selectors for Player 1 and Player 2', () => {
    const draft = {
      ...mockCfgDraft,
      multiseat: {
        ...mockCfgDraft.multiseat,
        seat1_gamepad: 'auto',
        seat2_gamepad: 'auto',
      }
    };

    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    expect(screen.getByText('Player 1 Controller')).toBeInTheDocument();
    expect(screen.getByText('Player 2 Controller')).toBeInTheDocument();
  });

  it('renders dual keyboard and mouse allocation selectors and cursor toggles', () => {
    const draft = {
      ...mockCfgDraft,
      multiseat: {
        ...mockCfgDraft.multiseat,
        seat1_keyboard: 'auto',
        seat2_keyboard: 'auto',
        seat1_mouse: 'auto',
        seat2_mouse: 'auto',
      }
    };

    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    expect(screen.getByText('Dual Keyboards & Mice (Experimental)')).toBeInTheDocument();
    expect(screen.getByText('Player 1 Keyboard')).toBeInTheDocument();
    expect(screen.getByText('Player 2 Keyboard')).toBeInTheDocument();
    expect(screen.getByText('Player 1 Mouse')).toBeInTheDocument();
    expect(screen.getByText('Player 2 Mouse')).toBeInTheDocument();
  });

  it('reflects and updates software cursor and pointer switches', async () => {
    const draft = {
      ...mockCfgDraft,
      multiseat: {
        ...mockCfgDraft.multiseat,
        seat2_force_grab_cursor: false,
        seat2_grab_keyboard: false,
        seat2_show_cursor: true,
      }
    };

    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    const switches = screen.getAllByRole('switch');
    expect(switches.length).toBeGreaterThanOrEqual(3);

    // Toggle Force Grab Cursor
    await fireEvent.click(switches[0]);
    expect(draft.multiseat.seat2_force_grab_cursor).toBe(true);
  });

  it('selects visual split-screen layout options and updates config draft', async () => {
    const draft = { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } };
    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    const quadBtn = screen.getByText('2x2 Quad Grid');
    expect(quadBtn).toBeInTheDocument();

    await fireEvent.click(quadBtn);
    expect(draft.multiseat.default_layout).toBe('quad_grid');

    const threeBtn = screen.getByText('3-Player Split');
    await fireEvent.click(threeBtn);
    expect(draft.multiseat.default_layout).toBe('three_player');
  });

  it('auto-expands seats when selecting 4-player or 3-player layout', async () => {
    const draft = { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat, default_layout: 'side_by_side' } };
    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    const quadBtn = screen.getByText('2x2 Quad Grid');
    await fireEvent.click(quadBtn);

    expect(draft.multiseat.default_layout).toBe('quad_grid');
    expect(screen.getByText('Player 3 (Co-op Guest)')).toBeInTheDocument();
    expect(screen.getByText('Player 4 (Co-op Guest)')).toBeInTheDocument();
  });

  it('renders screen portion selectors and swap halves button for split-screen layouts', async () => {
    const draft = { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat, default_layout: 'side_by_side' } };
    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    const swapBtn = screen.getByRole('button', { name: /Swap Halves ⇄/i });
    expect(swapBtn).toBeInTheDocument();

    const portionLabels = screen.getAllByText('Screen Portion');
    expect(portionLabels.length).toBeGreaterThanOrEqual(2);

    await fireEvent.click(swapBtn);
  });
});

