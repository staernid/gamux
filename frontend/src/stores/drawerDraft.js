import { writable, get } from 'svelte/store';
import { saveGameProfile, resetGameProfile, rollbackGame, refreshGame, showInspector } from './library.js';
import { notificationStore } from './notifications.js';
import { AppAPI } from '../api/app.js';
import { filterLaunchCandidates, getExecutablePath } from '../utils/game.js';
import * as envDraft from './drawer/envDraft.js';

// Re-export environment draft helpers for backward compatibility
export const computeEnvEntries = envDraft.computeEnvEntries;
export const generateNextCustomKey = envDraft.generateNextCustomKey;

// Bidirectional casing adapters for Go LaunchProfile struct tags <-> JS camelCase
const SPECIAL_SNAKE_TO_CAMEL = {
  game_dir: 'gameDir',
  exe_path: 'exePath',
  exe_args: 'exeArgs',
  app_id: 'appId',
  operating_mode: 'operatingMode',
  wine_prefix: 'winePrefix',
  proton_path: 'protonPath',
  dll_overrides: 'dllOverrides',
  mangohud: 'mangoHud',
  mango_hud: 'mangoHud',
  disable_overlay: 'disableOverlay',
  offline_mode: 'offlineMode',
  custom_env: 'customEnv',
  source_override: 'sourceOverride',
  metadata_provider: 'metadataProvider',
  metadata_id: 'metadataId',
  gamescope_width: 'gamescopeWidth',
  gamescope_height: 'gamescopeHeight',
  gamescope_fps: 'gamescopeFps',
  working_dir: 'workingDir',
  playtime_seconds: 'playtimeSeconds',
  last_played: 'lastPlayed',
  custom_cover_url: 'customCoverUrl',
  custom_banner_url: 'customBannerUrl',
  custom_hero_url: 'customHeroUrl',
  custom_logo_url: 'customLogoUrl',
  custom_icon_url: 'customIconUrl',
  locked_artwork: 'lockedArtwork',
  branch_password: 'branchPassword',
  inhibit_sleep: 'inhibitSleep',
  prefer_sdl: 'preferSdl',
  enable_wayland: 'enableWayland',
};

const SPECIAL_CAMEL_TO_SNAKE = {
  gameDir: 'game_dir',
  exePath: 'exe_path',
  exeArgs: 'exe_args',
  appId: 'app_id',
  operatingMode: 'operating_mode',
  winePrefix: 'wine_prefix',
  protonPath: 'proton_path',
  dllOverrides: 'dll_overrides',
  mangoHud: 'mangohud',
  disableOverlay: 'disable_overlay',
  offlineMode: 'offline_mode',
  customEnv: 'custom_env',
  sourceOverride: 'source_override',
  metadataProvider: 'metadata_provider',
  metadataId: 'metadata_id',
  gamescopeWidth: 'gamescope_width',
  gamescopeHeight: 'gamescope_height',
  gamescopeFps: 'gamescope_fps',
  workingDir: 'working_dir',
  playtimeSeconds: 'playtime_seconds',
  lastPlayed: 'last_played',
  customCoverUrl: 'custom_cover_url',
  customBannerUrl: 'custom_banner_url',
  customHeroUrl: 'custom_hero_url',
  customLogoUrl: 'custom_logo_url',
  customIconUrl: 'custom_icon_url',
  lockedArtwork: 'locked_artwork',
  branchPassword: 'branch_password',
  inhibitSleep: 'inhibit_sleep',
  preferSdl: 'prefer_sdl',
  enableWayland: 'enable_wayland',
};

export function snakeToCamelKey(key) {
  if (SPECIAL_SNAKE_TO_CAMEL[key]) return SPECIAL_SNAKE_TO_CAMEL[key];
  return key.replace(/_([a-z0-9])/g, (_, letter) => letter.toUpperCase());
}

