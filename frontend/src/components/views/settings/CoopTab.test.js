import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import CoopTab from './CoopTab.svelte';
import { multiseatStore } from '../../../stores/multiseat.js';

describe('CoopTab Component', () => {
  const mockCfgDraft = {
    multiseat: {
      player2_account_name: 'PlayerTwo',
      player2_account_steamid: '76561198998877665',
      player2_language: 'english',
      player2_ip_country: 'US',
      player2_wine_prefix: '',
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

  it('renders Player 2 Profile inputs and Quick Setup button', () => {
    render(CoopTab, {
      cfgDraft: { ...mockCfgDraft },
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    expect(screen.getByText('Player 2 Profile (Co-op Guest)')).toBeInTheDocument();
    expect(screen.getByLabelText('Guest Display Name')).toBeInTheDocument();
    expect(screen.getByLabelText('Guest Steam ID (64-Bit)')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Quick Setup/i })).toBeInTheDocument();
  });

  it('generates a new randomized 17-digit SteamID for Player 2', async () => {
    const draft = { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } };
    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    const randomBtn = screen.getByRole('button', { name: /Random/i });
    await fireEvent.click(randomBtn);

    expect(draft.multiseat.player2_account_steamid).toMatch(/^\d{17}$/);
    expect(draft.multiseat.player2_account_steamid).not.toBe('76561198998877665');
  });

  it('clones Player 1 settings on Quick Setup click', async () => {
    const draft = { ...mockCfgDraft, multiseat: { ...mockCfgDraft.multiseat } };
    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft, account_name: 'MasterChief', ip_country: 'DE' },
      languagesList,
      countriesList
    });

    const quickBtn = screen.getByRole('button', { name: /Quick Setup/i });
    await fireEvent.click(quickBtn);

    expect(draft.multiseat.player2_account_name).toBe('MasterChief-Coop');
    expect(draft.multiseat.player2_ip_country).toBe('DE');
    expect(draft.multiseat.player2_account_steamid).toMatch(/^\d{17}$/);
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

    expect(screen.getByLabelText('Player 1 Controller')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 2 Controller')).toBeInTheDocument();
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
        seat2_force_grab_cursor: false,
        seat2_grab_keyboard: false,
        seat2_show_cursor: true,
        seat2_compositor: 'auto',
      }
    };

    render(CoopTab, {
      cfgDraft: draft,
      userCfgDraft: { ...mockUserCfgDraft },
      languagesList,
      countriesList
    });

    expect(screen.getByText('Dual Keyboards & Mice (Experimental)')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 1 Keyboard')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 2 Keyboard')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 1 Mouse')).toBeInTheDocument();
    expect(screen.getByLabelText('Player 2 Mouse')).toBeInTheDocument();
    expect(screen.getByText('Player 2 Pointer & Compositor Options')).toBeInTheDocument();
    expect(screen.getByText('Force Grab Cursor')).toBeInTheDocument();
    expect(screen.getByText('Grab Keyboard (-g)')).toBeInTheDocument();
    expect(screen.getByText('Software Cursor')).toBeInTheDocument();
    expect(screen.getByText(/Cage Wayland compositor/i)).toBeInTheDocument();
  });
});
