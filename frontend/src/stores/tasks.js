import { writable, derived } from 'svelte/store';
import { AppAPI } from '../api/app.js';
import { generateId } from '../utils/format.js';
import { notificationStore } from './notifications.js';

export function isActiveStatus(status) {
  return status === 'running' || status === 'downloading';
}

export function isQueuedStatus(status) {
  return status === 'queued' || status === 'pending';
}

export function isPausedStatus(status) {
  return status === 'paused';
}

export function isTerminalStatus(status) {
  return status === 'completed' || status === 'failed' || status === 'cancelled';
}

export function isPendingOrActiveStatus(status) {
  return isActiveStatus(status) || isQueuedStatus(status);
}

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

export function extractTaskProgress(source) {
  if (!source) {
    return {
      percent: 0,
      currentBytes: 0,
      totalBytes: 0,
      speedBps: 0,
      etaSeconds: 0,
      currentItems: 0,
      totalItems: 0,
    };
  }

  const prog = source.progress || {};

  const percent = typeof source.percent === 'number'
    ? source.percent
    : (typeof prog.percent === 'number' ? prog.percent : 0);

  const currentBytes = typeof source.currentBytes === 'number'
    ? source.currentBytes
    : (source.current_bytes ?? prog.current_bytes ?? prog.currentBytes ?? source.bytes_downloaded ?? 0);

  const totalBytes = typeof source.totalBytes === 'number'
    ? source.totalBytes
    : (source.total_bytes ?? prog.total_bytes ?? prog.totalBytes ?? 0);

  const speedBps = typeof source.speedBps === 'number'
    ? source.speedBps
    : (typeof source.bytes_per_sec === 'number'
      ? source.bytes_per_sec
      : (typeof source.speed_bps === 'number'
        ? source.speed_bps
        : (typeof prog.bytes_per_sec === 'number'
          ? prog.bytes_per_sec
          : (typeof prog.bytesPerSec === 'number'
            ? prog.bytesPerSec
            : (typeof prog.speedBps === 'number'
              ? prog.speedBps
              : (source.bytesPerSec ?? 0))))));

  const etaSeconds = typeof source.etaSeconds === 'number'
    ? source.etaSeconds
    : (prog.eta_seconds ?? prog.etaSeconds ?? source.eta_seconds ?? 0);

  const currentItems = typeof source.currentItems === 'number'
    ? source.currentItems
    : (prog.current_items ?? prog.currentItems ?? source.current_items ?? 0);

  const totalItems = typeof source.totalItems === 'number'
    ? source.totalItems
    : (prog.total_items ?? prog.totalItems ?? source.total_items ?? 0);

  return {
    percent,
    currentBytes,
    totalBytes,
    speedBps,
    etaSeconds,
    currentItems,
    totalItems,
  };
}

function normalizeTask(t) {
  if (!t) return t;
  const isStopped = isPausedStatus(t.status) || isTerminalStatus(t.status);
  const prog = extractTaskProgress(t);
  const speed = isStopped ? 0 : prog.speedBps;
  const eta = isStopped ? 0 : prog.etaSeconds;
  const gameName = t.game_name || t.gameName || t.title || 'Task';
  const isDl = isDownloadTask(t);
  const taskType = t.type || (isDl ? 'download' : (t.operation || 'operation'));
  const appId = String(t.app_id ?? t.appId ?? '');
  const gameDir = t.game_dir || t.gameDir || '';

  return {
    id: t.id || t.task_id,
    type: taskType,
    title: t.title || gameName,
    gameName,
    gameDir,
    appId,
    stage: t.stage || 'Running',
    details: t.details || '',
    status: t.status,
    percent: prog.percent,
    currentBytes: prog.currentBytes,
    totalBytes: prog.totalBytes,
    speedBps: speed,
    bytesPerSec: speed,
    etaSeconds: eta,
    currentItems: prog.currentItems,
    totalItems: prog.totalItems,
    store: t.store || 'Steam',
    posterUrl: t.poster_url || t.posterUrl || '',
    depots: t.depots || [],
    capabilities: t.capabilities || 0,
    createdAt: t.created_at || t.createdAt || 0,
    finishedAt: t.finished_at || t.finishedAt || 0,
    error: t.error || '',
    progress: {
      percent: prog.percent,
      currentBytes: prog.currentBytes,
      totalBytes: prog.totalBytes,
      currentItems: prog.currentItems,
      totalItems: prog.totalItems,
      speedBps: speed,
      etaSeconds: eta,
    }
  };
}

