import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ArtworkPicker from './ArtworkPicker.svelte';
import { games, selectedGame } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';
import { AppAPI } from '../../api/app.js';

describe('ArtworkPicker Domain Component', () => {
  const mockGame = normalizeGame({
    name: 'Balatro',
    app_id: '2379780',
    store: 'Steam',
    game_dir: '/games/balatro',
    cover_url: 'https://cdn.steam.com/balatro.jpg'
  });

  beforeEach(() => {
    vi.restoreAllMocks();
    games.set([mockGame]);
    selectedGame.set(mockGame);
  });

  it('renders asset type selectors and search controls', async () => {
    vi.spyOn(AppAPI, 'queryArtworkOptions').mockResolvedValue([]);

    render(ArtworkPicker, { props: { game: mockGame, assetType: 'cover' } });

    expect(screen.getByText('Cover Art')).toBeInTheDocument();
    expect(screen.getByText('Banner Header')).toBeInTheDocument();
    expect(screen.getByText('Hero')).toBeInTheDocument();
    expect(screen.getByText('Logo')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Search artwork/i)).toBeInTheDocument();
    expect(screen.getByText(/Browse File.../i)).toBeInTheDocument();
  });

  it('queries artwork options and applies selection', async () => {
    const mockArtwork = [
      { url: 'https://images.steam.com/balatro_cover.jpg', provider: 'steam', style: 'official' }
    ];
    vi.spyOn(AppAPI, 'queryArtworkOptions').mockResolvedValue(mockArtwork);
    const applySpy = vi.spyOn(AppAPI, 'applyCustomArtwork').mockResolvedValue({
      cover_url: 'https://images.steam.com/balatro_cover.jpg'
    });

    const applyHandler = vi.fn();
    render(ArtworkPicker, { props: { game: mockGame, assetType: 'cover', onapply: applyHandler } });

    const selectBtn = await screen.findByRole('button', { name: /Select/i });
    expect(selectBtn).toBeInTheDocument();

    await fireEvent.click(selectBtn);

    expect(applySpy).toHaveBeenCalledWith('/games/balatro', 'cover', 'https://images.steam.com/balatro_cover.jpg');
    expect(applyHandler).toHaveBeenCalled();
  });

  it('allows applying custom artwork url', async () => {
    vi.spyOn(AppAPI, 'queryArtworkOptions').mockResolvedValue([]);
    const applySpy = vi.spyOn(AppAPI, 'applyCustomArtwork').mockResolvedValue({});

    render(ArtworkPicker, { props: { game: mockGame, assetType: 'cover' } });

    const urlInput = screen.getByPlaceholderText(/https:\/\/example.com\/custom-artwork.jpg/i);
    await fireEvent.input(urlInput, { target: { value: 'https://mycustomart.com/art.png' } });

    const applyUrlBtn = screen.getByRole('button', { name: /^Apply$/i });
    await fireEvent.click(applyUrlBtn);

    expect(applySpy).toHaveBeenCalledWith('/games/balatro', 'cover', 'https://mycustomart.com/art.png');
  });
});
