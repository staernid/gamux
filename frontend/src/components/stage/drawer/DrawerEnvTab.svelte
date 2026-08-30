<script>
  import { drawerDraft, computeEnvEntries } from '../../../stores/drawerDraft.js';
  import SettingSection from '../../common/SettingSection.svelte';

  export let globalCfg = {};

  $: envEntries = computeEnvEntries($drawerDraft.customEnv, globalCfg.global_env || {});

  const presetChips = [
    { key: 'MANGOHUD', value: '1', label: '+ MANGOHUD=1' },
    { key: 'PROTON_LOG', value: '1', label: '+ PROTON_LOG=1' },
    { key: 'PROTON_ENABLE_NVAPI', value: '1', label: '+ PROTON_ENABLE_NVAPI=1' },
    { key: 'DXVK_ASYNC', value: '1', label: '+ DXVK_ASYNC=1' },
    { key: 'WINE_FULLSCREEN_FSR', value: '1', label: '+ WINE_FULLSCREEN_FSR=1' },
    { key: 'DXVK_HUD', value: 'fps,frametimes', label: '+ DXVK_HUD=fps' }
  ];

  function applyPreset(key, val) {
    drawerDraft.updateCustomEnv(key, val);
  }
</script>

<div class="space-y-4">
  <!-- Quick Preset Chips -->
  <SettingSection
    title="Common Gaming Presets"
    description="One-click flags to inject common Proton, Wine, and DXVK environment variables."
  >
    <div class="flex items-center space-x-2 flex-wrap gap-y-2">
      {#each presetChips as chip}
        <button
          type="button"
          on:click={() => applyPreset(chip.key, chip.value)}
          class="px-2.5 py-1.5 rounded-xl bg-[#14141e] hover:bg-[#1e1e2c] border border-[#262638] hover:border-cyan-500/50 text-slate-300 hover:text-cyan-300 font-mono text-[11px] font-bold transition-all flex items-center space-x-1 cursor-pointer active:scale-95 shadow-sm"
        >
          <span>{chip.label}</span>
        </button>
      {/each}
    </div>
  </SettingSection>

  <!-- Environment Variables List -->
  <SettingSection
    title="Environment Variables"
    description="Global defaults and per-game overrides passed to runners and native executables."
  >
    <svelte:fragment slot="action">
      <button
        type="button"
        on:click={() => { drawerDraft.addCustomEnv(globalCfg.global_env || {}); }}
        class="bg-[#1c1c28] hover:bg-[#262638] text-white text-xs font-bold px-3 py-1.5 rounded-xl border border-[#323246] transition-colors flex items-center space-x-1.5 cursor-pointer"
      >
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>Add Variable</span>
      </button>
    </svelte:fragment>

    {#if envEntries.length > 0}
      <div class="space-y-2 max-h-96 overflow-y-auto pr-1 custom-scrollbar">
        {#each envEntries as entry}
          <div class="flex items-center space-x-2 bg-[#0a0a0f] p-2.5 rounded-xl border border-[#1e1e2c] hover:border-[#2a2a3e] transition-colors">
            <div class="w-1/3 flex items-center space-x-1.5 min-w-0">
              <span class="font-mono text-xs font-bold text-slate-200 truncate" title={entry.key}>{entry.key}</span>
              {#if entry.isInherited}
                <span class="text-[9px] bg-slate-800 text-slate-400 border border-slate-700 px-1.5 py-0.5 rounded font-mono flex-shrink-0">
                  Inherited
                </span>
              {:else if entry.isOverriding}
                <span class="text-[9px] bg-amber-500/20 text-amber-300 border border-amber-500/40 px-1.5 py-0.5 rounded font-mono flex-shrink-0">
                  Override
                </span>
              {/if}
            </div>

            <div class="flex-1 min-w-0">
              {#if entry.isInherited}
                <input
                  type="text"
                  value={entry.value}
                  disabled
                  class="w-full bg-[#07070a] border border-[#181822] rounded-xl px-3 py-1.5 text-slate-400 font-mono text-xs cursor-not-allowed"
                  title="Inherited from global system settings"
                />
              {:else}
                <input
                  type="text"
                  value={entry.value}
                  on:input={(e) => drawerDraft.updateCustomEnv(entry.key, e.target.value)}
                  placeholder="Value"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-1.5 text-slate-100 font-mono text-xs outline-none transition-colors"
                />
              {/if}
            </div>

            <div class="flex items-center space-x-1 flex-shrink-0">
              {#if entry.isInherited}
                <button
                  type="button"
                  on:click={() => { drawerDraft.overrideEnv(entry.key, entry.value); }}
                  class="bg-[#1c1c28] hover:bg-[#262638] text-slate-300 text-[11px] font-bold px-2.5 py-1.5 rounded-lg border border-[#303044] transition-colors cursor-pointer"
                  title="Override for this title"
                >
                  Override
                </button>
              {:else if entry.isOverriding}
                <button
                  type="button"
                  on:click={() => { drawerDraft.revertEnv(entry.key); }}
                  class="bg-[#241c20] hover:bg-[#34242c] text-amber-300 text-[11px] font-bold px-2.5 py-1.5 rounded-lg border border-amber-900/50 transition-colors cursor-pointer"
                  title="Revert to inherited global value"
                >
                  Revert
                </button>
              {:else}
                <button
                  type="button"
                  on:click={() => { drawerDraft.deleteCustomEnv(entry.key); }}
                  class="p-1.5 rounded-lg bg-[#181824] hover:bg-rose-950/40 text-slate-400 hover:text-rose-400 border border-[#28283a] transition-colors cursor-pointer"
                  title="Delete Variable"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="p-4 rounded-xl bg-[#0a0a0f] border border-[#1e1e2c] text-center text-slate-500 text-xs font-mono">
        No environment variables configured. Click "+ Add Variable" or select a preset above.
      </div>
    {/if}
  </SettingSection>
</div>
