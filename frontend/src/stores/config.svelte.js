import { AppAPI } from '../api/app.js';
import { notificationStore } from './notifications.js';

export class ConfigState {
  current = $state(null);
  uiScale = $state(18);
  skipInstallConfirm = $state(false);
  isDirty = $state(false);

  #subscribers = new Set();

  constructor() {
    if (typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem('gamux_ui_scale');
      if (stored) {
        const sz = parseInt(stored, 10);
        if (sz >= 12 && sz <= 32) {
          this.uiScale = sz;
        }
      }
      this.skipInstallConfirm = localStorage.getItem('gamux_skip_install_confirm') === 'true';
    }
  }

  setSkipInstallConfirm(skip) {
    this.skipInstallConfirm = Boolean(skip);
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('gamux_skip_install_confirm', String(Boolean(skip)));
    }
  }

  setScale(size) {
    const sz = Math.min(32, Math.max(12, parseInt(size, 10) || 18));
    if (typeof document !== 'undefined') {
      document.documentElement.style.fontSize = `${sz}px`;
    }
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('gamux_ui_scale', String(sz));
    }
    this.uiScale = sz;
  }

  setDirty(dirty) {
    this.isDirty = Boolean(dirty);
  }

  async load() {
    try {
      const cfg = await AppAPI.getConfig();
      this.current = cfg;
      this.#notify(cfg);
      return cfg;
    } catch (err) {
      notificationStore.error('Config Load Error', err);
    }
  }

  async reload() {
    return this.load();
  }

  async save(newCfg) {
    try {
      await AppAPI.saveConfig(newCfg);
      this.current = newCfg;
      this.isDirty = false;
      this.#notify(newCfg);
      notificationStore.success('Settings Saved', 'Configuration written to ~/.config/gamux/config.json');
      return true;
    } catch (err) {
      notificationStore.error('Failed to Save Settings', err);
      return false;
    }
  }

  #notify(val) {
    for (const fn of this.#subscribers) {
      try {
        fn(val);
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

export const configStore = new ConfigState();
