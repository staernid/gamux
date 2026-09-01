import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import DrawerNewsTab from './DrawerNewsTab.svelte';

describe('DrawerNewsTab Component', () => {
  const mockNews = [
    {
      title: 'Patch 2.13 Available Now',
      contents: '[b]Patch Highlights:[/b]\nAdds support for AMD FidelityFX Super Resolution 3.',
      date: 1726142400,
      feedlabel: 'Community Announcements',
      author: 'CD PROJEKT RED',
      url: 'https://store.steampowered.com/news/app/1091500/view/123'
    },
    {
      title: 'Night City Wire Special',
      contents: 'Deep dive into Phantom Liberty enhancements.',
      date: 1720000000,
      feedlabel: 'News',
      author: 'Editorial',
      url: 'https://store.steampowered.com/news/app/1091500/view/456'
    }
  ];

  it('renders loading state when isLoadingNews is true', () => {
    render(DrawerNewsTab, {
      props: {
        isLoadingNews: true,
        newsItems: []
      }
    });

    expect(screen.getByText(/Fetching official Steam news\.\.\./i)).toBeInTheDocument();
  });

  it('renders error message with retry button when newsError occurs', async () => {
    const refreshSpy = vi.fn();
    render(DrawerNewsTab, {
      props: {
        isLoadingNews: false,
        newsError: 'Failed to fetch Steam news',
        newsItems: [],
        onRefreshNews: refreshSpy
      }
    });

    expect(screen.getByText('Failed to fetch Steam news')).toBeInTheDocument();
    const retryBtn = screen.getByRole('button', { name: /Retry Fetch/i });
    await fireEvent.click(retryBtn);
    expect(refreshSpy).toHaveBeenCalled();
  });

  it('renders news items with titles, author, date, and feed badges', () => {
    render(DrawerNewsTab, {
      props: {
        isLoadingNews: false,
        newsItems: mockNews,
        expandedNewsIndex: 0
      }
    });

    expect(screen.getByText('Patch 2.13 Available Now')).toBeInTheDocument();
    expect(screen.getByText('Night City Wire Special')).toBeInTheDocument();
    expect(screen.getByText('Community Announcements')).toBeInTheDocument();
    expect(screen.getByText('News')).toBeInTheDocument();
    expect(screen.getByText(/CD PROJEKT RED/i)).toBeInTheDocument();
  });

  it('expands article contents and parses BBCode formatted text', () => {
    render(DrawerNewsTab, {
      props: {
        isLoadingNews: false,
        newsItems: mockNews,
        expandedNewsIndex: 0
      }
    });

    expect(screen.getByText(/Patch Highlights:/i)).toBeInTheDocument();
    expect(screen.getByText(/Adds support for AMD FidelityFX Super Resolution 3/i)).toBeInTheDocument();
  });

  it('calls onToggleExpand when title or expand button is clicked', async () => {
    const toggleSpy = vi.fn();
    render(DrawerNewsTab, {
      props: {
        isLoadingNews: false,
        newsItems: mockNews,
        expandedNewsIndex: 0,
        onToggleExpand: toggleSpy
      }
    });

    const secondItemBtn = screen.getByRole('button', { name: /Night City Wire Special/i });
    await fireEvent.click(secondItemBtn);
    expect(toggleSpy).toHaveBeenCalledWith(1);
  });
});
