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
    expect(screen.getByText('Standalone / DRM-Free')).toBeInTheDocument();
  });

  it('switches to Standalone mode explanation', async () => {
    render(MetadataLinkModal, { props: { game: mockGame } });

    const standaloneTab = screen.getByText('Standalone / DRM-Free');
    await fireEvent.click(standaloneTab);

    expect(screen.getByText(/Standalone \/ Custom Non-Store Configuration/i)).toBeInTheDocument();
    expect(screen.getByText(/Set as Standalone Game/i)).toBeInTheDocument();
  });
});
