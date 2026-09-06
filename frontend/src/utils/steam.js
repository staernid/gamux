/**
 * Steamworks persona and emulator identity helpers.
 */

export const STEAMID64_BASE = BigInt('76561197960265728');

/**
 * Generates a valid randomized 17-digit SteamID64 for offline emulator accounts.
 * @returns {string} 17-digit numeric string
 */
export function generateRandomSteamID() {
  const randomOffset = BigInt(Math.floor(Math.random() * 1000000000));
  return (STEAMID64_BASE + randomOffset).toString();
}

/**
 * Validates whether a given string is a valid 17-digit SteamID64.
 * @param {string|number} id
 * @returns {boolean}
 */
export function isValidSteamID64(id) {
  if (!id) return false;
  return /^\d{17}$/.test(String(id).trim());
}
