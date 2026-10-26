import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import HubOverviewTab, { clearOverviewHubCache } from './HubOverviewTab.svelte';
import { AppAPI } from '../../../api/app.js';

describe('HubOverviewTab Component', () => {
  const mockGame = {
    appId: '1086940',
    name: "Baldur's Gate 3",
    gameDir: '/games/bg3',
    store: 'steam',
    platform: 'win64',
    exePath: 'bin/bg3.exe',
    isInstalled: true,
    fileCount: 1500,
    diskSizeBytes: 120 * 1024 * 1024 * 1024,
    playtimeSeconds: 3600
  };

  beforeEach(() => {
    clearOverviewHubCache();
    vi.restoreAllMocks();
  });

  it('renders overview about and external links for steam game', () => {
    render(HubOverviewTab, { game: mockGame });

    expect(screen.getByText('ProtonDB')).toBeInTheDocument();
    expect(screen.getByText('PCGamingWiki')).toBeInTheDocument();
    expect(screen.getByText('SteamDB')).toBeInTheDocument();
  });

  it('fetches and displays achievements in spotlight ribbon', async () => {
    const mockAchievements = [
      { id: 'ACH_1', name: 'First Steps', description: 'Begin your journey', unlocked: true, icon_url: 'https://example.com/icon.png' },
      { id: 'ACH_2', name: 'Champion', description: 'Win a duel', unlocked: false, icon_gray_url: 'https://example.com/icon_gray.png' }
    ];
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue(mockAchievements);

    render(HubOverviewTab, { game: mockGame });

    const achTitle = await screen.findByText('Achievements');
    expect(achTitle).toBeInTheDocument();
    expect(screen.getByText('First Steps')).toBeInTheDocument();
  });

  it('fetches and displays news items in patch notes feed', async () => {
    const mockNews = [
      { id: '1', title: 'Patch 7 Release Notes', contents: 'Major overhaul of modding and cinematics.', date: 1700000000 }
    ];
    vi.spyOn(AppAPI, 'fetchNews').mockResolvedValue(mockNews);

    render(HubOverviewTab, { game: mockGame });

    const patchTitle = await screen.findByText('Patch 7 Release Notes');
    expect(patchTitle).toBeInTheDocument();
  });

  it('triggers onnavigateTab callback when clicking navigation buttons', async () => {
    const mockAchievements = [
      { id: 'ACH_1', name: 'First Steps', description: 'Begin your journey', unlocked: true }
    ];
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue(mockAchievements);
    const navSpy = vi.fn();

    render(HubOverviewTab, { game: mockGame, onnavigateTab: navSpy });

    const viewAllBtn = await screen.findByRole('button', { name: /View All/i });
    await fireEvent.click(viewAllBtn);

    expect(navSpy).toHaveBeenCalledWith({
      tab: 'community',
      subTab: 'achievements',
      newsIndex: 0
    });
  });
});
