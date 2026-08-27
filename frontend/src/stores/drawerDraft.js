import { writable, get } from 'svelte/store';
import { saveGameProfile, resetGameProfile, rollbackGame, refreshGame, showInspector } from './library.js';
import { notificationStore } from './notifications.js';
import { AppAPI } from '../api/app.js';

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
      mangoHud: null,
      wineFullscreenFSR: null,
      protonNVAPI: null,
      dxvkHud: '',
      dllOverrides: '',
      customEnv: {},
      mode: 'loader',
      language: '',
      optNormalize: globalCfg.normalize ?? true,
      optSteamless: globalCfg.steamless ?? true,
      optAchievements: globalCfg.achievements ?? true,
    };
  }

  const candidates = (game.launchCandidates || [])
    .filter(c => {
      const exe = (c.executable || '').toLowerCase();
      return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
    });

  const lp = game.launchProfile || null;
  let baseExe = lp?.exe_path || game.exePath || (candidates[0]?.executable || '');
  if (baseExe.toLowerCase().includes('steamclient_loader') || baseExe.toLowerCase().includes('coldclientloader')) {
    baseExe = candidates[0]?.executable || '';
  }

  return {
    gameDir: game.gameDir || '',
    name: game.name || '',
    platform: game.platform || '',
    appId: game.appId || '',
    exePath: baseExe,
    exeArgs: lp?.exe_args || game.exeArgs || candidates[0]?.arguments || '',
    operatingMode: lp?.operating_mode ?? '',
    runner: lp?.runner ?? '',
    winePrefix: lp?.wine_prefix || game.winePrefix || '',
    dllOverrides: lp?.dll_overrides || game.dllOverrides || '',
    mangoHud: (lp?.mangohud !== undefined && lp?.mangohud !== null) ? lp.mangohud : null,
    wineFullscreenFSR: (lp?.wine_fullscreen_fsr !== undefined && lp?.wine_fullscreen_fsr !== null) ? lp.wine_fullscreen_fsr : null,
    protonNVAPI: (lp?.proton_nvapi !== undefined && lp?.proton_nvapi !== null) ? lp.proton_nvapi : null,
    dxvkHud: lp?.dxvk_hud || '',
    customEnv: { ...(lp?.custom_env || {}) },
    mode: game.isPortable ? 'portable' : (game.isPatched ? 'loader' : 'loader'),
    language: game.currentLanguage || '',
    optNormalize: globalCfg.normalize ?? true,
    optSteamless: globalCfg.steamless ?? true,
    optAchievements: globalCfg.achievements ?? true,
  };
}

export function computeEnvEntries(customEnv = {}, globalEnv = {}) {
  const entries = [];
  const seen = new Set();

  // 1. Process global envs
  for (const [k, v] of Object.entries(globalEnv)) {
    seen.add(k);
    if (k in customEnv) {
      entries.push({
        key: k,
        value: customEnv[k],
        globalValue: v,
        isInherited: false,
        isOverriding: true
      });
    } else {
      entries.push({
        key: k,
        value: v,
        globalValue: v,
        isInherited: true,
        isOverriding: false
      });
    }
  }

  // 2. Process custom game envs not in global
  for (const [k, v] of Object.entries(customEnv)) {
    if (!seen.has(k)) {
      entries.push({
        key: k,
        value: v,
        globalValue: '',
        isInherited: false,
        isOverriding: false
      });
    }
  }

  return entries;
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
        exeArgs: candidate.arguments !== undefined ? candidate.arguments : state.exeArgs
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
      update(state => {
        let newKey = 'CUSTOM_VAR';
        let counter = 1;
        while (newKey in state.customEnv || (globalEnv && newKey in globalEnv)) {
          newKey = `CUSTOM_VAR_${counter++}`;
        }
        return {
          ...state,
          customEnv: { ...state.customEnv, [newKey]: '' }
        };
      });
    },

    updateCustomEnv(key, value) {
      update(state => ({
        ...state,
        customEnv: { ...state.customEnv, [key]: value }
      }));
    },

    overrideEnv(key, globalValue) {
      update(state => ({
        ...state,
        customEnv: { ...state.customEnv, [key]: globalValue }
      }));
    },

    revertEnv(key) {
      update(state => {
        const next = { ...state.customEnv };
        delete next[key];
        return { ...state, customEnv: next };
      });
    },

    deleteCustomEnv(key) {
      update(state => {
        const next = { ...state.customEnv };
        delete next[key];
        return { ...state, customEnv: next };
      });
    },

    async save(game) {
      if (!game?.gameDir) return false;
      const gameDir = game.gameDir;
      const draft = get(this);

      const launchProfile = {
        game_dir: gameDir,
        exe_path: (draft.exePath || '').trim(),
        exe_args: (draft.exeArgs || '').trim(),
        app_id: (draft.appId || '').trim(),
        name: (draft.name || game.name || '').trim(),
        platform: game.platform || '',
        operating_mode: draft.operatingMode,
        runner: draft.runner,
        wine_prefix: (draft.winePrefix || '').trim(),
        dll_overrides: (draft.dllOverrides || '').trim(),
        mangohud: draft.mangoHud,
        wine_fullscreen_fsr: draft.wineFullscreenFSR,
        proton_nvapi: draft.protonNVAPI,
        dxvk_hud: (draft.dxvkHud || '').trim(),
        custom_env: draft.customEnv,
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
    }
  };
}

export const drawerDraft = createDrawerDraftStore();
