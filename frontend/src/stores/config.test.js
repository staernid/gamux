import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { configStore } from './config.js';
import { AppAPI } from '../api/app.js';

describe('configStore', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('loads and reloads configuration from AppAPI', async () => {
    const mockCfg = { default_runner: 'wine', mangohud: true };
    const getSpy = vi.spyOn(AppAPI, 'getConfig').mockResolvedValue(mockCfg);

    const loaded = await configStore.load();
    expect(getSpy).toHaveBeenCalledTimes(1);
    expect(loaded).toEqual(mockCfg);
    expect(get(configStore)).toEqual(mockCfg);

    // Verify reload alias works identically
    mockCfg.default_runner = 'umu';
    const reloaded = await configStore.reload();
    expect(getSpy).toHaveBeenCalledTimes(2);
    expect(reloaded).toEqual(mockCfg);
    expect(get(configStore)).toEqual(mockCfg);
  });

  it('saves configuration via AppAPI', async () => {
    const newCfg = { default_runner: 'umu', language: 'english' };
    const saveSpy = vi.spyOn(AppAPI, 'saveConfig').mockResolvedValue(true);

    const ok = await configStore.save(newCfg);
    expect(ok).toBe(true);
    expect(saveSpy).toHaveBeenCalledWith(newCfg);
    expect(get(configStore)).toEqual(newCfg);
  });

  it('sets UI scale within bounds and updates document fontSize and localStorage', () => {
    configStore.setScale(24);
    expect(get(configStore.uiScale)).toBe(24);
    expect(localStorage.getItem('gamux_ui_scale')).toBe('24');
    expect(document.documentElement.style.fontSize).toBe('24px');

    // Clamp tests
    configStore.setScale(50);
    expect(get(configStore.uiScale)).toBe(32);

    configStore.setScale(5);
    expect(get(configStore.uiScale)).toBe(12);
  });

  it('manages isDirty state and clears on successful save', async () => {
    expect(get(configStore.isDirty)).toBe(false);

    configStore.setDirty(true);
    expect(get(configStore.isDirty)).toBe(true);

    vi.spyOn(AppAPI, 'saveConfig').mockResolvedValue(true);
    await configStore.save({ default_runner: 'umu' });
    expect(get(configStore.isDirty)).toBe(false);
  });
});
