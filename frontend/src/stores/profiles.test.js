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
      { id: 'main', index: 1, name: 'Alice', is_main: true },
      { id: 'p2', index: 2, name: 'Bob', is_main: false },
    ];
    vi.spyOn(AppAPI, 'getProfiles').mockResolvedValue(mockList);
    vi.spyOn(AppAPI, 'getActiveProfile').mockResolvedValue(mockList[0]);

    await profilesStore.load();
    const state = get(profilesStore);
    expect(state.profiles).toEqual(mockList);
    expect(state.activeProfile.name).toBe('Alice');
  });

  it('switches active profile', async () => {
    const setSpy = vi.spyOn(AppAPI, 'setActiveProfile').mockResolvedValue(true);
    profilesStore.setProfiles([
      { id: 'main', index: 1, name: 'Alice', is_main: true },
      { id: 'p2', index: 2, name: 'Bob', is_main: false },
    ]);

    const ok = await profilesStore.setActive('p2');
    expect(ok).toBe(true);
    expect(setSpy).toHaveBeenCalledWith('p2');
    expect(get(profilesStore).activeProfile.name).toBe('Bob');
  });

  it('creates and sets new profile', async () => {
    const newProf = { id: 'p3', index: 3, name: 'Charlie', is_main: false };
    vi.spyOn(AppAPI, 'createProfile').mockResolvedValue(newProf);
    vi.spyOn(AppAPI, 'setActiveProfile').mockResolvedValue(true);

    const created = await profilesStore.create('Charlie');
    expect(created).toEqual(newProf);
    expect(get(profilesStore).activeProfile.id).toBe('p3');
  });

  it('deletes secondary profile and falls back to main', async () => {
    vi.spyOn(AppAPI, 'deleteProfile').mockResolvedValue(true);
    profilesStore.setProfiles([
      { id: 'main', index: 1, name: 'Alice', is_main: true },
      { id: 'p2', index: 2, name: 'Bob', is_main: false },
    ]);
    profilesStore.setActiveProfile({ id: 'p2', index: 2, name: 'Bob', is_main: false });

    const ok = await profilesStore.delete('p2');
    expect(ok).toBe(true);
    expect(get(profilesStore).profiles.some(p => p.id === 'p2')).toBe(false);
    expect(get(profilesStore).activeProfile.id).toBe('main');
  });
});
