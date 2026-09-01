<script>
  import { createEventDispatcher } from 'svelte';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { modalStore } from '../../../stores/modal.js';
  import { getMetadataInfo, hasMetadataMatch } from '../../../utils/game.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SettingRow from '../../common/SettingRow.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import GameSourceWizard from './GameSourceWizard.svelte';
  import GameMetadataWizard from './GameMetadataWizard.svelte';

  export let game = null;
  export let overrideCount = 0;
  export let isSaving = false;
  export let saveSuccess = false;

  const dispatch = createEventDispatcher();

  let showSourceWizard = false;
  let showMetadataWizard = false;

  $: candidates = game?.launchCandidates || [];
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };
  $: hasMatch = game ? hasMetadataMatch(game) : false;

  function handleSave() {
    dispatch('save');
  }

  function handleReset() {
    dispatch('reset');
  }

  function handleLinkMeta(e) {
    dispatch('linkMeta', e.detail);
    showMetadataWizard = false;
  }

  function handleUnlinkMeta() {
    dispatch('unlinkMeta');
    showMetadataWizard = false;
  }

  function toggleSourceWizard() {
    showSourceWizard = !showSourceWizard;
    if (showSourceWizard) showMetadataWizard = false;
  }

  function toggleMetadataWizard() {
    showMetadataWizard = !showMetadataWizard;
    if (showMetadataWizard) showSourceWizard = false;
  }
</script>

