import { writable, derived } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { notificationStore } from './notifications.js';

export function isDownloadTask(t) {
  if (!t) return false;
  if (t.type === 'download' || t.operation === 'download' || t.status === 'downloading') return true;
  if (typeof t.id === 'string' && (t.id.startsWith('download_') || t.id.startsWith('transfer_') || t.id.startsWith('dl_') || t.id.startsWith('tr_mock_') || t.id.startsWith('diag_tr_'))) return true;
  if (t.type && ['apply', 'verify', 'update', 'tool_update', 'normalize', 'rollback', 'workshop'].includes(t.type)) {
    return false;
  }
  if (t.operation && ['apply', 'verify', 'update', 'tool_update', 'normalize', 'rollback', 'workshop'].includes(t.operation)) {
    return false;
  }
  return Boolean(t.bytes_downloaded || (t.speed_bps && t.speed_bps > 0) || (t.depots && t.depots.length > 0));
}

function normalizeTask(t) {
  if (!t) return t;
  const prog = t.progress || {};
  const isStopped = t.status === 'paused' || t.status === 'completed' || t.status === 'failed' || t.status === 'cancelled';
  const pct = typeof t.percent === 'number' ? t.percent : (prog.percent ?? 0);
  const currBytes = t.current_bytes ?? prog.current_bytes ?? t.bytes_downloaded ?? 0;
  const totBytes = t.total_bytes ?? prog.total_bytes ?? 0;
  const speed = isStopped ? 0 : (typeof t.speed_bps === 'number' ? t.speed_bps : (prog.bytes_per_sec ?? 0));
  const eta = isStopped ? 0 : (prog.eta_seconds ?? t.eta_seconds ?? 0);
  const currItems = prog.current_items ?? t.current_items ?? 0;
  const totItems = prog.total_items ?? t.total_items ?? 0;
  const gameName = t.game_name || t.gameName || t.title || 'Task';
  const isDl = isDownloadTask(t);
  const taskType = t.type || (isDl ? 'download' : (t.operation || 'operation'));

  return {
    ...t,
    id: t.id || t.task_id,
    type: taskType,
    gameName,
    game_name: gameName,
    game_dir: t.game_dir || t.gameDir || '',
    gameDir: t.game_dir || t.gameDir || '',
    app_id: t.app_id || t.appID || '',
    appID: t.app_id || t.appID || '',
    stage: t.stage || 'Running',
    details: t.details || '',
    status: t.status,
    percent: pct,
    bytes_downloaded: currBytes,
    current_bytes: currBytes,
    currentBytes: currBytes,
    total_bytes: totBytes,
    totalBytes: totBytes,
    speed_bps: speed,
    bytesPerSec: speed,
    eta_seconds: eta,
    etaSeconds: eta,
    current_items: currItems,
    currentItems: currItems,
    total_items: totItems,
    totalItems: totItems,
    progress: {
      percent: pct,
      current_bytes: currBytes,
      total_bytes: totBytes,
      current_items: currItems,
      total_items: totItems,
      bytes_per_sec: speed,
      eta_seconds: eta,
    }
  };
}

