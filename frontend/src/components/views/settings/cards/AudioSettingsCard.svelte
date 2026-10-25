<script>
  import { configStore } from '../../../../stores/config.js';
  import {
    isSfxEnabled,
    setSfxEnabled,
    getSfxVolume,
    setSfxVolume,
    isMuteInBackground,
    setMuteInBackground,
    playSelect
  } from '../../../../utils/audio.js';
  import SeamlessSwitch from '../../../common/SeamlessSwitch.svelte';
  import InheritSelect from '../../../common/InheritSelect.svelte';
  import ThemeSwitch from '../../../common/ThemeSwitch.svelte';
  import { UI_SCALE_OPTIONS } from '../../../../constants/options.js';

  const uiScaleOptions = UI_SCALE_OPTIONS;

  let sfxEnabled = $state(isSfxEnabled());
  let sfxVolume = $state(getSfxVolume());
  let sfxMuteInBackground = $state(isMuteInBackground());

  function handleScaleChange(size) {
    configStore.setScale(size);
  }

  function handleSfxToggle(e) {
    setSfxEnabled(e.detail);
    sfxEnabled = isSfxEnabled();
  }

  function handleMuteBgToggle(e) {
    setMuteInBackground(e.detail);
    sfxMuteInBackground = isMuteInBackground();
  }

  function handleVolumeInput(e) {
    const vol = parseFloat(e.target.value);
    setSfxVolume(vol);
    sfxVolume = getSfxVolume();
  }

  function handleTestSound() {
    playSelect();
  }
</script>

<div class="space-y-4">
  <!-- Display & Interface Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="space-y-0.5">
      <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
        Display &amp; Interface
      </h3>
      <p class="text-xs text-neutral-400 leading-relaxed">
        Interface theme and scaling options.
      </p>
    </div>
    <div class="space-y-2.5">
      <ThemeSwitch />
      <InheritSelect
        label="Interface Scale"
        description="Base UI scale and font sizing for handhelds, desktops, or TVs (applies immediately)"
        value={configStore.uiScale}
        options={uiScaleOptions}
        onchange={(val) => handleScaleChange(val)}
      />
    </div>
  </div>

  <!-- Audio & Sound Effects Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="space-y-0.5">
      <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
        Audio &amp; Sound Effects
      </h3>
      <p class="text-xs text-neutral-400 leading-relaxed">
        Configure UI sound effects, mute behavior, and volume.
      </p>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
      <SeamlessSwitch
        label="UI Sound Effects"
        description="Synthesized mechanical clicks and chimes for navigation and interactions"
        value={sfxEnabled}
        onchange={handleSfxToggle}
      />

      <SeamlessSwitch
        label="Mute in Background"
        description="Automatically silence UI audio when the Gamux window loses focus"
        value={sfxMuteInBackground}
        onchange={handleMuteBgToggle}
      />
    </div>

    {#if sfxEnabled}
      <div class="p-2.5 bg-surface-header border border-surface-border rounded-xl flex items-center justify-between gap-3 animate-fade-in mt-2.5">
        <div class="space-y-0.5 min-w-0">
          <span class="text-xs font-semibold text-neutral-200">Sound Effects Volume</span>
          <p class="text-2xs text-neutral-400">Master output volume for synthetic interface sounds</p>
        </div>
        <div class="flex items-center space-x-2.5 flex-shrink-0">
          <span class="text-xs font-mono text-neutral-300 w-10 text-right">{Math.round(sfxVolume * 100)}%</span>
          <input
            type="range"
            min="0"
            max="1"
            step="0.05"
            value={sfxVolume}
            oninput={handleVolumeInput}
            class="w-24 sm:w-32 accent-white cursor-pointer"
            aria-label="Sound Effects Volume"
          />
          <button
            type="button"
            onclick={handleTestSound}
            class="px-2.5 py-1 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors cursor-pointer"
            title="Play test audio chime"
          >
            Test
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
