/**
 * Utilities for Depot and DLC categorization, labeling, and selection.
 */

/**
 * @param {any} depot
 * @returns {string}
 */
export function getContentCategory(depot) {
  if (!depot) return 'core';
  if (depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0) || depot.content_category === 'dlc') return 'dlc';
  if (depot.content_category) return depot.content_category;
  if (depot.category === 'soundtrack') return 'soundtrack';
  return 'core';
}

/**
 * @param {any} depot
 * @returns {string}
 */
export function getDepotLanguage(depot) {
  if (!depot) return '';
  if (depot.language) return depot.language;
  const cat = getContentCategory(depot);
  if (cat.startsWith('locale:')) {
    return cat.slice(7);
  }
  return '';
}

/**
 * @param {any} depot
 * @returns {string}
 */
export function getCategoryLabel(depot) {
  if (!depot) return 'Core Content';
  const cat = getContentCategory(depot);
  if (cat.startsWith('locale:')) {
    const lang = getDepotLanguage(depot);
    const capLang = lang ? lang.charAt(0).toUpperCase() + lang.slice(1) : 'Locale';
    return `${capLang} Voice / Audio`;
  }
  if (cat === 'optional_hd') {
    return 'High-Res / Optional Assets';
  }
  if (cat === 'soundtrack') {
    return 'Official Soundtrack';
  }
  if (cat === 'dlc' || depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0)) {
    if (depot.dlc_app_id) {
      return `DLC #${depot.dlc_app_id}`;
    }
    return 'DLC Content';
  }
  switch (depot.category) {
    case 'common': return 'Base Content';
    case 'windows': return 'Windows Binaries';
    case 'linux': return 'Linux Native';
    case 'macos': return 'macOS Bundle';
    default: return 'Core Content';
  }
}

/**
 * @param {any} depot
 * @returns {string}
 */
export function getCategoryBadgeClass(depot) {
  const cat = getContentCategory(depot);
  if (cat.startsWith('locale:')) {
    return 'bg-blue-950/60 text-blue-300 border-blue-800/50';
  }
  if (cat === 'optional_hd') {
    return 'bg-amber-950/60 text-amber-300 border-amber-800/50';
  }
  if (cat === 'soundtrack') {
    return 'bg-purple-950/60 text-purple-300 border-purple-800/50';
  }
  if (cat === 'dlc' || depot.is_dlc || (depot.dlc_app_id && Number(depot.dlc_app_id) > 0)) {
    return 'bg-emerald-950/60 text-emerald-300 border-emerald-800/50';
  }
  return 'bg-surface-header text-neutral-300 border-surface-border';
}

/**
 * @param {any} depot
 * @param {string} configuredLanguage
 * @returns {boolean}
 */
export function isDefaultLanguageDepot(depot, configuredLanguage) {
  const lang = getDepotLanguage(depot).toLowerCase();
  if (!lang || !configuredLanguage) return false;
  const conf = configuredLanguage.toLowerCase();
  return lang === conf || conf.startsWith(lang) || lang.startsWith(conf);
}

/**
 * @param {any[]} depotList
 * @param {string} configuredLanguage
 * @returns {Set<number>}
 */
export function computeDefaultDepotSelections(depotList, configuredLanguage) {
  const selected = new Set();
  const confLang = (configuredLanguage || '').toLowerCase();
  for (const d of depotList) {
    const cat = getContentCategory(d);
    if (cat === 'core' || cat === 'common' || cat === 'dlc' || d.is_dlc) {
      selected.add(Number(d.depot_id));
    } else if (cat.startsWith('locale:')) {
      const lang = getDepotLanguage(d).toLowerCase();
      if (lang === confLang || confLang.startsWith(lang) || lang.startsWith(confLang)) {
        selected.add(Number(d.depot_id));
      }
    }
  }
  const hasLang = Array.from(selected).some(id => {
    const d = depotList.find(x => Number(x.depot_id) === id);
    return d && getContentCategory(d).startsWith('locale:');
  });
  if (!hasLang) {
    const engDepot = depotList.find(d => getDepotLanguage(d).toLowerCase() === 'english');
    if (engDepot) selected.add(Number(engDepot.depot_id));
  }
  return selected;
}

/**
 * @param {any[]} depots
 * @param {string} selectedCategoryFilter
 * @returns {any[]}
 */
export function filterDepotsByCategory(depots, selectedCategoryFilter) {
  return (depots || []).filter(depot => {
    if (selectedCategoryFilter === 'all') return true;
    const cat = getContentCategory(depot);
    if (selectedCategoryFilter === 'languages') return cat.startsWith('locale:');
    if (selectedCategoryFilter === 'optional') return cat === 'optional_hd';
    if (selectedCategoryFilter === 'soundtrack') return cat === 'soundtrack';
    if (selectedCategoryFilter === 'dlc') return cat === 'dlc';
    if (selectedCategoryFilter === 'core') return cat === 'core' || (!cat.startsWith('locale:') && cat !== 'optional_hd' && cat !== 'soundtrack' && cat !== 'dlc');
    return true;
  });
}

/**
 * @param {any[]} depots
 * @returns {{ language: any[], optional: any[], soundtrack: any[], dlc: any[], core: any[] }}
 */
export function groupDepotsByCategory(depots) {
  const language = [];
  const optional = [];
  const soundtrack = [];
  const dlc = [];
  const core = [];

  for (const d of depots || []) {
    const cat = getContentCategory(d);
    if (cat.startsWith('locale:')) {
      language.push(d);
    } else if (cat === 'optional_hd') {
      optional.push(d);
    } else if (cat === 'soundtrack') {
      soundtrack.push(d);
    } else if (cat === 'dlc') {
      dlc.push(d);
    } else {
      core.push(d);
    }
  }
  return { language, optional, soundtrack, dlc, core };
}

/**
 * @param {any[]} depots
 * @param {Set<number>} selectedDepotIds
 * @param {string} newLang
 * @returns {Set<number>}
 */
export function updateSelectedLanguageDepots(depots, selectedDepotIds, newLang) {
  const next = new Set(selectedDepotIds);
  const langLower = (newLang || '').toLowerCase();
  for (const d of depots || []) {
    if (getContentCategory(d).startsWith('locale:')) {
      next.delete(Number(d.depot_id));
    }
  }
  for (const d of depots || []) {
    if (getContentCategory(d).startsWith('locale:')) {
      const l = getDepotLanguage(d).toLowerCase();
      if (l === langLower || langLower.startsWith(l)) {
        next.add(Number(d.depot_id));
      }
    }
  }
  return next;
}

