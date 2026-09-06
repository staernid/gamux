import { describe, it, expect } from 'vitest';
import { generateRandomSteamID, isValidSteamID64, STEAMID64_BASE } from './steam.js';

describe('steam utilities', () => {
  it('generates a valid 17-digit SteamID64 starting with the standard base', () => {
    const id = generateRandomSteamID();
    expect(typeof id).toBe('string');
    expect(id).toHaveLength(17);
    expect(isValidSteamID64(id)).toBe(true);

    const num = BigInt(id);
    expect(num).toBeGreaterThanOrEqual(STEAMID64_BASE);
  });

  it('generates unique random IDs across multiple calls', () => {
    const set = new Set();
    for (let i = 0; i < 20; i++) {
      set.add(generateRandomSteamID());
    }
    expect(set.size).toBe(20);
  });

  it('validates 17-digit numeric strings correctly', () => {
    expect(isValidSteamID64('76561197960287930')).toBe(true);
    expect(isValidSteamID64('76561198998877665')).toBe(true);
    expect(isValidSteamID64('12345678901234567')).toBe(true);

    // Invalid
    expect(isValidSteamID64('')).toBe(false);
    expect(isValidSteamID64(null)).toBe(false);
    expect(isValidSteamID64(undefined)).toBe(false);
    expect(isValidSteamID64('7656119796028793')).toBe(false); // 16 digits
    expect(isValidSteamID64('765611979602879301')).toBe(false); // 18 digits
    expect(isValidSteamID64('7656119796028793a')).toBe(false); // non-digits
  });
});
