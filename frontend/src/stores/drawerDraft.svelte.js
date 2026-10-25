import { saveGameProfile, resetGameProfile, rollbackGame, showInspector } from './library.js';
import { notificationStore } from './notifications.js';
import { AppAPI } from '../api/app.js';
import {
  camelToSnakeKey,
  filterLaunchCandidates,
  getExecutablePath,
  normalizeLaunchProfile,
  snakeToCamelKey,
} from '../utils/game.js';
import * as envDraft from './drawer/envDraft.js';

// Re-export environment & casing helpers for backward compatibility
export const computeEnvEntries = envDraft.computeEnvEntries;
export const generateNextCustomKey = envDraft.generateNextCustomKey;
export { snakeToCamelKey, camelToSnakeKey };

export function createInitialDraft(game = null, globalCfg = {}) {
  const baseDraft = {
    gameDir: '',
    name: '',
    platform: 'win64',
    appId: '',
    exePath: '',
    exeArgs: '',
    operatingMode: '',
    runner: '',
    winePrefix: '',
    protonPath: '',
    mangoHud: null,
    disableOverlay: null,
    offlineMode: null,
    inhibitSleep: null,
    preferSdl: null,
    enableWayland: null,
    dllOverrides: '',
    customEnv: {},
    language: '',
    sourceOverride: '',
    metadataProvider: '',
    metadataId: '',
    gamescope: null,
    gamescopeWidth: 0,
    gamescopeHeight: 0,
    gamescopeFps: 0,
    workingDir: '',
    branch: 'public',
    branchPassword: '',
    optNormalize: globalCfg.normalize ?? true,
    optSteamless: globalCfg.steamless ?? true,
    optAchievements: globalCfg.achievements ?? true,
  };

  if (!game) {
    baseDraft.platform = '';
    return baseDraft;
  }

  const candidates = filterLaunchCandidates(game.launchCandidates || []);
  const baseExe = getExecutablePath(game, candidates);
  const lp = normalizeLaunchProfile(game.launchProfile || game.launch_profile) || {};

  const draft = {
    ...baseDraft,
    gameDir: game.gameDir || '',
    name: lp.name || '',
    platform: lp.platform || game.platform || 'win64',
    appId: lp.appId || '',
    exePath: baseExe,
    exeArgs: lp.exeArgs || '',
    branch: game.branch || lp.branch || 'public',
    branchPassword: game.branchPassword || lp.branchPassword || '',
  };

  // Dynamically map all profile fields from normalized lp to camelCase in draft
  for (const [camelKey, val] of Object.entries(lp)) {
    if (val === undefined || val === null) continue;
    if (camelKey === 'customEnv' && val && typeof val === 'object') {
      draft.customEnv = { ...val };
    } else {
      draft[camelKey] = val;
    }
  }

  // Handle explicit nulls for boolean overrides
  if (lp.mangoHud !== undefined) draft.mangoHud = lp.mangoHud;
  if (lp.disableOverlay !== undefined) draft.disableOverlay = lp.disableOverlay;
  if (lp.offlineMode !== undefined) draft.offlineMode = lp.offlineMode;
  if (lp.inhibitSleep !== undefined) draft.inhibitSleep = lp.inhibitSleep;
  if (lp.preferSdl !== undefined) draft.preferSdl = lp.preferSdl;
  if (lp.enableWayland !== undefined) draft.enableWayland = lp.enableWayland;

  if (!draft.exePath && baseExe) {
    draft.exePath = baseExe;
  }

  draft.gameDir = game.gameDir || draft.gameDir || '';
  draft.platform = draft.platform || game.platform || 'win64';

  return draft;
}

export class DrawerDraftState {
  current = $state(createInitialDraft());
  dirty = $state(false);

  #subscribers = new Set();
  #dirtySubscribers = new Set();

