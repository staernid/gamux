import { writable, get } from 'svelte/store';
import { saveGameProfile, resetGameProfile, rollbackGame, refreshGame, showInspector } from './library.js';
import { notificationStore } from './notifications.js';
import { AppAPI } from '../api/app.js';
import * as envDraft from './drawer/envDraft.js';

// Re-export environment draft helpers for backward compatibility
export const computeEnvEntries = envDraft.computeEnvEntries;
export const generateNextCustomKey = envDraft.generateNextCustomKey;

export function createInitialDraft(game = null, globalCfg = {}) {
  if (!game) {
    return {
      gameDir: '',
      name: '',
      platform: '',
      appId: '',
      exePath: '',
      exeArgs: '',
      operatingMode: '',
      runner: '',
      winePrefix: '',
      protonPath: '',
      mangoHud: null,
      dllOverrides: '',
      customEnv: {},
      language: '',
      metadataProvider: '',
      metadataId: '',
      optNormalize: globalCfg.normalize ?? true,
      optSteamless: globalCfg.steamless ?? true,
      optAchievements: globalCfg.achievements ?? true,
    };
  }

  const candidates = (game.launchCandidates || []).filter(c => {
    const exe = (c.executable || '').toLowerCase();
    return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
  });

  const lp = game.launchProfile || null;
  let baseExe = lp?.exe_path || game.exePath || candidates[0]?.executable || '';
  if (baseExe.toLowerCase().includes('steamclient_loader') || baseExe.toLowerCase().includes('coldclientloader')) {
    baseExe = candidates[0]?.executable || '';
  }

  return {
    gameDir: game.gameDir || '',
    name: game.name || '',
    platform: game.platform || 'win64',
    appId: game.appId || '',
    exePath: baseExe,
    exeArgs: lp?.exe_args || game.exeArgs || candidates[0]?.arguments || '',
    operatingMode: lp?.operating_mode || lp?.operatingMode || game.operatingMode || '',
    runner: lp?.runner || lp?.Runner || game.runner || '',
    winePrefix: lp?.wine_prefix || lp?.winePrefix || game.winePrefix || '',
    protonPath: lp?.proton_path || lp?.protonPath || game.protonPath || '',
    dllOverrides: lp?.dll_overrides || lp?.dllOverrides || '',
    mangoHud: lp?.mangohud ?? lp?.mangoHud ?? null,
    customEnv: { ...(lp?.custom_env || lp?.customEnv || {}) },
    language: lp?.language || '',
    sourceOverride: lp?.source_override || lp?.sourceOverride || '',
    metadataProvider: lp?.metadata_provider || lp?.metadataProvider || game.metadataProvider || '',
    metadataId: lp?.metadata_id || lp?.metadataId || game.metadataId || '',
    optNormalize: globalCfg.normalize ?? true,
    optSteamless: globalCfg.steamless ?? true,
    optAchievements: globalCfg.achievements ?? true,
  };
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

      const launchProfile = {
        game_dir: gameDir,
        exe_path: (draft.exePath || '').trim(),
        exe_args: (draft.exeArgs || '').trim(),
        app_id: (draft.appId || game.appId || '').trim(),
        name: (draft.name || game.name || '').trim(),
        platform: game.platform || 'win64',
        operating_mode: (draft.operatingMode || '').trim(),
        language: (draft.language || '').trim(),
        runner: (draft.runner || '').trim(),
        wine_prefix: (draft.winePrefix || '').trim(),
        proton_path: (draft.protonPath || '').trim(),
        dll_overrides: (draft.dllOverrides || '').trim(),
        mangohud: draft.mangoHud,
        custom_env: draft.customEnv || {},
        playtime_seconds: game.playtimeSeconds || 0,
        last_played: game.lastPlayed || 0,
        custom_cover_url: game.customCoverUrl || '',
        custom_banner_url: game.customBannerUrl || '',
        custom_hero_url: game.customHeroUrl || '',
        custom_logo_url: game.customLogoUrl || '',
        custom_icon_url: game.customIconUrl || '',
        locked_artwork: !!game.lockedArtwork,
        source_override: (draft.sourceOverride || '').trim(),
        metadata_provider: (draft.metadataProvider ?? game.metadataProvider ?? '').trim(),
        metadata_id: (draft.metadataId ?? game.metadataId ?? '').trim(),
      };

      try {
        await saveGameProfile(gameDir, launchProfile);
        await refreshGame(gameDir);
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

    async resetDefaults(game) {
      if (!game?.gameDir) return false;
      try {
        await resetGameProfile(game.gameDir);
        await refreshGame(game.gameDir);
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
