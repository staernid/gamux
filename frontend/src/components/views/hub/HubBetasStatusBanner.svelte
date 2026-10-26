<script>
  import { isActiveStatus, isPausedStatus } from '../../../stores/tasks.js';

  /**
   * @typedef {Object} Props
   * @property {string} effectiveAppId
   * @property {boolean} isVirtual
   * @property {boolean} isStub
   * @property {string} currentBranch
   * @property {string} targetBuildId
   * @property {boolean} isBuildMismatch
   * @property {string} cleanInstalledBuild
   * @property {boolean} isLoading
   * @property {boolean} isTaskActive
   * @property {import('../../../types/domain').Task | null} [activeGameTask]
   * @property {() => void} [onrefresh]
   * @property {() => void} [ontriggerUpdate]
   */

  /** @type {Props} */
  let {
    effectiveAppId = '',
    isVirtual = false,
    isStub = false,
    currentBranch = 'public',
    targetBuildId = '',
    isBuildMismatch = false,
    cleanInstalledBuild = '',
    isLoading = false,
    isTaskActive = false,
    activeGameTask = null,
    onrefresh = undefined,
    ontriggerUpdate = undefined
  } = $props();
</script>

<div class="bg-surface-card border border-surface-border rounded-2xl p-5 shadow-sm space-y-3">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center space-x-3">
      <div class="w-9 h-9 rounded-xl bg-surface-header border border-surface-border text-neutral-300 flex items-center justify-center flex-shrink-0">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v10M8 7a3 3 0 100-6 3 3 0 000 6zm0 10a3 3 0 100 6 3 3 0 000-6zm8-6a3 3 0 100-6 3 3 0 000 6zm-8 4h3a5 5 0 005-5V7" />
        </svg>
      </div>
      <div>
        <div class="flex items-center space-x-2">
          <h2 class="text-sm font-bold text-white">Steam Betas & Branches</h2>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-header border border-surface-border text-neutral-400">
            #{effectiveAppId}
          </span>
        </div>
        <p class="text-xs text-neutral-400 leading-relaxed mt-0.5">
          Select testing branches, Vulkan/DirectX variants, or legacy versions provided by the game developers.
        </p>
      </div>
    </div>

    <button
      type="button"
      onclick={onrefresh}
      disabled={isLoading}
      class="px-3 py-1.5 rounded-xl bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white text-xs font-semibold transition-colors flex items-center space-x-1.5 self-start sm:self-auto cursor-pointer disabled:opacity-50"
      title="Refresh branch list from Steam"
    >
      <svg class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
      <span>Refresh</span>
    </button>
  </div>

  <!-- Context Notice Badge -->
  <div class="pt-2 border-t border-surface-border flex flex-wrap items-center justify-between gap-2 text-xs">
    <div class="flex items-center space-x-2">
      {#if isVirtual}
        <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-neutral-800 text-neutral-300 border border-neutral-700">
          Virtual Store Title
        </span>
        <span class="text-neutral-400 text-xs">
          Selected branch will be saved when adding to library or downloading.
        </span>
      {:else if isStub}
        <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-neutral-800 text-neutral-300 border border-neutral-700">
          Library Stub
        </span>
        <span class="text-neutral-400 text-xs">
          Configured in launch profile. Downloads will target branch <strong class="text-neutral-200">{currentBranch}</strong>{#if targetBuildId} (Build #{targetBuildId}){/if}.
        </span>
      {:else}
        <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-surface-header text-neutral-300 border border-surface-border">
          Installed Game
        </span>
        <span class="text-neutral-400 text-xs">
          Configured in <code class="font-mono text-neutral-300">appmanifest_{effectiveAppId}.acf</code> & launch profile.
        </span>
      {/if}
    </div>

    <!-- Current Branch Badge -->
    <div class="flex items-center space-x-1.5 font-mono text-2xs">
      <span class="text-neutral-500">Active Branch:</span>
      <span class="px-2 py-0.5 rounded bg-surface-header border border-surface-border text-white font-bold">
        {currentBranch}
      </span>
      {#if targetBuildId}
        <span class="text-neutral-500 font-mono">#{targetBuildId}</span>
      {/if}
    </div>
  </div>

  <!-- Build Mismatch / Update Pending Banner for Installed Game -->
  {#if isBuildMismatch}
    <div class="p-3.5 rounded-xl bg-surface-header border border-surface-borderHighlight flex flex-col sm:flex-row sm:items-center justify-between gap-3 animate-fade-in">
      <div class="space-y-0.5">
        <div class="flex items-center space-x-2">
          <span class="w-2 h-2 rounded-full bg-amber-400"></span>
          <span class="text-xs font-bold text-neutral-100">Files Pending Branch Sync</span>
        </div>
        <p class="text-2xs text-neutral-400 font-mono">
          Installed files: #{cleanInstalledBuild} &rarr; Target branch files: #{targetBuildId}
        </p>
        {#if activeGameTask && (isActiveStatus(activeGameTask.status) || isPausedStatus(activeGameTask.status))}
          <p class="text-3xs text-neutral-400 font-mono">
            Active task: {activeGameTask.status === 'paused' ? 'Paused - ' : ''}{activeGameTask.stage || activeGameTask.details || 'Syncing files...'}
          </p>
        {/if}
      </div>

      <button
        type="button"
        onclick={ontriggerUpdate}
        disabled={isTaskActive}
        class="px-4 py-2 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 font-black text-xs transition-colors flex items-center space-x-1.5 self-start sm:self-auto cursor-pointer disabled:opacity-50"
      >
        {#if isTaskActive}
          {#if activeGameTask?.status === 'paused'}
            <span>Download Paused</span>
          {:else}
            <svg class="animate-spin w-3.5 h-3.5 text-neutral-950" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Download in Progress...</span>
          {/if}
        {:else}
          <svg class="w-3.5 h-3.5 text-neutral-950" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>Switch Files to Branch</span>
        {/if}
      </button>
    </div>
  {/if}
</div>
