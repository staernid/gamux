import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import PersonaTab from './PersonaTab.svelte';
import { AppAPI } from '../../../api/app.js';

describe('PersonaTab Component', () => {
  const defaultUserCfg = {
    account_name: 'OriginalPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US',
  };

  const mockLanguages = [
    { value: 'english', label: 'English' },
    { value: 'german', label: 'German' },
  ];

  const mockCountries = [
    { value: 'US', label: 'United States' },
    { value: 'DE', label: 'Germany' },
  ];

  it('renders persona configuration fields', () => {
    render(PersonaTab, {
      props: {
        userCfgDraft: { ...defaultUserCfg },
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    expect(screen.getByText(/Steam & Persona Emulation/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/GamuxPlayer/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue('OriginalPlayer')).toBeInTheDocument();
    expect(screen.getByDisplayValue('76561197960287930')).toBeInTheDocument();
  });

  it('imports profile via 1-click Steam sync', async () => {
    vi.spyOn(AppAPI, 'resolveSteamProfile').mockResolvedValue({
      steam_id: '76561197960435530',
      persona_name: 'Robin Walker',
      avatar_url: 'https://avatars.steamstatic.com/robin.jpg',
      country_code: 'AU',
      profile_url: 'https://steamcommunity.com/id/robinwalker',
    });

    const draft = { ...defaultUserCfg };
    render(PersonaTab, {
      props: {
        userCfgDraft: draft,
        languagesList: mockLanguages,
        countriesList: mockCountries,
      },
    });

    const importInput = screen.getByPlaceholderText(/e\.g\. gaben or https:\/\/steamcommunity\.com\/id/i);
    await fireEvent.input(importInput, { target: { value: 'robinwalker' } });

    const importBtn = screen.getByRole('button', { name: /Import/i });
    await fireEvent.click(importBtn);

    await waitFor(() => {
      expect(screen.getByText(/Imported profile: Robin Walker \(76561197960435530\)/i)).toBeInTheDocument();
    });

    // Check that draft values were updated
    expect(screen.getByDisplayValue('Robin Walker')).toBeInTheDocument();
    expect(screen.getByDisplayValue('76561197960435530')).toBeInTheDocument();
  });
});
