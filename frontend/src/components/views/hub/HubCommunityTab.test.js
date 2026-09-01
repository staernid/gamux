import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import HubCommunityTab from './HubCommunityTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { normalizeGame } from '../../../utils/game.js';

describe('HubCommunityTab Component', () => {
  const mockSteamGame = normalizeGame({
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    store: 'Steam',
    game_dir: '/games/cyberpunk'
  });

  const mockNews = [
    {
      title: 'Patch 2.13 Available Now',
      contents: 'Adds support for AMD FidelityFX Super Resolution 3.',
      date: 1726142400,
      feedlabel: 'Community Announcements',
      author: 'CD PROJEKT RED'
    },
    {
      title: 'Night City Wire Special',
      contents: 'Deep dive into Phantom Liberty enhancements.',
      date: 1720000000,
      feedlabel: 'News',
      author: 'Editorial'
    }
  ];

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders default achievements tab initially', async () => {
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue([
      { id: 'ach_1', name: 'The Fool', description: 'Become a mercenary.', unlocked: true, unlock_time: 1720000000 }
    ]);

    render(HubCommunityTab, { props: { game: mockSteamGame } });

    expect(screen.getByRole('button', { name: /^Achievements$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /News & Updates/i })).toBeInTheDocument();
    expect(await screen.findByText('The Fool')).toBeInTheDocument();
  });

  it('fetches and displays news articles when switching to News & Updates tab', async () => {
    const fetchSpy = vi.spyOn(AppAPI, 'fetchNews').mockResolvedValue(mockNews);

    render(HubCommunityTab, { props: { game: mockSteamGame } });

    const newsTabBtn = screen.getByRole('button', { name: /News & Updates/i });
    await fireEvent.click(newsTabBtn);

    expect(fetchSpy).toHaveBeenCalledWith('1091500', 15);
    expect(await screen.findByText('Patch 2.13 Available Now')).toBeInTheDocument();
    expect(screen.getByText('Night City Wire Special')).toBeInTheDocument();
    expect(screen.getByText('Community Announcements')).toBeInTheDocument();
  });

  it('resolves steam AppID when game is linked to steam metadata', async () => {
    const linkedGame = normalizeGame({
      name: 'Custom RPG',
      app_id: '0',
      store: 'Standalone',
      game_dir: '/games/rpg',
      metadata_provider: 'steam',
      metadata_id: '1091500'
    });

    const fetchSpy = vi.spyOn(AppAPI, 'fetchNews').mockResolvedValue(mockNews);

    render(HubCommunityTab, { props: { game: linkedGame } });

    const newsTabBtn = screen.getByRole('button', { name: /News & Updates/i });
    await fireEvent.click(newsTabBtn);

    expect(fetchSpy).toHaveBeenCalledWith('1091500', 15);
    expect(await screen.findByText('Patch 2.13 Available Now')).toBeInTheDocument();
  });

  it('hides News & Updates sub-tab for unlinked standalone games', () => {
    const unlinkedGame = normalizeGame({
      name: 'Freeware Game',
      app_id: '0',
      store: 'Standalone',
      game_dir: '/games/freeware',
      metadata_provider: 'none',
      metadata_id: ''
    });

    render(HubCommunityTab, { props: { game: unlinkedGame } });

    expect(screen.getByRole('button', { name: /^Achievements$/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /News & Updates/i })).not.toBeInTheDocument();
  });
});
