import { AppAPI } from '../api/app.js';
import { generateId } from '../utils/format.js';
import { isSamePath } from '../utils/game.js';
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

export function isLiveStatus(status) {
  return isActiveStatus(status) || isQueuedStatus(status) || isPausedStatus(status);
}

export function taskMatchesGame(task, game) {
  if (!task || !game) return false;
  const tDir = task.gameDir || task.game_dir;
  const gDir = game.gameDir || game.game_dir;
  if (tDir && gDir && isSamePath(tDir, gDir)) {
    return true;
  }
  const tApp = String(task.appId || task.app_id || '').trim();
  const gApp = String(game.appId || game.app_id || '').trim();
  if (tApp && tApp !== '0' && gApp && gApp !== '0' && tApp === gApp) {
    return true;
  }
  return false;
}

export function isDownloadTask(t) {
  if (!t) return false;
  if (t.type === 'download' || t.status === 'downloading') return true;
  if (typeof t.id === 'string' && (t.id.startsWith('download_') || t.id.startsWith('transfer_') || t.id.startsWith('dl_') || t.id.startsWith('tr_mock_') || t.id.startsWith('diag_tr_'))) return true;
  if (t.type && ['apply', 'verify', 'update', 'tool_update', 'normalize', 'rollback', 'workshop'].includes(t.type)) {
    return false;
  }
  return false;
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

  const prog = source.progress || source;

  return {
    percent: typeof prog.percent === 'number' ? prog.percent : 0,
    currentBytes: prog.current_bytes ?? 0,
    totalBytes: prog.total_bytes ?? 0,
    speedBps: prog.bytes_per_sec ?? 0,
    etaSeconds: prog.eta_seconds ?? 0,
    currentItems: prog.current_items ?? 0,
    totalItems: prog.total_items ?? 0,
  };
}

export function normalizeTask(t) {
  if (!t) return t;
  const isStopped = isPausedStatus(t.status) || isTerminalStatus(t.status);
  const prog = extractTaskProgress(t);
  const speed = isStopped ? 0 : prog.speedBps;
  const eta = isStopped ? 0 : prog.etaSeconds;
  const gameName = t.game_name || t.title || 'Task';
  const appId = t.app_id ? String(t.app_id) : '';
  const gameDir = t.game_dir || '';

  return {
    id: t.id || '',
    type: t.type || (isDownloadTask(t) ? 'download' : 'operation'),
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
    capabilities: t.capabilities || 0,
    createdAt: t.created_at || 0,
    startedAt: t.started_at || 0,
    finishedAt: t.finished_at || 0,
    error: t.error || '',
    metadata: t.metadata || {},
    game: t.game || null,
    depots: t.depots || [],
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
    this._items = Array.isArray(val) ? val : [];
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
    const taskID = ev.task_id || (typeof ev.id === 'string' && (ev.id.startsWith('task_') || ev.id.startsWith('dl_') || ev.id.startsWith('transfer_') || ev.id.startsWith('op_')) ? ev.id : null);
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
    let item = idx >= 0 ? { ...this.items[idx] } : {
      id: taskID,
      type: isDl ? 'download' : (ev.operation || 'operation'),
      title: ev.game_name || 'Task',
      gameName: ev.game_name || '',
      gameDir: ev.game_dir || '',
      appId: String(ev.app_id || ''),
      status: 'queued',
      stage: ev.stage || 'Queued',
      details: ev.details || '',
      percent: 0,
      currentBytes: 0,
      totalBytes: 0,
      speedBps: 0,
      bytesPerSec: 0,
      etaSeconds: 0,
      currentItems: 0,
      totalItems: 0,
      capabilities: 0,
      createdAt: ev.timestamp || now,
      startedAt: 0,
      finishedAt: 0,
      error: '',
      progress: {
        percent: 0,
        currentBytes: 0,
        totalBytes: 0,
        currentItems: 0,
        totalItems: 0,
        speedBps: 0,
        etaSeconds: 0,
      }
    };

    if (isDl) item.type = 'download';

    if (ev.progress) {
      const prog = extractTaskProgress(ev.progress);
      if (typeof ev.progress.percent === 'number') item.percent = prog.percent;
      if (prog.currentBytes > 0) item.currentBytes = prog.currentBytes;
      if (prog.totalBytes > 0) item.totalBytes = prog.totalBytes;
      if (prog.speedBps > 0) {
        item.speedBps = prog.speedBps;
        item.bytesPerSec = prog.speedBps;
      }
      if (prog.etaSeconds > 0) item.etaSeconds = prog.etaSeconds;
      if (prog.currentItems > 0) item.currentItems = prog.currentItems;
      if (prog.totalItems > 0) item.totalItems = prog.totalItems;
      item.progress = {
        percent: item.percent,
        currentBytes: item.currentBytes,
        totalBytes: item.totalBytes,
        currentItems: item.currentItems,
        totalItems: item.totalItems,
        speedBps: item.speedBps,
        etaSeconds: item.etaSeconds,
      };
    }

    switch (ev.type) {
      case 'task:started':
        item.status = 'running';
        item.startedAt = ev.timestamp || now;
        break;

      case 'transfer:started':
        item.status = 'downloading';
        item.startedAt = ev.timestamp || now;
        break;

      case 'task:stage_change':
      case 'task:progress':
        if (item.status !== 'paused' && item.status !== 'completed' && item.status !== 'failed' && item.status !== 'cancelled') {
          item.status = (ev.stage === 'Completed' || ev.progress?.percent === 100) ? 'completed' : 'running';
        }
        break;

      case 'transfer:progress':
        if (item.status !== 'paused' && item.status !== 'completed' && item.status !== 'failed' && item.status !== 'cancelled') {
          item.status = (ev.stage === 'Completed' || ev.progress?.percent === 100) ? 'completed' : 'downloading';
        }
        break;

      case 'task:paused':
      case 'transfer:paused':
        item.status = 'paused';
        item.speedBps = 0;
        item.bytesPerSec = 0;
        if (item.progress) item.progress.speedBps = 0;
        break;

      case 'task:resumed':
        item.status = item.stage === 'Queued' ? 'queued' : 'running';
        break;

      case 'transfer:resumed':
        item.status = item.stage === 'Queued' ? 'queued' : 'downloading';
        break;

      case 'task:completed':
      case 'transfer:completed':
        item.status = 'completed';
        item.percent = 100;
        item.speedBps = 0;
        item.bytesPerSec = 0;
        if (item.progress) {
          item.progress.speedBps = 0;
          item.progress.percent = 100;
        }
        item.finishedAt = ev.timestamp || now;
        notificationStore.success(`${item.title || 'Task'} Complete`, item.details || 'Completed successfully.', item.gameName);
        break;

      case 'task:failed':
      case 'transfer:failed':
        item.status = 'failed';
        item.speedBps = 0;
        item.bytesPerSec = 0;
        if (item.progress) item.progress.speedBps = 0;
        item.error = ev.error || ev.details || 'Task failed';
        item.finishedAt = ev.timestamp || now;
        notificationStore.error(`${item.title || 'Task'} Failed`, item.error, item.gameName);
        break;

      case 'task:cancelled':
      case 'transfer:cancelled':
        item.status = 'cancelled';
        item.speedBps = 0;
        item.bytesPerSec = 0;
        if (item.progress) item.progress.speedBps = 0;
        item.finishedAt = ev.timestamp || now;
        break;
    }

    const task = item;
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
    const appIdStr = data.app_id ? String(data.app_id) : '';
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
