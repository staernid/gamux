<script>
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import StepperRow from '../../common/StepperRow.svelte';

  export let globalCfg = {};

  $: runnerOptions = [
    { value: '', label: `Default (${globalCfg.default_runner ? globalCfg.default_runner.toUpperCase() : 'UMU'})` },
    { value: 'umu', label: 'UMU (umu-run)' },
    { value: 'wine', label: 'Direct Wine' },
    { value: 'proton', label: 'Direct Proton' },
    { value: 'linux', label: 'Linux Native' }
  ];

  $: triStateOptions = (enabledByDefault) => [
    { value: null, label: `Inherited (${enabledByDefault ? 'On' : 'Off'})` },
    { value: true, label: 'Force Enabled' },
    { value: false, label: 'Force Disabled' }
  ];

  function handleStepperChange() {
    // Modified in-memory draft
  }
</script>

<div class="space-y-3">
  <!-- Primary Execution Configuration -->
  <div class="space-y-2">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      Execution & Compatibility
    </span>

    <StepperRow
      label="Runner Engine"
      options={runnerOptions}
      bind:value={$drawerDraft.runner}
      on:change={handleStepperChange}
    />
  </div>

  <!-- Wine Prefix Path -->
  <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
    <label for="drawerPrefix" class="block font-bold text-slate-200 text-xs">
      Wine Prefix Path
    </label>
    <input
      id="drawerPrefix"
      type="text"
      bind:value={$drawerDraft.winePrefix}
      placeholder="Inherited ({globalCfg.wine_prefix || '~/.local/share/wineprefixes/default'})"
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
    />
  </div>

  <!-- Wine DLL Overrides -->
  <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
    <div class="flex items-center justify-between">
      <label for="drawerDllOverrides" class="block font-bold text-slate-200 text-xs">
        Wine DLL Overrides (WINEDLLOVERRIDES)
      </label>
      <span class="text-[10px] text-slate-500 font-mono">e.g. steam_api64=n,b;d3d11=n,b</span>
    </div>
    <input
      id="drawerDllOverrides"
      type="text"
      bind:value={$drawerDraft.dllOverrides}
      placeholder="e.g. steam_api64=n,b;d3d11=n,b;dxgi=n,b"
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
    />
    <p class="text-[10px] text-slate-400">
      Instruct Wine/Proton to prioritize native (n) or builtin (b) DLL libraries without external process injection.
    </p>
  </div>

  <!-- Performance & Graphics Overlays -->
  <div class="space-y-2 pt-2 border-t border-[#20202c]">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      Graphics & Performance Overlays
    </span>

    <StepperRow
      label="MangoHud"
      options={triStateOptions(!!globalCfg.mangohud)}
      bind:value={$drawerDraft.mangoHud}
      on:change={handleStepperChange}
    />

    <StepperRow
      label="Wine Fullscreen FSR"
      options={triStateOptions(!!globalCfg.wine_fullscreen_fsr)}
      bind:value={$drawerDraft.wineFullscreenFSR}
      on:change={handleStepperChange}
    />

    <StepperRow
      label="Proton NVAPI / DLSS"
      options={triStateOptions(!!globalCfg.proton_nvapi)}
      bind:value={$drawerDraft.protonNVAPI}
      on:change={handleStepperChange}
    />

    <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
      <label for="drawerDxvk" class="block font-bold text-slate-200 text-xs">
        DXVK HUD Flags
      </label>
      <input
        id="drawerDxvk"
        type="text"
        bind:value={$drawerDraft.dxvkHud}
        placeholder="Inherited ({globalCfg.dxvk_hud || 'fps,devinfo'})"
        class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
      />
    </div>
  </div>
</div>
