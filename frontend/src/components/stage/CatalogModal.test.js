import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import CatalogModal from './CatalogModal.svelte';
import { games, selectedGame, activeView } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';

describe('CatalogModal Component', () => {
  const mockGames = [
    { name: 'Astroneer', app_id: '361420', store: 'Steam', game_dir: '/games/astroneer', is_installed: true },
    { name: 'Balatro', app_id: '2379780', store: 'Steam', game_dir: '/games/balatro', is_installed: true },
    { name: 'Cyberpunk 2077', app_id: '', store: 'GOG', gog_id: '1091500', game_dir: '/games/cyberpunk', is_installed: false },
  ].map(normalizeGame);

  beforeEach(() => {
    games.set(mockGames);
    selectedGame.set(mockGames[0]);
    activeView.set('library');
  });

  it('renders catalog header with wireframe globe icon, online discovery state, and search input', () => {
    render(CatalogModal);

    expect(screen.getByText(/Online Catalog/i)).toBeInTheDocument();
    expect(screen.getByText(/Steam · GOG · RAWG/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i)).toBeInTheDocument();
    expect(screen.getByText(/Online Store Catalog Discovery/i)).toBeInTheDocument();
  });

  it('performs debounced online store search and renders store results', async () => {
    const { AppAPI } = await import('../../api/app.js');
    const searchSpy = vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam', release_date: '2024' },
      { id: '2379780', name: 'Balatro', store: 'steam', release_date: '2024' },
    ]);

    render(CatalogModal);

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Hades' } });

    // Wait for debounced search execution
    await new Promise(r => setTimeout(r, 300));
    expect(searchSpy).toHaveBeenCalledWith('Hades');

    const hadesItems = await screen.findAllByText('Hades II');
    expect(hadesItems.length).toBeGreaterThan(0);

    const balatroItems = await screen.findAllByText('Balatro');
    expect(balatroItems.length).toBeGreaterThan(0);

    // Balatro matches local library and is installed
    expect(screen.getByText(/Installed/i)).toBeInTheDocument();
  });

  it('selects virtual store title and navigates to game-hub on click', async () => {
    const { AppAPI } = await import('../../api/app.js');
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam', release_date: '2024' },
    ]);

    const closeSpy = vi.fn();
    render(CatalogModal, { props: { onclose: closeSpy } });

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Hades' } });

    await new Promise(r => setTimeout(r, 300));
    const hadesBtn = (await screen.findAllByText('Hades II'))[0];
    await fireEvent.click(hadesBtn);

    let current = null;
    selectedGame.subscribe(v => current = v)();
    expect(current?.name).toBe('Hades II');
    expect(current?.appId).toBe('1145350');
    expect(current?.isVirtual).toBe(true);

    let currentView = null;
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('game-hub');
    expect(closeSpy).toHaveBeenCalled();
  });

  it('selects matching local title if store result matches installed library game', async () => {
    const { AppAPI } = await import('../../api/app.js');
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '2379780', name: 'Balatro', store: 'steam', release_date: '2024' },
    ]);

    render(CatalogModal);

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Balatro' } });

    await new Promise(r => setTimeout(r, 300));
    const balatroBtn = (await screen.findAllByText('Balatro'))[0];
    await fireEvent.click(balatroBtn);

    let current = null;
    selectedGame.subscribe(v => current = v)();
    expect(current?.name).toBe('Balatro');
    expect(current?.gameDir).toBe('/games/balatro');
    expect(current?.isVirtual).toBeFalsy();
  });

  it('dispatches close event on close button click or escape key', async () => {
    const closeSpy = vi.fn();
    render(CatalogModal, { props: { onclose: closeSpy } });

    const closeBtn = screen.getByRole('button', { name: /Esc Close/i });
    await fireEvent.click(closeBtn);
    expect(closeSpy).toHaveBeenCalled();

    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(closeSpy).toHaveBeenCalledTimes(2);
  });

  it('triggers search when clicking a quick discovery seed in empty state', async () => {
    const { AppAPI } = await import('../../api/app.js');
    const searchSpy = vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam', release_date: '2024' },
    ]);

    render(CatalogModal);

    const hadesSeed = screen.getByRole('button', { name: 'Hades II' });
    await fireEvent.click(hadesSeed);

    await new Promise(r => setTimeout(r, 300));
    expect(searchSpy).toHaveBeenCalledWith('Hades II');

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    expect(input.value).toBe('Hades II');
  });

  it('filters search results using store provider chips', async () => {
    const { AppAPI } = await import('../../api/app.js');
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam', release_date: '2024' },
      { id: '1091500', name: 'Cyberpunk 2077', store: 'gog', release_date: '2020' },
    ]);

    render(CatalogModal);

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Game' } });

    await new Promise(r => setTimeout(r, 300));
    expect((await screen.findAllByText('Hades II')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('Cyberpunk 2077')).length).toBeGreaterThan(0);

    // Filter to GOG only
    const gogFilterBtn = screen.getByRole('button', { name: /GOG \(1\)/i });
    await fireEvent.click(gogFilterBtn);

    expect(screen.queryByText('Hades II')).not.toBeInTheDocument();
    expect(screen.getAllByText('Cyberpunk 2077').length).toBeGreaterThan(0);

    // Switch back to All
    const allFilterBtn = screen.getByRole('button', { name: /All \(2\)/i });
    await fireEvent.click(allFilterBtn);

    expect(screen.getAllByText('Hades II').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Cyberpunk 2077').length).toBeGreaterThan(0);
  });

  it('navigates grid using arrow keys and activates selection with Enter', async () => {
    const { AppAPI } = await import('../../api/app.js');
    vi.spyOn(AppAPI, 'searchStoreGames').mockResolvedValue([
      { id: '1145350', name: 'Hades II', store: 'steam', release_date: '2024' },
      { id: '2379780', name: 'Balatro', store: 'steam', release_date: '2024' },
    ]);

    const closeSpy = vi.fn();
    render(CatalogModal, { props: { onclose: closeSpy } });

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Roguelike' } });
    await new Promise(r => setTimeout(r, 300));

    // Initially focusedIndex is 0 (Hades II)
    // Press ArrowRight to focus index 1 (Balatro)
    await fireEvent.keyDown(window, { key: 'ArrowRight' });
    // Press Enter to activate focused index
    await fireEvent.keyDown(window, { key: 'Enter' });

    let current = null;
    selectedGame.subscribe(v => current = v)();
    expect(current?.name).toBe('Balatro');
    expect(closeSpy).toHaveBeenCalled();
  });

  it('surfaces notification error when store search fails', async () => {
    const { AppAPI } = await import('../../api/app.js');
    const { notificationStore } = await import('../../stores/notifications.js');
    const errorSpy = vi.spyOn(notificationStore, 'error');
    vi.spyOn(AppAPI, 'searchStoreGames').mockRejectedValueOnce(new Error('Search server offline'));

    render(CatalogModal);

    const input = screen.getByPlaceholderText(/Search Steam, GOG, RAWG catalogs/i);
    await fireEvent.input(input, { target: { value: 'Crash' } });

    await new Promise(r => setTimeout(r, 300));
    expect(errorSpy).toHaveBeenCalledWith('Store Search Failed', expect.any(Error));
  });
});
