<script>
  import SeamlessSwitch from '../../../common/SeamlessSwitch.svelte';
  import InheritSelect from '../../../common/InheritSelect.svelte';

  let { gamescopeDraft = $bindable(), cfgDraft = $bindable() } = $props();

  let targetDraft = $derived(gamescopeDraft || cfgDraft || {});

  let userSelectedRes = $state(null);
  let computedRes = $derived.by(() => {
    const w = targetDraft.gamescope_width || 0;
    const h = targetDraft.gamescope_height || 0;
    if (!w && !h) return { preset: 'auto', isCustom: false };
    if (w === 3840 && h === 2160) return { preset: '3840x2160', isCustom: false };
    if (w === 3440 && h === 1440) return { preset: '3440x1440', isCustom: false };
    if (w === 2560 && h === 1440) return { preset: '2560x1440', isCustom: false };
    if (w === 1920 && h === 1080) return { preset: '1920x1080', isCustom: false };
    if (w === 1280 && h === 720) return { preset: '1280x720', isCustom: false };
    return { preset: 'custom', isCustom: true };
  });
  let selectedResPreset = $derived(userSelectedRes ?? computedRes.preset);
  let isCustomRes = $derived(selectedResPreset === 'custom' || computedRes.isCustom);

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
    userSelectedRes = val;
    if (val === 'auto') {
      setDimensions(0, 0);
    } else if (val === 'custom') {
      if (!targetDraft.gamescope_width && !targetDraft.gamescope_height) {
        setDimensions(1920, 1080);
      }
    } else {
      const parts = val.split('x').map(Number);
      setDimensions(parts[0], parts[1]);
      userSelectedRes = null;
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

  let userSelectedFps = $state(null);
  let computedFps = $derived.by(() => {
    const fps = targetDraft.gamescope_fps || 0;
    if ([0, 60, 100, 120, 144, 165, 240].includes(fps)) {
      return { preset: String(fps), isCustom: false };
    }
    return { preset: 'custom', isCustom: true };
  });
  let selectedFpsPreset = $derived(userSelectedFps ?? computedFps.preset);
  let isCustomFps = $derived(selectedFpsPreset === 'custom' || computedFps.isCustom);

  function setFps(fps) {
    if (gamescopeDraft) gamescopeDraft.gamescope_fps = fps;
    if (cfgDraft) cfgDraft.gamescope_fps = fps;
  }

  function handleFpsChange(val) {
    userSelectedFps = val;
    if (val === 'custom') {
      if (!targetDraft.gamescope_fps) setFps(60);
    } else {
      setFps(Number(val));
      userSelectedFps = null;
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

<div class="pt-4 border-t border-surface-border space-y-3">
  <div class="space-y-0.5">
    <h4 class="text-xs font-bold text-neutral-200 uppercase tracking-wider font-mono">
      Gamescope &amp; Screen Resolution
    </h4>
    <p class="text-xs text-neutral-400 leading-relaxed">
      Configures base display resolution and target refresh rate for Gamescope. This resolution is applied 1:1 for single-player launches (when enabled) and automatically partitioned across split-screen co-op seats.
    </p>
  </div>

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
