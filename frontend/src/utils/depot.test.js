import { describe, it, expect } from 'vitest';
import {
  getContentCategory,
  getDepotLanguage,
  getCategoryLabel,
  getCategoryBadgeClass,
  isDefaultLanguageDepot,
  computeDefaultDepotSelections,
  filterDepotsByCategory,
  groupDepotsByCategory,
  updateSelectedLanguageDepots,
} from './depot.js';

describe('depot utils', () => {
  it('identifies content category correctly', () => {
    expect(getContentCategory(null)).toBe('core');
    expect(getContentCategory({ is_dlc: true })).toBe('dlc');
    expect(getContentCategory({ dlc_app_id: 12345 })).toBe('dlc');
    expect(getContentCategory({ content_category: 'dlc' })).toBe('dlc');
    expect(getContentCategory({ content_category: 'locale:german' })).toBe('locale:german');
    expect(getContentCategory({ category: 'soundtrack' })).toBe('soundtrack');
    expect(getContentCategory({ category: 'common' })).toBe('core');
  });

  it('resolves depot language', () => {
    expect(getDepotLanguage(null)).toBe('');
    expect(getDepotLanguage({ language: 'french' })).toBe('french');
    expect(getDepotLanguage({ content_category: 'locale:spanish' })).toBe('spanish');
    expect(getDepotLanguage({ category: 'common' })).toBe('');
  });

  it('formats category labels and badge classes', () => {
    expect(getCategoryLabel({ content_category: 'locale:german' })).toBe('German Voice / Audio');
    expect(getCategoryLabel({ content_category: 'optional_hd' })).toBe('High-Res / Optional Assets');
    expect(getCategoryLabel({ category: 'soundtrack' })).toBe('Official Soundtrack');
    expect(getCategoryLabel({ dlc_app_id: 999 })).toBe('DLC #999');
    expect(getCategoryLabel({ is_dlc: true })).toBe('DLC Content');
    expect(getCategoryLabel({ category: 'windows' })).toBe('Windows Binaries');
    expect(getCategoryLabel({ category: 'linux' })).toBe('Linux Native');
    expect(getCategoryLabel({ category: 'macos' })).toBe('macOS Bundle');
    expect(getCategoryLabel({ category: 'common' })).toBe('Base Content');

    expect(getCategoryBadgeClass({ content_category: 'locale:german' })).toContain('text-blue-300');
    expect(getCategoryBadgeClass({ content_category: 'optional_hd' })).toContain('text-amber-300');
    expect(getCategoryBadgeClass({ category: 'soundtrack' })).toContain('text-purple-300');
    expect(getCategoryBadgeClass({ is_dlc: true })).toContain('text-emerald-300');
    expect(getCategoryBadgeClass({ category: 'common' })).toContain('text-neutral-300');
  });

  it('matches default language depot', () => {
    expect(isDefaultLanguageDepot({ language: 'english' }, 'english')).toBe(true);
    expect(isDefaultLanguageDepot({ content_category: 'locale:english' }, 'english')).toBe(true);
    expect(isDefaultLanguageDepot({ language: 'german' }, 'english')).toBe(false);
    expect(isDefaultLanguageDepot(null, 'english')).toBe(false);
  });

  it('computes default selections and filters depots', () => {
    const sample = [
      { depot_id: 1, content_category: 'core' },
      { depot_id: 2, content_category: 'locale:german' },
      { depot_id: 3, content_category: 'locale:english' },
      { depot_id: 4, category: 'soundtrack' },
      { depot_id: 5, content_category: 'optional_hd' },
      { depot_id: 6, is_dlc: true },
    ];

    const germanSelections = computeDefaultDepotSelections(sample, 'german');
    expect(germanSelections.has(1)).toBe(true);
    expect(germanSelections.has(2)).toBe(true);
    expect(germanSelections.has(3)).toBe(false);
    expect(germanSelections.has(6)).toBe(true);

    const filteredLanguages = filterDepotsByCategory(sample, 'languages');
    expect(filteredLanguages.map(d => d.depot_id)).toEqual([2, 3]);

    const filteredSoundtrack = filterDepotsByCategory(sample, 'soundtrack');
    expect(filteredSoundtrack.map(d => d.depot_id)).toEqual([4]);

    const filteredOptional = filterDepotsByCategory(sample, 'optional');
    expect(filteredOptional.map(d => d.depot_id)).toEqual([5]);

    const filteredDlc = filterDepotsByCategory(sample, 'dlc');
    expect(filteredDlc.map(d => d.depot_id)).toEqual([6]);

    const filteredCore = filterDepotsByCategory(sample, 'core');
    expect(filteredCore.map(d => d.depot_id)).toEqual([1]);

    const grouped = groupDepotsByCategory(sample);
    expect(grouped.language.length).toBe(2);
    expect(grouped.soundtrack.length).toBe(1);
    expect(grouped.optional.length).toBe(1);
    expect(grouped.dlc.length).toBe(1);
    expect(grouped.core.length).toBe(1);

    const updatedLang = updateSelectedLanguageDepots(sample, new Set([1, 2]), 'english');
    expect(updatedLang.has(1)).toBe(true);
    expect(updatedLang.has(2)).toBe(false);
    expect(updatedLang.has(3)).toBe(true);
  });
});
