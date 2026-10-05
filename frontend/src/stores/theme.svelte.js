/**
 * Theme store supporting 3 modes: 'system' | 'dark' | 'light'.
 * Tracks OS prefers-color-scheme changes when in 'system' mode and synchronizes documentElement classes and color-scheme.
 */

export class ThemeState {
  mode = $state('system');
  systemIsDark = $state(true);

  #subscribers = new Set();
  #mediaQuery = null;
  #mediaHandler = null;

  constructor() {
    if (typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem('gamux_theme');
      if (stored === 'dark' || stored === 'light' || stored === 'system') {
        this.mode = stored;
      }
    }

    if (typeof window !== 'undefined' && window.matchMedia) {
      this.#mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      this.systemIsDark = Boolean(this.#mediaQuery.matches);
    }
  }

  get resolved() {
    if (this.mode === 'system') {
      return this.systemIsDark ? 'dark' : 'light';
    }
    return this.mode;
  }

  handleSystemThemeChange(isDark) {
    this.systemIsDark = Boolean(isDark);
    if (this.mode === 'system') {
      this.#applyDom();
      this.#notify();
    }
  }

  init() {
    if (typeof window !== 'undefined' && window.matchMedia) {
      if (!this.#mediaQuery) {
        this.#mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      }
      this.systemIsDark = Boolean(this.#mediaQuery.matches);

      if (!this.#mediaHandler && this.#mediaQuery.addEventListener) {
        this.#mediaHandler = (e) => {
          this.handleSystemThemeChange(e.matches);
        };
        this.#mediaQuery.addEventListener('change', this.#mediaHandler);
      }
    }

    this.#applyDom();
    this.#notify();
  }

  setMode(newMode) {
    if (newMode !== 'system' && newMode !== 'dark' && newMode !== 'light') {
      return;
    }
    this.mode = newMode;

    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('gamux_theme', newMode);
    }

    this.#applyDom();
    this.#notify();
  }

  cycleMode() {
    const cycleMap = {
      system: 'light',
      light: 'dark',
      dark: 'system'
    };
    const next = cycleMap[this.mode] || 'system';
    this.setMode(next);
    return next;
  }

  #applyDom() {
    if (typeof document === 'undefined') return;

    const res = this.resolved;
    const root = document.documentElement;
    if (res === 'dark') {
      root.classList.add('dark');
      root.classList.remove('light');
      root.style.colorScheme = 'dark';
    } else {
      root.classList.remove('dark');
      root.classList.add('light');
      root.style.colorScheme = 'light';
    }
  }

  #notify() {
    for (const fn of this.#subscribers) {
      try {
        fn(this.mode);
      } catch {}
    }
  }

  // Svelte store contract interop
  subscribe = (fn) => {
    this.#subscribers.add(fn);
    fn(this.mode);
    return () => this.#subscribers.delete(fn);
  };
}

export const themeStore = new ThemeState();
