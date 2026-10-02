<script>
  import { run } from 'svelte/legacy';

    import { isSteamApp } from '../../../utils/game.js';
  import HubDLCCard from './cards/HubDLCCard.svelte';
  import HubIntegrityCard from './cards/HubIntegrityCard.svelte';
  import HubLogsCard from './cards/HubLogsCard.svelte';
  import HubBetasTab from './HubBetasTab.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   * @property {string} [activeSubTab] - 'depots' | 'branch' | 'integrity' | 'logs'
   * @property {boolean} [isGameRunning]
   * @property {boolean} [isTaskActive]
   * @property {any} [activeGameTask]
   * @property {boolean} [isCheckingIntegrity]
   * @property {boolean} [isNormalizing]
   * @property {string} [integrityError]
   * @property {any} [integrityReport]
   */

  /** @type {Props} */
  let {
    game = null,
    activeSubTab = 'depots',
    isGameRunning = false,
    isTaskActive = false,
    activeGameTask = null,
    isCheckingIntegrity = false,
    isNormalizing = false,
    integrityError = '',
    integrityReport = null
  ,
    onrunIntegrityCheck,
    onupdateGame,
    onnormalizeDirectory
  } = $props();

  let effectiveAppId = $derived((game?.appId && isSteamApp(game.appId))
    ? String(game.appId).trim()
    : ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId) ? String(game.metadataId).trim() : ''));
  let isSteam = $derived(Boolean(effectiveAppId));

  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);

  let activeFilesSubTab = $state(activeSubTab || (isInstalled ? 'depots' : (isSteam ? 'branch' : 'depots')));
  let prevActiveSubTab = $state(activeSubTab);

  run(() => {
    if (activeSubTab !== prevActiveSubTab) {
      prevActiveSubTab = activeSubTab;
      activeFilesSubTab = activeSubTab || 'depots';
    }
  });

  function handleRunIntegrityCheck() {
    onrunIntegrityCheck?.();
  }

  function handleUpdateGame(e) {
    onupdateGame?.(e?.detail);
  }

  function handleNormalizeDirectory() {
    onnormalizeDirectory?.();
  }
</script>

<div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-xl space-y-6">
  <!-- Files Sub-Pills -->
  <div class="flex items-center space-x-2 bg-surface-header p-1.5 rounded-xl border border-surface-border flex-wrap gap-y-1">
    <button
      type="button"
      onclick={() => { activeFilesSubTab = 'depots'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'depots' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
      </svg>
      <span>DLC & Depots</span>
    </button>

    {#if isSteam}
      <button
        type="button"
        onclick={() => { activeFilesSubTab = 'branch'; }}
        class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'branch' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v10M8 7a3 3 0 100-6 3 3 0 000 6zm0 10a3 3 0 100 6 3 3 0 000-6zm8-6a3 3 0 100-6 3 3 0 000 6zm-8 4h3a5 5 0 005-5V7" />
        </svg>
        <span>Branch</span>
        {#if game?.branch && game.branch !== 'public'}
          <span class="px-1.5 py-0.5 rounded-full bg-neutral-800 text-neutral-200 text-3xs font-mono border border-neutral-700">
            {game.branch}
          </span>
        {/if}
      </button>
    {/if}

    <button
      type="button"
      onclick={() => { activeFilesSubTab = 'integrity'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'integrity' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
      <span>Integrity & Repair</span>
    </button>

    <button
      type="button"
      onclick={() => { activeFilesSubTab = 'logs'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'logs' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
    >
      <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <span>Execution Logs</span>
      {#if isGameRunning}
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Game currently executing"></span>
      {/if}
    </button>
  </div>

  <!-- Files Sub-View Content -->
  {#if activeFilesSubTab === 'branch'}
    <HubBetasTab
      {game}
      {isTaskActive}
      {activeGameTask}
      onupdateGame={handleUpdateGame}
    />
  {:else if activeFilesSubTab === 'depots'}
    {#if !isInstalled}
      <!-- Uninstalled State Placeholder for disk-dependent subtabs -->
      <div class="bg-surface-header border border-surface-border rounded-xl p-8 text-center space-y-3">
        <div class="w-10 h-10 rounded-xl bg-surface-cardHover border border-surface-borderHighlight text-neutral-300 flex items-center justify-center mx-auto">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        </div>
        <h3 class="font-bold text-white text-sm">Game Files Not Installed</h3>
        <p class="text-xs text-neutral-400 leading-relaxed max-w-md mx-auto">
          Download and install {game?.name || 'this game'} to inspect installed DLCs, verify depot checksums, or review runtime execution logs. You can inspect available content depots and select language packs below.
        </p>
      </div>
    {/if}
    <HubDLCCard {game} />
  {:else if !isInstalled}
    <!-- Uninstalled State Placeholder for disk-dependent subtabs -->
    <div class="bg-surface-header border border-surface-border rounded-xl p-8 text-center space-y-3">
      <div class="w-10 h-10 rounded-xl bg-surface-cardHover border border-surface-borderHighlight text-neutral-300 flex items-center justify-center mx-auto">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
      </div>
      <h3 class="font-bold text-white text-sm">Game Files Not Installed</h3>
      <p class="text-xs text-neutral-400 leading-relaxed max-w-md mx-auto">
        Download and install {game?.name || 'this game'} to inspect installed DLCs, verify depot checksums, or review runtime execution logs.
      </p>
    </div>
  {:else if activeFilesSubTab === 'integrity'}
    <HubIntegrityCard
      {activeGameTask}
      {isCheckingIntegrity}
      {isNormalizing}
      isUpdating={isTaskActive}
      {integrityError}
      {integrityReport}
      onRunIntegrityCheck={handleRunIntegrityCheck}
      onUpdateGame={handleUpdateGame}
      onNormalizeDirectory={handleNormalizeDirectory}
    />
  {:else if activeFilesSubTab === 'logs'}
    <HubLogsCard {game} />
  {/if}
</div>
