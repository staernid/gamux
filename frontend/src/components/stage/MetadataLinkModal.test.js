import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import MetadataLinkModal from './MetadataLinkModal.svelte';
import { normalizeGame } from '../../utils/game.js';

describe('MetadataLinkModal Component', () => {
  const mockGame = normalizeGame({
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    store: 'Steam',
    game_dir: '/games/cyberpunk'
  });

  it('renders provider tabs and search controls', () => {
    render(MetadataLinkModal, { props: { game: mockGame } });

    expect(screen.getAllByText(/Change Metadata Provider/i).length).toBeGreaterThan(0);
    expect(screen.getByText('Steam')).toBeInTheDocument();
    expect(screen.getByText('GOG.com')).toBeInTheDocument();
    expect(screen.getByText('RAWG Video Games Database')).toBeInTheDocument();
    expect(screen.getByText('None (Unlinked)')).toBeInTheDocument();
  });

  it('switches to None (Unlinked) explanation', async () => {
    render(MetadataLinkModal, { props: { game: mockGame } });

    const noneTab = screen.getByText('None (Unlinked)');
    await fireEvent.click(noneTab);

    expect(screen.getByText(/None \(Unlinked Metadata\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Set None \(Unlinked\)/i)).toBeInTheDocument();
  });

  it('renders Reset to Auto button to unset metadata overrides', () => {
    render(MetadataLinkModal, { props: { game: mockGame } });

    const resetButton = screen.getByText('Reset to Auto');
    expect(resetButton).toBeInTheDocument();
  });
});
