import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import SearchGridModal from './SearchGridModal.svelte';
import { games, selectedGame } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';

describe('SearchGridModal Component', () => {
  const mockGames = [
    { name: 'Astroneer', app_id: '361420', store: 'Steam', game_dir: '/games/astroneer' },
    { name: 'Balatro', app_id: '2379780', store: 'Steam', game_dir: '/games/balatro' },
    { name: 'Cyberpunk 2077', app_id: '', store: 'GOG', game_dir: '/games/cyberpunk' },
  ].map(normalizeGame);

  beforeEach(() => {
    games.set(mockGames);
    selectedGame.set(mockGames[0]);
  });

  it('renders all games and search input', () => {
    render(SearchGridModal);

    expect(screen.getByPlaceholderText(/Type to filter.*titles/i)).toBeInTheDocument();
    expect(screen.getAllByText('Astroneer').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Balatro').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Cyberpunk 2077').length).toBeGreaterThan(0);
  });

  it('filters titles based on search query', async () => {
    render(SearchGridModal);

    const input = screen.getByPlaceholderText(/Type to filter.*titles/i);
    await fireEvent.input(input, { target: { value: 'Bala' } });

    expect(screen.getAllByText('Balatro').length).toBeGreaterThan(0);
    expect(screen.queryByText('Astroneer')).not.toBeInTheDocument();
  });

  it('selects game on item click', async () => {
    render(SearchGridModal);

    const balatroBtn = screen.getAllByText('Balatro')[0];
    await fireEvent.click(balatroBtn);

    let current = null;
    selectedGame.subscribe(v => current = v)();
    expect(current?.name).toBe('Balatro');

    let currentView = null;
    const { activeView } = await import('../../stores/library.js');
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('game-hub');
  });

  it('renders sort chips and updates sort order on click', async () => {
    render(SearchGridModal);

    const playtimeChip = screen.getByText('Playtime');
    expect(playtimeChip).toBeInTheDocument();
    await fireEvent.click(playtimeChip);

    let currentSort = null;
    const { sortBy } = await import('../../stores/library.js');
    sortBy.subscribe(v => currentSort = v)();
    expect(currentSort).toBe('playtime_desc');
  });

  it('switches to Store Catalog mode and searches online titles', async () => {
    const { AppAPI } = await import('../../api/app.js');
    const searchSpy = vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam', release_date: '2024' },
      { id: '1091500', name: 'Cyberpunk 2077', store: 'steam', release_date: '2020' },
    ]);

    render(SearchGridModal);

    const storeTab = screen.getByText(/Store Catalog/i);
    await fireEvent.click(storeTab);

    expect(screen.getByText(/Online Store Catalog Discovery/i)).toBeInTheDocument();

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Hades' } });

    // Wait for debounced search execution
    await new Promise(r => setTimeout(r, 300));
    expect(searchSpy).toHaveBeenCalledWith('Hades');

    const hadesItems = await screen.findAllByText('Hades II');
    expect(hadesItems.length).toBeGreaterThan(0);

    await fireEvent.click(hadesItems[0]);

    let current = null;
    selectedGame.subscribe(v => current = v)();
    expect(current?.name).toBe('Hades II');
    expect(current?.appId).toBe('1145350');
    expect(current?.isVirtual).toBe(true);
    expect(current?.isInstalled).toBe(false);

    let currentView = null;
    const { activeView } = await import('../../stores/library.js');
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('game-hub');
  });

  it('switches to dedicated command mode when query starts with >', async () => {
    render(SearchGridModal);

    const input = screen.getByPlaceholderText(/Type to filter.*titles/i);
    await fireEvent.input(input, { target: { value: '>' } });

    expect(screen.getByText(/Commands & Actions/i)).toBeInTheDocument();
    expect(screen.getByText('Go to Settings')).toBeInTheDocument();
    expect(screen.getByText('Scan Library Folders')).toBeInTheDocument();

    // Filter to a specific command
    await fireEvent.input(input, { target: { value: '>settings' } });
    const settingsBtn = screen.getByText('Go to Settings');
    await fireEvent.click(settingsBtn);

    let currentView = null;
    const { activeView } = await import('../../stores/library.js');
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('settings');
  });

  it('shows suggested commands when search query matches command keywords', async () => {
    render(SearchGridModal);

    const input = screen.getByPlaceholderText(/Type to filter.*titles/i);
    await fireEvent.input(input, { target: { value: 'tasks' } });

    expect(screen.getByText(/Suggested Commands/i)).toBeInTheDocument();
    const tasksCmdBtn = screen.getByText('Go to Tasks');
    await fireEvent.click(tasksCmdBtn);

    let currentView = null;
    const { activeView } = await import('../../stores/library.js');
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('tasks');
  });
});



