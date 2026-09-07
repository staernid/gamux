import { describe, it, expect } from 'vitest';

export function parseAppIdFromTarget(target) {
  if (!target) return '';
  const clean = String(target).trim();
  // 1. gamux://app/123 or gamux://game/123 or gamux://123
  const gamuxMatch = clean.match(/^gamux:\/\/(?:app\/|game\/)?(\d+)/i);
  if (gamuxMatch) return gamuxMatch[1];
  // 2. steam://store/123 or steam://rungameid/123
  const steamMatch = clean.match(/^steam:\/\/(?:store\/|rungameid\/)?(\d+)/i);
  if (steamMatch) return steamMatch[1];
  // 3. https://store.steampowered.com/app/123
  const webMatch = clean.match(/\/app\/(\d+)/i);
  if (webMatch) return webMatch[1];
  // 4. Raw numeric AppID
  if (/^\d+$/.test(clean)) return clean;
  return '';
}

describe('parseAppIdFromTarget', () => {
  it('parses gamux://app/<appid>', () => {
    expect(parseAppIdFromTarget('gamux://app/730')).toBe('730');
    expect(parseAppIdFromTarget('gamux://app/1245620/')).toBe('1245620');
  });

  it('parses gamux://game/<appid> and gamux://<appid>', () => {
    expect(parseAppIdFromTarget('gamux://game/730')).toBe('730');
    expect(parseAppIdFromTarget('gamux://570')).toBe('570');
  });

  it('parses steam:// protocol links', () => {
    expect(parseAppIdFromTarget('steam://store/730')).toBe('730');
    expect(parseAppIdFromTarget('steam://rungameid/730')).toBe('730');
  });

  it('parses full web steam store URLs', () => {
    expect(parseAppIdFromTarget('https://store.steampowered.com/app/730/CounterStrike_2/')).toBe('730');
    expect(parseAppIdFromTarget('http://store.steampowered.com/app/1086940')).toBe('1086940');
  });

  it('parses raw numeric string AppIDs', () => {
    expect(parseAppIdFromTarget('730')).toBe('730');
    expect(parseAppIdFromTarget('  1245620  ')).toBe('1245620');
  });

  it('returns empty string for invalid targets', () => {
    expect(parseAppIdFromTarget('')).toBe('');
    expect(parseAppIdFromTarget(null)).toBe('');
    expect(parseAppIdFromTarget('https://google.com')).toBe('');
    expect(parseAppIdFromTarget('gamux://settings')).toBe('');
  });
});
