import { AppAPI } from '../api/app.js';
import { notificationStore } from '../stores/notifications.js';

/**
 * Prompts the user with a native directory picker dialog.
 *
 * @param {string} [title='Select Directory'] Dialog title or prompt
 * @param {string} [defaultPath=''] Initial directory path
 * @returns {Promise<string|null>} The selected absolute path, or null if canceled
 */
export async function selectDirectoryDialog(title = 'Select Directory', defaultPath = '') {
  try {
    if (typeof AppAPI?.selectDirectory !== 'function') {
      throw new Error('Directory selection API is not available');
    }
    const selected = await AppAPI.selectDirectory(title, defaultPath || '');
    if (selected && typeof selected === 'string' && selected.trim()) {
      return selected.trim();
    }
    return null;
  } catch (err) {
    notificationStore.error(
      'Directory Selection Failed',
      err?.message || String(err)
    );
    return null;
  }
}
