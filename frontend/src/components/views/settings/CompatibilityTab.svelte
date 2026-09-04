<script>
  import { AppAPI } from '../../../api/app.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import GhostInput from '../../common/GhostInput.svelte';

  export let cfgDraft;
  export let runnersList = [];

  let showAdvancedCompat = false;
  let newEnvKey = '';
  let newEnvVal = '';

  async function handleBrowseWinePrefix() {
    try {
      const selected = await AppAPI.selectDirectory('Select Wine Prefix Directory', cfgDraft.wine_prefix || '');
      if (selected) {
        cfgDraft.wine_prefix = selected;
        cfgDraft = { ...cfgDraft };
      }
    } catch {
      // Handled
    }
  }

  function addGlobalEnv() {
    const k = newEnvKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
    const v = newEnvVal.trim();
    if (!k) return;
    cfgDraft.global_env = { ...(cfgDraft.global_env || {}), [k]: v };
    cfgDraft = { ...cfgDraft };
    newEnvKey = '';
    newEnvVal = '';
  }

  function removeGlobalEnv(k) {
    const next = { ...(cfgDraft.global_env || {}) };
    delete next[k];
    cfgDraft.global_env = next;
    cfgDraft = { ...cfgDraft };
  }

  const operatingModesList = [
    { value: 'loader', label: 'Loader Mode (Clean / Non-Invasive - Recommended)' },
    { value: 'portable', label: 'Portable Mode (Injected DLL Replacement)' },
    { value: 'none', label: 'DRM-Free / None' }
  ];

  let selectedResPreset = 'auto';
  let isCustomRes = false;

  $: {
    const w = cfgDraft?.gamescope_width || 0;
    const h = cfgDraft?.gamescope_height || 0;
    if (!w && !h) {
      selectedResPreset = 'auto';
      isCustomRes = false;
    } else if (w === 3840 && h === 2160) {
      selectedResPreset = '3840x2160';
      isCustomRes = false;
    } else if (w === 3440 && h === 1440) {
      selectedResPreset = '3440x1440';
      isCustomRes = false;
    } else if (w === 2560 && h === 1440) {
      selectedResPreset = '2560x1440';
      isCustomRes = false;
    } else if (w === 1920 && h === 1080) {
      selectedResPreset = '1920x1080';
      isCustomRes = false;
    } else if (w === 1280 && h === 720) {
      selectedResPreset = '1280x720';
      isCustomRes = false;
    } else {
      selectedResPreset = 'custom';
      isCustomRes = true;
    }
  }

  function handleResChange(val) {
    selectedResPreset = val;
    if (val === 'auto') {
      cfgDraft.gamescope_width = 0;
      cfgDraft.gamescope_height = 0;
      isCustomRes = false;
    } else if (val === 'custom') {
      isCustomRes = true;
    } else {
      isCustomRes = false;
      const parts = val.split('x').map(Number);
      cfgDraft.gamescope_width = parts[0];
      cfgDraft.gamescope_height = parts[1];
    }
    cfgDraft = { ...cfgDraft };
  }

  const resPresets = [
    { value: 'auto', label: 'Auto-Detect Native Display' },
    { value: '3840x2160', label: '3840 × 2160 (4K UHD)' },
    { value: '3440x1440', label: '3440 × 1440 (UWQHD 21:9)' },
    { value: '2560x1440', label: '2560 × 1440 (1440p QHD)' },
    { value: '1920x1080', label: '1920 × 1080 (1080p FHD)' },
    { value: '1280x720', label: '1280 × 720 (720p HD)' },
    { value: 'custom', label: 'Custom Resolution...' }
  ];

  let selectedFpsPreset = '0';
  let isCustomFps = false;

  $: {
    const fps = cfgDraft?.gamescope_fps || 0;
    if ([0, 60, 100, 120, 144, 165, 240].includes(fps)) {
      selectedFpsPreset = String(fps);
      isCustomFps = false;
    } else {
      selectedFpsPreset = 'custom';
      isCustomFps = true;
    }
  }

  function handleFpsChange(val) {
    selectedFpsPreset = val;
    if (val === 'custom') {
      isCustomFps = true;
    } else {
      isCustomFps = false;
      cfgDraft.gamescope_fps = Number(val);
      cfgDraft = { ...cfgDraft };
    }
  }

  const fpsPresets = [
    { value: '0', label: 'Display Default / Uncapped' },
    { value: '60', label: '60 FPS / Hz' },
    { value: '100', label: '100 FPS / Hz' },
    { value: '120', label: '120 FPS / Hz' },
    { value: '144', label: '144 FPS / Hz' },
    { value: '165', label: '165 FPS / Hz' },
    { value: '240', label: '240 FPS / Hz' },
    { value: 'custom', label: 'Custom Limit...' }
  ];
</script>

