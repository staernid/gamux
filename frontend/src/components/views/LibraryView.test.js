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

  it('filters titles using status filter chips (Ready / Unpatched)', async () => {
    const { activeFilter } = await import('../../stores/library.js');
    render(LibraryView);

    const readyChip = screen.getByRole('button', { name: /Ready/i });
    expect(readyChip).toBeInTheDocument();
    await fireEvent.click(readyChip);

    let currentFilter = '';
    activeFilter.subscribe(v => currentFilter = v)();
    expect(currentFilter).toBe('patched');

    const unpatchedChip = screen.getByRole('button', { name: /Unpatched/i });
    await fireEvent.click(unpatchedChip);
    activeFilter.subscribe(v => currentFilter = v)();
    expect(currentFilter).toBe('clean');
  });
});
