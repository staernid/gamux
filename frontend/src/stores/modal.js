import { writable } from 'svelte/store';

function createModalStore() {
  const { subscribe, set } = writable(null);

  return {
    subscribe,
    open(type, props = {}) {
      set({ type, props });
    },
    close() {
      set(null);
    }
  };
}

export const modalStore = createModalStore();
