<script>
  import { modalStore } from '../../../stores/modal.js';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { getGameSource, getMetadataInfo } from '../../../utils/game.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SettingRow from '../../common/SettingRow.svelte';

  export let game = null;
  export let candidates = [];
  export let onUpdateGame = () => {};

  $: gameSource = game ? getGameSource(game) : 'Standalone';
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };

  let isCustomPath = false;

  $: {
    if ($drawerDraft.exePath) {
      isCustomPath = !candidates.some(c => c.executable === $drawerDraft.exePath);
    }
  }

  function handleSelectExe(exe) {
    isCustomPath = false;
    const found = candidates.find(c => c.executable === exe);
    if (found) {
      drawerDraft.selectCandidate(found);
    } else {
      drawerDraft.setField('exePath', exe);
    }
  }
</script>

<div class="space-y-4">
  <!-- Game Title -->
  <SettingSection title="Game Identification">
    <SettingRow label="Game Title" description="Display name used across your library, hub, and notifications." forId="drawerGameName" inline={false}>
      <input
        id="drawerGameName"
        type="text"
        bind:value={$drawerDraft.name}
        placeholder={game?.name || "Game Title"}
        class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs transition-colors"
      />
    </SettingRow>
  </SettingSection>

  <!-- Executable & Launch Configuration -->
  <SettingSection
    title="Target Executable"
    description="The main executable launched when you click Play."
  >
    <!-- Detected Candidates Cards -->
    <div class="space-y-2">
      {#if candidates.length > 0}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          {#each candidates as cand, idx}
            {@const isSelected = $drawerDraft.exePath === cand.executable}
            <button
              type="button"
              on:click={() => handleSelectExe(cand.executable)}
              class="p-3 rounded-xl border text-left transition-all flex items-start justify-between gap-2 cursor-pointer {isSelected ? 'bg-cyan-950/40 border-cyan-500/70 shadow-lg shadow-cyan-950/30' : 'bg-[#0a0a0f] border-[#1e1e2c] hover:border-[#2a2a3e] text-slate-300'}"
            >
              <div class="space-y-1 min-w-0 flex-1">
                <div class="flex items-center space-x-1.5 flex-wrap">
                  <span class="font-bold text-xs {isSelected ? 'text-white' : 'text-slate-200'} truncate">
                    {cand.name || cand.executable.split('/').pop()}
                  </span>
                  {#if idx === 0}
                    <span class="px-1.5 py-0.2 rounded text-[9px] font-bold bg-cyan-500/20 text-cyan-300 border border-cyan-500/40">
                      Detected Main
                    </span>
                  {/if}
                </div>
                <p class="font-mono text-[10px] text-slate-400 truncate">
                  {cand.executable}
                </p>
              </div>

              <div class="w-4 h-4 rounded-full border flex items-center justify-center flex-shrink-0 mt-0.5 {isSelected ? 'border-cyan-400 bg-cyan-500' : 'border-[#303044]'}" aria-hidden="true">
                {#if isSelected}
                  <div class="w-1.5 h-1.5 rounded-full bg-black"></div>
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}

      <!-- Custom Executable Path Picker -->
      <div class="pt-2">
        <label for="drawerCustomExe" class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-1.5">
          Custom Executable Path
        </label>
        <div class="flex items-center space-x-2">
          <input
            id="drawerCustomExe"
            type="text"
            bind:value={$drawerDraft.exePath}
            placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
            class="flex-1 bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 font-mono text-xs outline-none transition-colors"
          />
          <button
            type="button"
            on:click={async () => { await drawerDraft.browseExecutable(game?.gameDir); }}
            class="bg-[#1c1c28] hover:bg-[#262638] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323246] text-xs flex-shrink-0 cursor-pointer"
          >
            Browse
          </button>
        </div>
      </div>
    </div>

    <!-- Launch Arguments -->
    <div class="pt-2 border-t border-[#1c1c28]">
      <SettingRow label="Launch Arguments" description="Custom command-line flags passed directly to the binary." forId="drawerArgs" inline={false}>
        <input
          id="drawerArgs"
          type="text"
          bind:value={$drawerDraft.exeArgs}
          placeholder="e.g. -dx11 -fullscreen -novid"
          class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 font-mono text-xs outline-none transition-colors"
        />
      </SettingRow>
    </div>
  </SettingSection>

  <!-- Update Available Banner -->
  {#if game?.hasUpdatesOrIssues || game?.hasUpdate}
    <div class="bg-gradient-to-r from-emerald-950/50 to-[#0e1715] border border-emerald-600/50 p-4 rounded-2xl flex items-center justify-between gap-3 shadow-lg">
      <div class="space-y-0.5 min-w-0">
        <strong class="text-emerald-300 font-bold text-xs block">Steam Update or Modified Files Detected</strong>
        <p class="text-[11px] text-slate-400">Download missing or delta depot files and re-apply emulator setup automatically.</p>
      </div>
      <button
        type="button"
        on:click={onUpdateGame}
        class="bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-black text-xs px-4 py-2 rounded-xl transition-colors flex-shrink-0 shadow-lg shadow-emerald-950/40 cursor-pointer"
      >
        Update Game
      </button>
    </div>
  {/if}

  <!-- Source & Metadata Identity Card -->
  <SettingSection title="Source & Metadata Identity">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <div class="flex items-center space-x-2 flex-wrap">
          <span class="px-2 py-0.5 rounded-md bg-[#161622] border border-[#27273a] text-slate-300 text-[10px] font-bold uppercase">
            {gameSource} Source
          </span>
          {#if metaInfo.provider && metaInfo.provider !== 'None'}
            <span class="text-cyan-400 text-xs font-mono font-bold">
              {metaInfo.provider}{metaInfo.id ? ` #${metaInfo.id}` : ''}
            </span>
          {/if}
        </div>
        <p class="text-[11px] text-slate-400">
          Match Steam/SteamGridDB metadata for box art, hero banners, and achievement icons.
        </p>
      </div>

      <div class="flex items-center space-x-2 flex-shrink-0">
        <button
          type="button"
          on:click={() => modalStore.open('metadata-link', { game })}
          class="bg-[#1c1c28] hover:bg-[#262638] text-slate-200 text-xs font-bold px-3.5 py-1.5 rounded-xl border border-[#323246] transition-colors cursor-pointer"
        >
          Link Metadata
        </button>
        <button
          type="button"
          on:click={() => modalStore.open('artwork-picker', { game })}
          class="bg-cyan-950/40 hover:bg-cyan-900/60 text-cyan-300 text-xs font-bold px-3.5 py-1.5 rounded-xl border border-cyan-800/50 transition-colors cursor-pointer"
        >
          Artwork
        </button>
      </div>
    </div>
  </SettingSection>
</div>
