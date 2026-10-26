<script>
  import { onMount } from 'svelte';
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { modalStore } from '../../../stores/modal.js';
  import { AppAPI } from '../../../api/app.js';
  import { getMetadataInfo, hasMetadataMatch, isSamePath } from '../../../utils/game.js';
  import SettingRow from '../../common/SettingRow.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';
  import EnvDiffEditor from '../../common/EnvDiffEditor.svelte';
  import { PLATFORM_OPTIONS } from '../../../constants/options.js';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').Game} [game]
   * @property {Record<string, unknown>} [globalCfg]
   * @property {number} [overrideCount]
   * @property {boolean} [isSaving]
   * @property {boolean} [saveSuccess]
   */

  /** @type {Props} */
  let {
    game = null,
    globalCfg = {},
    overrideCount = 0,
    isSaving = false,
    saveSuccess = false
  ,
    onreset = null,
    onlinkMeta = null,
    onunlinkMeta = null
  } = $props();
  const platformOptions = PLATFORM_OPTIONS;

  let filterOnlyOverrides = $state(false);
  let showPreview = $state(false);
  let launchPreview = $state(null);
  let isLoadingPreview = $state(false);
  let previewCopied = $state(false);
  let previewCopiedTimer = null;

  let candidates = $derived(game?.launchCandidates || []);
  let metaInfo = $derived(game ? getMetadataInfo(game) : { provider: 'None', id: '' });
  let hasMatch = $derived(game ? hasMetadataMatch(game) : false);
  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);

  let isLinux = $derived($drawerDraft.platform === 'linux');
  let effectiveRunner = $derived(isLinux ? 'linux' : ($drawerDraft.runner || game?.runner || globalCfg?.default_runner || 'umu'));
  let isWine = $derived(!isLinux && effectiveRunner === 'wine');
  let isSteamGame = $derived(Boolean(
    (game?.store && game.store.toLowerCase() === 'steam') ||
    (game?.source && game.source.toLowerCase() === 'steam') ||
    ($drawerDraft.appId && (!game?.store || game.store.toLowerCase() === 'steam'))
  ));

  async function loadPreview() {
    if (!game?.gameDir) return;
    isLoadingPreview = true;
    try {
      const res = await AppAPI.previewGameLaunch(game.gameDir);
      if (res) {
        launchPreview = res;
      }
    } catch {
      // Quiet fallback
    } finally {
      isLoadingPreview = false;
    }
  }

  let previewDebounce = null;
  function schedulePreviewUpdate() {
    if (previewDebounce) clearTimeout(previewDebounce);
    previewDebounce = setTimeout(() => {
      loadPreview();
    }, 300);
  }

  // Debounce launch preview recalculation when draft config changes
  $effect(() => {
    if ($drawerDraft && game?.gameDir) {
      schedulePreviewUpdate();
    }
  });

  onMount(() => {
    loadPreview();
  });

  function handleResetAll() {
    onreset?.();
    schedulePreviewUpdate();
  }

  async function copyLaunchCommand() {
    if (!launchPreview?.command) return;
    try {
      await navigator.clipboard.writeText(launchPreview.command);
      previewCopied = true;
      if (previewCopiedTimer) clearTimeout(previewCopiedTimer);
      previewCopiedTimer = setTimeout(() => {
        previewCopied = false;
      }, 2000);
    } catch {
      // Fallback
    }
  }
</script>

