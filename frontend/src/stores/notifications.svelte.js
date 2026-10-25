import { generateId } from '../utils/format.js';

const STORAGE_KEY = 'gamux_notifications_v2';

function formatErrorMessage(err) {
  if (!err) return 'Unknown error';
  if (typeof err === 'string') return err;
  if (err.message) return err.message;
  return String(err);
}

export class NotificationsState {
  _items = $state([]);
  _toasts = $state([]);
  _undoQueue = $state([]);

  #itemSubscribers = new Set();
  #toastSubscribers = new Set();
  #undoSubscribers = new Set();
  #unreadSubscribers = new Set();
  #storeSubscribers = new Set();

  constructor() {
    let initialItems = [];
    try {
      if (typeof localStorage !== 'undefined') {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (raw) initialItems = JSON.parse(raw);
      }
    } catch {
      initialItems = [];
    }
    this._items = Array.isArray(initialItems) ? initialItems : [];
  }

  #save(arr) {
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(arr));
      }
    } catch {}
  }

  #notifyItems() {
    for (const fn of this.#itemSubscribers) {
      try {
        fn(this._items);
      } catch {}
    }
    const unread = this._items.filter(i => !i.read).length;
    for (const fn of this.#unreadSubscribers) {
      try {
        fn(unread);
      } catch {}
    }
    this.#notifyStore();
  }

  #notifyToasts() {
    for (const fn of this.#toastSubscribers) {
      try {
        fn(this._toasts);
      } catch {}
    }
    this.#notifyStore();
  }

  #notifyUndo() {
    for (const fn of this.#undoSubscribers) {
      try {
        fn(this._undoQueue);
      } catch {}
    }
    this.#notifyStore();
  }

  #notifyStore() {
    for (const fn of this.#storeSubscribers) {
      try {
        fn(this);
      } catch {}
    }
  }

  addNotif = ({
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
  }) => {
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
      const updated = [notif, ...this._items];
      if (updated.length > 60) updated.pop();
      this._items = updated;
      this.#save(updated);
      this.#notifyItems();
    }

    // Add to active toast list
    const toastId = notif.id;
    this._toasts = [...this._toasts, { ...notif, id: toastId }];
    this.#notifyToasts();

    if (duration > 0) {
      setTimeout(() => {
        this._toasts = this._toasts.filter(x => x.id !== toastId);
        this.#notifyToasts();
      }, duration);
    }
  };

  clearAllUndo = () => {
    this._undoQueue.forEach(item => {
      if (item._interval) clearInterval(item._interval);
      if (item._timeout) clearTimeout(item._timeout);
    });
    this._undoQueue = [];
    this.#notifyUndo();
  };

  pushUndo = ({ message, action, onCommit, duration = 6000 }) => {
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
      this._undoQueue = this._undoQueue.map(it => {
        if (it.id === id) {
          return { ...it, remaining: it.remaining - 1 };
        }
        return it;
      });
      this.#notifyUndo();
    }, 1000);

    const timeoutId = setTimeout(() => {
      clearInterval(intervalId);
      this._undoQueue = this._undoQueue.filter(it => it.id !== id);
      this.#notifyUndo();
      if (typeof onCommit === 'function') onCommit();
    }, duration);

    item._interval = intervalId;
    item._timeout = timeoutId;

    this._undoQueue = [item, ...this._undoQueue];
    this.#notifyUndo();
    return item;
  };

  undo = (message, action, onCommit, duration = 6000) => {
    return this.pushUndo({ message, action, onCommit, duration });
  };

  triggerUndo = (id) => {
    let triggered = null;
    const found = this._undoQueue.find(it => it.id === id);
    if (found) {
      triggered = found;
      if (found._interval) clearInterval(found._interval);
      if (found._timeout) clearTimeout(found._timeout);
    }
    this._undoQueue = this._undoQueue.filter(it => it.id !== id);
    this.#notifyUndo();
    if (triggered && typeof triggered.action === 'function') {
      triggered.action();
    }
  };

  dismissUndo = (id) => {
    let dismissed = null;
    const found = this._undoQueue.find(it => it.id === id);
    if (found) {
      dismissed = found;
      if (found._interval) clearInterval(found._interval);
      if (found._timeout) clearTimeout(found._timeout);
    }
    this._undoQueue = this._undoQueue.filter(it => it.id !== id);
    this.#notifyUndo();
    if (dismissed && typeof dismissed.onCommit === 'function') {
      dismissed.onCommit();
    }
  };

  items = {
    subscribe: (fn) => {
      this.#itemSubscribers.add(fn);
      fn(this._items);
      return () => this.#itemSubscribers.delete(fn);
    },
    add: (opts) => this.addNotif(opts),
    markAllRead: () => {
      const updated = this._items.map(i => ({ ...i, read: true }));
      this._items = updated;
      this.#save(updated);
      this.#notifyItems();
    },
    clearAll: () => {
      this._items = [];
      this.#save([]);
      this.#notifyItems();
    }
  };

  toasts = {
    subscribe: (fn) => {
      this.#toastSubscribers.add(fn);
      fn(this._toasts);
      return () => this.#toastSubscribers.delete(fn);
    },
    dismiss: (id) => {
      this._toasts = this._toasts.filter(x => x.id !== id);
      this.#notifyToasts();
    }
  };

  undoQueue = {
    subscribe: (fn) => {
      this.#undoSubscribers.add(fn);
      fn(this._undoQueue);
      return () => this.#undoSubscribers.delete(fn);
    },
    clearAll: () => this.clearAllUndo()
  };

  unreadCount = {
    subscribe: (fn) => {
      this.#unreadSubscribers.add(fn);
      fn(this._items.filter(i => !i.read).length);
      return () => this.#unreadSubscribers.delete(fn);
    }
  };

  get unread() {
    return this._items.filter(i => !i.read).length;
  }

  // Fast transient HUD on-screen display (OSD) pill
  osd = (message, opts = {}) => {
    this.addNotif({
      type: 'osd',
      message,
      title: opts.title || '',
      position: opts.position || 'top-center',
      duration: opts.duration || 1400,
      transient: true,
      icon: opts.icon || 'sort',
      ...opts,
    });
  };

  // Convenience 1-liner helpers
  error = (title, err, gameName = '') => {
    this.addNotif({ type: 'error', title, message: formatErrorMessage(err), gameName });
  };

  success = (title, message, gameName = '') => {
    this.addNotif({ type: 'success', title, message, gameName });
  };

  warn = (title, message, gameName = '') => {
    this.addNotif({ type: 'warning', title, message, gameName });
  };

  info = (title, message, gameName = '') => {
    this.addNotif({ type: 'info', title, message, gameName });
  };

  // Svelte store contract interop
  subscribe = (fn) => {
    this.#storeSubscribers.add(fn);
    fn(this);
    return () => this.#storeSubscribers.delete(fn);
  };
}

export const notificationStore = new NotificationsState();
