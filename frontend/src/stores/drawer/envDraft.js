/**
 * Environment Variable Draft Helpers
 * Provides pure helper functions for computing inherited and overriding variables
 * and immutably updating per-game environment dictionaries.
 */

export function computeEnvEntries(customEnv = {}, globalEnv = {}) {
  const entries = [];
  const seen = new Set();

  // 1. Process global envs with potential game-specific overrides
  for (const [k, v] of Object.entries(globalEnv || {})) {
    seen.add(k);
    if (customEnv && k in customEnv) {
      entries.push({
        key: k,
        value: customEnv[k],
        globalValue: v,
        isInherited: false,
        isOverriding: true,
      });
    } else {
      entries.push({
        key: k,
        value: v,
        globalValue: v,
        isInherited: true,
        isOverriding: false,
      });
    }
  }

  // 2. Process custom game envs not present in global configuration
  for (const [k, v] of Object.entries(customEnv || {})) {
    if (!seen.has(k)) {
      entries.push({
        key: k,
        value: v,
        globalValue: '',
        isInherited: false,
        isOverriding: false,
      });
    }
  }

  return entries;
}

export function generateNextCustomKey(customEnv = {}, globalEnv = {}) {
  let newKey = 'CUSTOM_VAR';
  let counter = 1;
  while ((customEnv && newKey in customEnv) || (globalEnv && newKey in globalEnv)) {
    newKey = `CUSTOM_VAR_${counter++}`;
  }
  return newKey;
}

export function addCustomEnv(currentCustomEnv = {}, globalEnv = {}) {
  const nextKey = generateNextCustomKey(currentCustomEnv, globalEnv);
  return { ...currentCustomEnv, [nextKey]: '' };
}

export function updateCustomEnv(currentCustomEnv = {}, key, value) {
  return { ...currentCustomEnv, [key]: value };
}

export function overrideEnv(currentCustomEnv = {}, key, globalValue) {
  return { ...currentCustomEnv, [key]: globalValue };
}

export function revertEnv(currentCustomEnv = {}, key) {
  const next = { ...currentCustomEnv };
  delete next[key];
  return next;
}

export const deleteCustomEnv = revertEnv;
