import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import Toast from './Toast.svelte';
import { notificationStore } from '../../stores/notifications.js';

describe('Toast Component with Undo Snackbars', () => {
  beforeEach(() => {
    notificationStore.undoQueue.clearAll();
  });

  it('renders active undo snackbars with countdown and triggers undo action on click', async () => {
    const undoAction = vi.fn();
    notificationStore.undo('Reset launch profile for Hades II', undoAction);

    render(Toast);

    expect(screen.getByTestId('undo-snackbar-container')).toBeInTheDocument();
    expect(screen.getByText('Reset launch profile for Hades II')).toBeInTheDocument();

    const undoBtn = screen.getByTestId('undo-action-btn');
    expect(undoBtn).toHaveTextContent(/Undo \(6s\)/);

    await fireEvent.click(undoBtn);

    expect(undoAction).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId('undo-snackbar-container')).not.toBeInTheDocument();
  });

  it('dismisses undo snackbar when close icon is clicked', async () => {
    const undoAction = vi.fn();
    notificationStore.undo('Unlinked Steam metadata', undoAction);

    render(Toast);

    const dismissBtn = screen.getByTestId('dismiss-undo-btn');
    await fireEvent.click(dismissBtn);

    expect(undoAction).not.toHaveBeenCalled();
    expect(screen.queryByTestId('undo-snackbar-container')).not.toBeInTheDocument();
  });
});
