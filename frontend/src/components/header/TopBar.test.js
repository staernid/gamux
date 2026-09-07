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

  it('renders navigation buttons (1: Shelf, 2: Tasks, 3: Settings)', () => {
    render(TopBar);

    expect(screen.getByTitle(/Shelf \(1\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Tasks \(2\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Settings \(3\)/i)).toBeInTheDocument();
  });

  it('navigates to tasks view on button click', async () => {
    render(TopBar);

    const taskBtn = screen.getByTitle(/Tasks \(2\)/i);
    await fireEvent.click(taskBtn);

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

  it('morphs first button into Back to Shelf when activeView is game-hub', async () => {
    activeView.set('game-hub');
    render(TopBar);

    const hubBtn = screen.getByTitle(/Back to Shelf/i);
    expect(hubBtn).toBeInTheDocument();

    await fireEvent.click(hubBtn);
    let currentView = '';
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('library');
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

  it('renders search and command palette button and opens search-grid modal on click', async () => {
    const { modalStore } = await import('../../stores/modal.js');
    const openSpy = vi.spyOn(modalStore, 'open');

    render(TopBar);

    const searchBtn = screen.getByTitle(/Search & Commands \(Ctrl\+K\)/i);
    expect(searchBtn).toBeInTheDocument();

    await fireEvent.click(searchBtn);
    expect(openSpy).toHaveBeenCalledWith('search-grid');
  });
});

