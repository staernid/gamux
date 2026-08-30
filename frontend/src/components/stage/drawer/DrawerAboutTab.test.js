import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import DrawerAboutTab from './DrawerAboutTab.svelte';
import { AppAPI } from '../../../api/app.js';
import { normalizeGame } from '../../../utils/game.js';

describe('DrawerAboutTab Component', () => {
  it('loads and renders metadata description for Steam game', async () => {
    const game = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '/games/cyberpunk',
      metadata_provider: 'steam',
      metadata_id: '1091500'
    });

    render(DrawerAboutTab, { props: { game } });

    expect(await screen.findByText(/Cyberpunk 2077 is an open-world/i)).toBeInTheDocument();
    expect(screen.getByText('CD PROJEKT RED')).toBeInTheDocument();
  });

  it('updates description when metadata provider changes to RAWG and caches responses', async () => {
    const spy = vi.spyOn(AppAPI, 'getGameMetadata');

    const steamGame = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '/games/cyberpunk',
      metadata_provider: 'steam',
      metadata_id: '1091500'
    });

    const { rerender } = render(DrawerAboutTab, { props: { game: steamGame } });

    expect(await screen.findByText(/Cyberpunk 2077 is an open-world/i)).toBeInTheDocument();
    expect(spy).toHaveBeenCalledWith('steam', '1091500');

    // Switch provider to RAWG
    const rawgGame = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      game_dir: '/games/cyberpunk',
      metadata_provider: 'rawg',
      metadata_id: '41494'
    });

    await rerender({ game: rawgGame });

    expect(await screen.findByText(/Detailed RAWG catalog synopsis/i)).toBeInTheDocument();
    expect(spy).toHaveBeenCalledWith('rawg', '41494');

    // Switch back to Steam (should resolve from cache without duplicate network call)
    const callCountBefore = spy.mock.calls.length;
    await rerender({ game: steamGame });
    expect(await screen.findByText(/Cyberpunk 2077 is an open-world/i)).toBeInTheDocument();
    expect(spy.mock.calls.length).toBe(callCountBefore); // No new network call, served from cache!
  });

  it('renders unlinked fallback when metadata provider is None', async () => {
    const unlinkedGame = normalizeGame({
      name: 'My Custom Game',
      app_id: '0',
      store: 'Standalone',
      game_dir: '/games/custom',
      metadata_provider: 'none',
      metadata_id: '0'
    });

    render(DrawerAboutTab, { props: { game: unlinkedGame } });

    expect(screen.getByText('My Custom Game')).toBeInTheDocument();
    expect(screen.getByText(/No official store metadata linked yet/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Link Metadata/i })).toBeInTheDocument();
  });
});
