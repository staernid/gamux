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
});

