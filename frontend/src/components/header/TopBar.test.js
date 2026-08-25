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

  it('renders navigation buttons (1: Shelf, 2: Downloads, 3: Activity, 4: Settings)', () => {
    render(TopBar);

    expect(screen.getByTitle(/Shelf \(1\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Downloads \(2\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Activity \(3\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Settings \(4\)/i)).toBeInTheDocument();
  });

  it('navigates to downloads view on button click', async () => {
    render(TopBar);

    const dlBtn = screen.getByTitle(/Downloads \(2\)/i);
    await fireEvent.click(dlBtn);

    let currentView = '';
    activeView.subscribe(v => currentView = v)();
    expect(currentView).toBe('downloads');
  });

  it('displays active downloads badge count when downloads are running', async () => {
    render(TopBar);

    tasksStore.items.set([
      { id: 'dl_1', status: 'downloading', game_name: 'Test Game' }
    ]);

    const badge = await screen.findByText('1');
    expect(badge).toBeInTheDocument();
  });
});
