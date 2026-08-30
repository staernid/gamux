import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import InheritSwitcher from './InheritSwitcher.svelte';

describe('InheritSwitcher Component', () => {
  it('renders label, description, and tri-state inherit badge', () => {
    render(InheritSwitcher, {
      props: {
        label: 'MangoHud Overlay',
        description: 'FPS and frametime metrics',
        value: null,
        inheritedValue: true,
        tag: 'HUD'
      }
    });

    expect(screen.getByText('MangoHud Overlay')).toBeInTheDocument();
    expect(screen.getByText('FPS and frametime metrics')).toBeInTheDocument();
    expect(screen.getByText('Inherit')).toBeInTheDocument();
    expect(screen.getByText('(On)')).toBeInTheDocument();
  });

  it('switches between states when buttons are clicked', async () => {
    render(InheritSwitcher, {
      props: {
        label: 'FSR Upscaling',
        value: null,
        inheritedValue: false
      }
    });

    const onBtn = screen.getByRole('button', { name: /^On$/i });
    const offBtn = screen.getByRole('button', { name: /^Off$/i });
    const inheritBtn = screen.getByRole('button', { name: /Inherit/i });

    await fireEvent.click(onBtn);
    expect(onBtn.className).toContain('bg-');

    await fireEvent.click(offBtn);
    expect(offBtn.className).toContain('bg-');

    await fireEvent.click(inheritBtn);
    expect(inheritBtn.className).toContain('bg-');
  });
});
