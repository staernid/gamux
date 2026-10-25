export class ModalState {
  current = $state(null);
  #subscribers = new Set();

  open = (type, props = {}) => {
    this.current = { type, props };
    this.#notify();
  };

  close = () => {
    this.current = null;
    this.#notify();
  };

  #notify() {
    for (const fn of this.#subscribers) {
      try {
        fn(this.current);
      } catch {}
    }
  }

  // Svelte store contract interop
  subscribe = (fn) => {
    this.#subscribers.add(fn);
    fn(this.current);
    return () => this.#subscribers.delete(fn);
  };
}

export const modalStore = new ModalState();
