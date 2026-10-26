import { describe, it, expect } from 'vitest';
import { parseTabParam, getAdjacentTab, SETTINGS_TABS, DEFAULT_CFG_DRAFT, DEFAULT_USER_CFG_DRAFT } from './settings.js';

describe('settings utils', () => {
  describe('DEFAULT_CFG_DRAFT', () => {
    it('contains expected default values', () => {
      expect(DEFAULT_CFG_DRAFT.default_runner).toBe('umu');
      expect(DEFAULT_CFG_DRAFT.multiseat).toBeDefined();
      expect(DEFAULT_CFG_DRAFT.multiseat.seat2_show_cursor).toBe(true);
    });

    it('defines default GSE user config', () => {
      expect(DEFAULT_USER_CFG_DRAFT.account_name).toBe('GamuxPlayer');
    });
  });

  describe('parseTabParam', () => {
    it('returns null for null or empty input', () => {
      expect(parseTabParam(null)).toBeNull();
      expect(parseTabParam('')).toBeNull();
    });

    it('resolves general aliases', () => {
      expect(parseTabParam('general')).toEqual({ activeTab: 'general' });
      expect(parseTabParam('network')).toEqual({ activeTab: 'general' });
      expect(parseTabParam('ui')).toEqual({ activeTab: 'general' });
    });

    it('resolves storage and credential aliases', () => {
      expect(parseTabParam('storage')).toEqual({ activeTab: 'storage' });
      expect(parseTabParam('library')).toEqual({ activeTab: 'storage' });
      expect(parseTabParam('keys')).toEqual({ activeTab: 'storage' });
    });

    it('resolves compatibility aliases', () => {
      expect(parseTabParam('compatibility')).toEqual({ activeTab: 'compatibility' });
      expect(parseTabParam('compat')).toEqual({ activeTab: 'compatibility' });
    });

    it('resolves profiles aliases', () => {
      expect(parseTabParam('profiles')).toEqual({ activeTab: 'profiles' });
      expect(parseTabParam('persona')).toEqual({ activeTab: 'profiles' });
    });

    it('resolves advanced subtab aliases', () => {
      expect(parseTabParam('advanced')).toEqual({ activeTab: 'advanced' });
      expect(parseTabParam('coop')).toEqual({ activeTab: 'advanced', advancedSubtab: 'coop' });
      expect(parseTabParam('multiseat')).toEqual({ activeTab: 'advanced', advancedSubtab: 'coop' });
      expect(parseTabParam('maintenance')).toEqual({ activeTab: 'advanced', advancedSubtab: 'maintenance' });
      expect(parseTabParam('tools')).toEqual({ activeTab: 'advanced', advancedSubtab: 'maintenance' });
      expect(parseTabParam('logs')).toEqual({ activeTab: 'advanced', advancedSubtab: 'logs' });
      expect(parseTabParam('schema')).toEqual({ activeTab: 'advanced', advancedSubtab: 'schema' });
    });
  });

  describe('getAdjacentTab', () => {
    it('cycles forward through tabs', () => {
      expect(getAdjacentTab('general', 'next')).toBe('storage');
      expect(getAdjacentTab('advanced', 'next')).toBe('general');
    });

    it('cycles backward through tabs', () => {
      expect(getAdjacentTab('general', 'prev')).toBe('advanced');
      expect(getAdjacentTab('storage', 'prev')).toBe('general');
    });

    it('defaults to first tab if current tab is unknown', () => {
      expect(getAdjacentTab('unknown', 'next')).toBe(SETTINGS_TABS[0]);
    });
  });
});
