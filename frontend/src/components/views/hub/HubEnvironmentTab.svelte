<script>
  import { createEventDispatcher } from 'svelte';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import EnvDiffEditor from '../../common/EnvDiffEditor.svelte';
  import HubActionToolbar from './HubActionToolbar.svelte';
  import { RUNNER_OPTIONS, OPERATING_MODE_OPTIONS, PLATFORM_OPTIONS, LANGUAGE_OPTIONS } from '../../../constants/options.js';

  export let game = {};
  export let globalCfg = {};
  export let overrideCount = 0;
  export let isSaving = false;
  export let saveSuccess = false;
  export let runnerOptions = RUNNER_OPTIONS;
  export let operatingModeOptions = OPERATING_MODE_OPTIONS;
  export let gameLanguageOptions = LANGUAGE_OPTIONS;

  const platformOptions = PLATFORM_OPTIONS;

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
  <HubActionToolbar
    title="Runtime & Environment"
    {overrideCount}
    {isSaving}
    {saveSuccess}
    on:save={handleSave}
    on:reset={handleReset}
  />

  <!-- Sub-Tab Content -->
  <div class="bg-surface-card border border-surface-border rounded-xl p-4 sm:p-5 shadow-xl space-y-4">
    <div class="space-y-3">
      <SettingSection title="Runner & Platform">
        <div class="space-y-2.5">
          <InheritSelect
            label="Target Platform"
            bind:value={$drawerDraft.platform}
            options={platformOptions}
          />

          {#if isLinux}
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-2.5 sm:p-3 rounded-xl bg-surface-header border border-surface-border opacity-60">
              <div class="space-y-0.5">
                <span class="text-xs font-semibold text-neutral-200">Runner Layer</span>
                <p class="text-2xs text-neutral-400">Native direct execution on host Linux system</p>
              </div>
              <span class="text-xs font-mono text-neutral-400 px-2.5 py-1 bg-surface-input border border-surface-border rounded-lg">
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

            <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5 pt-2 border-t border-surface-border">
              <SeamlessSwitch
                label="Offline Mode"
                description="Report disconnected offline status in Steamworks emulation"
                bind:value={$drawerDraft.offlineMode}
                inheritedValue={Boolean(globalCfg?.offline_mode)}
              />

              <SeamlessSwitch
                label="Disable In-Game Overlay"
                description="Disable Goldberg Steam overlay hooks and telemetry"
                bind:value={$drawerDraft.disableOverlay}
                inheritedValue={Boolean(globalCfg?.disable_overlay)}
              />
            </div>
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
