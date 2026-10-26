<script>
  import HubOverviewTab from './HubOverviewTab.svelte';
  import HubCommunityTab from './HubCommunityTab.svelte';
  import HubConfigTab from './HubConfigTab.svelte';
  import HubFilesTab from './HubFilesTab.svelte';
  import HubModsTab from './HubModsTab.svelte';
  import HubSavesCard from './cards/HubSavesCard.svelte';

  /**
   * @typedef {Object} Props
   * @property {string} [activeTab]
   * @property {import('../../../types/domain').Game} [game]
   * @property {any} [globalCfg]
   * @property {number} [overrideCount]
   * @property {boolean} [isSaving]
   * @property {boolean} [saveSuccess]
   * @property {string} [communitySubTab]
   * @property {number} [communityNewsIndex]
   * @property {string} [filesSubTab]
   * @property {boolean} [isGameRunning]
   * @property {boolean} [isTaskActive]
   * @property {import('../../../types/domain').Task | null} [activeGameTask]
   * @property {import('../../../types/domain').Task | null} [verifyTask]
   * @property {boolean} [isCheckingIntegrity]
   * @property {boolean} [isNormalizing]
   * @property {string} [integrityError]
   * @property {any} [integrityReport]
   * @property {() => void} [onsaveSettings]
   * @property {() => void} [onresetSettings]
   * @property {(e: any) => void} [onlinkMeta]
   * @property {() => void} [onunlinkMeta]
   * @property {() => void} [onrunIntegrityCheck]
   * @property {(e: any) => void} [onupdateGame]
   * @property {() => void} [onnormalizeDirectory]
   * @property {(e: any) => void} [onnavigateTab]
   * @property {() => void} [onprimaryAction]
   */

  /** @type {Props} */
  let {
    activeTab = $bindable('overview'),
    game = null,
    globalCfg = {},
    overrideCount = 0,
    isSaving = false,
    saveSuccess = false,
    communitySubTab = 'achievements',
    communityNewsIndex = 0,
    filesSubTab = 'depots',
    isGameRunning = false,
    isTaskActive = false,
    activeGameTask = null,
    verifyTask = null,
    isCheckingIntegrity = false,
    isNormalizing = false,
    integrityError = '',
    integrityReport = null,
    onsaveSettings = undefined,
    onresetSettings = undefined,
    onlinkMeta = undefined,
    onunlinkMeta = undefined,
    onrunIntegrityCheck = undefined,
    onupdateGame = undefined,
    onnormalizeDirectory = undefined,
    onnavigateTab = undefined,
    onprimaryAction = undefined
  } = $props();

  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);
</script>

<div class="flex-1 pb-12">
  {#if activeTab === 'overview'}
    <HubOverviewTab {game} {onnavigateTab} />
  {:else if activeTab === 'config'}
    <HubConfigTab
      {game}
      {globalCfg}
      {overrideCount}
      {isSaving}
      {saveSuccess}
      onsave={onsaveSettings}
      onreset={onresetSettings}
      onlinkMeta={onlinkMeta}
      onunlinkMeta={onunlinkMeta}
    />
  {:else if activeTab === 'files'}
    <HubFilesTab
      {game}
      activeSubTab={filesSubTab}
      {isGameRunning}
      {isTaskActive}
      {activeGameTask}
      activeVerifyTask={verifyTask}
      {isCheckingIntegrity}
      {isNormalizing}
      {integrityError}
      {integrityReport}
      onrunIntegrityCheck={onrunIntegrityCheck}
      onupdateGame={onupdateGame}
      onnormalizeDirectory={onnormalizeDirectory}
    />
  {:else if activeTab === 'mods'}
    <HubModsTab {game} />
  {:else if activeTab === 'community'}
    <div class="space-y-4">
      <button
        type="button"
        onclick={() => { activeTab = 'overview'; }}
        class="text-xs font-bold text-neutral-400 hover:text-white flex items-center space-x-1.5 cursor-pointer pb-1"
      >
        <span>← Back to Overview</span>
      </button>
      <HubCommunityTab
        {game}
        initialSubTab={communitySubTab}
        initialNewsIndex={communityNewsIndex}
      />
    </div>
  {:else if activeTab === 'saves'}
    <HubSavesCard {game} />
  {:else if !isInstalled}
    <!-- Uninstalled State Placeholder for Technical Tabs -->
    <div class="bg-surface-card border border-surface-border rounded-2xl p-10 text-center space-y-4 max-w-lg mx-auto mt-8">
      <div class="w-12 h-12 rounded-2xl bg-surface-cardHover border border-surface-borderHighlight text-neutral-300 flex items-center justify-center mx-auto">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
      </div>
      <div class="space-y-1">
        <h3 class="font-bold text-white text-base">Game Files Not Installed</h3>
        <p class="text-xs text-neutral-400 leading-relaxed">Download and install {game?.name} to inspect files and run diagnostic verification.</p>
      </div>
      <button
        type="button"
        onclick={onprimaryAction}
        class="px-5 py-2.5 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 font-black text-xs shadow-lg cursor-pointer"
      >
        Download Game Now
      </button>
    </div>
  {/if}
</div>
