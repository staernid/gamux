import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import SteamPersonaCard from './SteamPersonaCard.svelte';
import { AppAPI } from '../../../api/app.js';

describe('SteamPersonaCard Component', () => {
  const mockLanguages = [
    { value: 'english', label: 'English' },
    { value: 'german', label: 'German' },
  ];

  const mockCountries = [
    { value: 'US', label: 'United States' },
    { value: 'DE', label: 'Germany' },
  ];

  it('renders default persona fields and labels', () => {
    render(SteamPersonaCard, {
      props: {
        accountName: 'TestPlayer',
        accountSteamId: '76561197960287930',
        language: 'english',
        ipCountry: 'US',
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    expect(screen.getByText(/Auto-Sync from Steam Profile/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue('TestPlayer')).toBeInTheDocument();
    expect(screen.getByDisplayValue('76561197960287930')).toBeInTheDocument();
  });

  it('renders custom labels for guest/co-op mode', () => {
    render(SteamPersonaCard, {
      props: {
        accountName: 'CoopGuest',
        accountSteamId: '76561198998877665',
        nameLabel: 'Guest Display Name',
        steamidLabel: 'Guest Steam ID (64-Bit)',
        syncTitle: 'Auto-Sync Player 2 Profile',
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    expect(screen.getByText('Auto-Sync Player 2 Profile')).toBeInTheDocument();
    expect(screen.getByLabelText('Guest Display Name')).toBeInTheDocument();
    expect(screen.getByLabelText('Guest Steam ID (64-Bit)')).toBeInTheDocument();
  });

  it('displays collision warning when collisionSteamId matches accountSteamId', () => {
    render(SteamPersonaCard, {
      props: {
        accountName: 'PlayerTwo',
        accountSteamId: '76561197960287930',
        collisionSteamId: '76561197960287930',
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    expect(screen.getByText(/Collision: Player 1 and Player 2 have the same SteamID/i)).toBeInTheDocument();
  });

  it('displays format warning for invalid SteamID64 length', () => {
    render(SteamPersonaCard, {
      props: {
        accountName: 'PlayerTwo',
        accountSteamId: '12345',
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    expect(screen.getByText(/Warning: Standard SteamID64 format should be exactly 17 digits/i)).toBeInTheDocument();
  });

  it('imports profile via 1-click Steam sync', async () => {
    vi.spyOn(AppAPI, 'resolveSteamProfile').mockResolvedValue({
      steam_id: '76561197960435530',
      persona_name: 'Robin Walker',
      avatar_url: 'https://avatars.steamstatic.com/robin.jpg',
      country_code: 'AU',
      profile_url: 'https://steamcommunity.com/id/robinwalker',
    });

    render(SteamPersonaCard, {
      props: {
        accountName: 'InitialName',
        accountSteamId: '76561197960287930',
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    const input = screen.getByPlaceholderText(/e\.g\. gaben or https:\/\/steamcommunity\.com\/id/i);
    await fireEvent.input(input, { target: { value: 'robinwalker' } });

    const importBtn = screen.getByRole('button', { name: /Import/i });
    await fireEvent.click(importBtn);

    await waitFor(() => {
      expect(screen.getByText(/Imported profile: Robin Walker \(76561197960435530\)/i)).toBeInTheDocument();
    });

    expect(screen.getByDisplayValue('Robin Walker')).toBeInTheDocument();
    expect(screen.getByDisplayValue('76561197960435530')).toBeInTheDocument();
  });
});