<div class="space-y-4">
  <!-- Action Toolbar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 bg-[#121218] border border-[#222232] rounded-xl p-2.5 shadow-xl">
    <div class="flex items-center space-x-2 px-2">
      <span class="text-xs font-bold text-slate-200">General Configuration</span>
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

  <!-- Tab Content -->
  <div class="bg-[#121218] border border-[#222232] rounded-xl p-4 sm:p-5 shadow-xl space-y-4">
    <div class="space-y-3">
      <SettingSection title="Game Identification">
        <SettingRow label="Title Override" forId="draftGameName" inline={false}>
          <input
            id="draftGameName"
            type="text"
            bind:value={$drawerDraft.name}
            placeholder={game?.name || "Game Title"}
            class="w-full bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-lg px-3 py-2 text-slate-100 outline-none text-xs transition-colors"
          />
        </SettingRow>
      </SettingSection>

      <SettingSection title="Identity, Source & Artwork">
        <!-- 3 Pillar Action Cards -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <!-- 1. Physical Source Pillar -->
          <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 {showSourceWizard ? 'border-cyan-500/50 bg-[#12121c]' : ''}">
            <div class="space-y-1">
              <div class="flex items-center justify-between">
                <span class="text-[10px] uppercase font-mono font-bold tracking-wider text-slate-400 flex items-center gap-1.5">
                  <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7" />
                  </svg>
                  Physical Source & Store
                </span>
                <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-slate-800 text-slate-200 border border-slate-700">
                  {$drawerDraft.sourceOverride || game?.source || game?.store || 'Standalone'}
                  {#if $drawerDraft.appId || (game?.appId && game.appId !== '0')}
                    <span class="text-cyan-400 font-mono ml-1">#{$drawerDraft.appId || game?.appId}</span>
                  {/if}
                </span>
              </div>
              <p class="text-[11px] text-slate-400 leading-snug">
                File layout, DRM emulation, and runner execution pipeline.
              </p>
            </div>
            <button
              type="button"
              on:click={toggleSourceWizard}
              class="w-full py-1.5 px-3 rounded-lg bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <span>{showSourceWizard ? 'Close Matcher' : (($drawerDraft.appId || (game?.appId && game.appId !== '0')) ? 'Match Store...' : 'Match Store...')}</span>
            </button>
          </div>

          <!-- 2. Store Metadata Pillar -->
          <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 {showMetadataWizard ? 'border-purple-500/50 bg-[#12121c]' : ''}">
            <div class="space-y-1">
              <div class="flex items-center justify-between">
                <span class="text-[10px] uppercase font-mono font-bold tracking-wider text-slate-400 flex items-center gap-1.5">
                  <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                  </svg>
                  Store Metadata
                </span>
                {#if hasMatch}
                  <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-sky-500/15 text-sky-300 border border-sky-500/30">
                    {metaInfo.provider} {metaInfo.id ? '#' + metaInfo.id : ''}
                  </span>
                {:else}
                  <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-amber-500/15 text-amber-300 border border-amber-500/30">
                    Unlinked
                  </span>
                {/if}
              </div>
              <p class="text-[11px] text-slate-400 leading-snug">
                Game synopsis, release date, developer credits, and patch notes.
              </p>
            </div>
            <button
              type="button"
              on:click={toggleMetadataWizard}
              class="w-full py-1.5 px-3 rounded-lg bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <span>{showMetadataWizard ? 'Close Matcher' : (hasMatch ? 'Change Match...' : 'Match Metadata...')}</span>
            </button>
          </div>

          <!-- 3. Artwork Hub Pillar -->
          <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col justify-between space-y-3">
            <div class="space-y-1">
              <div class="flex items-center justify-between">
                <span class="text-[10px] uppercase font-mono font-bold tracking-wider text-slate-400 flex items-center gap-1.5">
                  <svg class="w-3.5 h-3.5 text-pink-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  Artwork Hub
                </span>
                <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-pink-500/15 text-pink-300 border border-pink-500/30">
                  {game?.lockedArtwork ? 'Custom' : 'Active'}
                </span>
              </div>
              <p class="text-[11px] text-slate-400 leading-snug">
                Covers (600×900), Hero banners, Grid banners, and Logos.
              </p>
            </div>
            <button
              type="button"
              on:click={() => modalStore.open('artwork-picker', { game })}
              class="w-full py-1.5 px-3 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-bold transition shadow flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span>Open Artwork Hub...</span>
            </button>
          </div>
        </div>

        {#if showSourceWizard}
          <GameSourceWizard {game} on:close={() => showSourceWizard = false} />
        {/if}

        {#if showMetadataWizard}
          <GameMetadataWizard
            {game}
            on:linkMeta={handleLinkMeta}
            on:unlinkMeta={handleUnlinkMeta}
            on:close={() => showMetadataWizard = false}
          />
        {/if}
      </SettingSection>

      <SettingSection title="Target Executable">
        {#if candidates.length > 0}
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 pb-2">
            {#each candidates as cand, idx}
              {@const isSelected = $drawerDraft.exePath === cand.executable}
              <button
                type="button"
                on:click={() => drawerDraft.selectCandidate(cand)}
                class="p-2.5 rounded-lg border text-left transition-all flex items-start justify-between gap-2 cursor-pointer {isSelected ? 'bg-sky-950/40 border-sky-500/70 shadow-sm shadow-sky-950/30' : 'bg-[#0d0d14] border-[#1e1e2c] hover:border-[#2a2a3e] text-slate-300'}"
              >
                <div class="space-y-0.5 min-w-0 flex-1">
                  <div class="flex items-center space-x-1.5 flex-wrap">
                    <span class="font-bold text-xs {isSelected ? 'text-white' : 'text-slate-200'} truncate">
                      {cand.name || cand.executable.split('/').pop()}
                    </span>
                    {#if idx === 0}
                      <span class="px-1.5 py-0.2 rounded text-[9px] font-bold bg-sky-500/20 text-sky-300 border border-sky-500/40">
                        Primary
                      </span>
                    {/if}
                  </div>
                  <p class="font-mono text-[10px] text-slate-400 truncate">
                    {cand.executable}
                  </p>
                </div>

                <div class="w-3.5 h-3.5 rounded-full border flex items-center justify-center flex-shrink-0 mt-0.5 {isSelected ? 'border-sky-400 bg-sky-500' : 'border-[#303044]'}" aria-hidden="true">
                  {#if isSelected}
                    <div class="w-1.5 h-1.5 rounded-full bg-black"></div>
                  {/if}
                </div>
              </button>
            {/each}
          </div>
        {/if}

        <div class="flex items-center space-x-2 pt-1 pb-3">
          <input
            type="text"
            bind:value={$drawerDraft.exePath}
            placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
            class="flex-1 bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-lg px-3 py-2 text-slate-100 font-mono text-xs outline-none transition-colors"
          />
          <button
            type="button"
            on:click={async () => { await drawerDraft.browseExecutable(game?.gameDir); }}
            class="bg-[#1c1c28] hover:bg-[#262638] text-slate-200 px-3.5 py-2 rounded-lg font-bold border border-[#323246] text-xs flex-shrink-0 cursor-pointer"
          >
            Browse
          </button>
        </div>

        <div class="pt-3 border-t border-[#1a1a26]">
          <GhostInput
            label="Launch Arguments"
            description="Command-line flags passed directly to game executable"
            bind:value={$drawerDraft.exeArgs}
            globalPlaceholder="e.g. -dx11 -fullscreen -novid"
            mono={true}
          />
        </div>
      </SettingSection>
    </div>
  </div>
</div>
