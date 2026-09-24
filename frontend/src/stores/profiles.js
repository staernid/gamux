import { writable, get } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { notificationStore } from './notifications.js';

function createProfilesStore() {
  const { subscribe, set, update } = writable({
    profiles: [],
    activeProfile: {
      id: 'main',
      index: 1,
      name: 'GamuxPlayer',
      steam_id: '76561197960287930',
      language: 'english',
      ip_country: 'US',
      avatar_url: '',
      is_main: true,
    },
    isLoading: false,
  });

  async function load() {
    update(s => ({ ...s, isLoading: true }));
    try {
      const [list, active] = await Promise.all([
        AppAPI.getProfiles(),
        AppAPI.getActiveProfile()
      ]);
      update(s => ({
        ...s,
        profiles: Array.isArray(list) && list.length > 0 ? list : s.profiles,
        activeProfile: active && active.id ? active : s.activeProfile,
        isLoading: false,
      }));
    } catch (err) {
      console.warn('Failed to load profiles:', err);
      update(s => ({ ...s, isLoading: false }));
    }
  }

  async function setActive(id) {
    try {
      await AppAPI.setActiveProfile(id);
      const state = get({ subscribe });
      const target = state.profiles.find(p => p.id === id);
      if (target) {
        update(s => ({ ...s, activeProfile: target }));
      }
      notificationStore.info('Profile Switched', `Active profile is now ${target?.name || id}`);
      return true;
    } catch (err) {
      notificationStore.error('Failed to Switch Profile', err);
      return false;
    }
  }

  async function create(name, steamID = '', language = 'english', country = 'US', avatarURL = '') {
    try {
      const created = await AppAPI.createProfile(name, steamID, language, country, avatarURL);
      if (created) {
        update(s => ({
          ...s,
          profiles: [...s.profiles, created],
          activeProfile: created
        }));
        await setActive(created.id);
        notificationStore.success('Profile Created', `Created profile ${created.name}`);
        return created;
      }
    } catch (err) {
      notificationStore.error('Failed to Create Profile', err);
      return null;
    }
  }

  async function save(p) {
    try {
      await AppAPI.saveProfile(p);
      update(s => ({
        ...s,
        profiles: s.profiles.map(item => item.id === p.id ? { ...p } : item),
        activeProfile: s.activeProfile?.id === p.id ? { ...p } : s.activeProfile
      }));
      notificationStore.success('Profile Saved', `Updated profile ${p.name}`);
      return true;
    } catch (err) {
      notificationStore.error('Failed to Save Profile', err);
      return false;
    }
  }

  async function remove(id) {
    try {
      await AppAPI.deleteProfile(id);
      update(s => {
        const filtered = s.profiles.filter(p => p.id !== id);
        let nextActive = s.activeProfile;
        if (s.activeProfile?.id === id) {
          nextActive = filtered.find(p => p.is_main) || filtered[0] || s.activeProfile;
        }
        return {
          ...s,
          profiles: filtered,
          activeProfile: nextActive
        };
      });
      notificationStore.info('Profile Deleted', 'Profile removed successfully');
      return true;
    } catch (err) {
      notificationStore.error('Failed to Delete Profile', err);
      return false;
    }
  }

  // Subscribe to backend events if runtime available
  if (typeof window !== 'undefined' && window.runtime?.EventsOn) {
    window.runtime.EventsOn('gamux:active-profile-changed', (p) => {
      if (p && p.id) {
        update(s => ({ ...s, activeProfile: p }));
      }
    });
    window.runtime.EventsOn('gamux:profiles-updated', (list) => {
      if (Array.isArray(list)) {
        update(s => ({ ...s, profiles: list }));
      }
    });
  }

  return {
    subscribe,
    load,
    setActive,
    create,
    save,
    delete: remove,
    // Direct store access helper
    setProfiles(list) {
      update(s => ({ ...s, profiles: list }));
    },
    setActiveProfile(p) {
      update(s => ({ ...s, activeProfile: p }));
    }
  };
}

export const profilesStore = createProfilesStore();