<div class="space-y-6 animate-fade-in text-neutral-200">
  {#if !isInstalled}
    <div class="p-3.5 bg-surface-card border border-surface-border rounded-2xl flex items-center space-x-3 text-xs text-neutral-300 shadow-sm">
      <span class="w-2 h-2 rounded-full bg-sky-400 flex-shrink-0 animate-pulse"></span>
      <p class="leading-relaxed">
        <strong class="text-white font-bold">Pre-installation Configuration:</strong> Customize runner, wine prefix, launch arguments, or environment variables now. These options will be applied when the game is installed and launched.
      </p>
    </div>
  {/if}

  <!-- Inheritance Status & Auto-Save Banner -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-4 sm:p-5 shadow-sm space-y-3">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <!-- Status Badge & Description -->
      <div class="flex items-center space-x-3 min-w-0">
        <div class="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 {overrideCount > 0 ? 'bg-neutral-800 border border-neutral-600 text-white' : 'bg-emerald-950/40 border border-emerald-800/40 text-emerald-400'}">
          {#if overrideCount > 0}
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
            </svg>
          {:else}
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          {/if}
        </div>
        <div class="space-y-0.5 min-w-0">
          <div class="flex items-center space-x-2 flex-wrap">
            <h3 class="font-bold text-white text-sm">
              {#if overrideCount > 0}
                {overrideCount} Custom Override{overrideCount > 1 ? 's' : ''} Active
              {:else}
                Inheriting System Defaults
              {/if}
            </h3>
            {#if isSaving}
              <span class="text-2xs font-mono text-neutral-400 flex items-center space-x-1 animate-pulse">
                <svg class="animate-spin w-3 h-3" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Saving...</span>
              </span>
            {:else if saveSuccess}
              <span class="text-2xs font-mono font-semibold text-emerald-400 flex items-center space-x-1 animate-fade-in">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                <span>Saved</span>
              </span>
            {/if}
          </div>
          <p class="text-xs text-neutral-400 leading-relaxed truncate">
            {#if overrideCount > 0}
              Game overrides take precedence over global settings. Changes save automatically.
            {:else}
              Running on global defaults (Runner: <span class="font-mono text-neutral-300">{effectiveRunner}</span> · Mode: <span class="font-mono text-neutral-300">{$drawerDraft.operatingMode || 'loader'}</span>).
            {/if}
          </p>
        </div>
      </div>

      <!-- Controls & Filter Toolbar -->
      <div class="flex items-center space-x-2 flex-shrink-0">
        {#if overrideCount > 0}
          <button
            type="button"
            onclick={() => { filterOnlyOverrides = !filterOnlyOverrides; }}
            class="px-3 py-1.5 rounded-lg text-xs font-bold border transition cursor-pointer flex items-center space-x-1.5 {filterOnlyOverrides ? 'bg-neutral-800 border-neutral-500 text-white' : 'bg-surface-header border-surface-border text-neutral-400 hover:text-white'}"
          >
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
            </svg>
            <span>{filterOnlyOverrides ? 'Showing Overrides' : 'Filter Overrides'}</span>
          </button>

          <button
            type="button"
            onclick={handleResetAll}
            class="px-3 py-1.5 rounded-lg text-xs font-bold bg-neutral-900 hover:bg-neutral-800 border border-neutral-700 text-neutral-300 hover:text-white transition cursor-pointer flex items-center space-x-1.5"
            title="Reset all settings for this title back to global defaults"
          >
            <span>↺</span>
            <span>Reset All</span>
          </button>
        {/if}
      </div>
    </div>

    <!-- Effective Execution Dry Run Card -->
    <div class="pt-3 border-t border-surface-border">
      <button
        type="button"
        class="w-full flex items-center justify-between pb-1.5 cursor-pointer select-none text-left"
        onclick={() => { showPreview = !showPreview; }}
      >
        <span class="text-3xs uppercase font-mono font-bold tracking-wider text-neutral-400 flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          Effective Launch Command (Dry Run)
        </span>
        <span class="text-3xs font-mono text-neutral-400 hover:text-neutral-200">
          {showPreview ? 'Hide ▲' : 'Inspect ▼'}
        </span>
      </button>

      {#if showPreview}
        <div class="mt-1 p-3 rounded-xl bg-[#09090b] border border-surface-border space-y-2">
          {#if isLoadingPreview && !launchPreview}
            <div class="text-xs font-mono text-neutral-500 animate-pulse">Resolving launch specification...</div>
          {:else if launchPreview?.command}
            <div class="flex items-start justify-between gap-3">
              <pre class="font-mono text-2xs text-emerald-400/90 whitespace-pre-wrap break-all leading-relaxed flex-1 select-all">{launchPreview.command}</pre>
              <button
                type="button"
                onclick={copyLaunchCommand}
                class="px-2.5 py-1 rounded bg-neutral-900 hover:bg-neutral-800 border border-neutral-700 text-3xs font-mono font-bold text-neutral-300 hover:text-white transition flex-shrink-0 cursor-pointer flex items-center space-x-1"
              >
                {#if previewCopied}
                  <span class="text-emerald-400">✓ Copied</span>
                {:else}
                  <span>Copy</span>
                {/if}
              </button>
            </div>
          {:else}
            <div class="text-xs font-mono text-neutral-500">No executable target resolved yet.</div>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  {#if !filterOnlyOverrides}
    <!-- 1. Executable & Launch Arguments Card -->
    <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
      <div class="flex items-center space-x-2 text-xs font-mono font-bold uppercase tracking-wider text-neutral-400">
        <svg class="w-4 h-4 text-neutral-400 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
        </svg>
        <span>Executable & Arguments</span>
      </div>

      <div class="space-y-4">
        <SettingRow label="Title Override" forId="draftGameName" inline={false}>
          <input
            id="draftGameName"
            type="text"
            bind:value={$drawerDraft.name}
            placeholder={game?.name || "Game Title"}
            class="w-full bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-lg px-3 py-2 text-neutral-100 outline-none text-xs transition-colors"
          />
        </SettingRow>

        <SettingRow label="Launch Arguments" forId="draftExeArgs" inline={false}>
          <input
            id="draftExeArgs"
            type="text"
            bind:value={$drawerDraft.exeArgs}
            placeholder="e.g. -dx11 -novid %command%"
            class="w-full bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-lg px-3 py-2 text-neutral-100 outline-none text-xs font-mono transition-colors"
          />
        </SettingRow>

        <div class="pt-2">
          <span class="block text-xs font-semibold text-neutral-200 mb-2">Target Executable</span>
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
                        <span class="px-1.5 py-0.2 rounded text-4xs font-bold bg-neutral-800 text-neutral-200 border border-neutral-700">Primary</span>
                      {/if}
                    </div>
                    <p class="font-mono text-3xs text-neutral-400 truncate">{cand.executable}</p>
                  </div>
                  <div class="w-3.5 h-3.5 rounded-full border flex items-center justify-center flex-shrink-0 mt-0.5 {isSelected ? 'border-white bg-white' : 'border-surface-borderHighlight'}" aria-hidden="true">
                    {#if isSelected}
                      <div class="w-1.5 h-1.5 rounded-full bg-neutral-950"></div>
                    {/if}
                  </div>
                </button>
              {/each}
            </div>
          {/if}

          <div class="flex items-center space-x-2 mt-2">
            <input
              type="text"
              bind:value={$drawerDraft.exePath}
              placeholder="Relative or absolute path to game executable"
              class="flex-1 bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-lg px-3 py-2 text-neutral-100 outline-none text-xs font-mono transition-colors"
            />
            <button
              type="button"
              onclick={() => drawerDraft.browseExecutable(game?.gameDir)}
              class="px-3.5 py-2 rounded-lg bg-surface-cardHover hover:bg-surface-elevated border border-surface-border text-neutral-200 hover:text-white text-xs font-bold transition cursor-pointer flex-shrink-0"
            >
              Browse...
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- 2. Runtime & Compatibility Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="flex items-center space-x-2 text-xs font-mono font-bold uppercase tracking-wider text-neutral-400">
      <svg class="w-4 h-4 text-neutral-400 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Runtime & Compatibility</span>
    </div>

    <div class="space-y-3">
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
          includeKeys={isWine ? ['runner', 'wine_prefix', 'dll_overrides', 'mangohud'] : ['runner', 'dll_overrides', 'mangohud']}
          scope="game"
          groupByCategory={false}
          bind:values={$drawerDraft}
          inheritedValues={globalCfg}
        />
      {/if}

      {#if isSteamGame}
        <div class="pt-3 border-t border-surface-border space-y-2">
          <span class="block text-3xs font-mono font-bold uppercase tracking-wider text-neutral-400">Steam Emulation Options</span>
          <PropertySheet
            includeKeys={['operating_mode', 'language', 'offline_mode', 'disable_overlay']}
            scope="game"
            groupByCategory={false}
            bind:values={$drawerDraft}
            inheritedValues={globalCfg}
          />
        </div>
      {/if}
    </div>
  </div>

  <!-- 3. Environment Variables Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="flex items-center space-x-2 text-xs font-mono font-bold uppercase tracking-wider text-neutral-400">
      <svg class="w-4 h-4 text-neutral-400 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
      </svg>
      <span>Environment Variables</span>
    </div>

    <EnvDiffEditor
      globalEnv={globalCfg?.global_env || {}}
      bind:customEnv={$drawerDraft.customEnv}
    />
  </div>

  {#if !filterOnlyOverrides}
    <!-- 4. Store Identity & Artwork Card -->
    <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
      <div class="flex items-center space-x-2 text-xs font-mono font-bold uppercase tracking-wider text-neutral-400">
        <svg class="w-4 h-4 text-neutral-400 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>Store Identity & Artwork</span>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <!-- Physical Source Pillar -->
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
            <p class="text-2xs text-neutral-400 leading-snug">File layout, DRM emulation, and runner execution.</p>
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

        <!-- Store Metadata Pillar -->
        <div class="bg-surface-header border border-surface-border p-3.5 rounded-xl flex flex-col justify-between space-y-3">
          <div class="space-y-1">
            <div class="flex items-center justify-between gap-2 min-w-0">
              <span class="text-3xs uppercase font-mono font-bold tracking-wider text-neutral-400 flex items-center gap-1.5 truncate">
                <svg class="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                </svg>
                Store Metadata
              </span>
              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-cardHover text-neutral-200 border border-surface-border whitespace-nowrap flex-shrink-0">
                {hasMatch ? metaInfo.provider : 'Unlinked'}
                {#if hasMatch && metaInfo.id}
                  <span class="text-neutral-400 font-mono ml-1">#{metaInfo.id}</span>
                {/if}
              </span>
            </div>
            <p class="text-2xs text-neutral-400 leading-snug">Synopsis, release date, and credits.</p>
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

        <!-- Artwork Hub Pillar -->
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
            <p class="text-2xs text-neutral-400 leading-snug">Covers, banners, heroes, and logos.</p>
          </div>
          <button
            type="button"
            onclick={() => modalStore.open('artwork-picker', { game, initialTab: 'cover' })}
            class="w-full py-1.5 px-3 rounded-lg bg-surface-cardHover hover:bg-surface-elevated border border-surface-border text-neutral-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
          >
            <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>Open Artwork Hub...</span>
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>
