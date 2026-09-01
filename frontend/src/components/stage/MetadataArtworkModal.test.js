import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import MetadataArtworkModal from './MetadataArtworkModal.svelte';
import { games, selectedGame } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';
import { AppAPI } from '../../api/app.js';

describe('MetadataArtworkModal Component', () => {
  const mockGame = normalizeGame({
    name: 'Hades II',
    app_id: '1145350',
    store: 'Steam',
    game_dir: '/games/hades2',
    cover_url: 'https://cdn.steam.com/hades2.jpg'
  });

  beforeEach(() => {
    vi.restoreAllMocks();
    games.set([mockGame]);
    selectedGame.set(mockGame);
  });

  it('renders metadata tab by default with game info', async () => {
    vi.spyOn(AppAPI, 'listMetadataProviders').mockResolvedValue([
      { id: 'steam', display_name: 'Steam' },
      { id: 'gog', display_name: 'GOG.com' },
    ]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam' }
    ]);

    render(MetadataArtworkModal, { props: { game: mockGame, initialTab: 'metadata' } });

    expect(screen.getByText(/1. Store & Metadata Link/i)).toBeInTheDocument();
    expect(screen.getByText(/2. Artwork Gallery & Custom Art/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Search STEAM catalogue/i)).toBeInTheDocument();
  });

  it('switches between metadata and artwork tabs', async () => {
    vi.spyOn(AppAPI, 'listMetadataProviders').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'getArtworkOptions').mockResolvedValue([]);

    render(MetadataArtworkModal, { props: { game: mockGame, initialTab: 'metadata' } });

    const artworkTabBtn = screen.getByText(/2. Artwork Gallery & Custom Art/i);
    await fireEvent.click(artworkTabBtn);

    expect(screen.getByText(/Cover Art/i)).toBeInTheDocument();
    expect(screen.getByText(/Hero/i)).toBeInTheDocument();
    expect(screen.getByText(/Banner Header/i)).toBeInTheDocument();
    expect(screen.getByText(/Logo/i)).toBeInTheDocument();
  });

  it('renders artwork tab directly when initialTab is artwork', () => {
    vi.spyOn(AppAPI, 'listMetadataProviders').mockResolvedValue([]);
    vi.spyOn(AppAPI, 'getArtworkOptions').mockResolvedValue([]);

    render(MetadataArtworkModal, { props: { game: mockGame, initialTab: 'artwork' } });

    expect(screen.getByText(/Cover Art/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/https:\/\/example.com\/custom-artwork.jpg/i)).toBeInTheDocument();
  });
});
