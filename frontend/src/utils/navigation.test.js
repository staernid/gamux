import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { openExternalUrl, copyToClipboard } from './navigation.js';
import { AppAPI } from '../api/app.js';
import { notificationStore } from '../stores/notifications.js';

describe('navigation utility', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('openExternalUrl', () => {
    it('returns false when given an empty or invalid URL', async () => {
      expect(await openExternalUrl('')).toBe(false);
      expect(await openExternalUrl(null)).toBe(false);
      expect(await openExternalUrl('   ')).toBe(false);
    });

    it('opens URL via AppAPI.openURL when available and notifies if title provided', async () => {
      const openSpy = vi.spyOn(AppAPI, 'openURL').mockResolvedValue();
      const notifySpy = vi.spyOn(notificationStore, 'info');

      const result = await openExternalUrl('https://example.com', { title: 'Example Site' });
      expect(result).toBe(true);
      expect(openSpy).toHaveBeenCalledWith('https://example.com');
      expect(notifySpy).toHaveBeenCalledWith('Opening Link', 'Launched Example Site in web browser');
    });

    it('falls back to window.open when AppAPI.openURL throws or is not available', async () => {
      vi.spyOn(AppAPI, 'openURL').mockImplementation(() => { throw new Error('API down'); });
      const windowOpenSpy = vi.spyOn(window, 'open').mockImplementation(() => {});
      const notifySpy = vi.spyOn(notificationStore, 'error');

      const result = await openExternalUrl('https://example.com');
      expect(result).toBe(false);
      expect(notifySpy).toHaveBeenCalledWith('Failed to open link', 'API down');
    });
  });

  describe('copyToClipboard', () => {
    it('returns false for empty text', async () => {
      expect(await copyToClipboard('')).toBe(false);
    });

    it('copies text via navigator.clipboard and notifies', async () => {
      const writeTextMock = vi.fn().mockResolvedValue();
      Object.assign(navigator, {
        clipboard: {
          writeText: writeTextMock
        }
      });
      const notifySpy = vi.spyOn(notificationStore, 'info');

      const result = await copyToClipboard('test-content', 'Link');
      expect(result).toBe(true);
      expect(writeTextMock).toHaveBeenCalledWith('test-content');
      expect(notifySpy).toHaveBeenCalledWith('Copied', 'Copied Link to clipboard');
    });
  });
});
