import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import App from './App.svelte';
import { games, selectedGame, activeView } from './stores/library.js';
import { normalizeGame } from './utils/game.js';

describe('App Root Shell', () => {
  const mockGames = [
    { name: 'Balatro', app_id: '2379780', store: 'Steam', game_dir: '/games/balatro', exe_path: 'Balatro.exe', is_ready: true, is_patched: true, is_installed: true, file_count: 5 },
    { name: 'Astroneer', app_id: '361420', store: 'Steam', game_dir: '/games/astroneer', exe_path: 'Astro.exe', is_ready: true, is_patched: true, is_installed: true, file_count: 10 },
  ].map(normalizeGame);

  beforeEach(() => {
    localStorage.clear();
    games.set(mockGames);
    selectedGame.set(mockGames[0]);
    activeView.set('library');
  });

  it('renders TopBar and Library view by default', () => {
    render(App);

    expect(screen.getByTitle(/Shelf \(1\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Tasks \(2\)/i)).toBeInTheDocument();
    expect(screen.getByTitle(/Settings \(3\)/i)).toBeInTheDocument();
  });

  it('switches views smoothly via activeView store', async () => {
    render(App);

    // Switch to tasks view
    activeView.set('tasks');
    expect(await screen.findByRole('heading', { name: /Tasks/i })).toBeInTheDocument();

    // Switch to settings view
    activeView.set('settings');
    expect(await screen.findByText('System Settings')).toBeInTheDocument();

    // Switch to game-hub
    selectedGame.set(mockGames[0]);
    activeView.set('game-hub');
    expect(await screen.findByRole('button', { name: /Back to Shelf/i })).toBeInTheDocument();
    expect(await screen.findByText('Play Now')).toBeInTheDocument();

    // Escape returns to library
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(await screen.findByTitle(/Shelf \(1\)/i)).toBeInTheDocument();
  });
});


