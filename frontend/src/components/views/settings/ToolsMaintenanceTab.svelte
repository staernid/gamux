<script>
  import { activeView } from '../../../stores/library.js';
  import { modalStore } from '../../../stores/modal.js';
  import { configStore } from '../../../stores/config.js';
  import { tasksStore } from '../../../stores/tasks.js';
  import { findActiveBatchTask } from '../../../utils/tools.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import ToolCard from './ToolCard.svelte';
  import BatchOperationCard from './BatchOperationCard.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').ToolStatus[]} [toolStatuses]
   * @property {boolean} [isCheckingTools]
   * @property {Record<string, boolean>} [updatingTools]
   * @property {boolean} [isSubmittingBatch]
   * @property {number} [gamesCount]
   * @property {number} [unpatchedGamesCount]
   */

  /** @type {Props} */
  let {
    toolStatuses = [],
    isCheckingTools = false,
    updatingTools = {},
    isSubmittingBatch = false,
    gamesCount = 0,
    unpatchedGamesCount = 0,
    oncheckAll = null,
    onupdateTool = null,
    onbatchAchievements = null,
    onbatchArtwork = null,
    onbatchApply = null,
    onbatchNormalize = null,
    onbatchVerify = null
  } = $props();

  const { items: taskList } = tasksStore;

  let achievementsTask = $derived(findActiveBatchTask('achievements', $taskList));
  let artworkTask = $derived(findActiveBatchTask('artwork', $taskList));
  let applyTask = $derived(findActiveBatchTask('apply', $taskList));
  let normalizeTask = $derived(findActiveBatchTask('normalize', $taskList));
  let verifyTask = $derived(findActiveBatchTask('verify', $taskList));

  function handleViewTasks() {
    if (configStore.isDirty) {
      modalStore.open('confirm', {
        title: 'Unsaved Changes',
        message: 'You have unsaved changes in Settings. Do you want to discard them or keep editing?',
        confirmLabel: 'Discard Changes',
        cancelLabel: 'Keep Editing',
        danger: true,
        onConfirm: () => {
          configStore.setDirty(false);
          $activeView = 'tasks';
        }
      });
      return;
    }
    $activeView = 'tasks';
  }
</script>

<div class="space-y-3">
  <!-- Toolchain Management -->
  <SettingSection
    title="Helper Tools"
    tooltip="External tools for Steam emulation, DRM removal, and Proton execution."
  >
    {#snippet action()}
      <button
        type="button"
        onclick={() => oncheckAll?.()}
        disabled={isCheckingTools}
        class="px-2.5 py-1 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isCheckingTools}
          <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Checking...</span>
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Check All</span>
        {/if}
      </button>
    {/snippet}

    <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5">
      {#each toolStatuses as tool}
        <ToolCard
          {tool}
          isUpdating={Boolean(updatingTools[tool.key])}
          onupdate={(data) => onupdateTool?.(data)}
        />
      {/each}
    </div>
  </SettingSection>

  <!-- Batch Maintenance Operations -->
  <SettingSection
    title="Batch Maintenance"
    tooltip="Automated multi-game library operations executed in the background."
  >
    {#snippet action()}
      <button
        type="button"
        onclick={handleViewTasks}
        class="px-2 py-0.5 rounded-lg bg-surface-card hover:bg-surface-cardHover border border-surface-border text-2xs text-neutral-300 hover:text-white transition-colors flex items-center space-x-1 cursor-pointer"
        title="View active background tasks"
      >
        <span>View Tasks</span>
        <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
        </svg>
      </button>
    {/snippet}

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
      <!-- 1. Batch Achievements -->
      <BatchOperationCard
        title="Achievements"
        description="Download missing achievement schemas and icons across library."
        actionLabel="Download Missing"
        task={achievementsTask}
        disabled={isSubmittingBatch || gamesCount === 0}
        onrun={() => onbatchAchievements?.()}
        onviewTasks={handleViewTasks}
      >
        {#snippet icon()}
          <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
          </svg>
        {/snippet}
      </BatchOperationCard>

      <!-- 2. Batch Artwork -->
      <BatchOperationCard
        title="Artwork"
        description="Resolve and cache missing posters, hero banners, and logos."
        actionLabel="Refresh Artwork"
        task={artworkTask}
        disabled={isSubmittingBatch || gamesCount === 0}
        onrun={() => onbatchArtwork?.()}
        onviewTasks={handleViewTasks}
      >
        {#snippet icon()}
          <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
        {/snippet}
      </BatchOperationCard>

      <!-- 3. Batch Apply DRM & Emulator -->
      <BatchOperationCard
        title="Apply Setup"
        description="Apply Goldberg emulator and unpacker to unpatched games."
        actionLabel="Apply All"
        countBadge={String(unpatchedGamesCount)}
        task={applyTask}
        disabled={isSubmittingBatch || gamesCount === 0}
        onrun={() => onbatchApply?.()}
        onviewTasks={handleViewTasks}
      >
        {#snippet icon()}
          <svg class="w-3.5 h-3.5 text-neutral-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
        {/snippet}
      </BatchOperationCard>

      <!-- 4. Batch Normalize -->
      <BatchOperationCard
        title="Normalize Folders"
        description="Standardize directory names to match official Steam titles."
        actionLabel="Normalize"
        task={normalizeTask}
        disabled={isSubmittingBatch || gamesCount === 0}
        onrun={() => onbatchNormalize?.()}
        onviewTasks={handleViewTasks}
      >
        {#snippet icon()}
          <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        {/snippet}
      </BatchOperationCard>

      <!-- 5. Batch Verify Manifests -->
      <BatchOperationCard
        title="Verify Manifests"
        description="Verify SHA-1 file integrity against official depot manifests."
        actionLabel="Verify All Manifests"
        className="sm:col-span-2 lg:col-span-2"
        task={verifyTask}
        disabled={isSubmittingBatch || gamesCount === 0}
        onrun={() => onbatchVerify?.()}
        onviewTasks={handleViewTasks}
      >
        {#snippet icon()}
          <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        {/snippet}
      </BatchOperationCard>
    </div>
  </SettingSection>
</div>
