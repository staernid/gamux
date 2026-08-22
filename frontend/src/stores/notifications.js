import { writable, derived } from 'svelte/store';

const STORAGE_KEY = 'gamux_notifications_v2';

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

  return {
    items: {
      subscribe: items.subscribe,
      add({ type = 'info', title, message, gameName, details }) {
        const notif = {
          id: 'notif_' + Date.now() + '_' + Math.random().toString(36).substr(2, 4),
          type, // 'success' | 'error' | 'warning' | 'info'
          title: title || 'Notification',
          message: message || '',
          gameName: gameName || '',
          details: details || '',
          timestamp: Date.now(),
          read: false,
        };

        items.update(list => {
          const updated = [notif, ...list];
          if (updated.length > 60) updated.pop();
          save(updated);
          return updated;
        });

        // Add to active toast list
        const toastId = notif.id;
        toasts.update(t => [...t, { ...notif, id: toastId }]);
        setTimeout(() => {
          toasts.update(t => t.filter(x => x.id !== toastId));
        }, 4500);
      },
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
    unreadCount: derived(items, $items => $items.filter(i => !i.read).length)
  };
}

export const notificationStore = createNotificationStore();
