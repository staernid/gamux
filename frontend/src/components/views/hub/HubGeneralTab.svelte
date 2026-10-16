<script>
    import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { modalStore } from '../../../stores/modal.js';
  import { getMetadataInfo, hasMetadataMatch, isSamePath } from '../../../utils/game.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SettingRow from '../../common/SettingRow.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import HubActionToolbar from './HubActionToolbar.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   * @property {number} [overrideCount]
   * @property {boolean} [isSaving]
   * @property {boolean} [saveSuccess]
   */

  /** @type {Props} */
  let {
    game = null,
    overrideCount = 0,
    isSaving = false,
    saveSuccess = false
  ,
    onsave,
    onreset,
    onlinkMeta,
    onunlinkMeta
  } = $props();

  let candidates = $derived(game?.launchCandidates || []);
  let metaInfo = $derived(game ? getMetadataInfo(game) : { provider: 'None', id: '' });
  let hasMatch = $derived(game ? hasMetadataMatch(game) : false);

  function handleSave() {
    onsave?.();
  }

  function handleReset() {
    onreset?.();
  }
</script>

<div class="space-y-4">
  <!-- Action Toolbar -->
  <HubActionToolbar
    title="General Configuration"
    {overrideCount}
    {isSaving}
    {saveSuccess}
    onsave={handleSave}
    onreset={handleReset}
  />

  <!-- Tab Content -->
  <div class="bg-surface-card border border-surface-border rounded-xl p-4 sm:p-5 shadow-xl space-y-4">
    <div class="space-y-3">
      <SettingSection title="Game Identification">
        <SettingRow label="Title Override" forId="draftGameName" inline={false}>
          <input
            id="draftGameName"
            type="text"
            bind:value={$drawerDraft.name}
            placeholder={game?.name || "Game Title"}
            class="w-full bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-lg px-3 py-2 text-neutral-100 outline-none text-xs transition-colors"
          />
        </SettingRow>
      </SettingSection>

      <SettingSection title="Identity, Source & Artwork">
        <!-- 3 Pillar Action Cards -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <!-- 1. Physical Source Pillar -->
          <div class="bg-surface-header border border-surface-border p-3.5 rounded-xl flex flex-col justify-between space-y-3">
            <div class="space-y-1">
              <div class="flex items-center justify-between gap-2 min-w-0">
                <span class="text-3xs uppercase font-mono font-bold tracking-wider text-neutral-400 flex items-center gap-1.5 truncate">
                  <svg class="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7" />
                  </svg>
                  Physical Source
                </span>
                <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-cardHover text-neutral-200 border border-surface-border whitespace-nowrap flex-shrink-0">
                  {$drawerDraft.sourceOverride || game?.source || game?.store || 'Standalone'}
                  {#if $drawerDraft.appId || (game?.appId && game.appId !== '0')}
                    <span class="text-neutral-400 font-mono ml-1">#{$drawerDraft.appId || game?.appId}</span>
                  {/if}
                </span>
              </div>
              <p class="text-2xs text-neutral-400 leading-snug">
                File layout, DRM emulation, and runner execution pipeline.
              </p>
            </div>
            <button
              type="button"
              onclick={() => modalStore.open('metadata-artwork', { game, initialTab: 'metadata' })}
              class="w-full py-1.5 px-3 rounded-lg bg-surface-cardHover hover:bg-surface-elevated border border-surface-border text-neutral-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <span>Match Store...</span>
            </button>
          </div>

          <!-- 2. Store Metadata Pillar -->
          <div class="bg-surface-header border border-surface-border p-3.5 rounded-xl flex flex-col justify-between space-y-3">
            <div class="space-y-1">
              <div class="flex items-center justify-between gap-2 min-w-0">
                <span class="text-3xs uppercase font-mono font-bold tracking-wider text-neutral-400 flex items-center gap-1.5 truncate">
                  <svg class="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                  </svg>
                  Store Metadata
                </span>
                {#if hasMatch}
                  <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-cardHover text-neutral-200 border border-surface-border whitespace-nowrap flex-shrink-0">
                    {metaInfo.provider}
                    {#if metaInfo.id}
                      <span class="text-neutral-400 font-mono ml-1">#{metaInfo.id}</span>
                    {/if}
                  </span>
                {:else}
                  <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-cardHover text-neutral-400 border border-surface-border whitespace-nowrap flex-shrink-0">
                    Unlinked
                  </span>
                {/if}
              </div>
              <p class="text-2xs text-neutral-400 leading-snug">
                Game synopsis, release date, developer credits, and patch notes.
              </p>
            </div>
            <button
              type="button"
              onclick={() => modalStore.open('metadata-link', { game })}
              class="w-full py-1.5 px-3 rounded-lg bg-surface-cardHover hover:bg-surface-elevated border border-surface-border text-neutral-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <span>{hasMatch ? 'Change Match...' : 'Match Metadata...'}</span>
            </button>
          </div>

          <!-- 3. Artwork Hub Pillar -->
          <div class="bg-surface-header border border-surface-border p-3.5 rounded-xl flex flex-col justify-between space-y-3">
            <div class="space-y-1">
              <div class="flex items-center justify-between gap-2 min-w-0">
                <span class="text-3xs uppercase font-mono font-bold tracking-wider text-neutral-400 flex items-center gap-1.5 truncate">
                  <svg class="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  Artwork Hub
                </span>
                <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-cardHover text-neutral-300 border border-surface-border whitespace-nowrap flex-shrink-0">
                  {game?.lockedArtwork ? 'Custom' : 'Active'}
                </span>
              </div>
              <p class="text-2xs text-neutral-400 leading-snug">
                Covers (600×900), Hero banners, Grid banners, and Logos.
              </p>
            </div>
            <button
              type="button"
              onclick={() => modalStore.open('artwork-picker', { game, initialTab: 'cover' })}
              class="w-full py-1.5 px-3 rounded-lg bg-surface-cardHover hover:bg-surface-elevated border border-surface-border text-neutral-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span>Open Artwork Hub...</span>
            </button>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Target Executable">
        {#if candidates.length > 0}
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 pb-2">
            {#each candidates as cand, idx}
              {@const isSelected = isSamePath($drawerDraft.exePath, cand.executable)}
              <button
                type="button"
                onclick={() => drawerDraft.selectCandidate(cand)}
                class="p-2.5 rounded-lg border text-left transition-all flex items-start justify-between gap-2 cursor-pointer {isSelected ? 'bg-surface-cardHover border-neutral-300 shadow-md text-white' : 'bg-surface-header border-surface-border hover:border-surface-borderHighlight text-neutral-300'}"
              >
                <div class="space-y-0.5 min-w-0 flex-1">
                  <div class="flex items-center space-x-1.5 flex-wrap">
                    <span class="font-bold text-xs {isSelected ? 'text-white' : 'text-neutral-200'} truncate">
                      {cand.name || cand.executable.split('/').pop()}
                    </span>
                    {#if idx === 0}
                      <span class="px-1.5 py-0.2 rounded text-4xs font-bold bg-neutral-800 text-neutral-200 border border-neutral-700">
                        Primary
                      </span>
                    {/if}
                    {#if cand.arguments}
                      <span class="px-1.5 py-0.2 rounded text-4xs font-mono bg-neutral-900 text-neutral-400 border border-neutral-800 truncate max-w-[120px]" title={cand.arguments}>
                        {cand.arguments}
                      </span>
                    {/if}
                  </div>
                  <p class="font-mono text-3xs text-neutral-400 truncate">
                    {cand.executable}
                  </p>
                </div>

                <div class="w-3.5 h-3.5 rounded-full border flex items-center justify-center flex-shrink-0 mt-0.5 {isSelected ? 'border-white bg-white' : 'border-surface-borderHighlight'}" aria-hidden="true">
                  {#if isSelected}
                    <div class="w-1.5 h-1.5 rounded-full bg-neutral-950"></div>
                  {/if}
                </div>
              </button>
            {/each}
          </div>
        {:else if !$drawerDraft.exePath}
          <div class="mb-3 p-3 rounded-lg border border-amber-900/40 bg-amber-950/20 flex items-start gap-2.5 text-amber-200/90 text-xs">
            <svg class="w-4 h-4 flex-shrink-0 mt-0.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div class="space-y-0.5 min-w-0">
              <p class="font-semibold text-amber-300">No Target Executable Configured</p>
              <p class="text-2xs text-amber-400/80 leading-relaxed">
                Gamux could not detect an authoritative launch executable for this title. Please browse or specify the main game executable below before launching.
              </p>
            </div>
          </div>
        {/if}

        <div class="flex items-center space-x-2 pt-1 pb-3">
          <input
            type="text"
            bind:value={$drawerDraft.exePath}
            placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
            class="flex-1 bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-lg px-3 py-2 text-neutral-100 font-mono text-xs outline-none transition-colors"
          />
          <button
            type="button"
            onclick={async () => { await drawerDraft.browseExecutable(game?.gameDir); }}
            class="bg-surface-cardHover hover:bg-surface-elevated text-neutral-200 px-3.5 py-2 rounded-lg font-bold border border-surface-borderHighlight text-xs flex-shrink-0 cursor-pointer"
          >
            Browse
          </button>
        </div>

        <div class="pt-3 border-t border-surface-border">
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
