<script>
  import { createEventDispatcher } from 'svelte';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import EnvDiffEditor from '../../common/EnvDiffEditor.svelte';

  export let game = {};
  export let globalCfg = {};
  export let overrideCount = 0;
  export let isSaving = false;
  export let saveSuccess = false;
  export let runnerOptions = [
    { value: 'umu', label: 'UMU' },
    { value: 'wine', label: 'Wine' }
  ];
  export let operatingModeOptions = [
    { value: 'loader', label: 'Loader Mode (Clean / Non-Invasive)' },
    { value: 'portable', label: 'Portable Mode (Injected DLL Replacement)' },
    { value: 'none', label: 'DRM-Free / None' }
  ];
  export let gameLanguageOptions = [];

  const platformOptions = [
    { value: 'win64', label: 'Windows' },
    { value: 'linux', label: 'Linux (Native)' }
  ];

  const dispatch = createEventDispatcher();

  function handleSave() {
    dispatch('save');
  }

  function handleReset() {
    dispatch('reset');
  }

  $: isLinux = $drawerDraft.platform === 'linux';
  $: effectiveRunner = isLinux ? 'linux' : ($drawerDraft.runner || game?.runner || globalCfg?.default_runner || 'umu');
  $: isWine = !isLinux && effectiveRunner === 'wine';
  $: isSteamGame = Boolean(
    (game?.store && game.store.toLowerCase() === 'steam') ||
    (game?.source && game.source.toLowerCase() === 'steam') ||
    ($drawerDraft.appId && (!game?.store || game.store.toLowerCase() === 'steam'))
  );
</script>

<div class="space-y-4">
  <!-- Action Toolbar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 bg-[#121218] border border-[#222232] rounded-xl p-2.5 shadow-xl">
    <div class="flex items-center space-x-2 px-2">
      <span class="text-xs font-bold text-slate-200">Runtime & Environment</span>
    </div>

    <!-- Status Badge & Save Actions -->
    <div class="flex items-center space-x-2 flex-shrink-0">
      {#if overrideCount > 0}
        <span class="px-2 py-0.5 rounded-lg text-xs font-mono font-bold bg-sky-500/15 text-sky-300 border border-sky-500/30 flex items-center space-x-1">
          <span class="w-1.5 h-1.5 rounded-full bg-sky-400"></span>
          <span>{overrideCount} Override{overrideCount === 1 ? '' : 's'}</span>
        </span>
      {:else}
        <span class="px-2 py-0.5 rounded-lg text-xs font-mono text-slate-400 bg-[#0c0c10] border border-[#1e1e2c]">
          Inheriting Defaults
        </span>
      {/if}

      {#if saveSuccess}
        <span class="text-xs font-mono font-bold text-emerald-400 flex items-center space-x-1 animate-fade-in">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <span>Saved</span>
        </span>
      {/if}

      <button
        type="button"
        on:click={handleReset}
        class="p-1.5 bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-400 hover:text-white rounded-lg text-xs transition-colors cursor-pointer"
        title="Reset all settings to system defaults"
        aria-label="Reset all settings to system defaults"
      >
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
      </button>

      <button
        type="button"
        on:click={handleSave}
        disabled={isSaving}
        class="px-3.5 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs shadow-md transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isSaving}
          <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Saving...</span>
        {:else}
          <span>Save</span>
        {/if}
      </button>
    </div>
  </div>

  <!-- Sub-Tab Content -->
  <div class="bg-[#121218] border border-[#222232] rounded-xl p-4 sm:p-5 shadow-xl space-y-4">
    <div class="space-y-3">
      <SettingSection title="Runner & Platform">
        <div class="space-y-2.5">
          <InheritSelect
            label="Target Platform"
            bind:value={$drawerDraft.platform}
            options={platformOptions}
          />

          {#if isLinux}
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-2.5 sm:p-3 rounded-xl bg-[#0d0d14] border border-[#1e1e2c] opacity-60">
              <div class="space-y-0.5">
                <span class="text-xs font-semibold text-slate-200">Runner Layer</span>
                <p class="text-[11px] text-slate-400">Native direct execution on host Linux system</p>
              </div>
              <span class="text-xs font-mono text-slate-400 px-2.5 py-1 bg-[#14141e] border border-[#262638] rounded-lg">
                Linux Native
              </span>
            </div>
          {:else}
            <InheritSelect
              label="Runner Layer"
              bind:value={$drawerDraft.runner}
              globalValue={game?.runner || globalCfg?.default_runner || 'umu'}
              options={runnerOptions}
            />

            {#if isWine}
              <GhostInput
                label="Wine Prefix"
                description="Custom isolated Wine bottle location"
                bind:value={$drawerDraft.winePrefix}
                globalPlaceholder={globalCfg?.wine_prefix || "Isolated: steam_settings/pfx"}
                mono={true}
              />
            {/if}

            <GhostInput
              label="DLL Overrides"
              description="Custom native/builtin DLL loading rules (WINEDLLOVERRIDES)"
              bind:value={$drawerDraft.dllOverrides}
              globalPlaceholder="e.g. version=n,b;dinput8=n,b;dxgi=n,b"
              mono={true}
            />
          {/if}
        </div>
      </SettingSection>

      <SettingSection title="Graphics & Overlays">
        <div class="space-y-2.5">
          <SeamlessSwitch
            label="MangoHud Overlay"
            description="Display real-time FPS, frame timing, and GPU/CPU telemetry HUD"
            bind:value={$drawerDraft.mangoHud}
            inheritedValue={Boolean(globalCfg?.mangohud)}
          />
        </div>
      </SettingSection>

      {#if isSteamGame}
        <SettingSection title="Steam Emulation">
          <div class="space-y-2.5">
            <InheritSelect
              label="Operating Mode"
              bind:value={$drawerDraft.operatingMode}
              globalValue={globalCfg?.operating_mode || 'loader'}
              options={operatingModeOptions}
            />

            <InheritSelect
              label="Game Language"
              bind:value={$drawerDraft.language}
              globalValue={globalCfg?.language || 'auto'}
              options={gameLanguageOptions}
            />
          </div>
        </SettingSection>
      {/if}

      <SettingSection title="Environment Variables">
        <EnvDiffEditor
          globalEnv={globalCfg?.global_env || {}}
          bind:customEnv={$drawerDraft.customEnv}
        />
      </SettingSection>
    </div>
  </div>
</div>
