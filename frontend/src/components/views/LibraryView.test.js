import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import LibraryView from './LibraryView.svelte';
import { games, selectedGame, activeView, searchQuery } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';

describe('LibraryView Component', () => {
  const mockGames = [
    { name: 'Astroneer', app_id: '361420', store: 'Steam', game_dir: '/games/astroneer' },
    { name: 'Balatro', app_id: '2379780', store: 'Steam', game_dir: '/games/balatro' },
    { name: 'Cyberpunk 2077', app_id: '', store: 'GOG', game_dir: '/games/cyberpunk' },
  ].map(normalizeGame);

  beforeEach(() => {
    localStorage.clear();
    games.set(mockGames);
    selectedGame.set(mockGames[0]);
    activeView.set('library');
    searchQuery.set('');
  });

  it('renders desktop grid view by default with game titles and search toolbar', () => {
    render(LibraryView);

    expect(screen.getByPlaceholderText(/Search library/i)).toBeInTheDocument();
    expect(screen.getAllByText('Astroneer').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Balatro').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Cyberpunk 2077').length).toBeGreaterThan(0);
  });

  it('filters titles based on search query', async () => {
    render(LibraryView);

    const input = screen.getByPlaceholderText(/Search library/i);
    await fireEvent.input(input, { target: { value: 'Balatro' } });

    expect(screen.getByText('Balatro')).toBeInTheDocument();
    expect(screen.queryByText('Astroneer')).not.toBeInTheDocument();
  });

  it('navigates to Game Hub when a game card is clicked', async () => {
    render(LibraryView);

    const balatroBtn = screen.getByText('Balatro').closest('button');
    expect(balatroBtn).toBeInTheDocument();
    await fireEvent.click(balatroBtn);

    let currentView = null;
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('game-hub');

    let currentSel = null;
    selectedGame.subscribe(v => currentSel = v)();
    expect(currentSel?.name).toBe('Balatro');
  });

  it('filters titles using status filter dropdown (Ready / Unpatched)', async () => {
    const { activeFilter } = await import('../../stores/library.js');
    render(LibraryView);

    const filterSelect = screen.getByLabelText(/Filter games by status/i);
    expect(filterSelect).toBeInTheDocument();

    await fireEvent.change(filterSelect, { target: { value: 'patched' } });
    let currentFilter = '';
    activeFilter.subscribe(v => currentFilter = v)();
    expect(currentFilter).toBe('patched');

    await fireEvent.change(filterSelect, { target: { value: 'clean' } });
    activeFilter.subscribe(v => currentFilter = v)();
    expect(currentFilter).toBe('clean');
  });

  it('renders online catalog mode with discovery suggestions when mode is catalog', () => {
    render(LibraryView, { props: { mode: 'catalog' } });

    expect(screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i)).toBeInTheDocument();
    expect(screen.getByText(/Online Store Catalog Discovery/i)).toBeInTheDocument();
    expect(screen.getByText('Hades II')).toBeInTheDocument();
  });

  it('searches store games in catalog mode and allows selecting a game', async () => {
    const { AppAPI } = await import('../../api/app.js');
    const searchSpy = vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1091500', name: 'Cyberpunk 2077', store: 'steam', poster_url: 'https://img.test/cp.jpg' }
    ]);

    render(LibraryView, { props: { mode: 'catalog' } });

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Cyberpunk' } });

    // Wait for debounced search
    await vi.waitFor(() => expect(searchSpy).toHaveBeenCalledWith('Cyberpunk'));

    const cardTitle = await screen.findByText('Cyberpunk 2077');
    expect(cardTitle).toBeInTheDocument();

    const cardButton = cardTitle.closest('button');
    await fireEvent.click(cardButton);

    let currentView = null;
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('game-hub');

    searchSpy.mockRestore();
  });
});
