import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import CatalogCard from './CatalogCard.svelte';

describe('CatalogCard Component', () => {
  const mockItem = {
    id: '1145350',
    name: 'Hades II',
    store: 'steam',
    poster_url: 'https://example.com/hades2.jpg',
    release_date: '2024-05-06',
  };

  it('renders title, uppercase store badge, and release year', () => {
    render(CatalogCard, { props: { item: mockItem } });

    expect(screen.getByText('Hades II')).toBeInTheDocument();
    expect(screen.getByText('STEAM')).toBeInTheDocument();
    expect(screen.getByText(/2024/)).toBeInTheDocument();
  });

  it('renders fallback representation when poster image errors', async () => {
    render(CatalogCard, { props: { item: mockItem } });

    const img = screen.getByRole('img');
    await fireEvent.error(img);

    // After image error, fallback placeholder with gamepad icon and title appears
    const allTitles = screen.getAllByText('Hades II');
    expect(allTitles.length).toBeGreaterThanOrEqual(2);
  });

  it('renders Installed badge when matching installed local title', () => {
    const localMatch = { isInstalled: true, name: 'Hades II' };
    render(CatalogCard, { props: { item: mockItem, localMatch } });

    expect(screen.getByText('Installed')).toBeInTheDocument();
    expect(screen.getByText(/Open in Library/i)).toBeInTheDocument();
  });

  it('renders In Library badge when matching uninstalled local title', () => {
    const localMatch = { isInstalled: false, name: 'Hades II' };
    render(CatalogCard, { props: { item: mockItem, localMatch } });

    expect(screen.getByText('In Library')).toBeInTheDocument();
  });

  it('renders Open in Hub action cue when title is not in library', () => {
    render(CatalogCard, { props: { item: mockItem, localMatch: null } });

    expect(screen.getByText(/Open in Hub/i)).toBeInTheDocument();
  });

  it('dispatches select on click and focus on mouseenter', async () => {
    const onselect = vi.fn();
    const onfocus = vi.fn();
    render(CatalogCard, { props: { item: mockItem, onselect, onfocus } });

    const btn = screen.getByRole('button');
    await fireEvent.click(btn);
    expect(onselect).toHaveBeenCalledWith(mockItem);

    await fireEvent.mouseEnter(btn);
    expect(onfocus).toHaveBeenCalled();
  });
});
