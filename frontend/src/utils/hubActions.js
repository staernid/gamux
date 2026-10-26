import { get } from 'svelte/store';
import {
  selectedGame,
  activeView,
  queueApplyGame,
  setGameRunning,
  refreshGame,
  libraryPath,
  libraryPaths,
  updateGameInStore,
  trashGame,
  ensureGameStub
} from '../stores/library.js';
import { tasksStore, isPendingOrActiveStatus } from '../stores/tasks.js';
import { modalStore } from '../stores/modal.js';
import { configStore } from '../stores/config.js';
import { notificationStore } from '../stores/notifications.js';
import { multiseatStore, getActiveSession } from '../stores/multiseat.js';
import { profilesStore } from '../stores/profiles.js';
import { drawerDraft } from '../stores/drawerDraft.js';
import { AppAPI } from '../api/app.js';

/**
 * Handles launching or downloading a game entity based on its installed state.
 */
export async function executePrimaryAction({
  game,
  activeGameTask,
  isGameRunning,
  userCfg,
  globalCfg,
  drawerDraftExePath,
  drawerDraftExeArgs,
}) {
  if (!game) return;

  if (activeGameTask) {
    if (activeGameTask.status === 'paused') {
      await tasksStore.resume(activeGameTask.id);
      return;
    }
    if (isPendingOrActiveStatus(activeGameTask.status)) {
      await tasksStore.pause(activeGameTask.id);
      return;
    }
  }

  if (isGameRunning) {
    try {
      const activeSession = getActiveSession(get(multiseatStore).activeSessions, game.gameDir);
      if (activeSession?.is_running) {
        await multiseatStore.stopSession(game.gameDir);
      } else {
        await AppAPI.stopGame(game.gameDir);
      }
      setGameRunning(game.gameDir, false);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Stop Game',
        message: err?.message || String(err)
      });
    }
    return;
  }

  const isInstalled = game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0));
  if (!isInstalled) {
    const appID = parseInt(game.appId, 10);
    if (!appID || isNaN(appID)) {
      notificationStore.items.add({
        type: 'error',
        title: 'Identifier Required',
        message: 'This title does not have a valid Steam AppID for depot downloading.'
      });
      return;
    }

    if (!configStore.skipInstallConfirm) {
      modalStore.open('install', { game });
      return;
    }

    let dest = game.gameDir;
    const paths = get(libraryPaths);
    const configuredLib = (paths && paths.length > 0) ? paths[0] : (get(libraryPath) || '~/Games');
    const lib = configuredLib;

    if (!dest || game.isVirtual) {
      try {
        const stubDir = await ensureGameStub(game, lib);
        if (stubDir) {
          dest = stubDir;
        }
      } catch {
        return;
      }
    }

    try {
      await tasksStore.start({
        app_id: appID,
        game_name: game.name,
        target_dir: dest,
        platform: game.platform || 'win64',
        language: (game.currentLanguage || configStore.current?.language || 'english').toLowerCase(),
        auto_apply: true,
        store_type: (game.store || 'steam').toLowerCase(),
        branch: game.branch || 'public',
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Download Failed',
        message: err?.message || String(err)
      });
    }
    return;
  }

  const isReady = game.isReady || game.isPatched || !!game.exePath;
  if (!isReady) {
    await queueApplyGame(game);
    return;
  }

  const targetExe = game.exePath || drawerDraftExePath || '';
  if (!targetExe) {
    notificationStore.warn('No Target Executable', 'Please configure or browse for a target executable in the General tab before launching.');
    return;
  }

  try {
    if (get(multiseatStore).isCoopEnabled) {
      await multiseatStore.launchSession(game, userCfg || {});
    } else {
      const isNative = (game.platform || '').toLowerCase().includes('linux') || game.runner === 'linux';
      const profId = profilesStore.activeProfile?.id || 'main';
      await AppAPI.launchGame({
        name: game.name,
        game_dir: game.gameDir,
        app_id: game.appId,
        platform: game.platform || 'win64',
        runner: isNative ? 'linux' : (game.runner || globalCfg?.default_runner || 'umu'),
        exe_path: targetExe,
        exe_args: game.exeArgs || drawerDraftExeArgs || '',
        profile_id: profId,
      });
      setGameRunning(game.gameDir, true, 0, 0, {
        profileId: profId,
        gameName: game.name,
        appId: game.appId,
      });
    }
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Launch Failed',
      message: err?.message || String(err)
    });
  }
}

/**
 * Prompts user confirmation to move game to trash.
 */
