import { writable, derived } from 'svelte/store';

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
  }) {
    const notif = {
      id: 'notif_' + Date.now() + '_' + Math.random().toString(36).substr(2, 4),
      type, // 'success' | 'error' | 'warning' | 'info' | 'osd'
      title: title || '',
      message: message || '',
      gameName: gameName || '',
      details: details || '',
      position, // 'top-center' | 'bottom-center' | 'bottom-right'
      duration,
      transient,
      icon,
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
