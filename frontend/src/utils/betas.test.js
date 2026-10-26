import { describe, it, expect } from 'vitest';
import { getEffectiveAppId, sortBranches, filterBranches, isBranchBuildMismatch } from './betas.js';

describe('betas utils', () => {
  describe('getEffectiveAppId', () => {
    it('returns game.appId if valid Steam App ID', () => {
      expect(getEffectiveAppId({ appId: '1091500' })).toBe('1091500');
      expect(getEffectiveAppId({ appId: 730 })).toBe('730');
    });

    it('falls back to steam metadataId if appId is invalid/missing', () => {
      expect(getEffectiveAppId({ appId: '0', metadataProvider: 'steam', metadataId: '400' })).toBe('400');
      expect(getEffectiveAppId({ metadataProvider: 'Steam', metadataId: '570' })).toBe('570');
    });

    it('returns empty string if neither is steam app', () => {
      expect(getEffectiveAppId({ appId: '0', metadataProvider: 'gog', metadataId: '123' })).toBe('');
      expect(getEffectiveAppId(null)).toBe('');
    });
  });

  describe('sortBranches', () => {
    const branches = [
      { name: 'staging' },
      { name: 'public' },
      { name: 'beta' },
      { name: 'alpha' }
    ];

    it('puts current branch first, then public, then alphabetical', () => {
      const sorted = sortBranches(branches, 'beta');
      expect(sorted.map(b => b.name)).toEqual(['beta', 'public', 'alpha', 'staging']);
    });

    it('handles currentBranch = public correctly', () => {
      const sorted = sortBranches(branches, 'public');
      expect(sorted.map(b => b.name)).toEqual(['public', 'alpha', 'beta', 'staging']);
    });
  });

  describe('filterBranches', () => {
    const branches = [
      { name: 'public', description: 'Default public branch', build_id: '100' },
      { name: 'beta', description: 'Testing beta branch', build_id: '200' },
      { name: 'experimental', description: 'Bleeding edge', build_id: '300' }
    ];

    it('returns all when query is empty', () => {
      expect(filterBranches(branches, '')).toHaveLength(3);
      expect(filterBranches(branches, '  ')).toHaveLength(3);
    });

    it('filters by name, description, or build_id case-insensitively', () => {
      expect(filterBranches(branches, 'BETA')).toHaveLength(1);
      expect(filterBranches(branches, 'edge')).toHaveLength(1);
      expect(filterBranches(branches, '100')).toHaveLength(1);
      expect(filterBranches(branches, 'xyz')).toHaveLength(0);
    });
  });

  describe('isBranchBuildMismatch', () => {
    it('returns true when installed build differs from target build', () => {
      expect(isBranchBuildMismatch(true, '1200000', '1205000')).toBe(true);
    });

    it('returns false when not installed', () => {
      expect(isBranchBuildMismatch(false, '1200000', '1205000')).toBe(false);
    });

    it('returns false when installed build is 0, unknown, or empty (Absence is Not an Error)', () => {
      expect(isBranchBuildMismatch(true, '0', '1205000')).toBe(false);
      expect(isBranchBuildMismatch(true, 'unknown', '1205000')).toBe(false);
      expect(isBranchBuildMismatch(true, '', '1205000')).toBe(false);
      expect(isBranchBuildMismatch(true, null, '1205000')).toBe(false);
    });

    it('returns false when target build is 0 or empty', () => {
      expect(isBranchBuildMismatch(true, '1200000', '0')).toBe(false);
      expect(isBranchBuildMismatch(true, '1200000', '')).toBe(false);
    });

    it('returns false when builds match', () => {
      expect(isBranchBuildMismatch(true, '1200000', '1200000')).toBe(false);
    });
  });
});
