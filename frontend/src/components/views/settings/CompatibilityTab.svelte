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
    </div>

    <!-- Advanced Compatibility Options -->
    <div class="pt-3 mt-3 border-t border-[#27272a]">
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
                class="px-3 py-1.5 bg-[#18181b] hover:bg-[#222225] border border-[#27272a] text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
              >
                Browse
              </button>
            </GhostInput>
          </div>

          <!-- Global Environment Variables -->
          <div class="pt-2 border-t border-[#27272a] space-y-2">
            <span class="text-xs font-bold text-neutral-200 uppercase tracking-wider block">
              Global Environment Variables
            </span>

            {#if Object.keys(cfgDraft.global_env || {}).length > 0}
              <div class="grid grid-cols-1 md:grid-cols-2 gap-1.5">
                {#each Object.keys(cfgDraft.global_env) as ek}
                  <div class="flex items-center justify-between p-2 bg-[#101013] border border-[#27272a] rounded-lg">
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
                class="flex-1 bg-[#141417] border border-[#27272a] focus:border-neutral-400 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono uppercase outline-none transition-colors"
              />
              <input
                type="text"
                placeholder="value"
                bind:value={newEnvVal}
                on:keydown={(e) => { if (e.key === 'Enter') addGlobalEnv(); }}
                class="flex-1 bg-[#141417] border border-[#27272a] focus:border-neutral-400 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono outline-none transition-colors"
              />
              <button
                type="button"
                on:click={addGlobalEnv}
                class="px-3 py-1.5 bg-[#18181b] hover:bg-[#222225] border border-[#27272a] text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex-shrink-0 cursor-pointer"
              >
                Add
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </SettingSection>
</div>
