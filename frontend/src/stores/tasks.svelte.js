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

export function normalizeTask(t) {
  if (!t) return t;
  const isStopped = isPausedStatus(t.status) || isTerminalStatus(t.status);
  const prog = extractTaskProgress(t);
  const speed = isStopped ? 0 : prog.speedBps;
  const eta = isStopped ? 0 : prog.etaSeconds;
  const gameName = t.game_name || t.gameName || t.title || 'Task';
  const isDl = isDownloadTask(t);
  const taskType = t.type || (isDl ? 'download' : (t.operation || 'operation'));
  const appId = String(t.app_id ?? t.appId ?? t.game_id ?? '');
  const gameDir = t.game_dir || t.gameDir || t.target_dir || t.targetDir || '';

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

export class TasksState {
  _items = $state([]);
  speedLimit = $state(0);
  isRefreshing = $state(false);

  #subscribers = new Set();

  #notifySubscribers() {
    for (const fn of this.#subscribers) {
      try {
        fn(this._items);
      } catch (e) {
        console.error('Error notifying tasks subscriber:', e);
      }
    }
  }

  get items() {
    return new Proxy(this._items, {
      get: (target, prop, receiver) => {
        if (prop === 'subscribe') {
          return (fn) => {
            this.#subscribers.add(fn);
            fn(this._items);
            return () => this.#subscribers.delete(fn);
          };
        }
        if (prop === 'set') {
          return (val) => {
            this.items = val;
          };
        }
        if (prop === 'update') {
          return (fn) => {
            this.items = fn(this._items);
          };
        }
        const val = Reflect.get(target, prop, receiver);
        return typeof val === 'function' ? val.bind(target) : val;
      }
    });
  }

  set items(val) {
    this._items = Array.isArray(val) ? val.map(normalizeTask) : [];
    this.#notifySubscribers();
  }

  // Derived reactive views (Svelte 5 runes)
  active = $derived(this._items.filter((t) => isActiveStatus(t.status)));
  queued = $derived(this._items.filter((t) => isQueuedStatus(t.status)));
  paused = $derived(this._items.filter((t) => isPausedStatus(t.status)));
  completed = $derived(this._items.filter((t) => isTerminalStatus(t.status)));

  activeDownloads = $derived(this._items.filter((t) => isDownloadTask(t) && isActiveStatus(t.status)));
  queuedDownloads = $derived(this._items.filter((t) => isDownloadTask(t) && isQueuedStatus(t.status)));
  pausedDownloads = $derived(this._items.filter((t) => isDownloadTask(t) && isPausedStatus(t.status)));
  completedDownloads = $derived(this._items.filter((t) => isDownloadTask(t) && isTerminalStatus(t.status)));

  activeOperations = $derived(this._items.filter((t) => !isDownloadTask(t) && isActiveStatus(t.status)));
  queuedOperations = $derived(this._items.filter((t) => !isDownloadTask(t) && isQueuedStatus(t.status)));

  activeTask = $derived(this._items.find((t) => isActiveStatus(t.status)) || null);

  totalSpeedBps = $derived(
    this._items
      .filter((t) => isDownloadTask(t) && isActiveStatus(t.status))
      .reduce((sum, t) => sum + (t.speedBps || t.progress?.bytesPerSec || 0), 0)
  );

  activeCount = $derived(
    this._items.filter((t) => isPendingOrActiveStatus(t.status)).length
  );

  activeTransfersCount = $derived(
    this._items.filter((t) => isDownloadTask(t) && isPendingOrActiveStatus(t.status)).length
  );

  handleTaskEvent(ev) {
    if (!ev || !ev.type) return;
    const taskID = ev.task_id || ev.taskId || (typeof ev.id === 'string' && (ev.id.startsWith('task_') || ev.id.startsWith('dl_') || ev.id.startsWith('transfer_') || ev.id.startsWith('op_')) ? ev.id : null);
    if (!taskID) {
      if (ev.stage || ev.details || ev.progress) {
        const active = this.items.find((t) => t.status === 'running' || t.status === 'downloading');
        if (active) {
          if (ev.stage) active.stage = ev.stage;
          if (ev.details) active.details = ev.details;
          if (typeof ev.progress?.percent === 'number') active.percent = Math.round(ev.progress.percent);
          this.items = [...this.items];
        }
      }
      return;
    }

    const now = Date.now();
    const isDl = ev.operation === 'download' || (ev.type && ev.type.startsWith('transfer:')) || (typeof taskID === 'string' && (taskID.startsWith('transfer_') || taskID.startsWith('dl_') || taskID.startsWith('download_')));

    const idx = this.items.findIndex((t) => t.id === taskID);
    let raw = idx >= 0 ? { ...this.items[idx] } : {
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
    const updated = [...this.items];
    if (idx >= 0) {
      updated[idx] = task;
    } else {
      updated.unshift(task);
    }
    this.items = updated;
  }

  handleTransferEvent(ev) {
    this.handleTaskEvent(ev);
  }

  handleEvent(ev) {
    this.handleTaskEvent(ev);
  }

  async refresh() {
    this.isRefreshing = true;
    try {
      const data = await AppAPI.listTasks();
      if (Array.isArray(data)) {
        this.items = data.map(normalizeTask);
      }
      const limit = await AppAPI.getTransferSpeedLimit();
      if (typeof limit === 'number') {
        this.speedLimit = limit;
      }
    } catch (err) {
      console.warn('Failed to refresh tasks:', err);
    } finally {
      this.isRefreshing = false;
    }
  }

  async start(opts) {
    try {
      const id = await AppAPI.startDownloadTransfer(opts);
      notificationStore.info('Transfer Enqueued', `Download started for ${opts.game_name || opts.app_id}`);
      await this.refresh();
      return id;
    } catch (err) {
      notificationStore.error('Failed to Start Download', err);
      throw err;
    }
  }

  async pause(id) {
    try {
      await AppAPI.pauseTask(id);
      this.items = this.items.map((t) =>
        t.id === id ? { ...t, status: 'paused', stage: 'Paused', speedBps: 0, bytesPerSec: 0 } : t
      );
    } catch (err) {
      notificationStore.error('Pause Failed', err);
    }
  }

  async resume(id) {
    try {
      await AppAPI.resumeTask(id);
      this.items = this.items.map((t) =>
        t.id === id ? { ...t, status: 'downloading', stage: 'Resuming...' } : t
      );
    } catch (err) {
      notificationStore.error('Resume Failed', err);
    }
  }

  async pauseAll() {
    const targets = this.items.filter((t) => isDownloadTask(t) && (t.status === 'running' || t.status === 'downloading'));
    await Promise.allSettled(targets.map((t) => this.pause(t.id)));
  }

  async resumeAll() {
    const targets = this.items.filter((t) => isDownloadTask(t) && t.status === 'paused');
    await Promise.allSettled(targets.map((t) => this.resume(t.id)));
  }

  async cancel(id) {
    try {
      await AppAPI.cancelTask(id);
    } catch (err) {
      console.warn('Backend cancelTask notice:', id, err);
    } finally {
      this.items = this.items.map((t) =>
        t.id === id ? { ...t, status: 'cancelled', stage: 'Cancelled', speedBps: 0, bytesPerSec: 0 } : t
      );
    }
  }

  cancelTask(id) {
    return this.cancel(id);
  }

  async remove(id) {
    try {
      await AppAPI.removeTask(id);
      this.items = this.items.filter((t) => t.id !== id);
    } catch (err) {
      notificationStore.error('Remove Failed', err);
    }
  }

  async reorder(ids) {
    if (!Array.isArray(ids) || ids.length === 0) return;
    const orderMap = new Map();
    ids.forEach((id, idx) => orderMap.set(id, idx));
    this.items = [...this.items].sort((a, b) => {
      const orderA = orderMap.has(a.id) ? orderMap.get(a.id) : 9999;
      const orderB = orderMap.has(b.id) ? orderMap.get(b.id) : 9999;
      return orderA - orderB;
    });
    try {
      await AppAPI.reorderTasks(ids);
      await this.refresh();
    } catch (err) {
      console.warn('Failed to reorder tasks:', err);
    }
  }

  async clearFinished() {
    try {
      await AppAPI.clearFinishedTasks();
      this.items = this.items.filter((t) => !isTerminalStatus(t.status));
    } catch (err) {
      console.warn('Failed to clear finished tasks:', err);
    }
  }

  clearHistory() {
    return this.clearFinished();
  }

  async setLimit(bps) {
    try {
      await AppAPI.setTransferSpeedLimit(bps);
      this.speedLimit = bps;
    } catch (err) {
      notificationStore.error('Speed Limit Error', err);
    }
  }

  updateStepProgress(data) {
    if (!data) return;
    const activeOp = this.items.find((t) => !isDownloadTask(t) && (t.status === 'running' || t.status === 'queued'));
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
      this.items = [...this.items];
    }
  }

  updateDownloadProgress(data) {
    if (!data) return;
    const appIdStr = data.app_id ? String(data.app_id) : (data.appId ? String(data.appId) : '');
    const dl = this.items.find((t) => isDownloadTask(t) && (
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
      this.items = [...this.items];
    }
  }

  enqueue({ type = 'operation', title, gameName, gameDir, run, onSuccess, onError }) {
    const id = generateId('task');
    this.handleTaskEvent({
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
          this.handleTaskEvent({
            type: 'task:completed',
            id,
            task_id: id,
            details: typeof res === 'string' ? res : 'Complete',
          });
          if (onSuccess) onSuccess(res);
        })
        .catch((err) => {
          this.handleTaskEvent({
            type: 'task:failed',
            id,
            task_id: id,
            error: err?.message || String(err),
          });
          if (onError) onError(err);
        });
    }
  }

  reset() {
    this.items = [];
  }
}

export const tasksStore = new TasksState();
