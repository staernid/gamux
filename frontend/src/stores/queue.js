import { writable, derived } from 'svelte/store';
import { notificationStore } from './notifications.js';

function createTaskQueueStore() {
  const queue = writable([]);
  const activeTask = writable(null);

  let currentQueue = [];
  let currentActive = null;

  queue.subscribe(val => { currentQueue = val; });
  activeTask.subscribe(val => { currentActive = val; });

  async function processNext() {
    if (currentQueue.length === 0) {
      activeTask.set(null);
      return;
    }

    const next = currentQueue[0];
    queue.update(q => q.slice(1));
    next.status = 'running';
    next.stepText = 'Starting...';
    activeTask.set(next);

    try {
      const result = await next.run(next);
      notificationStore.items.add({
        type: 'success',
        title: `${next.title} Complete`,
        message: typeof result === 'string' ? result : 'Operation finished successfully.',
        gameName: next.gameName
      });

      if (next.onSuccess) {
        await next.onSuccess(result);
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: `${next.title} Failed`,
        message: err?.message || String(err),
        gameName: next.gameName
      });

      if (next.onError) {
        next.onError(err);
      }
    } finally {
      activeTask.set(null);
      processNext();
    }
  }

  function enqueue({ type = 'operation', title, gameName, gameDir, run, onSuccess, onError }) {
    const task = {
      id: 'task_' + Date.now() + '_' + Math.random().toString(36).substr(2, 4),
      type,
      title,
      gameName: gameName || '',
      gameDir: gameDir || '',
      percent: 0,
      step: 1,
      totalSteps: 5,
      stepText: 'Queued in background...',
      details: 'Waiting in line...',
      run,
      onSuccess,
      onError,
      status: 'pending',
    };

    queue.update(q => [...q, task]);
    notificationStore.items.add({
      type: 'info',
      title: 'Task Queued',
      message: `${title} added to queue.`,
      gameName: gameName || ''
    });

    if (!currentActive) {
      processNext();
    }
  }

  function updateStepProgress(data) {
    if (!currentActive) return;
    activeTask.update(t => {
      if (!t) return null;
      const step = data.step || 1;
      const total = data.total || 5;
      const pct = Math.round((step / total) * 100);
      return {
        ...t,
        step,
        totalSteps: total,
        stepText: data.title || `Step ${step} of ${total}`,
        details: data.details || '',
        percent: pct
      };
    });
  }

  function updateDownloadProgress(data) {
    if (!currentActive) return;
    activeTask.update(t => {
      if (!t) return null;
      const pct = Math.round(data.percent || 0);
      return {
        ...t,
        percent: pct,
        stepText: `Downloading ${pct}% [${data.current || 0}/${data.total || 0}]`,
        details: data.item || 'Transferring chunks...'
      };
    });
  }

  return {
    queue: { subscribe: queue.subscribe },
    activeTask: { subscribe: activeTask.subscribe },
    totalCount: derived([queue, activeTask], ([$queue, $active]) => ($active ? 1 : 0) + $queue.length),
    enqueue,
    updateStepProgress,
    updateDownloadProgress
  };
}

export const taskQueue = createTaskQueueStore();
