import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import MetadataLinker from './MetadataLinker.svelte';
import { games, selectedGame } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';
import { AppAPI } from '../../api/app.js';

describe('MetadataLinker Domain Component', () => {
  const mockGame = normalizeGame({
    name: 'Astroneer',
    app_id: '361420',
    store: 'Steam',
    game_dir: '/games/astroneer',
  });

  beforeEach(() => {
    vi.restoreAllMocks();
    games.set([mockGame]);
    selectedGame.set(mockGame);
  });

  it('renders store provider options and search input', async () => {
    vi.spyOn(AppAPI, 'listMetadataProviders').mockResolvedValue([
      { id: 'steam', display_name: 'Steam' },
      { id: 'gog', display_name: 'GOG.com' }
    ]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([]);

    render(MetadataLinker, { props: { game: mockGame } });

    expect(screen.getByText('Steam')).toBeInTheDocument();
    expect(screen.getByText('Reset to Auto')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Search STEAM catalogue/i)).toBeInTheDocument();
    expect(screen.getByText('Steam AppID:')).toBeInTheDocument();
  });

  it('searches store candidates and links game on click', async () => {
    vi.spyOn(AppAPI, 'listMetadataProviders').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '361420', name: 'Astroneer', store: 'steam' }
    ]);
    const linkSpy = vi.spyOn(AppAPI, 'linkGameMetadata').mockResolvedValue({
      app_id: '361420',
      name: 'Astroneer'
    });

    const linkedHandler = vi.fn();
    render(MetadataLinker, { props: { game: mockGame, onlinked: linkedHandler } });

    const linkBtn = await screen.findByRole('button', { name: /Link Game/i });
    expect(linkBtn).toBeInTheDocument();

    await fireEvent.click(linkBtn);

    expect(linkSpy).toHaveBeenCalledWith('/games/astroneer', 'steam', '361420');
    expect(linkedHandler).toHaveBeenCalled();
  });

  it('switches to standalone DRM-free mode', async () => {
    vi.spyOn(AppAPI, 'listMetadataProviders').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([]);
    const linkSpy = vi.spyOn(AppAPI, 'linkGameMetadata').mockResolvedValue({});

    render(MetadataLinker, { props: { game: mockGame } });

    const standaloneBtn = screen.getByText('Standalone / DRM-Free');
    await fireEvent.click(standaloneBtn);

    expect(screen.getByText('Standalone / DRM-Free Mode')).toBeInTheDocument();

    const setStandaloneBtn = screen.getByRole('button', { name: /Set as Standalone/i });
    await fireEvent.click(setStandaloneBtn);

    expect(linkSpy).toHaveBeenCalledWith('/games/astroneer', 'standalone', '0');
  });
});