  #notify() {
    for (const fn of this.#subscribers) {
      try { fn(this.current); } catch {}
    }
  }

  #notifyDirty() {
    for (const fn of this.#dirtySubscribers) {
      try { fn(this.dirty); } catch {}
    }
  }

  init(game, globalCfg = {}) {
    this.current = createInitialDraft(game, globalCfg);
    this.dirty = false;
    this.#notifyDirty();
    this.#notify();
  }

  markSaved() {
    this.dirty = false;
    this.#notifyDirty();
  }

  syncIdentity(game) {
    if (!game) return;
    const lp = normalizeLaunchProfile(game.launchProfile || game.launch_profile) || {};
    this.current.appId = lp.appId || game.appId || this.current.appId;
    this.current.metadataProvider = lp.metadataProvider || game.metadataProvider || this.current.metadataProvider;
    this.current.metadataId = lp.metadataId || game.metadataId || this.current.metadataId;
    this.current.name = lp.name || game.name || this.current.name;
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  setField(key, value) {
    this.current[key] = value;
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  selectCandidate(candidate) {
    if (!candidate) return;
    if (candidate.executable) {
      this.current.exePath = candidate.executable;
    }
    if (candidate.arguments !== undefined) {
      this.current.exeArgs = candidate.arguments;
    }
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  async browseExecutable(gameDir) {
    if (!gameDir) return;
    try {
      const selected = await AppAPI.selectFile('Select Game Executable', 'Executables', '*');
      if (selected) {
        let rel = selected;
        if (selected.startsWith(gameDir)) {
          rel = selected.slice(gameDir.length).replace(/^[/\\]+/, '');
        }
        this.setField('exePath', rel);
      }
    } catch (err) {
      notificationStore.error('File Selection Failed', err);
    }
  }

  addCustomEnv(globalEnv = {}) {
    this.current.customEnv = envDraft.addCustomEnv(this.current.customEnv, globalEnv);
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  updateCustomEnv(key, value) {
    this.current.customEnv = envDraft.updateCustomEnv(this.current.customEnv, key, value);
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  overrideEnv(key, globalValue) {
    this.current.customEnv = envDraft.overrideEnv(this.current.customEnv, key, globalValue);
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  revertEnv(key) {
    this.current.customEnv = envDraft.revertEnv(this.current.customEnv, key);
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  deleteCustomEnv(key) {
    this.current.customEnv = envDraft.deleteCustomEnv(this.current.customEnv, key);
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  }

  async save(game) {
    if (!game?.gameDir) return false;
    const gameDir = game.gameDir;
    const draft = this.current;

    const IGNORED_SAVE_KEYS = new Set([
      'optNormalize',
      'optSteamless',
      'optAchievements',
      'gameDir'
    ]);

    const launchProfile = {
      game_dir: gameDir,
    };

    for (const [k, v] of Object.entries(draft)) {
      if (IGNORED_SAVE_KEYS.has(k)) continue;
      const snakeKey = camelToSnakeKey(k);
      if (typeof v === 'string') {
        launchProfile[snakeKey] = v.trim();
      } else if (v !== undefined) {
        launchProfile[snakeKey] = v;
      }
    }

    // Preserve game-level metadata and timestamps
    launchProfile.playtime_seconds = game.playtimeSeconds ?? 0;
    launchProfile.last_played = game.lastPlayed ?? 0;
    launchProfile.custom_cover_url = game.customCoverUrl ?? '';
    launchProfile.custom_banner_url = game.customBannerUrl ?? '';
    launchProfile.custom_hero_url = game.customHeroUrl ?? '';
    launchProfile.custom_logo_url = game.customLogoUrl ?? '';
    launchProfile.custom_icon_url = game.customIconUrl ?? '';
    launchProfile.locked_artwork = Boolean(game.lockedArtwork);

    const lp = normalizeLaunchProfile(game.launchProfile || game.launch_profile) || {};
    if (!launchProfile.metadata_provider && (lp.metadataProvider || game.metadataProvider)) {
      launchProfile.metadata_provider = lp.metadataProvider || game.metadataProvider || '';
    }
    if (!launchProfile.metadata_id && (lp.metadataId || game.metadataId)) {
      launchProfile.metadata_id = lp.metadataId || game.metadataId || '';
    }
    if (!launchProfile.store_app_id && lp.storeAppId) {
      launchProfile.store_app_id = lp.storeAppId || '';
    }

    try {
      const updated = await saveGameProfile(gameDir, launchProfile);
      if (updated) {
        this.init(updated);
      } else {
        this.markSaved();
      }
      showInspector.set(false);
      notificationStore.success('Settings Saved', `Updated configuration for ${game.name || 'game'}`);
      return true;
    } catch (err) {
      notificationStore.error('Save Failed', err);
      return false;
    }
  }

  discard() {
    showInspector.set(false);
  }

  async resetDefaults(game, globalCfg = {}) {
    if (!game?.gameDir) return false;
    try {
      const updated = await resetGameProfile(game.gameDir);
      this.init(updated || { ...game, launchProfile: null }, globalCfg);
      notificationStore.info('Defaults Restored', `Restored default configuration for ${game.name || 'game'}`);
      return true;
    } catch (err) {
      notificationStore.error('Reset Failed', err);
      return false;
    }
  }

  rollback(game) {
    if (!game?.gameDir) return;
    rollbackGame(game.gameDir, false);
    showInspector.set(false);
  }

  // Svelte store contract interop
  subscribe = (fn) => {
    this.#subscribers.add(fn);
    fn(this.current);
    return () => this.#subscribers.delete(fn);
  };

  set = (val) => {
    this.current = val;
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  };

  update = (fn) => {
    this.current = fn(this.current);
    this.dirty = true;
    this.#notifyDirty();
    this.#notify();
  };

  get isDirty() {
    return {
      subscribe: (fn) => {
        this.#dirtySubscribers.add(fn);
        fn(this.dirty);
        return () => this.#dirtySubscribers.delete(fn);
      }
    };
  }
}

export function createAutosave(draftStore, saveFn, delay = 400) {
  let timer = null;

  function cancel() {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
  }

  function isStoreDirty() {
    if (typeof draftStore.dirty === 'boolean') {
      return draftStore.dirty;
    }
    let d = false;
    if (draftStore.isDirty?.subscribe) {
      draftStore.isDirty.subscribe(v => { d = v; })();
    }
    return d;
  }

  function trigger() {
    cancel();
    if (!isStoreDirty()) return;
    timer = setTimeout(async () => {
      timer = null;
      if (isStoreDirty()) {
        await saveFn();
      }
    }, delay);
  }

  const unsubscribe = draftStore.subscribe(() => {
    if (isStoreDirty()) {
      trigger();
    } else {
      cancel();
    }
  });

  return {
    trigger,
    cancel,
    destroy() {
      cancel();
      unsubscribe();
    }
  };
}

export const drawerDraft = new DrawerDraftState();
