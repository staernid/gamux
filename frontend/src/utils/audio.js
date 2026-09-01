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

function withAudioContext(fn) {
  if (!sfxEnabled || sfxVolume <= 0) return;
  const ctx = getAudioContext();
  if (!ctx) return;
  if (ctx.state === 'suspended') {
    ctx.resume().catch(() => {});
  }
  try {
    fn(ctx);
  } catch {}
}

/**
 * Tactile Mechanical Switch / Dial Ratchet Click
 * Synthesizes a crisp, rich physical switch strike:
 * 1. Mechanical switch friction / contact noise burst (12ms)
 * 2. Crisp high-mid tactile snap (2200Hz -> 650Hz over 28ms)
 * 3. Acoustic body resonant "tock" (950Hz -> 280Hz over 36ms)
 * 4. Low-end tactile thump (220Hz -> 70Hz over 35ms)
 */
export function playTick() {
  withAudioContext((ctx) => {
    const now = ctx.currentTime;
    if (now - lastTickTime < 0.02) return;
    lastTickTime = now;

    // Psychoacoustic loudness compensation for short impulse duration
    const vol = Math.min(1.0, sfxVolume * 2.0);

    // 1. Mechanical switch friction / contact noise burst (12ms)
    const sampleRate = ctx.sampleRate || 44100;
    const noiseLen = Math.max(128, Math.floor(sampleRate * 0.014));
    const noiseBuffer = ctx.createBuffer(1, noiseLen, sampleRate);
    const noiseData = noiseBuffer.getChannelData(0);
    for (let i = 0; i < noiseLen; i++) {
      noiseData[i] = (Math.random() * 2 - 1) * Math.exp(-i / (noiseLen * 0.28));
    }

    const noiseSrc = ctx.createBufferSource();
    noiseSrc.buffer = noiseBuffer;

    const noiseFilter = ctx.createBiquadFilter();
    noiseFilter.type = 'bandpass';
    noiseFilter.frequency.setValueAtTime(3600, now);
    noiseFilter.Q.setValueAtTime(2.5, now);

    const noiseGain = ctx.createGain();
    noiseGain.gain.setValueAtTime(vol * 0.7, now);
    noiseGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.012);

    noiseSrc.connect(noiseFilter);
    noiseFilter.connect(noiseGain);
    noiseGain.connect(ctx.destination);
    noiseSrc.start(now);

    // 2. Crisp high-mid tactile snap (2200Hz -> 650Hz over 28ms)
    const snapOsc = ctx.createOscillator();
    const snapGain = ctx.createGain();
    snapOsc.type = 'triangle';
    snapOsc.frequency.setValueAtTime(2200, now);
    snapOsc.frequency.exponentialRampToValueAtTime(650, now + 0.028);

    snapGain.gain.setValueAtTime(vol * 0.85, now);
    snapGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.028);

    snapOsc.connect(snapGain);
    snapGain.connect(ctx.destination);
    snapOsc.start(now);
    snapOsc.stop(now + 0.03);

    // 3. Acoustic body resonant "tock" (950Hz -> 280Hz over 36ms)
    const bodyOsc = ctx.createOscillator();
    const bodyGain = ctx.createGain();
    bodyOsc.type = 'sine';
    bodyOsc.frequency.setValueAtTime(950, now);
    bodyOsc.frequency.exponentialRampToValueAtTime(280, now + 0.036);

    bodyGain.gain.setValueAtTime(vol * 0.65, now);
    bodyGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.036);

    bodyOsc.connect(bodyGain);
    bodyGain.connect(ctx.destination);
    bodyOsc.start(now);
    bodyOsc.stop(now + 0.038);

    // 4. Low-end tactile thump (220Hz -> 70Hz over 35ms)
    const thumpOsc = ctx.createOscillator();
    const thumpGain = ctx.createGain();
    thumpOsc.type = 'sine';
    thumpOsc.frequency.setValueAtTime(220, now);
    thumpOsc.frequency.exponentialRampToValueAtTime(70, now + 0.035);

    thumpGain.gain.setValueAtTime(vol * 0.5, now);
    thumpGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.035);

    thumpOsc.connect(thumpGain);
    thumpGain.connect(ctx.destination);
    thumpOsc.start(now);
    thumpOsc.stop(now + 0.037);
  });
}

/**
 * Harmonious acoustic chime / pop for launching or opening game details
 */
export function playSelect() {
  withAudioContext((ctx) => {
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
  });
}

/**
 * Soft descending tone for escaping / backing out
 */
export function playBack() {
  withAudioContext((ctx) => {
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
  });
}

/**
 * Soft navigation blip for tab cycling
 */
export function playNav() {
  withAudioContext((ctx) => {
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
  });
}
