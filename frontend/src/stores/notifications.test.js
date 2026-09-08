import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { notificationStore } from './notifications.js';

describe('notificationStore Undo Queue', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    notificationStore.undoQueue.clearAll();
  });

  afterEach(() => {
    notificationStore.undoQueue.clearAll();
    vi.useRealTimers();
  });

  it('pushes an undo item with 6-second countdown timer', () => {
    const onUndo = vi.fn();
    notificationStore.undo('Reset launch profile for Balatro', onUndo);

    const queue = get(notificationStore.undoQueue);
    expect(queue).toHaveLength(1);
    expect(queue[0].message).toBe('Reset launch profile for Balatro');
    expect(queue[0].remaining).toBe(6);
    expect(queue[0].action).toBe(onUndo);
  });

  it('counts down remaining seconds and automatically dismisses after 6 seconds', () => {
    const onCommit = vi.fn();
    notificationStore.pushUndo({
      message: 'Unlinked metadata',
      action: vi.fn(),
      onCommit,
      duration: 6000
    });

    expect(get(notificationStore.undoQueue)[0].remaining).toBe(6);

    // Advance 2 seconds
    vi.advanceTimersByTime(2000);
    expect(get(notificationStore.undoQueue)[0].remaining).toBe(4);

    // Advance remaining 4 seconds
    vi.advanceTimersByTime(4000);
    expect(get(notificationStore.undoQueue)).toHaveLength(0);
    expect(onCommit).toHaveBeenCalledTimes(1);
  });

  it('executes action callback and removes item on triggerUndo', () => {
    const onUndo = vi.fn();
    const item = notificationStore.undo('Removed game stub', onUndo);

    expect(get(notificationStore.undoQueue)).toHaveLength(1);

    notificationStore.triggerUndo(item.id);

    expect(onUndo).toHaveBeenCalledTimes(1);
    expect(get(notificationStore.undoQueue)).toHaveLength(0);
  });

  it('dismisses item without calling undo action on dismissUndo', () => {
    const onUndo = vi.fn();
    const item = notificationStore.undo('Test action', onUndo);

    notificationStore.dismissUndo(item.id);

    expect(onUndo).not.toHaveBeenCalled();
    expect(get(notificationStore.undoQueue)).toHaveLength(0);
  });
});
