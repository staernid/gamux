<script>
  import { createEventDispatcher } from 'svelte';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import EnvDiffEditor from '../../common/EnvDiffEditor.svelte';
  import HubActionToolbar from './HubActionToolbar.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';
  import { PLATFORM_OPTIONS } from '../../../constants/options.js';

  export let game = {};
  export let globalCfg = {};
  export let overrideCount = 0;
  export let isSaving = false;
  export let saveSuccess = false;

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
            <PropertySheet
              includeKeys={isWine ? ['runner', 'wine_prefix', 'dll_overrides'] : ['runner', 'dll_overrides']}
              scope="game"
              groupByCategory={false}
              bind:values={$drawerDraft}
              inheritedValues={globalCfg}
            />
          {/if}
        </div>
      </SettingSection>

      <SettingSection title="Graphics & Overlays">
        <PropertySheet
          includeKeys={['mangohud']}
          scope="game"
          groupByCategory={false}
          bind:values={$drawerDraft}
          inheritedValues={globalCfg}
        />
      </SettingSection>

      {#if isSteamGame}
        <SettingSection title="Steam Emulation">
          <PropertySheet
            includeKeys={['operating_mode', 'language', 'offline_mode', 'disable_overlay']}
            scope="game"
            groupByCategory={false}
            bind:values={$drawerDraft}
            inheritedValues={globalCfg}
          />
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
