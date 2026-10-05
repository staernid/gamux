import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { ThemeState, themeStore } from './theme.svelte.js';

describe('ThemeState', () => {
  let addEventListenerSpy;
  let listeners = [];

  beforeEach(() => {
    localStorage.clear();
    listeners = [];
    addEventListenerSpy = vi.fn((event, handler) => {
      listeners.push({ event, handler });
    });

    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: true, // Default to dark system theme in mock
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: addEventListenerSpy,
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    document.documentElement.className = '';
    document.documentElement.style.colorScheme = '';
  });

  it('defaults to system mode and resolves to system dark if matchMedia is dark', () => {
    const store = new ThemeState();
    expect(store.mode).toBe('system');
    expect(store.resolved).toBe('dark');
  });

  it('reads stored preference from localStorage', () => {
    localStorage.setItem('gamux_theme', 'light');
    const store = new ThemeState();
    expect(store.mode).toBe('light');
    expect(store.resolved).toBe('light');
  });

  it('sets mode to dark, light, and system correctly', () => {
    const store = new ThemeState();
    store.init();

    store.setMode('light');
    expect(store.mode).toBe('light');
    expect(store.resolved).toBe('light');
    expect(localStorage.getItem('gamux_theme')).toBe('light');
    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe('light');

    store.setMode('dark');
    expect(store.mode).toBe('dark');
    expect(store.resolved).toBe('dark');
    expect(localStorage.getItem('gamux_theme')).toBe('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.documentElement.classList.contains('light')).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe('dark');

    // Invalid mode is ignored
    store.setMode('invalid-mode');
    expect(store.mode).toBe('dark');
  });

  it('cycles mode properly: system -> light -> dark -> system', () => {
    const store = new ThemeState();
    store.setMode('system');

    expect(store.cycleMode()).toBe('light');
    expect(store.mode).toBe('light');

    expect(store.cycleMode()).toBe('dark');
    expect(store.mode).toBe('dark');

    expect(store.cycleMode()).toBe('system');
    expect(store.mode).toBe('system');
  });

  it('updates DOM and resolved value when OS theme changes in system mode', () => {
    const store = new ThemeState();
    store.init();
    expect(store.mode).toBe('system');
    expect(store.resolved).toBe('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);

    // Simulate OS switching to light
    const changeHandler = listeners.find((l) => l.event === 'change')?.handler;
    expect(changeHandler).toBeDefined();

    changeHandler({ matches: false });
    expect(store.systemIsDark).toBe(false);
    expect(store.resolved).toBe('light');
    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe('light');

    // Test direct handleSystemThemeChange event
    store.handleSystemThemeChange(true);
    expect(store.systemIsDark).toBe(true);
    expect(store.resolved).toBe('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('notifies subscribers on change', () => {
    const store = new ThemeState();
    const subscriber = vi.fn();
    const unsubscribe = store.subscribe(subscriber);

    expect(subscriber).toHaveBeenCalledWith('system');

    store.setMode('light');
    expect(subscriber).toHaveBeenCalledWith('light');

    unsubscribe();
    store.setMode('dark');
    expect(subscriber).toHaveBeenCalledTimes(2);
  });
});
