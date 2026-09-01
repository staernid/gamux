import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Icon from './Icon.svelte';

describe('Icon Component', () => {
  it('renders spinner icon with animation class', () => {
    const { container } = render(Icon, { props: { name: 'spinner', className: 'w-5 h-5 text-sky-400' } });
    const svg = container.querySelector('svg');
    expect(svg).toBeInTheDocument();
    expect(svg.classList.contains('animate-spin')).toBe(true);
    expect(svg.classList.contains('w-5')).toBe(true);
  });

  it('renders close and search icons properly', () => {
    const { container: c1 } = render(Icon, { props: { name: 'close' } });
    expect(c1.querySelector('svg')).toBeInTheDocument();

    const { container: c2 } = render(Icon, { props: { name: 'search' } });
    expect(c2.querySelector('svg')).toBeInTheDocument();
  });
});