<div class="space-y-3">
  <SettingSection title="Runtime & Compatibility">
    <div class="space-y-2.5">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
        <InheritSelect
          label="Default Runner"
          bind:value={cfgDraft.default_runner}
          options={runnersList}
        />

        <InheritSelect
          label="Default Operating Mode"
          bind:value={cfgDraft.operating_mode}
          options={operatingModesList}
        />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
        <SeamlessSwitch
          label="MangoHud Overlay"
          description="FPS and performance telemetry overlay"
          bind:value={cfgDraft.mangohud}
        />

        <SeamlessSwitch
          label="DRM Unpacking"
          description="Automatically run Steamless unpacker during setup"
          bind:value={cfgDraft.steamless}
        />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5 pt-2 border-t border-surface-border">
        <SeamlessSwitch
          label="Offline Mode"
          description="Report disconnected offline status in Steamworks emulation"
          bind:value={cfgDraft.offline_mode}
        />

        <SeamlessSwitch
          label="Disable In-Game Overlay"
          description="Disable Goldberg Steam overlay hooks (prevents crashes with graphics hooks)"
          bind:value={cfgDraft.disable_overlay}
        />
      </div>
    </div>

    <!-- Advanced Compatibility Options -->
    <div class="pt-3 mt-3 border-t border-surface-border">
      <button
        type="button"
        on:click={() => { showAdvancedCompat = !showAdvancedCompat; }}
        class="text-xs font-semibold text-neutral-400 hover:text-neutral-200 flex items-center space-x-1.5 cursor-pointer"
      >
        <svg class="w-3.5 h-3.5 transform transition-transform {showAdvancedCompat ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
        <span>Advanced Compatibility & Environment</span>
      </button>

      {#if showAdvancedCompat}
        <div class="pt-3 space-y-3 animate-fade-in">
          <div class="space-y-3">
            <GhostInput
              label="Default Wine Prefix"
              description="Global isolated Wine/Proton prefix directory for non-sandboxed titles"
              bind:value={cfgDraft.wine_prefix}
              placeholder="~/.local/share/wineprefixes/default"
              mono={true}
            >
              <button
                slot="action"
                type="button"
                on:click={handleBrowseWinePrefix}
                class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
              >
                Browse
              </button>
            </GhostInput>
          </div>

          <!-- Global Environment Variables -->
          <div class="pt-2 border-t border-surface-border space-y-2">
            <span class="text-xs font-bold text-neutral-200 uppercase tracking-wider block">
              Global Environment Variables
            </span>

            {#if Object.keys(cfgDraft.global_env || {}).length > 0}
              <div class="grid grid-cols-1 md:grid-cols-2 gap-1.5">
                {#each Object.keys(cfgDraft.global_env) as ek}
                  <div class="flex items-center justify-between p-2 bg-[#101013] border border-surface-border rounded-lg">
                    <div class="flex items-center space-x-2 font-mono text-xs min-w-0 truncate">
                      <span class="text-white font-bold truncate">{ek}</span>
                      <span class="text-neutral-500">=</span>
                      <span class="text-neutral-300 truncate">{cfgDraft.global_env[ek]}</span>
                    </div>
                    <button
                      type="button"
                      on:click={() => removeGlobalEnv(ek)}
                      class="p-1 text-neutral-400 hover:text-rose-400 transition-colors cursor-pointer flex-shrink-0"
                      aria-label={`Remove global environment variable ${ek}`}
                    >
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                {/each}
              </div>
            {/if}

            <div class="flex items-center gap-2 pt-1">
              <input
                type="text"
                placeholder="VAR_NAME"
                bind:value={newEnvKey}
                on:keydown={(e) => { if (e.key === 'Enter') addGlobalEnv(); }}
                class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono uppercase outline-none transition-colors"
              />
              <input
                type="text"
                placeholder="value"
                bind:value={newEnvVal}
                on:keydown={(e) => { if (e.key === 'Enter') addGlobalEnv(); }}
                class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono outline-none transition-colors"
              />
              <button
                type="button"
                on:click={addGlobalEnv}
                class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex-shrink-0 cursor-pointer"
              >
                Add
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </SettingSection>

  <SettingSection title="Gamescope & Screen Resolution">
    <div class="space-y-3">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Configures base display resolution and target refresh rate for Gamescope. This resolution is applied 1:1 for single-player launches (when enabled) and automatically partitioned across split-screen co-op seats.
      </p>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
        <SeamlessSwitch
          label="Enable Single-Player Gamescope"
          description="Wrap single-player game launches in Gamescope compositor (off by default)"
          bind:value={cfgDraft.gamescope}
        />

        <div class="space-y-1.5">
          <InheritSelect
            label="Base Screen Resolution"
            value={selectedResPreset}
            on:change={(e) => handleResChange(e.detail)}
            options={resPresets}
          />
          {#if isCustomRes}
            <div class="grid grid-cols-2 gap-2 pt-1">
              <div>
                <label for="custom-gamescope-w" class="block text-2xs font-semibold text-neutral-400 mb-1">Width (px)</label>
                <input
                  id="custom-gamescope-w"
                  type="number"
                  bind:value={cfgDraft.gamescope_width}
                  class="w-full bg-surface-input px-2.5 py-1.5 rounded-lg text-xs text-white border border-surface-border focus:outline-none focus:border-neutral-400"
                  placeholder="3440"
                />
              </div>
              <div>
                <label for="custom-gamescope-h" class="block text-2xs font-semibold text-neutral-400 mb-1">Height (px)</label>
                <input
                  id="custom-gamescope-h"
                  type="number"
                  bind:value={cfgDraft.gamescope_height}
                  class="w-full bg-surface-input px-2.5 py-1.5 rounded-lg text-xs text-white border border-surface-border focus:outline-none focus:border-neutral-400"
                  placeholder="1440"
                />
              </div>
            </div>
          {/if}
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5 pt-2 border-t border-surface-border">
        <div class="space-y-1.5">
          <InheritSelect
            label="Target Refresh Rate / FPS"
            value={selectedFpsPreset}
            on:change={(e) => handleFpsChange(e.detail)}
            options={fpsPresets}
          />
          {#if isCustomFps}
            <div class="pt-1">
              <label for="custom-gamescope-fps" class="block text-2xs font-semibold text-neutral-400 mb-1">Target FPS (Hz)</label>
              <input
                id="custom-gamescope-fps"
                type="number"
                bind:value={cfgDraft.gamescope_fps}
                class="w-full bg-surface-input px-2.5 py-1.5 rounded-lg text-xs text-white border border-surface-border focus:outline-none focus:border-neutral-400"
                placeholder="100"
              />
            </div>
          {/if}
        </div>
      </div>
    </div>
  </SettingSection>
</div>
