import { describe, it, expect, vi, beforeEach } from 'vitest';
import { selectDirectoryDialog } from './filesystem.js';
import { AppAPI } from '../api/app.js';
import { notificationStore } from '../stores/notifications.js';

describe('filesystem utility', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls AppAPI.selectDirectory and returns the selected path', async () => {
    const spy = vi.spyOn(AppAPI, 'selectDirectory').mockResolvedValue('/home/user/games');

    const res = await selectDirectoryDialog('Choose Path', '/home/user');
    expect(spy).toHaveBeenCalledWith('Choose Path', '/home/user');
    expect(res).toBe('/home/user/games');
  });

  it('returns null when dialog is canceled (empty response)', async () => {
    vi.spyOn(AppAPI, 'selectDirectory').mockResolvedValue('');

    const res = await selectDirectoryDialog('Choose Path');
    expect(res).toBeNull();
  });

  it('catches errors, triggers notification, and returns null', async () => {
    vi.spyOn(AppAPI, 'selectDirectory').mockRejectedValue(new Error('Permission denied'));
    const notifySpy = vi.spyOn(notificationStore, 'error');

    const res = await selectDirectoryDialog('Choose Path');
    expect(res).toBeNull();
    expect(notifySpy).toHaveBeenCalledWith('Directory Selection Failed', 'Permission denied');
  });
});
