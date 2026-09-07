import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubAchievementsCard from './HubAchievementsCard.svelte';
import { AppAPI } from '../../../../api/app.js';

describe('HubAchievementsCard Component', () => {
  const mockAchievements = [
    {
      id: 'ACH_COMMON',
      name: 'Common Feat',
      description: 'Done by many',
      icon_url: '',
      icon_gray_url: '',
      hidden: false,
      unlocked: true,
      unlock_time: 1700000000,
      global_percent: 68.5,
      rarity: 'common',
    },
    {
      id: 'ACH_RARE',
      name: 'Ultra Rare Mystery',
      description: 'Found by few',
      icon_url: '',
      icon_gray_url: '',
      hidden: false,
      unlocked: false,
      unlock_time: 0,
      global_percent: 2.1,
      rarity: 'ultra_rare',
    },
  ];

  it('renders achievement list with rarity badges', async () => {
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue(mockAchievements);

    render(HubAchievementsCard, { props: { gameDir: '/mock/game/dir' } });

    await waitFor(() => {
      expect(screen.getByText(/Common Feat/i)).toBeInTheDocument();
      expect(screen.getByText(/Ultra Rare Mystery/i)).toBeInTheDocument();
      expect(screen.getByText(/68.5%/i)).toBeInTheDocument();
      expect(screen.getByText(/2.1%/i)).toBeInTheDocument();
    });
  });

  it('toggles rarest-first sorting', async () => {
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue(mockAchievements);

    const { container } = render(HubAchievementsCard, { props: { gameDir: '/mock/game/dir' } });

    await waitFor(() => {
      expect(screen.getByText(/Default Order/i)).toBeInTheDocument();
    });

    // Toggle sort to rarest first
    const sortBtn = screen.getByTitle(/Toggle sorting between default and rarest first/i);
    await fireEvent.click(sortBtn);

    expect(screen.getByText(/Rarest First/i)).toBeInTheDocument();

    // After clicking rarest first, Ultra Rare Mystery (2.1%) should appear before Common Feat (68.5%)
    const titles = container.querySelectorAll('h5');
    expect(titles[0].textContent).toContain('Ultra Rare Mystery');
    expect(titles[1].textContent).toContain('Common Feat');
  });

  it('filters by unlocked and locked status', async () => {
    vi.spyOn(AppAPI, 'getGameAchievements').mockResolvedValue(mockAchievements);

    render(HubAchievementsCard, { props: { gameDir: '/mock/game/dir' } });

    await waitFor(() => {
      expect(screen.getByText(/Common Feat/i)).toBeInTheDocument();
    });

    // Click "Locked" filter (exact match start)
    const lockedBtn = screen.getByRole('button', { name: /^Locked/i });
    await fireEvent.click(lockedBtn);

    expect(screen.queryByText(/Common Feat/i)).not.toBeInTheDocument();
    expect(screen.getByText(/Ultra Rare Mystery/i)).toBeInTheDocument();
  });
});
