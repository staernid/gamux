// Zero-Asset Real-Time Synthesized Web Audio UI SFX Engine
let audioCtx = null;
let sfxEnabled = typeof localStorage !== 'undefined' ? localStorage.getItem('gamux_sfx_enabled') !== 'false' : true;
let sfxVolume = typeof localStorage !== 'undefined' ? parseFloat(localStorage.getItem('gamux_sfx_volume') || '0.35') : 0.35;
let lastTickTime = 0;

function initAudioContext() {
  if (typeof window === 'undefined') return null;
  if (!audioCtx) {
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (AudioContextClass) {
      audioCtx = new AudioContextClass();
    }
  }
  if (audioCtx && audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {});
  }
  return audioCtx;
}

if (typeof window !== 'undefined') {
  const unlock = () => {
    initAudioContext();
  };
  window.addEventListener('pointerdown', unlock, { passive: true });
  window.addEventListener('keydown', unlock, { passive: true });
  window.addEventListener('wheel', unlock, { passive: true });
}

function getAudioContext() {
  return initAudioContext();
}

export function isSfxEnabled() {
  return sfxEnabled;
}

export function setSfxEnabled(enabled) {
  sfxEnabled = Boolean(enabled);
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('gamux_sfx_enabled', String(sfxEnabled));
  }
}

export function getSfxVolume() {
  return sfxVolume;
}

export function setSfxVolume(vol) {
  sfxVolume = Math.max(0, Math.min(1, parseFloat(vol) || 0.35));
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('gamux_sfx_volume', String(sfxVolume));
  }
}

/**
 * Punchy, crisp tactile mechanical click / card step tick
 * Combines an 880Hz->260Hz snap with a 150Hz->55Hz tactile body thump (38ms duration)
 * Clean, audible on all soundcards and speakers without distortion
 */
export function playTick() {
  if (!sfxEnabled || sfxVolume <= 0) return;
  const ctx = getAudioContext();
  if (!ctx) return;

  const now = ctx.currentTime;
  if (now - lastTickTime < 0.035) return;
  lastTickTime = now;

  try {
    const duration = 0.038;

    // 1. High-mid tactile mechanical snap (880Hz -> 260Hz)
    const snapOsc = ctx.createOscillator();
    const snapGain = ctx.createGain();
    snapOsc.type = 'triangle';
    snapOsc.frequency.setValueAtTime(880, now);
    snapOsc.frequency.exponentialRampToValueAtTime(260, now + duration);

    const snapVol = sfxVolume * 0.65;
    snapGain.gain.setValueAtTime(snapVol, now);
    snapGain.gain.exponentialRampToValueAtTime(0.0001, now + duration);

    snapOsc.connect(snapGain);
    snapGain.connect(ctx.destination);
    snapOsc.start(now);
    snapOsc.stop(now + duration + 0.005);

    // 2. Low-frequency mechanical body thump (150Hz -> 55Hz)
    const bodyOsc = ctx.createOscillator();
    const bodyGain = ctx.createGain();
    bodyOsc.type = 'sine';
    bodyOsc.frequency.setValueAtTime(150, now);
    bodyOsc.frequency.exponentialRampToValueAtTime(55, now + duration + 0.01);

    const bodyVol = sfxVolume * 0.45;
    bodyGain.gain.setValueAtTime(bodyVol, now);
    bodyGain.gain.exponentialRampToValueAtTime(0.0001, now + duration + 0.01);

    bodyOsc.connect(bodyGain);
    bodyGain.connect(ctx.destination);
    bodyOsc.start(now);
    bodyOsc.stop(now + duration + 0.015);
  } catch {}
}

/**
 * Harmonious acoustic chime / pop for launching or opening game details
 */
export function playSelect() {
  if (!sfxEnabled || sfxVolume <= 0) return;
  const ctx = getAudioContext();
  if (!ctx) return;

  try {
    const now = ctx.currentTime;
    
    // Note 1 (D5 ~ 587Hz)
    const osc1 = ctx.createOscillator();
    const gain1 = ctx.createGain();
    osc1.type = 'sine';
    osc1.frequency.setValueAtTime(587.33, now);
    gain1.gain.setValueAtTime(sfxVolume * 0.5, now);
    gain1.gain.exponentialRampToValueAtTime(0.0001, now + 0.08);
    osc1.connect(gain1);
    gain1.connect(ctx.destination);
    osc1.start(now);
    osc1.stop(now + 0.085);

    // Note 2 (A5 ~ 880Hz, slightly delayed for chord strum)
    const osc2 = ctx.createOscillator();
    const gain2 = ctx.createGain();
    osc2.type = 'sine';
    osc2.frequency.setValueAtTime(880, now + 0.015);
    gain2.gain.setValueAtTime(sfxVolume * 0.45, now + 0.015);
    gain2.gain.exponentialRampToValueAtTime(0.0001, now + 0.1);
    osc2.connect(gain2);
    gain2.connect(ctx.destination);
    osc2.start(now + 0.015);
    osc2.stop(now + 0.105);
  } catch {}
}

/**
 * Soft descending tone for escaping / backing out
 */
export function playBack() {
  if (!sfxEnabled || sfxVolume <= 0) return;
  const ctx = getAudioContext();
  if (!ctx) return;

  try {
    const now = ctx.currentTime;
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();

    osc.type = 'sine';
    osc.frequency.setValueAtTime(480, now);
    osc.frequency.exponentialRampToValueAtTime(240, now + 0.065);

    gain.gain.setValueAtTime(sfxVolume * 0.4, now);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.065);

    osc.connect(gain);
    gain.connect(ctx.destination);

    osc.start(now);
    osc.stop(now + 0.07);
  } catch {}
}

/**
 * Soft navigation blip for tab cycling
 */
export function playNav() {
  if (!sfxEnabled || sfxVolume <= 0) return;
  const ctx = getAudioContext();
  if (!ctx) return;

  try {
    const now = ctx.currentTime;
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();

    osc.type = 'sine';
    osc.frequency.setValueAtTime(680, now);
    osc.frequency.exponentialRampToValueAtTime(420, now + 0.04);

    gain.gain.setValueAtTime(sfxVolume * 0.35, now);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.04);

    osc.connect(gain);
    gain.connect(ctx.destination);

    osc.start(now);
    osc.stop(now + 0.045);
  } catch {}
}
