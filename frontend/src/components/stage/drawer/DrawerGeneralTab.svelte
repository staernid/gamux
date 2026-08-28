<script>
  import { modalStore } from '../../../stores/modal.js';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { getGameSource, getMetadataInfo } from '../../../utils/game.js';
  import StepperRow from '../../common/StepperRow.svelte';

  export let game = null;
  export let candidates = [];
  export let onUpdateGame = () => {};

  $: gameSource = game ? getGameSource(game) : 'Standalone';
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };

  $: candidateOptions = [
    ...candidates.map(c => ({
      value: c.executable,
      label: c.name ? `${c.name} (${c.executable})` : c.executable
    })),
    ...(game?.exePath && !candidates.some(c => c.executable === game.exePath) ? [{ value: game.exePath, label: `${game.exePath} (Detected)` }] : []),
    { value: '__custom__', label: 'Custom Executable Path...' }
  ];

  function handleCandidateSelect(val) {
    if (val === '__custom__') return;
    const found = candidates.find(c => c.executable === val);
    if (found) {
      drawerDraft.selectCandidate(found);
    } else if (val) {
      drawerDraft.setField('exePath', val);
    }
  }
</script>

<div class="space-y-4">
  <!-- Game Title -->
  <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
    <label for="drawerGameName" class="block font-bold text-slate-200 text-xs">
      Game Title
    </label>
    <input
      id="drawerGameName"
      type="text"
      bind:value={$drawerDraft.name}
      placeholder={game?.name || "Game Title"}
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-sans text-xs transition-colors"
    />
  </div>

  <!-- Target Executable -->
  <div class="space-y-2">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      Launch Configuration
    </span>

    <StepperRow
      label="Target Executable"
      options={candidateOptions}
      value={candidates.some(c => c.executable === $drawerDraft.exePath) ? $drawerDraft.exePath : '__custom__'}
      on:change={(e) => handleCandidateSelect(e.detail)}
    />

    <div class="flex items-center space-x-2">
      <input
        type="text"
        bind:value={$drawerDraft.exePath}
        placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
        class="flex-1 bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-xl px-3 py-2 text-slate-100 font-mono text-xs outline-none transition-colors"
      />
      <button
        type="button"
        on:click={async () => { await drawerDraft.browseExecutable(game?.gameDir); }}
        class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244] text-xs flex-shrink-0 cursor-pointer"
      >
        Browse
      </button>
    </div>
  </div>

  <!-- Custom Launch Flags -->
  <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
    <label for="drawerArgs" class="block font-bold text-slate-200 text-xs">
      Launch Flags
    </label>
    <input
      id="drawerArgs"
      type="text"
      bind:value={$drawerDraft.exeArgs}
      placeholder="e.g. -dx11 -fullscreen -novid"
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
    />
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
        class="bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-bold text-xs px-4 py-2 rounded-xl transition-colors flex-shrink-0 shadow-lg shadow-emerald-950/40 cursor-pointer"
      >
        Update Game
      </button>
    </div>
  {/if}

  <!-- Source & Metadata Identity Card -->
  <div class="bg-[#0d0d12] border border-[#222230] p-3.5 rounded-xl flex items-center justify-between gap-3">
    <div class="space-y-0.5 min-w-0">
      <span class="block text-slate-400 font-bold text-[10px] uppercase tracking-wider">Source & Metadata Identity</span>
      <p class="font-mono text-slate-300 text-xs truncate flex items-center gap-2">
        <span class="px-1.5 py-0.5 rounded bg-[#161622] border border-[#27273a] text-slate-300 text-[10px] font-bold uppercase">{gameSource} Source</span>
        {#if metaInfo.provider && metaInfo.provider !== 'None'}
          <span class="text-sky-400 text-[11px] font-semibold">{metaInfo.provider}{metaInfo.id ? ` #${metaInfo.id}` : ''}</span>
        {/if}
      </p>
    </div>
    <div class="flex items-center space-x-2 flex-shrink-0">
      <button
        type="button"
        on:click={() => modalStore.open('metadata-link', { game })}
        class="bg-[#1a1a26] hover:bg-[#252536] text-slate-200 text-[11px] font-bold px-3 py-1.5 rounded-lg border border-[#2e2e42] transition-colors cursor-pointer"
      >
        Link Metadata
      </button>
      <button
        type="button"
        on:click={() => modalStore.open('artwork-picker', { game })}
        class="bg-sky-950/40 hover:bg-sky-900/60 text-sky-300 text-[11px] font-bold px-3 py-1.5 rounded-lg border border-sky-800/50 transition-colors cursor-pointer"
      >
        Artwork
      </button>
    </div>
  </div>
</div>
