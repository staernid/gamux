import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import SwitcherRow from './SwitcherRow.svelte';

describe('SwitcherRow Component', () => {
  it('renders label, description, and binary ON/OFF state', async () => {
    const { component } = render(SwitcherRow, {
      props: {
        label: 'Test Option',
        description: 'Test Description',
        tag: 'TEST',
        value: true,
        color: 'emerald'
      }
    });

    expect(screen.getByText('Test Option')).toBeInTheDocument();
    expect(screen.getByText('Test Description')).toBeInTheDocument();
    expect(screen.getByText('TEST')).toBeInTheDocument();
    expect(screen.getByText('ON')).toBeInTheDocument();

    const btn = screen.getByRole('button');
    await fireEvent.click(btn);
    expect(screen.getByText('OFF')).toBeInTheDocument();
  });

  it('cycles tri-state options: AUTO -> ON -> OFF -> AUTO', async () => {
    render(SwitcherRow, {
      props: {
        label: 'MangoHud HUD',
        tag: 'HUD',
        triState: true,
        value: null,
        inheritedValue: false,
        color: 'emerald'
      }
    });

    // Initial is AUTO
    expect(screen.getByText('AUTO')).toBeInTheDocument();
    const btn = screen.getByRole('button');

    // Click 1: turns ON
    await fireEvent.click(btn);
    expect(screen.getByText('ON')).toBeInTheDocument();

    // Click 2: turns OFF
    await fireEvent.click(btn);
    expect(screen.getByText('OFF')).toBeInTheDocument();

    // Click 3: turns AUTO
    await fireEvent.click(btn);
    expect(screen.getByText('AUTO')).toBeInTheDocument();
  });

  it('supports inverted labels for disable options', async () => {
    render(SwitcherRow, {
      props: {
        label: 'Disable Overlay',
        tag: 'OVERLAY',
        value: true, // disable_overlay = true means overlay is OFF
        invertLabel: true,
        color: 'rose'
      }
    });

    expect(screen.getByText('OVERLAY')).toBeInTheDocument();
    expect(screen.getByText('OFF')).toBeInTheDocument();
  });
});
