import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import ArchScrollbar from './ArchScrollbar.svelte';

describe('ArchScrollbar Component', () => {
  it('renders SVG arc track and indicator for library count > 1', () => {
    const { container } = render(ArchScrollbar, { props: { count: 10, currentPos: 3 } });

    const svg = container.querySelector('svg');
    expect(svg).toBeInTheDocument();
    expect(container.querySelectorAll('path').length).toBeGreaterThan(0);
  });

  it('does not render when count is 1 or 0', () => {
    const { container } = render(ArchScrollbar, { props: { count: 1, currentPos: 0 } });
    expect(container.querySelector('svg')).toBeNull();
  });
});
