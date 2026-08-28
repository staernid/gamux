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

    expect(screen.getByPlaceholderText(/Type to filter titles/i)).toBeInTheDocument();
    expect(screen.getAllByText('Astroneer').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Balatro').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Cyberpunk 2077').length).toBeGreaterThan(0);
  });

  it('filters titles based on search query', async () => {
    render(SearchGridModal);

    const input = screen.getByPlaceholderText(/Type to filter titles/i);
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
  });
});
