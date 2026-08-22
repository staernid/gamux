import { writable } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { notificationStore } from './notifications.js';

function createConfigStore() {
  const config = writable(null);
  const uiScale = writable(parseInt(localStorage.getItem('gamux_ui_scale') || '17', 10));

  function setScale(size) {
    const sz = Math.min(32, Math.max(12, parseInt(size, 10) || 17));
    document.documentElement.style.fontSize = `${sz}px`;
    localStorage.setItem('gamux_ui_scale', String(sz));
    uiScale.set(sz);
  }

  async function load() {
    try {
      const cfg = await AppAPI.getConfig();
      config.set(cfg);
      return cfg;
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Config Load Error',
        message: err?.message || String(err)
      });
    }
  }

  async function save(newCfg) {
    try {
      await AppAPI.saveConfig(newCfg);
      config.set(newCfg);
      notificationStore.items.add({
        type: 'success',
        title: 'Settings Saved',
        message: 'Configuration written to ~/.config/gamux/config.json'
      });
      return true;
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Save Settings',
        message: err?.message || String(err)
      });
      return false;
    }
  }

  return {
    subscribe: config.subscribe,
    uiScale: { subscribe: uiScale.subscribe, set: setScale },
    load,
    save,
    setScale
  };
}

export const configStore = createConfigStore();
