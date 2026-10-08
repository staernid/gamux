import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import TopBar from './TopBar.svelte';
import { activeView } from '../../stores/library.js';
import { tasksStore } from '../../stores/tasks.js';

describe('TopBar Component', () => {
  beforeEach(() => {
    activeView.set('library');
    tasksStore.reset();
  });

  it('renders navigation buttons (1: Shelf, 2: Downloads, 3: Selected Game, 4: Settings, 5: Catalog)', () => {
    render(TopBar);

    expect(screen.getByTitle(/Shelf \(1\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Downloads \(2\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Selected Game \(3\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Settings \(4\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Catalog \(5\)/i)).toBeInTheDocument();
  });

  it('navigates to downloads view on button click', async () => {
    render(TopBar);

    const downloadBtn = screen.getByTitle(/Downloads \(2\)/i);
    await fireEvent.click(downloadBtn);

    let currentView = '';
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('tasks');
  });

  it('displays active downloads badge count when downloads are running', async () => {
    render(TopBar);

    tasksStore.items.set([
      { id: 'dl_1', status: 'downloading', game_name: 'Test Game' }
    ]);

    const badge = await screen.findByText('1');
    expect(badge).toBeInTheDocument();
  });

  it('navigates to library on shelf click when activeView is game-hub and highlights gamepad', async () => {
    const { games, selectedGame } = await import('../../stores/library.js');
    const mockGame = { gameDir: '/games/hl2', name: 'Half-Life 2' };
    games.set([mockGame]);
    selectedGame.set(mockGame);
    activeView.set('game-hub');
    render(TopBar);

    const gamepadBtn = screen.getByTitle(/Selected: Half-Life 2/i);
    expect(gamepadBtn).toBeInTheDocument();
    expect(gamepadBtn.className).toContain('bg-surface-cardHover');

    const shelfBtn = screen.getByTitle(/Shelf \(1\)/i);
    await fireEvent.click(shelfBtn);
    let currentView = '';
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('library');
  });

  it('navigates to game-hub on middle gamepad button click', async () => {
    const { games, selectedGame } = await import('../../stores/library.js');
    const mockGame = { gameDir: '/games/portal', name: 'Portal' };
    games.set([mockGame]);
    selectedGame.set(mockGame);
    activeView.set('library');
    render(TopBar);

    const gamepadBtn = screen.getByTitle(/Selected: Portal/i);
    await fireEvent.click(gamepadBtn);

    let currentView = '';
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('game-hub');
  });

  it('renders vertically stacked running games on the right when games are active', async () => {
    const { games, runningGames } = await import('../../stores/library.js');
    games.set([
      { gameDir: '/games/game1', name: 'Game One' },
      { gameDir: '/games/game2', name: 'Game Two' }
    ]);
    runningGames.set(new Set(['/games/game1', '/games/game2']));

    render(TopBar);

    expect(screen.getByText('Game One')).toBeInTheDocument();
    expect(screen.getByText('Game Two')).toBeInTheDocument();
    const stopButtons = screen.getAllByRole('button', { name: /Stop/i });
    expect(stopButtons.length).toBe(2);

    runningGames.set(new Set());
  });

  it('renders profile pills with badges for secondary/co-op seats and omits badge for solo main', async () => {
    const { runningInstances } = await import('../../stores/library.js');
    runningInstances.set([
      { instanceId: '/games/hl2', gameDir: '/games/hl2', gameName: 'Half-Life 2', profileId: 'main', seatId: '' },
      { instanceId: '/games/portal2:p2', gameDir: '/games/portal2', gameName: 'Portal 2 (P2)', profileId: 'p2', seatId: 'p2' }
    ]);

    render(TopBar);

    expect(screen.getByText('Half-Life 2')).toBeInTheDocument();
    expect(screen.getByText('Portal 2 (P2)')).toBeInTheDocument();
    // P2 badge and/or avatar initials are shown
    expect(screen.getAllByText('P2').length).toBeGreaterThanOrEqual(1);

    runningInstances.set([]);
  });

  it('renders catalog button and navigates to catalog view on click', async () => {
    render(TopBar);

    const catalogBtn = screen.getByTitle(/Catalog \(5\)/i);
    expect(catalogBtn).toBeInTheDocument();

    await fireEvent.click(catalogBtn);
    let currentView = '';
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('catalog');
  });

  it('does not render visible text labels like "Back to Shelf" or "Ctrl+K" in navigation', () => {
    render(TopBar);
    expect(screen.queryByText('Back to Shelf')).not.toBeInTheDocument();
    expect(screen.queryByText('Shelf')).not.toBeInTheDocument();
    expect(screen.queryByText('Tasks')).not.toBeInTheDocument();
    expect(screen.queryByText('Downloads')).not.toBeInTheDocument();
    expect(screen.queryByText('Settings')).not.toBeInTheDocument();
    expect(screen.queryByText('Search')).not.toBeInTheDocument();
    expect(screen.queryByText(/Ctrl\+K/i)).not.toBeInTheDocument();

    activeView.set('game-hub');
    expect(screen.queryByText('Back to Shelf')).not.toBeInTheDocument();
  });
});

