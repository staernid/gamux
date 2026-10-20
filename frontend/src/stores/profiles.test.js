import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { profilesStore } from './profiles.js';
import { AppAPI } from '../api/app.js';

describe('profilesStore', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('loads profiles and active profile', async () => {
    const mockList = [
      { id: 'main', index: 1, name: 'Alice', is_main: true, steam_id: '123' },
      { id: 'p2', index: 2, name: 'Bob', is_main: false, steam_id: '456' },
    ];
    vi.spyOn(AppAPI, 'getProfiles').mockResolvedValue(mockList);
    vi.spyOn(AppAPI, 'getActiveProfile').mockResolvedValue(mockList[0]);

    await profilesStore.load();
    const state = get(profilesStore);
    expect(state.profiles).toHaveLength(2);
    expect(state.profiles[0].name).toBe('Alice');
    expect(state.profiles[0].isMain).toBe(true);
    expect(state.profiles[0].steamId).toBe('123');
    expect(state.profiles[0].is_main).toBeUndefined();
    expect(state.profiles[0].steam_id).toBeUndefined();
    expect(state.activeProfile.name).toBe('Alice');
    expect(state.activeProfile.isMain).toBe(true);
  });

  it('switches active profile', async () => {
    const setSpy = vi.spyOn(AppAPI, 'setActiveProfile').mockResolvedValue(true);
    profilesStore.setProfiles([
      { id: 'main', index: 1, name: 'Alice', isMain: true },
      { id: 'p2', index: 2, name: 'Bob', isMain: false },
    ]);

    const ok = await profilesStore.setActive('p2');
    expect(ok).toBe(true);
    expect(setSpy).toHaveBeenCalledWith('p2');
    expect(get(profilesStore).activeProfile.name).toBe('Bob');
  });

  it('creates and sets new profile', async () => {
    const newProf = { id: 'p3', index: 3, name: 'Charlie', is_main: false, steam_id: '789' };
    vi.spyOn(AppAPI, 'createProfile').mockResolvedValue(newProf);
    vi.spyOn(AppAPI, 'setActiveProfile').mockResolvedValue(true);

    const created = await profilesStore.create('Charlie');
    expect(created.name).toBe('Charlie');
    expect(created.isMain).toBe(false);
    expect(created.steamId).toBe('789');
    expect(created.is_main).toBeUndefined();
    expect(get(profilesStore).activeProfile.id).toBe('p3');
  });

  it('deletes secondary profile and falls back to main', async () => {
    vi.spyOn(AppAPI, 'deleteProfile').mockResolvedValue(true);
    profilesStore.setProfiles([
      { id: 'main', index: 1, name: 'Alice', isMain: true },
      { id: 'p2', index: 2, name: 'Bob', isMain: false },
    ]);
    profilesStore.setActiveProfile({ id: 'p2', index: 2, name: 'Bob', isMain: false });

    const ok = await profilesStore.delete('p2');
    expect(ok).toBe(true);
    expect(get(profilesStore).profiles.some(p => p.id === 'p2')).toBe(false);
    expect(get(profilesStore).activeProfile.id).toBe('main');
  });
});
