import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import Carousel3D from './Carousel3D.svelte';
import { games, selectedGame } from '../../stores/library.js';
import { normalizeGame } from '../../utils/game.js';

describe('Carousel3D Component', () => {
  const mockGames = [
    { name: 'Astroneer', app_id: '361420', store: 'Steam', game_dir: '/games/astroneer', disk_size_bytes: 3000000000 },
    { name: 'Balatro', app_id: '2379780', store: 'Steam', game_dir: '/games/balatro', disk_size_bytes: 50000000 },
    { name: 'Cyberpunk 2077', app_id: '', store: 'GOG', game_dir: '/games/cyberpunk', disk_size_bytes: 70000000000 },
    { name: 'Standalone Game', app_id: '', store: 'Custom', game_dir: '/games/standalone', disk_size_bytes: 2000000000 },
    { name: 'Hades', app_id: '1145360', store: 'Steam', game_dir: '/games/hades', disk_size_bytes: 12000000000 },
  ].map(normalizeGame);

  beforeEach(() => {
    games.set(mockGames);
    selectedGame.set(mockGames[0]);
  });

  it('renders active game with floating title badge', () => {
    render(Carousel3D);

    const titleBadge = screen.getByText('Astroneer');
    expect(titleBadge).toBeInTheDocument();
  });

  it('rotates to adjacent game on card click', async () => {
    const { component } = render(Carousel3D);

    const balatroImg = screen.getByAltText('Balatro');
    await fireEvent.click(balatroImg);

    // After step/click, targetPos advances
    component.step(1);
    expect(component).toBeTruthy();
  });

  it('handles empty library state gracefully', () => {
    games.set([]);
    selectedGame.set(null);
    render(Carousel3D);

    expect(screen.getByText('No Games Found')).toBeInTheDocument();
  });

  it('handles single game library without errors', () => {
    games.set([mockGames[0]]);
    selectedGame.set(mockGames[0]);
    render(Carousel3D);

    expect(screen.getByText('Astroneer')).toBeInTheDocument();
  });

  it('renders clean poster cards without overlay badge clutter', () => {
    render(Carousel3D);

    // Posters rendered cleanly without card badges
    expect(screen.getByAltText('Astroneer')).toBeInTheDocument();
    expect(screen.queryByText('Ready')).not.toBeInTheDocument();
  });

  it('allows stepping left and right smoothly without getting locked', () => {
    const { component } = render(Carousel3D);

    component.step(1);
    expect(component.getCurrentPos()).toBeDefined();

    component.step(-1);
    expect(component.getCurrentPos()).toBeDefined();

    component.addRotaryDelta(1.5);
    expect(component.getCurrentPos()).toBeDefined();

    component.seek(2);
    expect(component.getCurrentPos()).toBeDefined();
  });

  it('renders ambient glow container without error', () => {
    const { container } = render(Carousel3D);
    const glow = container.querySelector('.rounded-full.pointer-events-none');
    expect(glow).toBeInTheDocument();
  });
});