function createTasksStore() {
  const items = writable([]);
  const speedLimit = writable(0);
  const isRefreshing = writable(false);

  // Global partitions (all tasks)
  const active = derived(items, ($items) =>
    $items.filter((t) => t.status === 'running' || t.status === 'downloading')
  );

  const queued = derived(items, ($items) =>
    $items.filter((t) => t.status === 'queued' || t.status === 'pending')
  );

  const paused = derived(items, ($items) =>
    $items.filter((t) => t.status === 'paused')
  );

  const completed = derived(items, ($items) =>
    $items.filter((t) => t.status === 'completed' || t.status === 'failed' || t.status === 'cancelled')
  );

  // Download / Transfer specific partitions
  const activeDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && (t.status === 'running' || t.status === 'downloading'))
  );

  const queuedDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && (t.status === 'queued' || t.status === 'pending'))
  );

  const pausedDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && t.status === 'paused')
  );

  const completedDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && (t.status === 'completed' || t.status === 'failed' || t.status === 'cancelled'))
  );

  // Operations specific partitions
  const activeOperations = derived(items, ($items) =>
    $items.filter((t) => !isDownloadTask(t) && (t.status === 'running' || t.status === 'downloading'))
  );

  const queuedOperations = derived(items, ($items) =>
    $items.filter((t) => !isDownloadTask(t) && (t.status === 'queued' || t.status === 'pending'))
  );

  const activeTask = derived(items, ($items) =>
    $items.find((t) => t.status === 'running' || t.status === 'downloading') || null
  );

  const totalSpeedBps = derived(items, ($items) =>
    $items
      .filter((t) => isDownloadTask(t) && (t.status === 'running' || t.status === 'downloading'))
      .reduce((sum, t) => {
        const bps = t.speed_bps || t.progress?.bytes_per_sec || 0;
        return sum + bps;
      }, 0)
  );

  const activeCount = derived(items, ($items) =>
    $items.filter((t) => t.status === 'running' || t.status === 'downloading' || t.status === 'queued' || t.status === 'pending').length
  );

  const activeTransfersCount = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && (t.status === 'running' || t.status === 'downloading' || t.status === 'queued' || t.status === 'pending')).length
  );

  function handleTaskEvent(ev) {
    if (!ev || !ev.type) return;
    const taskID = ev.task_id || ev.taskId || (typeof ev.id === 'string' && (ev.id.startsWith('task_') || ev.id.startsWith('dl_') || ev.id.startsWith('transfer_') || ev.id.startsWith('op_')) ? ev.id : null);
    if (!taskID) {
      if (ev.stage || ev.details || ev.progress) {
        items.update((list) => {
          const active = list.find((t) => t.status === 'running' || t.status === 'downloading');
          if (active) {
            if (ev.stage) active.stage = ev.stage;
            if (ev.details) active.details = ev.details;
            if (typeof ev.progress?.percent === 'number') active.percent = Math.round(ev.progress.percent);
            return [...list];
          }
          return list;
        });
      }
      return;
    }

    const now = Date.now();
    const isDl = ev.operation === 'download' || (ev.type && ev.type.startsWith('transfer:')) || (typeof taskID === 'string' && (taskID.startsWith('transfer_') || taskID.startsWith('dl_') || taskID.startsWith('download_')));

    items.update((list) => {
      const idx = list.findIndex((t) => t.id === taskID);
      let raw = idx >= 0 ? { ...list[idx] } : {
        id: taskID,
        type: isDl ? 'download' : (ev.operation || 'operation'),
        title: ev.game_name || (ev.operation ? `${ev.operation.toUpperCase()} ${ev.game_name || ''}` : 'Task'),
        game_name: ev.game_name || '',
        game_dir: ev.game_dir || '',
        app_id: ev.app_id || '',
        status: 'queued',
        stage: ev.stage || 'Queued',
        details: ev.details || '',
        percent: 0,
        bytes_downloaded: 0,
        total_bytes: 0,
        speed_bps: 0,
        eta_seconds: 0,
        created_at: ev.timestamp || now
      };

      if (isDl) raw.type = 'download';

      if (ev.progress) {
        if (typeof ev.progress.percent === 'number') raw.percent = ev.progress.percent;
        if (typeof ev.progress.current_bytes === 'number') {
          raw.current_bytes = ev.progress.current_bytes;
          raw.bytes_downloaded = ev.progress.current_bytes;
        }
        if (typeof ev.progress.total_bytes === 'number') raw.total_bytes = ev.progress.total_bytes;
        if (typeof ev.progress.bytes_per_sec === 'number') raw.speed_bps = ev.progress.bytes_per_sec;
        if (typeof ev.progress.eta_seconds === 'number') raw.eta_seconds = ev.progress.eta_seconds;
        if (typeof ev.progress.current_items === 'number') raw.current_items = ev.progress.current_items;
        if (typeof ev.progress.total_items === 'number') raw.total_items = ev.progress.total_items;
        raw.progress = { ...raw.progress, ...ev.progress };
      }

      switch (ev.type) {
        case 'task:started':
          raw.status = 'running';
          raw.started_at = ev.timestamp || now;
          break;

        case 'transfer:started':
          raw.status = 'downloading';
          raw.started_at = ev.timestamp || now;
          break;

        case 'task:stage_change':
        case 'task:progress':
          if (raw.status !== 'paused' && raw.status !== 'completed' && raw.status !== 'failed' && raw.status !== 'cancelled') {
            raw.status = (ev.stage === 'Completed' || ev.progress?.percent === 100) ? 'completed' : 'running';
          }
          break;

        case 'transfer:progress':
          if (raw.status !== 'paused' && raw.status !== 'completed' && raw.status !== 'failed' && raw.status !== 'cancelled') {
            raw.status = (ev.stage === 'Completed' || ev.progress?.percent === 100) ? 'completed' : 'downloading';
          }
          break;

        case 'task:paused':
        case 'transfer:paused':
          raw.status = 'paused';
          raw.speed_bps = 0;
          if (raw.progress) raw.progress.bytes_per_sec = 0;
          break;

        case 'task:resumed':
          raw.status = raw.stage === 'Queued' ? 'queued' : 'running';
          break;

        case 'transfer:resumed':
          raw.status = raw.stage === 'Queued' ? 'queued' : 'downloading';
          break;

        case 'task:completed':
        case 'transfer:completed':
          raw.status = 'completed';
          raw.percent = 100;
          raw.speed_bps = 0;
          raw.finished_at = ev.timestamp || now;
          notificationStore.items.add({
            type: 'success',
            title: `${raw.title || 'Task'} Complete`,
            message: raw.details || 'Completed successfully.',
            gameName: raw.game_name
          });
          break;

        case 'task:failed':
        case 'transfer:failed':
          raw.status = 'failed';
          raw.speed_bps = 0;
          raw.error = ev.error || ev.details || 'Task failed';
          raw.finished_at = ev.timestamp || now;
          notificationStore.items.add({
            type: 'error',
            title: `${raw.title || 'Task'} Failed`,
            message: raw.error,
            gameName: raw.game_name
          });
          break;

        case 'task:cancelled':
        case 'transfer:cancelled':
          raw.status = 'cancelled';
          raw.speed_bps = 0;
          raw.finished_at = ev.timestamp || now;
          break;
      }

      const task = normalizeTask(raw);
      if (idx >= 0) {
        list[idx] = task;
        return [...list];
      } else {
        return [task, ...list];
      }
    });
  }

  async function refresh() {
    isRefreshing.set(true);
    try {
      const data = await AppAPI.listTasks();
      if (Array.isArray(data)) {
        items.set(data.map(normalizeTask));
      }
      const limit = await AppAPI.getTransferSpeedLimit();
      if (typeof limit === 'number') {
        speedLimit.set(limit);
      }
    } catch (err) {
      console.warn('Failed to refresh tasks:', err);
    } finally {
      isRefreshing.set(false);
    }
  }

  async function start(opts) {
    try {
      const id = await AppAPI.startDownloadTransfer(opts);
      notificationStore.items.add({
        type: 'info',
        title: 'Transfer Enqueued',
        message: `Download started for ${opts.game_name || opts.app_id}`
      });
      await refresh();
      return id;
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Start Download',
        message: err?.message || String(err)
      });
      throw err;
    }
  }

  async function pause(id) {
    try {
      await AppAPI.pauseTask(id);
      items.update((list) =>
        list.map((t) => (t.id === id ? { ...t, status: 'paused', stage: 'Paused', speed_bps: 0 } : t))
      );
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Pause Failed',
        message: err?.message || String(err)
      });
    }
  }

  async function resume(id) {
    try {
      await AppAPI.resumeTask(id);
      items.update((list) =>
        list.map((t) => (t.id === id ? { ...t, status: 'downloading', stage: 'Resuming...' } : t))
      );
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Resume Failed',
        message: err?.message || String(err)
      });
    }
  }

  async function cancel(id) {
    try {
      await AppAPI.cancelTask(id);
    } catch (err) {
      console.warn('Backend cancelTask notice:', id, err);
    } finally {
      items.update((list) =>
        list.map((t) => (t.id === id ? { ...t, status: 'cancelled', stage: 'Cancelled', speed_bps: 0 } : t))
      );
    }
  }

  async function remove(id) {
    try {
      await AppAPI.removeTask(id);
      items.update((list) => list.filter((t) => t.id !== id));
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Remove Failed',
        message: err?.message || String(err)
      });
    }
  }

  async function reorder(ids) {
    try {
      await AppAPI.reorderTasks(ids);
      await refresh();
    } catch (err) {
      console.warn('Failed to reorder tasks:', err);
    }
  }

  async function clearFinished() {
    try {
      await AppAPI.clearFinishedTasks();
      items.update((list) => list.filter((t) => t.status === 'running' || t.status === 'queued' || t.status === 'paused'));
    } catch (err) {
      console.warn('Failed to clear finished tasks:', err);
    }
  }

  async function setLimit(bps) {
    try {
      await AppAPI.setTransferSpeedLimit(bps);
      speedLimit.set(bps);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Speed Limit Error',
        message: err?.message || String(err)
      });
    }
  }

  function updateStepProgress(data) {
    if (!data) return;
    items.update((list) => {
      // Find currently active running operation
      const activeOp = list.find((t) => !isDownloadTask(t) && (t.status === 'running' || t.status === 'queued'));
      if (activeOp) {
        if (data.title) activeOp.stage = data.title;
        if (data.details) activeOp.details = data.details;
        const pct = data.percent || (data.step && data.total ? (data.step / data.total) * 100 : 0);
        activeOp.percent = Math.round(pct);
        if (activeOp.progress) {
          activeOp.progress.percent = activeOp.percent;
          activeOp.progress.current_items = data.step;
          activeOp.progress.total_items = data.total;
        }
        return [...list];
      }
      return list;
    });
  }

  function updateDownloadProgress(data) {
    if (!data) return;
    items.update((list) => {
      const appIdStr = data.app_id ? String(data.app_id) : '';
      const dl = list.find((t) => isDownloadTask(t) && (
        (appIdStr && (t.app_id === appIdStr || t.appID === appIdStr)) ||
        t.status === 'downloading' ||
        t.status === 'running'
      ));
      if (dl) {
        if (data.item) dl.details = data.item;
        if (typeof data.percent === 'number') dl.percent = data.percent;
        if (dl.progress) {
          if (typeof data.percent === 'number') dl.progress.percent = data.percent;
          if (typeof data.current === 'number') dl.progress.current_items = data.current;
          if (typeof data.total === 'number') dl.progress.total_items = data.total;
        }
        return [...list];
      }
      return list;
    });
  }

  function enqueue({ type = 'operation', title, gameName, gameDir, run, onSuccess, onError }) {
    const id = 'task_' + Date.now() + '_' + Math.random().toString(36).substr(2, 4);
    handleTaskEvent({
      type: 'task:started',
      id,
      task_id: id,
      operation: type,
      game_name: gameName,
      game_dir: gameDir,
      stage: 'Queued',
      details: 'Executing...',
    });

    if (run) {
      Promise.resolve()
        .then(() => run({ id, title, gameName, gameDir }))
        .then((res) => {
          handleTaskEvent({
            type: 'task:completed',
            id,
            task_id: id,
            details: typeof res === 'string' ? res : 'Complete',
          });
          if (onSuccess) onSuccess(res);
        })
        .catch((err) => {
          handleTaskEvent({
            type: 'task:failed',
            id,
            task_id: id,
            error: err?.message || String(err),
          });
          if (onError) onError(err);
        });
    }
  }

  return {
    items,
    active,
    queued,
    paused,
    completed,
    activeDownloads,
    queuedDownloads,
    pausedDownloads,
    completedDownloads,
    activeOperations,
    queuedOperations,
    activeTask,
    totalSpeedBps,
    activeCount,
    activeTransfersCount,
    speedLimit,
    isRefreshing,
    reset() {
      items.set([]);
    },
    handleTaskEvent,
    handleTransferEvent: handleTaskEvent,
    handleEvent: handleTaskEvent,
    refresh,
    start,
    pause,
    resume,
    cancel,
    remove,
    reorder,
    clearFinished,
    clearHistory: clearFinished,
    cancelTask: cancel,
    setLimit,
    enqueue,
    updateStepProgress,
    updateDownloadProgress,
  };
}

export const tasksStore = createTasksStore();
