import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import ArtworkPickerModal from './ArtworkPickerModal.svelte';
import { normalizeGame } from '../../utils/game.js';

describe('ArtworkPickerModal Component', () => {
  const mockGame = normalizeGame({
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    store: 'Steam',
    game_dir: '/games/cyberpunk'
  });

  it('renders header and asset type selectors', () => {
    render(ArtworkPickerModal, { props: { game: mockGame } });

    expect(screen.getAllByText(/Change Artwork/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/Cover Art/i)).toBeInTheDocument();
    expect(screen.getByText(/Banner Header/i)).toBeInTheDocument();
    expect(screen.getByText(/Hero Background/i)).toBeInTheDocument();
    expect(screen.getByText(/Title Logo/i)).toBeInTheDocument();
  });

  it('allows switching asset types', async () => {
    render(ArtworkPickerModal, { props: { game: mockGame } });

    const bannerBtn = screen.getByText(/Banner Header/i);
    await fireEvent.click(bannerBtn);
    expect(bannerBtn.className).toContain('bg-sky-600');
  });
});
