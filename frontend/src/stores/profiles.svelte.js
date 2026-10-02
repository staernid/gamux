import { AppAPI } from '../api/app.js';
import { notificationStore } from './notifications.js';

export class ProfilesState {
  profiles = $state([]);
  activeProfile = $state({
    id: 'main',
    index: 1,
    name: 'GamuxPlayer',
    steam_id: '76561197960287930',
    language: 'english',
    ip_country: 'US',
    avatar_url: '',
    is_main: true,
  });
  isLoading = $state(false);

  #subscribers = new Set();

  constructor() {
    if (typeof window !== 'undefined' && window.runtime?.EventsOn) {
      window.runtime.EventsOn('gamux:active-profile-changed', (p) => {
        if (p && p.id) {
          this.activeProfile = p;
          this.#notify();
        }
      });
      window.runtime.EventsOn('gamux:profiles-updated', (list) => {
        if (Array.isArray(list)) {
          this.profiles = list;
          this.#notify();
        }
      });
    }
  }

  async load() {
    this.isLoading = true;
    this.#notify();
    try {
      const [list, active] = await Promise.all([
        AppAPI.getProfiles(),
        AppAPI.getActiveProfile()
      ]);
      if (Array.isArray(list) && list.length > 0) {
        this.profiles = list;
      }
      if (active && active.id) {
        this.activeProfile = active;
      }
      this.isLoading = false;
      this.#notify();
    } catch (err) {
      console.warn('Failed to load profiles:', err);
      this.isLoading = false;
      this.#notify();
    }
  }

  async setActive(id) {
    try {
      await AppAPI.setActiveProfile(id);
      const target = this.profiles.find(p => p.id === id);
      if (target) {
        this.activeProfile = target;
      }
      this.#notify();
      notificationStore.info('Profile Switched', `Active profile is now ${target?.name || id}`);
      return true;
    } catch (err) {
      notificationStore.error('Failed to Switch Profile', err);
      return false;
    }
  }

  async create(name, steamID = '', language = 'english', country = 'US', avatarURL = '') {
    try {
      const created = await AppAPI.createProfile(name, steamID, language, country, avatarURL);
      if (created) {
        this.profiles = [...this.profiles, created];
        this.activeProfile = created;
        this.#notify();
        await this.setActive(created.id);
        notificationStore.success('Profile Created', `Created profile ${created.name}`);
        return created;
      }
    } catch (err) {
      notificationStore.error('Failed to Create Profile', err);
      return null;
    }
  }

  async save(p) {
    try {
      await AppAPI.saveProfile(p);
      this.profiles = this.profiles.map(item => item.id === p.id ? { ...p } : item);
      if (this.activeProfile?.id === p.id) {
        this.activeProfile = { ...p };
      }
      this.#notify();
      notificationStore.success('Profile Saved', `Updated profile ${p.name}`);
      return true;
    } catch (err) {
      notificationStore.error('Failed to Save Profile', err);
      return false;
    }
  }

  async delete(id) {
    try {
      await AppAPI.deleteProfile(id);
      const filtered = this.profiles.filter(p => p.id !== id);
      this.profiles = filtered;
      if (this.activeProfile?.id === id) {
        this.activeProfile = filtered.find(p => p.is_main) || filtered[0] || this.activeProfile;
      }
      this.#notify();
      notificationStore.info('Profile Deleted', 'Profile removed successfully');
      return true;
    } catch (err) {
      notificationStore.error('Failed to Delete Profile', err);
      return false;
    }
  }

  setProfiles(list) {
    this.profiles = list;
    this.#notify();
  }

  setActiveProfile(p) {
    this.activeProfile = p;
    this.#notify();
  }

  #notify() {
    const snapshot = {
      profiles: this.profiles,
      activeProfile: this.activeProfile,
      isLoading: this.isLoading,
    };
    for (const fn of this.#subscribers) {
      try {
        fn(snapshot);
      } catch {}
    }
  }

  // Svelte store contract interop
  subscribe = (fn) => {
    this.#subscribers.add(fn);
    fn({
      profiles: this.profiles,
      activeProfile: this.activeProfile,
      isLoading: this.isLoading,
    });
    return () => this.#subscribers.delete(fn);
  };
}

export const profilesStore = new ProfilesState();
