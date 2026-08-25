import { describe, it, expect } from 'vitest';
import { modalStore } from './modal.js';

describe('modalStore', () => {
  it('initializes as null', () => {
    let state = null;
    modalStore.subscribe(v => state = v)();
    expect(state).toBeNull();
  });

  it('opens modal with type and props', () => {
    modalStore.open('artwork-picker', { game: { name: 'Portal' } });

    let state = null;
    modalStore.subscribe(v => state = v)();
    expect(state).toEqual({
      type: 'artwork-picker',
      props: { game: { name: 'Portal' } }
    });
  });

  it('closes active modal', () => {
    modalStore.open('metadata-link');
    modalStore.close();

    let state = null;
    modalStore.subscribe(v => state = v)();
    expect(state).toBeNull();
  });
});
