import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import { get } from 'svelte/store';
import ConfirmModal from './ConfirmModal.svelte';
import { modalStore } from '../../stores/modal.js';

describe('ConfirmModal Component', () => {
  it('renders confirmation title, message, and action buttons', () => {
    render(ConfirmModal, {
      title: 'Discard Changes?',
      message: 'You have unsaved changes in Settings.',
      confirmLabel: 'Discard',
      cancelLabel: 'Keep Editing',
      danger: true
    });

    expect(screen.getByText('Discard Changes?')).toBeInTheDocument();
    expect(screen.getByText('You have unsaved changes in Settings.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Discard' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Keep Editing' })).toBeInTheDocument();
  });

  it('triggers onConfirm callback and closes modal on confirm click', async () => {
    const onConfirm = vi.fn();
    modalStore.open('confirm');

    render(ConfirmModal, {
      confirmLabel: 'Confirm',
      onConfirm
    });

    const confirmBtn = screen.getByRole('button', { name: 'Confirm' });
    await fireEvent.click(confirmBtn);

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(get(modalStore)).toBeNull();
  });

  it('triggers onCancel callback and closes modal on cancel click', async () => {
    const onCancel = vi.fn();
    modalStore.open('confirm');

    render(ConfirmModal, {
      cancelLabel: 'Cancel',
      onCancel
    });

    const cancelBtn = screen.getByRole('button', { name: 'Cancel' });
    await fireEvent.click(cancelBtn);

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(get(modalStore)).toBeNull();
  });

  it('closes modal on Escape key press', async () => {
    const onCancel = vi.fn();
    modalStore.open('confirm');

    render(ConfirmModal, {
      onCancel
    });

    await fireEvent.keyDown(window, { key: 'Escape' });

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(get(modalStore)).toBeNull();
  });

  it('focuses Cancel button when danger is true to prevent accidental confirmation', () => {
    render(ConfirmModal, {
      danger: true,
      confirmLabel: 'Delete',
      cancelLabel: 'Cancel'
    });

    const cancelBtn = screen.getByRole('button', { name: 'Cancel' });
    expect(document.activeElement).toBe(cancelBtn);
  });

  it('focuses Confirm button when danger is false', () => {
    render(ConfirmModal, {
      danger: false,
      confirmLabel: 'Proceed',
      cancelLabel: 'Cancel'
    });

    const confirmBtn = screen.getByRole('button', { name: 'Proceed' });
    expect(document.activeElement).toBe(confirmBtn);
  });
});
