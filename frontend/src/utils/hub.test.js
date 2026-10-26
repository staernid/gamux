import { describe, it, expect } from 'vitest';
import { cycleHubTab, findActiveGameTask, findVerifyTask, isSteamTitle, PRIMARY_HUB_TABS } from './hub.js';

describe('hub utils', () => {
  describe('cycleHubTab', () => {
    it('cycles forward through tabs and wraps around', () => {
      expect(cycleHubTab('overview', 1)).toBe('config');
      expect(cycleHubTab('saves', 1)).toBe('overview');
    });

    it('cycles backward through tabs and wraps around', () => {
      expect(cycleHubTab('overview', -1)).toBe('saves');
      expect(cycleHubTab('config', -1)).toBe('overview');
    });

    it('defaults to first tab if current tab is unknown', () => {
      expect(cycleHubTab('unknown', 1)).toBe(PRIMARY_HUB_TABS[0]);
    });
  });

  describe('findActiveGameTask', () => {
    const game = { appId: '1091500', name: 'Cyberpunk 2077', gameDir: '/games/cp2077' };

    it('finds live task for game', () => {
      const tasks = [
        { id: '1', status: 'completed', game_name: 'Cyberpunk 2077' },
        { id: '2', status: 'running', game_name: 'Cyberpunk 2077', game_dir: '/games/cp2077' },
        { id: '3', status: 'pending', game_name: 'Other Game' }
      ];
      expect(findActiveGameTask(tasks, game)?.id).toBe('2');
    });

    it('returns null if no matching active task', () => {
      const tasks = [
        { id: '1', status: 'completed', game_name: 'Cyberpunk 2077' }
      ];
      expect(findActiveGameTask(tasks, game)).toBeNull();
      expect(findActiveGameTask([], null)).toBeNull();
    });
  });

  describe('findVerifyTask', () => {
    const game = { appId: '1091500', name: 'Cyberpunk 2077', gameDir: '/games/cp2077' };

    it('finds latest verify task for game', () => {
      const tasks = [
        { id: '1', type: 'verify', status: 'completed', game_dir: '/games/cp2077' },
        { id: '2', type: 'download', status: 'running', game_dir: '/games/cp2077' },
        { id: '3', type: 'verify', status: 'running', game_dir: '/games/cp2077' }
      ];
      expect(findVerifyTask(tasks, game)?.id).toBe('3');
    });

    it('returns null when no verify task exists', () => {
      const tasks = [
        { id: '1', type: 'download', status: 'running', game_dir: '/games/cp2077' }
      ];
      expect(findVerifyTask(tasks, game)).toBeNull();
    });
  });

  describe('isSteamTitle', () => {
    it('returns true for game with valid steam appId', () => {
      expect(isSteamTitle({ appId: '1091500' })).toBe(true);
      expect(isSteamTitle({ appId: 730 })).toBe(true);
    });

    it('returns true for game with steam metadataProvider', () => {
      expect(isSteamTitle({ appId: '0', metadataProvider: 'steam', metadataId: '1091500' })).toBe(true);
      expect(isSteamTitle({ metadataProvider: 'Steam', metadataId: '570' })).toBe(true);
    });

    it('returns false for non-Steam games', () => {
      expect(isSteamTitle({ appId: '0', metadataProvider: 'gog', metadataId: '123' })).toBe(false);
      expect(isSteamTitle(null)).toBe(false);
    });
  });
});
