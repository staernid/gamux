<script>
  import { drawerDraft, computeEnvEntries } from '../../../stores/drawerDraft.js';

  export let globalCfg = {};

  $: envEntries = computeEnvEntries($drawerDraft.customEnv, globalCfg.global_env || {});
</script>

<div class="space-y-3">
  <!-- Environment Variables Header & Controls -->
  <div class="flex items-center justify-between">
    <div>
      <span class="block text-slate-300 font-bold uppercase tracking-wider text-[10px]">
        Environment Variables
      </span>
      <span class="text-[10px] text-slate-500">
        Global defaults with per-game overrides passed to runners and native binaries.
      </span>
    </div>
    <button
      type="button"
      on:click={() => { drawerDraft.addCustomEnv(globalCfg.global_env || {}); }}
      class="bg-[#20202c] hover:bg-[#2c2c3c] text-white text-[11px] font-bold px-3 py-1.5 rounded-xl border border-[#343446] transition-colors flex items-center space-x-1.5 cursor-pointer"
    >
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
      </svg>
      <span>Add Variable</span>
    </button>
  </div>

  {#if envEntries.length > 0}
    <div class="space-y-2 max-h-96 overflow-y-auto pr-1 custom-scrollbar">
      {#each envEntries as entry}
        <div class="flex items-center space-x-2 bg-[#09090d] p-2.5 rounded-xl border border-[#20202c]">
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
                class="w-full bg-[#07070a] border border-[#1e1e28] rounded-lg px-2.5 py-1.5 text-slate-400 font-mono text-xs cursor-not-allowed"
                title="Inherited from global system settings"
              />
            {:else}
              <input
                type="text"
                value={entry.value}
                on:input={(e) => drawerDraft.updateCustomEnv(entry.key, e.target.value)}
                placeholder="Value"
                class="w-full bg-[#0c0c10] border border-[#262636] rounded-lg px-2.5 py-1.5 text-slate-100 font-mono text-xs outline-none focus:border-sky-400 transition-colors"
              />
            {/if}
          </div>

          <div class="flex items-center space-x-1 flex-shrink-0">
            {#if entry.isInherited}
              <button
                type="button"
                on:click={() => { drawerDraft.overrideEnv(entry.key, entry.value); }}
                class="bg-[#1e1e2a] hover:bg-[#282838] text-slate-300 text-[10px] font-bold px-2.5 py-1.5 rounded-lg border border-[#303044] transition-colors cursor-pointer"
                title="Override for this title"
              >
                Override
              </button>
            {:else if entry.isOverriding}
              <button
                type="button"
                on:click={() => { drawerDraft.revertEnv(entry.key); }}
                class="bg-[#241c20] hover:bg-[#34242c] text-amber-300 text-[10px] font-bold px-2.5 py-1.5 rounded-lg border border-amber-900/50 transition-colors cursor-pointer"
                title="Revert to inherited global value"
              >
                Revert
              </button>
            {:else}
              <button
                type="button"
                on:click={() => { drawerDraft.deleteCustomEnv(entry.key); }}
                class="p-1.5 rounded-lg bg-[#20202c] hover:bg-rose-950/40 text-slate-400 hover:text-rose-400 border border-[#303042] transition-colors cursor-pointer"
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
    <div class="p-4 rounded-xl bg-[#0b0b10] border border-[#20202e] text-center text-slate-500 text-xs">
      No environment variables configured.
    </div>
  {/if}
</div>
