import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import PosterArt from './PosterArt.svelte';

describe('PosterArt Component', () => {
  it('renders thumbnail URL as primary image source for local art', () => {
    const { container } = render(PosterArt, {
      props: {
        name: 'Half-Life 2',
        appId: '220',
        localCoverUrl: '/art/half-life-2/cover.jpg?t=12345',
        thumbnailWidth: 300,
      },
    });

    const img = container.querySelector('img');
    expect(img).toBeInTheDocument();
    expect(img.getAttribute('src')).toBe('/art/thumb?id=half-life-2%2Fcover.jpg&w=300&t=12345');
    expect(img.getAttribute('alt')).toBe('Half-Life 2');
  });

  it('honors custom thumbnailWidth prop', () => {
    const { container } = render(PosterArt, {
      props: {
        name: 'Portal',
        appId: '400',
        localCoverUrl: '/art/portal/cover.jpg',
        thumbnailWidth: 200,
      },
    });

    const img = container.querySelector('img');
    expect(img).toBeInTheDocument();
    expect(img.getAttribute('src')).toBe('/art/thumb?id=portal%2Fcover.jpg&w=200');
  });

  it('bypasses thumbnail when thumbnailWidth is 0', () => {
    const { container } = render(PosterArt, {
      props: {
        name: 'Portal',
        appId: '400',
        localCoverUrl: '/art/portal/cover.jpg',
        thumbnailWidth: 0,
      },
    });

    const img = container.querySelector('img');
    expect(img).toBeInTheDocument();
    expect(img.getAttribute('src')).toBe('/art/portal/cover.jpg');
  });

  it('falls back to localCoverUrl and then remote CDN when thumbnail errors', async () => {
    const { container } = render(PosterArt, {
      props: {
        name: 'Portal 2',
        appId: '620',
        localCoverUrl: '/art/portal-2/cover.jpg',
        thumbnailWidth: 300,
      },
    });

    let img = container.querySelector('img');
    expect(img.getAttribute('src')).toBe('/art/thumb?id=portal-2%2Fcover.jpg&w=300');

    // Trigger error on thumbnail -> falls back to localCoverUrl
    await fireEvent.error(img);
    img = container.querySelector('img');
    expect(img.getAttribute('src')).toBe('/art/portal-2/cover.jpg');

    // Trigger error on localCoverUrl -> falls back to Steam CDN 600x900
    await fireEvent.error(img);
    img = container.querySelector('img');
    expect(img.getAttribute('src')).toBe('https://cdn.cloudflare.steamstatic.com/steam/apps/620/library_600x900.jpg');
  });

  it('renders initials placeholder when no URLs available or all error', async () => {
    const { container } = render(PosterArt, {
      props: {
        name: 'Custom Game',
        appId: '0',
        localCoverUrl: '',
        posterUrl: '',
      },
    });

    expect(container.querySelector('img')).toBeNull();
    expect(container.textContent).toContain('CG');
    expect(container.textContent).toContain('Custom Game');
  });

  it('renders clean fallback immediately when forced to none', () => {
    const { container } = render(PosterArt, {
      props: {
        name: 'Unset Art',
        appId: '123',
        posterUrl: 'none',
      },
    });

    expect(container.querySelector('img')).toBeNull();
    expect(container.textContent).toContain('UA');
  });
});
