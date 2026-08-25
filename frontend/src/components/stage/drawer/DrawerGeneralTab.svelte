<script>
  import { modalStore } from '../../../stores/modal.js';

  export let game = null;
  export let candidates = [];
  export let customExe = '';
  export let customArgs = '';
  export let envEntries = [];
  export let onSelectCandidate = () => {};
  export let onBrowseExecutable = () => {};
  export let onUpdateGame = () => {};
  export let envActions = {
    add: () => {},
    update: () => {},
    override: () => {},
    revert: () => {},
    delete: () => {}
  };
</script>

<div class="space-y-4">
  <!-- Launch Executable & Candidate Picker -->
  <div class="space-y-2">
    <label for="drawerMainExe" class="block text-slate-300 font-bold uppercase tracking-wider text-[10px]">
      Target Executable
    </label>
    <div class="flex space-x-2">
      <select
        id="drawerMainExe"
        on:change={onSelectCandidate}
        value={candidates.some(c => c.executable === customExe) ? customExe : '__custom__'}
        class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs focus:border-white"
      >
        {#if candidates.length > 0}
          {#each candidates as cand}
            <option value={cand.executable}>
              {cand.name ? `${cand.name} (${cand.executable})` : cand.executable}
            </option>
          {/each}
        {:else if game?.exePath}
          <option value={game.exePath}>{game.exePath} (Auto-detected)</option>
        {/if}
        <option value="__custom__">Custom Executable Path...</option>
      </select>

      <button
        type="button"
        on:click={onBrowseExecutable}
        class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244] text-xs flex-shrink-0"
      >
        Browse
      </button>
    </div>

    <input
      type="text"
      bind:value={customExe}
      placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
      class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-1.5 text-slate-200 font-mono outline-none focus:border-white text-xs"
    />
  </div>

  <!-- Launch Flags -->
  <div>
    <label for="drawerArgs" class="block text-slate-300 font-bold uppercase tracking-wider text-[10px] mb-1">
      Custom Launch Flags
    </label>
    <input
      id="drawerArgs"
      type="text"
      bind:value={customArgs}
      placeholder="-dx11 -fullscreen -novid"
      class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs focus:border-white"
    />
  </div>

  <!-- Directory Information Card -->
  <div class="bg-[#0b0b10] border border-[#202030] p-3 rounded-xl space-y-1">
    <span class="block text-slate-400 font-bold text-[10px] uppercase tracking-wider">Installed Directory</span>
    <p class="font-mono text-slate-300 text-xs truncate">{game?.gameDir || ''}</p>
  </div>

  <!-- Update Available Banner -->
  {#if game?.hasUpdatesOrIssues || game?.hasUpdate}
    <div class="bg-gradient-to-r from-emerald-950/40 to-[#0e1715] border border-emerald-700/50 p-3.5 rounded-xl flex items-center justify-between gap-3">
      <div class="space-y-0.5">
        <strong class="text-emerald-300 font-bold text-xs block">Steam Update or Modified Files Detected</strong>
        <p class="text-[11px] text-slate-400">Download missing or delta depot files and re-apply emulator setup automatically.</p>
      </div>
      <button
        type="button"
        on:click={onUpdateGame}
        class="bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-bold text-xs px-4 py-2 rounded-xl transition-colors flex-shrink-0 shadow-lg shadow-emerald-950/40"
      >
        Update Game
      </button>
    </div>
  {/if}

  <!-- Store & Artwork Identity Card -->
  <div class="bg-[#0b0b10] border border-[#202030] p-3 rounded-xl flex items-center justify-between gap-3">
    <div class="space-y-0.5 min-w-0">
      <span class="block text-slate-400 font-bold text-[10px] uppercase tracking-wider">Store & Artwork Identity</span>
      <p class="font-mono text-slate-300 text-xs truncate">
        {game?.store || 'Steam'} {game?.appId ? `(#${game.appId})` : ''}
      </p>
    </div>
    <div class="flex items-center space-x-2 flex-shrink-0">
      <button
        type="button"
        on:click={() => modalStore.open('metadata-link', { game })}
        class="bg-[#1a1a26] hover:bg-[#252536] text-slate-200 text-[11px] font-bold px-3 py-1.5 rounded-lg border border-[#2e2e42] transition-colors"
      >
        Change Store
      </button>
      <button
        type="button"
        on:click={() => modalStore.open('artwork-picker', { game })}
        class="bg-sky-950/40 hover:bg-sky-900/60 text-sky-300 text-[11px] font-bold px-3 py-1.5 rounded-lg border border-sky-800/50 transition-colors"
      >
        🎨 Change Artwork
      </button>
    </div>
  </div>

  <!-- Environment Variables -->
  <div class="space-y-2 pt-3 border-t border-[#20202c]">
    <div class="flex items-center justify-between">
      <div>
        <span class="block text-slate-300 font-bold uppercase tracking-wider text-[10px]">
          Environment Variables
        </span>
        <span class="text-[10px] text-slate-500">
          Unified hierarchy: Global defaults with per-game overrides.
        </span>
      </div>
      <button
        type="button"
        on:click={envActions.add}
        class="bg-[#20202c] hover:bg-[#2c2c3c] text-white text-[11px] font-bold px-3 py-1 rounded-lg border border-[#343446] transition-colors"
      >
        + Add Variable
      </button>
    </div>

    {#if envEntries.length > 0}
      <div class="space-y-2 max-h-48 overflow-y-auto pr-1">
        {#each envEntries as entry}
          <div class="flex items-center space-x-2 bg-[#09090d] p-2 rounded-xl border border-[#20202c]">
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
                  class="w-full bg-[#07070a] border border-[#1e1e28] rounded-lg px-2 py-1 text-slate-400 font-mono text-xs cursor-not-allowed"
                  title="Inherited from global system settings"
                />
              {:else}
                <input
                  type="text"
                  value={entry.value}
                  on:input={(e) => envActions.update(entry.key, e.target.value)}
                  placeholder="Value"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-lg px-2 py-1 text-slate-100 font-mono text-xs outline-none focus:border-white"
                />
              {/if}
            </div>

            <div class="flex items-center space-x-1 flex-shrink-0">
              {#if entry.isInherited}
                <button
                  type="button"
                  on:click={() => envActions.override(entry.key, entry.value)}
                  class="bg-[#1e1e2a] hover:bg-[#282838] text-slate-300 text-[10px] font-bold px-2 py-1 rounded-md border border-[#303044] transition-colors"
                  title="Override for this title"
                >
                  Override
                </button>
              {:else if entry.isOverriding}
                <button
                  type="button"
                  on:click={() => envActions.revert(entry.key)}
                  class="bg-[#241c20] hover:bg-[#34242c] text-amber-300 text-[10px] font-bold px-2 py-1 rounded-md border border-amber-900/50 transition-colors"
                  title="Revert to inherited global value"
                >
                  Revert
                </button>
              {:else}
                <button
                  type="button"
                  on:click={() => envActions.delete(entry.key)}
                  class="p-1 rounded bg-[#20202c] hover:bg-rose-950/40 text-slate-400 hover:text-rose-400 border border-[#303042] transition-colors"
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
      <div class="p-3 rounded-xl bg-[#0b0b10] border border-[#20202e] text-center text-slate-500 text-xs">
        No environment variables configured.
      </div>
    {/if}
  </div>
</div>
