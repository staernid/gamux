import { writable, derived } from 'svelte/store';
import { generateId } from '../utils/format.js';

const STORAGE_KEY = 'gamux_notifications_v2';

function formatErrorMessage(err) {
  if (!err) return 'Unknown error';
  if (typeof err === 'string') return err;
  if (err.message) return err.message;
  return String(err);
}

function createNotificationStore() {
  let initialItems = [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) initialItems = JSON.parse(raw);
  } catch {
    initialItems = [];
  }

  const items = writable(initialItems);
  const toasts = writable([]);

  function save(arr) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(arr));
    } catch {}
  }

  function addNotif({
    type = 'info',
    title,
    message,
    gameName,
    details,
    position = 'bottom-right',
    duration = 4500,
    transient = false,
    icon = '',
    actionLabel = '',
    actionView = '',
    actionTaskId = '',
    onAction = null,
  }) {
    const notif = {
      id: generateId('notif'),
      type, // 'success' | 'error' | 'warning' | 'info' | 'osd'
      title: title || '',
      message: message || '',
      gameName: gameName || '',
      details: details || '',
      position, // 'top-center' | 'bottom-center' | 'bottom-right'
      duration,
      transient,
      icon,
      actionLabel,
      actionView,
      actionTaskId,
      onAction,
      timestamp: Date.now(),
      read: false,
    };

    if (!transient) {
      items.update(list => {
        const updated = [notif, ...list];
        if (updated.length > 60) updated.pop();
        save(updated);
        return updated;
      });
    }

    // Add to active toast list
    const toastId = notif.id;
    toasts.update(t => [...t, { ...notif, id: toastId }]);
    if (duration > 0) {
      setTimeout(() => {
        toasts.update(t => t.filter(x => x.id !== toastId));
      }, duration);
    }
  }

  const undoQueue = writable([]);

  function clearAllUndo() {
    undoQueue.update(list => {
      list.forEach(item => {
        if (item._interval) clearInterval(item._interval);
        if (item._timeout) clearTimeout(item._timeout);
      });
      return [];
    });
  }

  function pushUndo({ message, action, onCommit, duration = 6000 }) {
    const id = generateId('undo');
    const remaining = Math.ceil(duration / 1000);

    const item = {
      id,
      message,
      action,
      onCommit,
      remaining,
      _interval: null,
      _timeout: null
    };

    const intervalId = setInterval(() => {
      undoQueue.update(list => {
        return list.map(it => {
          if (it.id === id) {
            return { ...it, remaining: it.remaining - 1 };
          }
          return it;
        });
      });
    }, 1000);

    const timeoutId = setTimeout(() => {
      clearInterval(intervalId);
      undoQueue.update(list => list.filter(it => it.id !== id));
      if (typeof onCommit === 'function') onCommit();
    }, duration);

    item._interval = intervalId;
    item._timeout = timeoutId;

    undoQueue.update(list => [item, ...list]);
    return item;
  }

  function undo(message, action, onCommit, duration = 6000) {
    return pushUndo({ message, action, onCommit, duration });
  }

  function triggerUndo(id) {
    let triggered = null;
    undoQueue.update(list => {
      const found = list.find(it => it.id === id);
      if (found) {
        triggered = found;
        if (found._interval) clearInterval(found._interval);
        if (found._timeout) clearTimeout(found._timeout);
      }
      return list.filter(it => it.id !== id);
    });
    if (triggered && typeof triggered.action === 'function') {
      triggered.action();
    }
  }

  function dismissUndo(id) {
    let dismissed = null;
    undoQueue.update(list => {
      const found = list.find(it => it.id === id);
      if (found) {
        dismissed = found;
        if (found._interval) clearInterval(found._interval);
        if (found._timeout) clearTimeout(found._timeout);
      }
      return list.filter(it => it.id !== id);
    });
    if (dismissed && typeof dismissed.onCommit === 'function') {
      dismissed.onCommit();
    }
  }

  return {
    items: {
      subscribe: items.subscribe,
      add: addNotif,
      markAllRead() {
        items.update(list => {
          const updated = list.map(i => ({ ...i, read: true }));
          save(updated);
          return updated;
        });
      },
      clearAll() {
        items.set([]);
        save([]);
      }
    },
    toasts: {
      subscribe: toasts.subscribe,
      dismiss(id) {
        toasts.update(t => t.filter(x => x.id !== id));
      }
    },
    undoQueue: {
      subscribe: undoQueue.subscribe,
      clearAll: clearAllUndo
    },
    pushUndo,
    undo,
    triggerUndo,
    dismissUndo,
    unreadCount: derived(items, $items => $items.filter(i => !i.read).length),

    // Fast transient HUD on-screen display (OSD) pill
    osd(message, opts = {}) {
      addNotif({
        type: 'osd',
        message,
        title: opts.title || '',
        position: opts.position || 'top-center',
        duration: opts.duration || 1400,
        transient: true,
        icon: opts.icon || 'sort',
        ...opts,
      });
    },

    // Convenience 1-liner helpers
    error(title, err, gameName = '') {
      addNotif({ type: 'error', title, message: formatErrorMessage(err), gameName });
    },
    success(title, message, gameName = '') {
      addNotif({ type: 'success', title, message, gameName });
    },
    warn(title, message, gameName = '') {
      addNotif({ type: 'warning', title, message, gameName });
    },
    info(title, message, gameName = '') {
      addNotif({ type: 'info', title, message, gameName });
    }
  };
}

export const notificationStore = createNotificationStore();