function createTasksStore() {
  const items = writable([]);
  const speedLimit = writable(0);
  const isRefreshing = writable(false);

  // Global partitions (all tasks)
  const active = derived(items, ($items) =>
    $items.filter((t) => isActiveStatus(t.status))
  );

  const queued = derived(items, ($items) =>
    $items.filter((t) => isQueuedStatus(t.status))
  );

  const paused = derived(items, ($items) =>
    $items.filter((t) => isPausedStatus(t.status))
  );

  const completed = derived(items, ($items) =>
    $items.filter((t) => isTerminalStatus(t.status))
  );

  // Download / Transfer specific partitions
  const activeDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && isActiveStatus(t.status))
  );

  const queuedDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && isQueuedStatus(t.status))
  );

  const pausedDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && isPausedStatus(t.status))
  );

  const completedDownloads = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && isTerminalStatus(t.status))
  );

  // Operations specific partitions
  const activeOperations = derived(items, ($items) =>
    $items.filter((t) => !isDownloadTask(t) && isActiveStatus(t.status))
  );

  const queuedOperations = derived(items, ($items) =>
    $items.filter((t) => !isDownloadTask(t) && isQueuedStatus(t.status))
  );

  const activeTask = derived(items, ($items) =>
    $items.find((t) => isActiveStatus(t.status)) || null
  );

  const totalSpeedBps = derived(items, ($items) =>
    $items
      .filter((t) => isDownloadTask(t) && isActiveStatus(t.status))
      .reduce((sum, t) => {
        const bps = t.speedBps || t.progress?.bytesPerSec || 0;
        return sum + bps;
      }, 0)
  );

  const activeCount = derived(items, ($items) =>
    $items.filter((t) => isPendingOrActiveStatus(t.status)).length
  );

  const activeTransfersCount = derived(items, ($items) =>
    $items.filter((t) => isDownloadTask(t) && isPendingOrActiveStatus(t.status)).length
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
        title: ev.game_name || ev.gameName || (ev.operation ? `${ev.operation.toUpperCase()} ${ev.game_name || ev.gameName || ''}` : 'Task'),
        gameName: ev.game_name || ev.gameName || '',
        gameDir: ev.game_dir || ev.gameDir || '',
        appId: String(ev.app_id || ev.appId || ''),
        status: 'queued',
        stage: ev.stage || 'Queued',
        details: ev.details || '',
        percent: 0,
        currentBytes: 0,
        totalBytes: 0,
        speedBps: 0,
        etaSeconds: 0,
        createdAt: ev.timestamp || now
      };

      if (isDl) raw.type = 'download';

      if (ev.progress) {
        raw.progress = { ...raw.progress, ...ev.progress };
        const prog = extractTaskProgress(ev.progress);
        if (typeof ev.progress.percent === 'number' || typeof ev.progress.progress === 'number') raw.percent = prog.percent;
        if (prog.currentBytes > 0 || typeof ev.progress.current_bytes === 'number' || typeof ev.progress.currentBytes === 'number') raw.currentBytes = prog.currentBytes;
        if (prog.totalBytes > 0 || typeof ev.progress.total_bytes === 'number' || typeof ev.progress.totalBytes === 'number') raw.totalBytes = prog.totalBytes;
        if (prog.speedBps > 0 || typeof ev.progress.bytes_per_sec === 'number' || typeof ev.progress.speedBps === 'number' || typeof ev.progress.speed_bps === 'number') raw.speedBps = prog.speedBps;
        if (prog.etaSeconds > 0 || typeof ev.progress.eta_seconds === 'number' || typeof ev.progress.etaSeconds === 'number') raw.etaSeconds = prog.etaSeconds;
        if (prog.currentItems > 0 || typeof ev.progress.current_items === 'number' || typeof ev.progress.currentItems === 'number') raw.currentItems = prog.currentItems;
        if (prog.totalItems > 0 || typeof ev.progress.total_items === 'number' || typeof ev.progress.totalItems === 'number') raw.totalItems = prog.totalItems;
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
          raw.speedBps = 0;
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
          raw.speedBps = 0;
          raw.finished_at = ev.timestamp || now;
          notificationStore.success(`${raw.title || 'Task'} Complete`, raw.details || 'Completed successfully.', raw.gameName);
          break;

        case 'task:failed':
        case 'transfer:failed':
          raw.status = 'failed';
          raw.speedBps = 0;
          raw.error = ev.error || ev.details || 'Task failed';
          raw.finished_at = ev.timestamp || now;
          notificationStore.error(`${raw.title || 'Task'} Failed`, raw.error, raw.gameName);
          break;

        case 'task:cancelled':
        case 'transfer:cancelled':
          raw.status = 'cancelled';
          raw.speedBps = 0;
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
      notificationStore.info('Transfer Enqueued', `Download started for ${opts.game_name || opts.app_id}`);
      await refresh();
      return id;
    } catch (err) {
      notificationStore.error('Failed to Start Download', err);
      throw err;
    }
  }

  async function pause(id) {
    try {
      await AppAPI.pauseTask(id);
      items.update((list) =>
        list.map((t) => (t.id === id ? { ...t, status: 'paused', stage: 'Paused', speedBps: 0, bytesPerSec: 0 } : t))
      );
    } catch (err) {
      notificationStore.error('Pause Failed', err);
    }
  }

  async function resume(id) {
    try {
      await AppAPI.resumeTask(id);
      items.update((list) =>
        list.map((t) => (t.id === id ? { ...t, status: 'downloading', stage: 'Resuming...' } : t))
      );
    } catch (err) {
      notificationStore.error('Resume Failed', err);
    }
  }

  async function cancel(id) {
    try {
      await AppAPI.cancelTask(id);
    } catch (err) {
      console.warn('Backend cancelTask notice:', id, err);
    } finally {
      items.update((list) =>
        list.map((t) => (t.id === id ? { ...t, status: 'cancelled', stage: 'Cancelled', speedBps: 0, bytesPerSec: 0 } : t))
      );
    }
  }

  async function remove(id) {
    try {
      await AppAPI.removeTask(id);
      items.update((list) => list.filter((t) => t.id !== id));
    } catch (err) {
      notificationStore.error('Remove Failed', err);
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
      items.update((list) => list.filter((t) => !isTerminalStatus(t.status)));
    } catch (err) {
      console.warn('Failed to clear finished tasks:', err);
    }
  }

  async function setLimit(bps) {
    try {
      await AppAPI.setTransferSpeedLimit(bps);
      speedLimit.set(bps);
    } catch (err) {
      notificationStore.error('Speed Limit Error', err);
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
          activeOp.progress.currentItems = data.step;
          activeOp.progress.totalItems = data.total;
        }
        return [...list];
      }
      return list;
    });
  }

  function updateDownloadProgress(data) {
    if (!data) return;
    items.update((list) => {
      const appIdStr = data.app_id ? String(data.app_id) : (data.appId ? String(data.appId) : '');
      const dl = list.find((t) => isDownloadTask(t) && (
        (appIdStr && t.appId === appIdStr) ||
        t.status === 'downloading' ||
        t.status === 'running'
      ));
      if (dl) {
        if (data.item) dl.details = data.item;
        if (typeof data.percent === 'number') dl.percent = data.percent;
        if (dl.progress) {
          if (typeof data.percent === 'number') dl.progress.percent = data.percent;
          if (typeof data.current === 'number') dl.progress.currentItems = data.current;
          if (typeof data.total === 'number') dl.progress.totalItems = data.total;
        }
        return [...list];
      }
      return list;
    });
  }

  function enqueue({ type = 'operation', title, gameName, gameDir, run, onSuccess, onError }) {
    const id = generateId('task');
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
