<script>
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import StepperRow from '../../common/StepperRow.svelte';

  export let game = null;
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

  function isCurrentPlatform(plat) {
    const current = ($drawerDraft.platform || game?.platform || 'win64').toLowerCase();
    const target = String(plat || '').toLowerCase();
    if (target.includes('win') && current.includes('win')) return true;
    if (target.includes('linux') && current.includes('linux')) return true;
    if (target.includes('mac') && current.includes('mac')) return true;
    return current === target;
  }

  function handleSwitchPlatform(plat) {
    const target = plat.toLowerCase().includes('linux') ? 'linux' : (plat.toLowerCase().includes('mac') ? 'macos' : 'win64');
    drawerDraft.setField('platform', target);
    if (target === 'linux') {
      drawerDraft.setField('runner', 'linux');
    } else if ($drawerDraft.runner === 'linux' || !$drawerDraft.runner) {
      drawerDraft.setField('runner', 'umu');
    }
    notificationStore.items.add({
      type: 'info',
      title: 'Target Platform Switched',
      message: `Configured platform to ${plat}. Click Save to apply launch settings.`
    });
  }

  function handleStepperChange() {
    // Modified in-memory draft
  }
</script>

<div class="space-y-3">
  <!-- Platform & Compatibility Switcher -->
  {#if game}
    <div class="p-3.5 bg-[#0d0d12] border border-[#222230] rounded-xl space-y-2.5">
      <div class="flex items-center justify-between flex-wrap gap-2">
        <div>
          <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
            Platform & Compatibility Tool
          </span>
          <span class="text-slate-100 font-bold text-xs">
            Active: <span class="text-sky-400 font-mono uppercase">{$drawerDraft.platform || game.platform || 'win64'}</span>
          </span>
        </div>
        <div class="flex items-center space-x-1.5 flex-wrap gap-1">
          {#each (game.available_platforms || game.availablePlatforms || ['Windows']) as plat}
            <button
              type="button"
              on:click={() => handleSwitchPlatform(plat)}
              class="px-2.5 py-1 rounded-lg text-[11px] font-bold transition-all border {isCurrentPlatform(plat) ? 'bg-sky-950/80 text-sky-300 border-sky-600/60 shadow-sm' : 'bg-[#181822] hover:bg-[#222232] text-slate-400 border-[#2e2e42]'}"
            >
              {plat}
            </button>
          {/each}
        </div>
      </div>
      <p class="text-[11px] text-slate-400 leading-relaxed">
        Steam compatibility layer. Choose between native Linux executables and Windows binaries running through UMU / Wine / Proton.
      </p>
    </div>
  {/if}

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
      placeholder={globalCfg.wine_prefix ? `Inherited (${globalCfg.wine_prefix})` : 'Default (Runner-managed prefix or ~/.wine)'}
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
    />
  </div>

  <!-- Wine DLL Overrides -->
  <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
    <label for="drawerDllOverrides" class="block font-bold text-slate-200 text-xs">
      Wine DLL Overrides (WINEDLLOVERRIDES)
    </label>
    <input
      id="drawerDllOverrides"
      type="text"
      bind:value={$drawerDraft.dllOverrides}
      placeholder="e.g. version=n,b;dinput8=n,b"
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
    />
    <p class="text-[10px] text-slate-400">
      Specify custom DLL overrides for mods or native libraries (e.g. version=n,b). Base graphics and Goldberg emulators are configured automatically.
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
  </div>
</div>