export function confirmTrashGame(game, isGameRunning) {
  if (!game?.gameDir || isGameRunning) return;
  const gameName = game.name || 'this game';
  const gameDir = game.gameDir;

  modalStore.open('confirm', {
    title: `Move "${gameName}" to Trash?`,
    message: `This will move the game files (${gameDir}) to your desktop Trash. Saves and emulation configuration will remain intact.`,
    confirmLabel: 'Move to Trash',
    cancelLabel: 'Keep Game',
    danger: true,
    onConfirm: async () => {
      try {
        await trashGame(gameDir);
        activeView.set('library');
        notificationStore.items.add({
          type: 'success',
          title: 'Moved to Trash',
          message: `${gameName} was moved to Trash.`
        });
      } catch (err) {
        notificationStore.error('Failed to Move Game to Trash', err);
      }
    }
  });
}

/**
 * Creates skeleton stub for game in the default library.
 */
export async function addGameStubToLibrary(game, libraryPathVal) {
  if (!game) return;
  const lib = libraryPathVal || '~/Games';
  try {
    const stub = await AppAPI.createGameStub(lib, game.appId, game.name, game.store || 'Steam', game.branch || 'public');
    if (stub) {
      const normalized = updateGameInStore(stub);
      if (normalized) {
        selectedGame.set(normalized);
      }
      notificationStore.items.add({
        type: 'success',
        title: 'Added to Library',
        message: `Saved skeleton metadata stub for ${game.name}`
      });
    }
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Failed to Add to Library',
      message: err?.message || String(err)
    });
  }
}

/**
 * Starts an update task for a game.
 */
export async function queueGameUpdate(game, branch) {
  if (!game?.gameDir) return;
  const branchToUpdate = branch || game.branch || 'public';
  if (game.appId && game.appId !== '0') {
    try {
      await tasksStore.start({
        app_id: parseInt(game.appId, 10),
        game_name: game.name,
        target_dir: game.gameDir,
        platform: 'win64',
        auto_apply: true,
        store_type: game.store || 'steam',
        branch: branchToUpdate,
      });
      notificationStore.items.add({
        type: 'info',
        title: 'Update Enqueued',
        message: `Update task started for ${game.name}${branchToUpdate !== 'public' ? ` (${branchToUpdate})` : ''}`,
        gameName: game.name
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Update Failed',
        message: err?.message || String(err)
      });
    }
  } else {
    try {
      await AppAPI.submitUpdate(game.gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Update Enqueued',
        message: `Update task started for ${game.name}`,
        gameName: game.name
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Update Failed',
        message: err?.message || String(err)
      });
    }
  }
}

/**
 * Links or unlinks external metadata provider for a title.
 */
export async function applyMetaLink(game, targetStore, targetId, globalCfg) {
  if (!game?.gameDir) return;
  try {
    const updated = await AppAPI.linkGameMetadata(game.gameDir, targetStore, targetId);
    notificationStore.items.add({
      type: 'success',
      title: 'Metadata Linked',
      message: targetStore === 'none' || targetStore === 'standalone' || !targetId || targetId === '0'
        ? `Unlinked metadata for ${game.name}`
        : `Linked ${game.name} to ${targetStore.toUpperCase()} (#${targetId})`
    });
    if (updated) {
      updateGameInStore(updated);
      drawerDraft.init(updated, globalCfg);
    }
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Linking Failed',
      message: err?.message || String(err)
    });
  }
}

/**
 * Restores auto-detected identity for a title.
 */
export async function unlinkMeta(game, globalCfg) {
  if (!game?.gameDir) return;
  try {
    const updated = await AppAPI.unlinkGameMetadata(game.gameDir);
    notificationStore.items.add({
      type: 'info',
      title: 'Metadata Reset',
      message: `Restored auto-detected identity for ${game.name}.`
    });
    if (updated) {
      updateGameInStore(updated);
      drawerDraft.init(updated, globalCfg);
    }
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Reset Failed',
      message: err?.message || String(err)
    });
  }
}

/**
 * Normalizes game directory folder name on disk.
 */
export async function normalizeDirectory(game) {
  if (!game?.gameDir) return;
  try {
    const newDir = await AppAPI.normalizeGameDirectory(game.gameDir);
    if (newDir && newDir !== game.gameDir) {
      await refreshGame(newDir);
    }
    notificationStore.items.add({
      type: 'success',
      title: 'Folder Normalized',
      message: `Normalized directory structure for ${game.name}`
    });
  } catch (err) {
    notificationStore.items.add({
      type: 'error',
      title: 'Normalization Failed',
      message: err?.message || String(err)
    });
  }
}
