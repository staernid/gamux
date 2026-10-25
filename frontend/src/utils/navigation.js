import { AppAPI } from '../api/app.js';
import { notificationStore } from '../stores/notifications.js';

/**
 * Opens an external URL in the user's default web browser.
 * Delegates to the Go backend (`App.OpenURL`); browser previews use the mock in api/app.js.
 *
 * @param {string} url The HTTP/HTTPS or external URI to open
 * @param {Object} [options]
 * @param {string} [options.title] Optional display label for feedback toasts
 * @param {boolean} [options.notify] Whether to show success toast (defaults to true if title provided)
 * @returns {Promise<boolean>} True if opened successfully, false otherwise
 */
export async function openExternalUrl(url, options = {}) {
  if (!url || typeof url !== 'string') return false;
  const trimmedUrl = url.trim();
  if (!trimmedUrl) return false;

  const { title = '', notify = Boolean(title) } = options;

  try {
    await AppAPI.openURL(trimmedUrl);

    if (notify) {
      notificationStore.info(
        'Opening Link',
        title ? `Launched ${title} in web browser` : `Launched ${trimmedUrl}`
      );
    }
    return true;
  } catch (err) {
    notificationStore.error(
      'Failed to open link',
      err?.message || String(err)
    );
    return false;
  }
}

/**
 * Copies plain text to the system clipboard with notification feedback.
 *
 * @param {string} text Text to copy
 * @param {string} [label='Text'] Label for toast message
 * @returns {Promise<boolean>}
 */
export async function copyToClipboard(text, label = 'URL') {
  if (!text) return false;
  try {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      notificationStore.info('Copied', `Copied ${label} to clipboard`);
      return true;
    }
    throw new Error('Clipboard API unavailable');
  } catch (err) {
    notificationStore.error('Copy Failed', `Could not copy ${label} to clipboard: ${err?.message || String(err)}`);
    return false;
  }
}
