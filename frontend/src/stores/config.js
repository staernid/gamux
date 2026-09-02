import { writable } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { notificationStore } from './notifications.js';

function createConfigStore() {
  const config = writable(null);
  const uiScale = writable(parseInt(localStorage.getItem('gamux_ui_scale') || '18', 10));
  const isDirty = writable(false);

  function setScale(size) {
    const sz = Math.min(32, Math.max(12, parseInt(size, 10) || 18));
    document.documentElement.style.fontSize = `${sz}px`;
    localStorage.setItem('gamux_ui_scale', String(sz));
    uiScale.set(sz);
  }

  function setDirty(dirty) {
    isDirty.set(Boolean(dirty));
  }

  async function load() {
    try {
      const cfg = await AppAPI.getConfig();
      config.set(cfg);
      return cfg;
    } catch (err) {
      notificationStore.error('Config Load Error', err);
    }
  }

  async function save(newCfg) {
    try {
      await AppAPI.saveConfig(newCfg);
      config.set(newCfg);
      isDirty.set(false);
      notificationStore.success('Settings Saved', 'Configuration written to ~/.config/gamux/config.json');
      return true;
    } catch (err) {
      notificationStore.error('Failed to Save Settings', err);
      return false;
    }
  }

  return {
    subscribe: config.subscribe,
    uiScale: { subscribe: uiScale.subscribe, set: setScale },
    isDirty: { subscribe: isDirty.subscribe, set: setDirty },
    load,
    reload: load,
    save,
    setScale,
    setDirty
  };
}

export const configStore = createConfigStore();