export function camelToSnakeKey(key) {
  if (SPECIAL_CAMEL_TO_SNAKE[key]) return SPECIAL_CAMEL_TO_SNAKE[key];
  return key.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
}

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
  const lp = game.launchProfile || game.launch_profile || {};

  const draft = {
    ...baseDraft,
    gameDir: game.gameDir || '',
    name: lp.name || '',
    platform: lp.platform || game.platform || 'win64',
    appId: lp.app_id || lp.appId || '',
    exePath: baseExe,
    exeArgs: lp.exe_args || lp.exeArgs || '',
    branch: game.branch || lp.branch || 'public',
    branchPassword: game.branchPassword || lp.branch_password || lp.branchPassword || '',
  };

  // Dynamically map all profile fields from lp to camelCase in draft
  for (const [rawKey, val] of Object.entries(lp)) {
    if (val === undefined || val === null) continue;
    const camelKey = snakeToCamelKey(rawKey);
    if (camelKey === 'customEnv' && val && typeof val === 'object') {
      draft.customEnv = { ...val };
    } else {
      draft[camelKey] = val;
    }
  }

  // Handle explicit nulls for boolean overrides (mangohud, disable_overlay, offline_mode)
  if (lp.mangohud !== undefined || lp.mangoHud !== undefined) {
    draft.mangoHud = lp.mangohud ?? lp.mangoHud;
  }
  if (lp.disable_overlay !== undefined || lp.disableOverlay !== undefined) {
    draft.disableOverlay = lp.disable_overlay ?? lp.disableOverlay;
  }
  if (lp.offline_mode !== undefined || lp.offlineMode !== undefined) {
    draft.offlineMode = lp.offline_mode ?? lp.offlineMode;
  }
  if (lp.inhibit_sleep !== undefined || lp.inhibitSleep !== undefined) {
    draft.inhibitSleep = lp.inhibit_sleep ?? lp.inhibitSleep;
  }
  if (lp.prefer_sdl !== undefined || lp.preferSdl !== undefined) {
    draft.preferSdl = lp.prefer_sdl ?? lp.preferSdl;
  }
  if (lp.enable_wayland !== undefined || lp.enableWayland !== undefined) {
    draft.enableWayland = lp.enable_wayland ?? lp.enableWayland;
  }

  if (!draft.exePath && baseExe) {
    draft.exePath = baseExe;
  }

  draft.gameDir = game.gameDir || draft.gameDir || '';
  draft.platform = draft.platform || game.platform || 'win64';

  return draft;
}

function createDrawerDraftStore() {
  const { subscribe, set, update } = writable(createInitialDraft());

  return {
    subscribe,
    set,
    update,

    init(game, globalCfg = {}) {
      set(createInitialDraft(game, globalCfg));
    },

    setField(key, value) {
      update(state => ({ ...state, [key]: value }));
    },

    selectCandidate(candidate) {
      if (!candidate) return;
      update(state => ({
        ...state,
        exePath: candidate.executable || state.exePath,
        exeArgs: candidate.arguments !== undefined ? candidate.arguments : state.exeArgs,
      }));
    },

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
    },

    addCustomEnv(globalEnv = {}) {
      update(state => ({
        ...state,
        customEnv: envDraft.addCustomEnv(state.customEnv, globalEnv),
      }));
    },

    updateCustomEnv(key, value) {
      update(state => ({
        ...state,
        customEnv: envDraft.updateCustomEnv(state.customEnv, key, value),
      }));
    },

    overrideEnv(key, globalValue) {
      update(state => ({
        ...state,
        customEnv: envDraft.overrideEnv(state.customEnv, key, globalValue),
      }));
    },

    revertEnv(key) {
      update(state => ({
        ...state,
        customEnv: envDraft.revertEnv(state.customEnv, key),
      }));
    },

    deleteCustomEnv(key) {
      update(state => ({
        ...state,
        customEnv: envDraft.deleteCustomEnv(state.customEnv, key),
      }));
    },

    async save(game) {
      if (!game?.gameDir) return false;
      const gameDir = game.gameDir;
      const draft = get(this);

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
      launchProfile.playtime_seconds = game.playtimeSeconds ?? game.playtime_seconds ?? 0;
      launchProfile.last_played = game.lastPlayed ?? game.last_played ?? 0;
      launchProfile.custom_cover_url = game.customCoverUrl ?? game.custom_cover_url ?? '';
      launchProfile.custom_banner_url = game.customBannerUrl ?? game.custom_banner_url ?? '';
      launchProfile.custom_hero_url = game.customHeroUrl ?? game.custom_hero_url ?? '';
      launchProfile.custom_logo_url = game.customLogoUrl ?? game.custom_logo_url ?? '';
      launchProfile.custom_icon_url = game.customIconUrl ?? game.custom_icon_url ?? '';
      launchProfile.locked_artwork = Boolean(game.lockedArtwork ?? game.locked_artwork);

      try {
        const updated = await saveGameProfile(gameDir, launchProfile);
        if (updated) {
          this.init(updated);
        }
        showInspector.set(false);
        notificationStore.success('Settings Saved', `Updated configuration for ${game.name || 'game'}`);
        return true;
      } catch (err) {
        notificationStore.error('Save Failed', err);
        return false;
      }
    },

    discard() {
      showInspector.set(false);
    },

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
    },

    rollback(game) {
      if (!game?.gameDir) return;
      rollbackGame(game.gameDir, false);
      showInspector.set(false);
    },
  };
}

export const drawerDraft = createDrawerDraftStore();
