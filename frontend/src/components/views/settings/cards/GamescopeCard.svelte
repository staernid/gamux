<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../../common/SeamlessSwitch.svelte';
  import InheritSelect from '../../../common/InheritSelect.svelte';

  let { gamescopeDraft = $bindable(), cfgDraft = $bindable() } = $props();

  let targetDraft = $derived(gamescopeDraft || cfgDraft || {});

  let selectedResPreset = $state('auto');
  let isCustomRes = $state(false);

  $effect(() => {
    const w = targetDraft.gamescope_width || 0;
    const h = targetDraft.gamescope_height || 0;
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
  });

  function setDimensions(w, h) {
    if (gamescopeDraft) {
      gamescopeDraft.gamescope_width = w;
      gamescopeDraft.gamescope_height = h;
    }
    if (cfgDraft) {
      cfgDraft.gamescope_width = w;
      cfgDraft.gamescope_height = h;
    }
  }

  function handleResChange(val) {
    selectedResPreset = val;
    if (val === 'auto') {
      setDimensions(0, 0);
      isCustomRes = false;
    } else if (val === 'custom') {
      isCustomRes = true;
    } else {
      isCustomRes = false;
      const parts = val.split('x').map(Number);
      setDimensions(parts[0], parts[1]);
    }
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

  let selectedFpsPreset = $state('0');
  let isCustomFps = $state(false);

  $effect(() => {
    const fps = targetDraft.gamescope_fps || 0;
    if ([0, 60, 100, 120, 144, 165, 240].includes(fps)) {
      selectedFpsPreset = String(fps);
      isCustomFps = false;
    } else {
      selectedFpsPreset = 'custom';
      isCustomFps = true;
    }
  });

  function setFps(fps) {
    if (gamescopeDraft) gamescopeDraft.gamescope_fps = fps;
    if (cfgDraft) cfgDraft.gamescope_fps = fps;
  }

  function handleFpsChange(val) {
    selectedFpsPreset = val;
    if (val === 'custom') {
      isCustomFps = true;
    } else {
      isCustomFps = false;
      setFps(Number(val));
    }
  }

  function handleGamescopeToggle(val) {
    if (gamescopeDraft) gamescopeDraft.gamescope = val;
    if (cfgDraft) cfgDraft.gamescope = val;
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

<SettingSection title="Gamescope & Screen Resolution">
  <div class="space-y-3">
    <p class="text-xs text-neutral-400 leading-relaxed">
      Configures base display resolution and target refresh rate for Gamescope. This resolution is applied 1:1 for single-player launches (when enabled) and automatically partitioned across split-screen co-op seats.
    </p>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
      <SeamlessSwitch
        label="Enable Single-Player Gamescope"
        description="Wrap single-player game launches in Gamescope compositor (off by default)"
        value={targetDraft.gamescope}
        onchange={handleGamescopeToggle}
      />

      <div class="space-y-1.5">
        <InheritSelect
          label="Base Screen Resolution"
          value={selectedResPreset}
          onchange={(val) => handleResChange(val)}
          options={resPresets}
        />
        {#if isCustomRes}
          <div class="grid grid-cols-2 gap-2 pt-1">
            <div>
              <label for="custom-gamescope-w" class="block text-2xs font-semibold text-neutral-400 mb-1">Width (px)</label>
              <input
                id="custom-gamescope-w"
                type="number"
                bind:value={targetDraft.gamescope_width}
                class="w-full bg-surface-input px-2.5 py-1.5 rounded-lg text-xs text-white border border-surface-border focus:outline-none focus:border-neutral-400"
                placeholder="3440"
              />
            </div>
            <div>
              <label for="custom-gamescope-h" class="block text-2xs font-semibold text-neutral-400 mb-1">Height (px)</label>
              <input
                id="custom-gamescope-h"
                type="number"
                bind:value={targetDraft.gamescope_height}
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
          onchange={(val) => handleFpsChange(val)}
          options={fpsPresets}
        />
        {#if isCustomFps}
          <div class="pt-1">
            <label for="custom-gamescope-fps" class="block text-2xs font-semibold text-neutral-400 mb-1">Target FPS (Hz)</label>
            <input
              id="custom-gamescope-fps"
              type="number"
              bind:value={targetDraft.gamescope_fps}
              class="w-full bg-surface-input px-2.5 py-1.5 rounded-lg text-xs text-white border border-surface-border focus:outline-none focus:border-neutral-400"
              placeholder="100"
            />
          </div>
        {/if}
      </div>
    </div>
  </div>
</SettingSection>
