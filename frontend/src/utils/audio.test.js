import { describe, it, expect, beforeEach } from 'vitest';
import {
  isSfxEnabled,
  setSfxEnabled,
  getSfxVolume,
  setSfxVolume,
  isMuteInBackground,
  setMuteInBackground,
  playTick,
  playSelect,
  playBack,
  playNav
} from './audio.js';

describe('Audio SFX Engine', () => {
  beforeEach(() => {
    setSfxEnabled(true);
    setSfxVolume(0.22);
    setMuteInBackground(false);
  });

  it('manages enabled state properly', () => {
    expect(isSfxEnabled()).toBe(true);
    setSfxEnabled(false);
    expect(isSfxEnabled()).toBe(false);
    setSfxEnabled(true);
    expect(isSfxEnabled()).toBe(true);
  });

  it('clamps volume properly', () => {
    setSfxVolume(0.5);
    expect(getSfxVolume()).toBe(0.5);
    setSfxVolume(1.5);
    expect(getSfxVolume()).toBe(1.0);
    setSfxVolume(-0.2);
    expect(getSfxVolume()).toBe(0);
  });

  it('manages mute in background properly', () => {
    expect(isMuteInBackground()).toBe(false);
    setMuteInBackground(true);
    expect(isMuteInBackground()).toBe(true);
    setMuteInBackground(false);
    expect(isMuteInBackground()).toBe(false);
  });

  it('runs play functions safely without crashing in test environment', () => {
    expect(() => {
      playTick();
      playSelect();
      playBack();
      playNav();
    }).not.toThrow();
  });
});
