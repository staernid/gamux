<script>
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SettingRow from '../../common/SettingRow.svelte';
  import InheritSwitcher from '../../common/InheritSwitcher.svelte';

  export let game = null;
  export let globalCfg = {};

  $: runnerOptions = [
    { value: '', label: `Default (${globalCfg.default_runner ? globalCfg.default_runner.toUpperCase() : 'UMU'})` },
    { value: 'umu', label: 'UMU (umu-run - Recommended)' },
    { value: 'wine', label: 'Direct Wine' },
    { value: 'proton', label: 'Direct Proton' },
    { value: 'linux', label: 'Linux Native' }
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
</script>

<div class="space-y-4">
  <!-- Platform & Compatibility Tool -->
  {#if game}
    <SettingSection
      title="Platform & Compatibility Tool"
      description="Choose between native Linux binaries and Windows executables running via Proton/Wine."
    >
      <div class="flex items-center justify-between flex-wrap gap-2">
        <span class="text-xs font-bold text-slate-200">
          Target Architecture: <span class="text-cyan-400 font-mono uppercase">{$drawerDraft.platform || game.platform || 'win64'}</span>
        </span>
        <div class="flex items-center space-x-1.5">
          {#each (game.available_platforms || game.availablePlatforms || ['Windows']) as plat}
            <button
              type="button"
              on:click={() => handleSwitchPlatform(plat)}
              class="px-3 py-1 rounded-xl text-xs font-bold transition-all border cursor-pointer {isCurrentPlatform(plat) ? 'bg-cyan-950/80 text-cyan-300 border-cyan-500/60 shadow-sm' : 'bg-[#181824] hover:bg-[#222232] text-slate-400 border-[#2e2e40]'}"
            >
              {plat}
            </button>
          {/each}
        </div>
      </div>
    </SettingSection>
  {/if}

  <!-- Execution & Compatibility -->
  <SettingSection
    title="Execution & Compatibility"
    description="Select compatibility layer runtime and prefix options."
  >
    <SettingRow label="Runner Engine" description="Compatibility runner used to launch Windows executables." forId="drawerRunner" inline={false}>
      <select
        id="drawerRunner"
        bind:value={$drawerDraft.runner}
        class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs"
      >
        {#each runnerOptions as opt}
          <option value={opt.value}>{opt.label}</option>
        {/each}
      </select>
    </SettingRow>

    <SettingRow label="Wine Prefix Path" description="Custom Wine bottle prefix (leave blank for runner default)." forId="drawerPrefix" inline={false}>
      <input
        id="drawerPrefix"
        type="text"
        bind:value={$drawerDraft.winePrefix}
        placeholder={globalCfg.wine_prefix ? `Inherited (${globalCfg.wine_prefix})` : 'Default (Runner-managed prefix)'}
        class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
      />
    </SettingRow>

    <SettingRow label="Wine DLL Overrides (WINEDLLOVERRIDES)" description="DLL load order (e.g. version=n,b;dinput8=n,b)." forId="drawerDllOverrides" inline={false}>
      <input
        id="drawerDllOverrides"
        type="text"
        bind:value={$drawerDraft.dllOverrides}
        placeholder="e.g. version=n,b;dinput8=n,b"
        class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
      />
    </SettingRow>
  </SettingSection>

  <!-- Graphics & Performance Overlays -->
  <SettingSection
    title="Graphics & Performance Overlays"
    description="Per-game overrides for telemetry, spatial upscaling, and DLSS integration."
  >
    <div class="space-y-2">
      <InheritSwitcher
        label="MangoHud Performance HUD"
        description="On-screen FPS, frametime graphs, temperatures, and hardware telemetry overlay."
        tag="HUD"
        color="emerald"
        inheritedValue={!!globalCfg.mangohud}
        bind:value={$drawerDraft.mangoHud}
      />

      <InheritSwitcher
        label="Wine Fullscreen FSR Upscaling"
        description="FidelityFX Super Resolution spatial upscaling for non-native display resolutions."
        tag="FSR"
        color="cyan"
        inheritedValue={!!globalCfg.wine_fullscreen_fsr}
        bind:value={$drawerDraft.wineFullscreenFSR}
      />

      <InheritSwitcher
        label="Proton NVAPI & DLSS Integration"
        description="Expose NVIDIA DLSS, Reflex, and NVAPI libraries through DXVK-NVAPI."
        tag="DLSS"
        color="emerald"
        inheritedValue={!!globalCfg.proton_nvapi}
        bind:value={$drawerDraft.protonNVAPI}
      />
    </div>
  </SettingSection>
</div>
