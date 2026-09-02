import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import StepperRow from './StepperRow.svelte';

describe('StepperRow Component', () => {
  const sampleOptions = [
    { value: 'low', label: 'Low' },
    { value: 'medium', label: 'Medium' },
    { value: 'high', label: 'High' }
  ];

  it('renders label, description, and current value', () => {
    render(StepperRow, {
      props: {
        label: 'Graphics Quality',
        description: 'Set preset rendering detail',
        options: sampleOptions,
        value: 'medium'
      }
    });

    expect(screen.getByText('Graphics Quality')).toBeInTheDocument();
    expect(screen.getByText('Set preset rendering detail')).toBeInTheDocument();
    expect(screen.getByText('Medium')).toBeInTheDocument();
  });

  it('does NOT step forward when clicking the capsule, text, or label zone', async () => {
    render(StepperRow, {
      props: {
        label: 'Graphics Quality',
        options: sampleOptions,
        value: 'medium'
      }
    });

    // Verify initial value is Medium
    expect(screen.getByText('Medium')).toBeInTheDocument();

    // Click label
    await fireEvent.click(screen.getByText('Graphics Quality'));
    expect(screen.getByText('Medium')).toBeInTheDocument();

    // Click value display in the center of the capsule
    await fireEvent.click(screen.getByText('Medium'));
    expect(screen.getByText('Medium')).toBeInTheDocument();

    // Click capsule slider element itself
    const slider = screen.getByRole('slider', { name: /Graphics Quality: Medium/i });
    await fireEvent.click(slider);
    expect(screen.getByText('Medium')).toBeInTheDocument();
  });

  it('steps forward only when explicitly clicking the right button', async () => {
    render(StepperRow, {
      props: {
        label: 'Graphics Quality',
        options: sampleOptions,
        value: 'medium'
      }
    });

    expect(screen.getByText('Medium')).toBeInTheDocument();

    const nextBtn = screen.getByRole('button', { name: /Next Option/i });
    await fireEvent.click(nextBtn);

    expect(screen.getByText('High')).toBeInTheDocument();
  });

  it('steps backward only when explicitly clicking the left button', async () => {
    render(StepperRow, {
      props: {
        label: 'Graphics Quality',
        options: sampleOptions,
        value: 'medium'
      }
    });

    expect(screen.getByText('Medium')).toBeInTheDocument();

    const prevBtn = screen.getByRole('button', { name: /Previous Option/i });
    await fireEvent.click(prevBtn);

    expect(screen.getByText('Low')).toBeInTheDocument();
  });

  it('supports keyboard stepping on focused slider', async () => {
    render(StepperRow, {
      props: {
        label: 'Graphics Quality',
        options: sampleOptions,
        value: 'medium'
      }
    });

    const slider = screen.getByRole('slider', { name: /Graphics Quality: Medium/i });

    // ArrowLeft -> Low
    await fireEvent.keyDown(slider, { key: 'ArrowLeft' });
    expect(screen.getByText('Low')).toBeInTheDocument();

    // ArrowRight -> Medium
    await fireEvent.keyDown(slider, { key: 'ArrowRight' });
    expect(screen.getByText('Medium')).toBeInTheDocument();
  });
});
