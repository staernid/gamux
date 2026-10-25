<script>
    import { AppAPI } from '../../../api/app.js';
  import { openExternalUrl } from '../../../utils/navigation.js';
  import { activeView } from '../../../stores/library.js';
  import { modalStore } from '../../../stores/modal.js';
  import { configStore } from '../../../stores/config.js';
  import { tasksStore, isPendingOrActiveStatus } from '../../../stores/tasks.js';
  import SettingSection from '../../common/SettingSection.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [toolStatuses]
   * @property {boolean} [isCheckingTools]
   * @property {any} [updatingTools]
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
    unpatchedGamesCount = 0
  ,
    oncheckAll = null,
    onupdateTool = null,
    onbatchAchievements = null,
    onbatchArtwork = null,
    onbatchApply = null,
    onbatchNormalize = null,
    onbatchVerify = null
  } = $props();
  const { items: taskList } = tasksStore;

  function checkIsInstalled(tool) {
    if (!tool) return false;
    return Boolean(tool.isInstalled);
  }

  function findActiveBatchTask(kind, items) {
    if (!Array.isArray(items)) return null;
    return items.find((t) => {
      if (!isPendingOrActiveStatus(t.status)) {
        return false;
      }
      const title = (t.title || '').toLowerCase();
      const type = (t.type || '').toLowerCase();
      const batchKind = (t.metadata?.batch_kind || '').toLowerCase();

      if (kind === 'achievements') {
        return batchKind === 'achievements' || title.includes('achievement');
      }
      if (kind === 'artwork') {
        return batchKind === 'artwork' || title.includes('artwork');
      }
      if (kind === 'apply') {
        return batchKind === 'apply' || (type === 'apply' && title.includes('batch')) || title.includes('apply setup') || title.includes('apply drm');
      }
      if (kind === 'normalize') {
        return batchKind === 'normalize' || (type === 'normalize' && title.includes('batch')) || title.includes('normalize');
      }
      if (kind === 'verify') {
        return batchKind === 'verify' || (type === 'verify' && title.includes('batch')) || title.includes('verify');
      }
      return false;
    }) || null;
  }

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

  function checkAllTools() {
    oncheckAll?.();
  }

  function updateTool(key, name) {
    onupdateTool?.({ key, name });
  }

  function runBatchAchievements() {
    onbatchAchievements?.();
  }

  function runBatchArtwork() {
    onbatchArtwork?.();
  }

  function runBatchApply() {
    onbatchApply?.();
  }

  function runBatchNormalize() {
    onbatchNormalize?.();
  }

  function runBatchVerify() {
    onbatchVerify?.();
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
          onclick={checkAllTools}
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
        {@const isInstalled = checkIsInstalled(tool)}
        {@const isUpdating = Boolean(updatingTools[tool.key])}
        <div class="bg-surface-header border border-surface-border p-3 rounded-xl flex flex-col justify-between gap-2.5">
          <div class="space-y-1 min-w-0">
            <div class="flex items-center justify-between gap-1.5">
              <span class="font-semibold text-xs text-white truncate">{tool.name}</span>
              {#if !isInstalled}
                <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold bg-amber-500/15 text-amber-300 border border-amber-500/30 flex-shrink-0">
                  Missing
                </span>
              {:else if tool.hasUpdate}
                <span class="px-1.5 py-0.5 rounded text-3xs font-mono font-bold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 flex-shrink-0">
                  Update
                </span>
              {/if}
            </div>

            <div class="text-2xs font-mono text-neutral-400 truncate">
              {#if !isInstalled}
                <span class="text-amber-400/90 font-medium">Not Installed</span>
                {#if tool.latestTag}
                  <span class="text-neutral-500">·</span>
                  <span class="text-emerald-400">{tool.latestTag}</span>
                {/if}
              {:else}
                <span>{tool.installedTag || 'Installed'}</span>
                {#if tool.latestTag}
                  <span class="text-neutral-500">·</span>
                  <span class="text-emerald-400">{tool.latestTag}</span>
                {/if}
              {/if}
            </div>
          </div>

          <div class="flex items-center space-x-1.5 pt-1 border-t border-surface-border">
            {#if tool.releaseUrl}
              <button
                type="button"
                onclick={() => openExternalUrl(tool.releaseUrl, { title: tool.name || 'GitHub Release' })}
                class="text-neutral-400 hover:text-white text-xs px-2 py-1 rounded-lg border border-surface-border bg-surface-input transition-colors cursor-pointer"
                title="Open GitHub Release"
                aria-label="Open GitHub Release"
              >
                ↗
              </button>
            {/if}
            <button
              type="button"
              onclick={() => updateTool(tool.key, tool.name)}
              disabled={isUpdating}
              class="flex-1 py-1.5 rounded-lg font-semibold text-xs transition-colors flex items-center justify-center space-x-1 cursor-pointer disabled:opacity-50 {isInstalled && !tool.hasUpdate ? 'bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-400 hover:text-white' : 'bg-neutral-800 hover:bg-neutral-700 text-white border border-neutral-600 shadow-sm'}"
            >
              {#if isUpdating}
                <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>{isInstalled ? (tool.hasUpdate ? 'Updating...' : 'Reinstalling...') : 'Installing...'}</span>
              {:else}
                <span>{isInstalled ? (tool.hasUpdate ? 'Update' : 'Reinstall') : 'Install'}</span>
              {/if}
            </button>
          </div>
        </div>
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
      <div class="bg-surface-header border border-surface-border hover:border-surface-borderHighlight p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <span class="font-bold text-neutral-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
            </svg>
            <span>Achievements</span>
          </span>
          <p class="text-xs text-neutral-400 leading-relaxed">
            Download missing achievement schemas and icons across library.
          </p>
          {#if achievementsTask}
            <div class="mt-2 pt-2 border-t border-surface-border space-y-1.5 animate-fade-in">
              <div class="flex items-center justify-between text-2xs">
                <span class="text-neutral-200 font-semibold truncate flex items-center space-x-1.5">
                  <svg class="animate-spin w-3 h-3 text-white flex-shrink-0" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span class="truncate">{achievementsTask.stage || achievementsTask.details || 'Processing...'}</span>
                </span>
                <span class="font-mono text-neutral-300 font-bold ml-2 flex-shrink-0">{Math.round(achievementsTask.percent || 0)}%</span>
              </div>
              <div class="w-full bg-surface-input rounded-full h-1.5 overflow-hidden border border-surface-border">
                <div class="bg-white h-1.5 rounded-full transition-all duration-300" style="width: {Math.max(2, Math.min(100, achievementsTask.percent || 0))}%"></div>
              </div>
              <div class="flex items-center justify-between text-3xs">
                <span class="truncate text-neutral-500">{achievementsTask.details || ''}</span>
                <button
                  type="button"
                  onclick={handleViewTasks}
                  class="text-neutral-300 hover:text-white font-bold flex items-center space-x-0.5 ml-2 flex-shrink-0 cursor-pointer"
                >
                  <span>View in Tasks</span>
                  <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
                </button>
              </div>
            </div>
          {/if}
        </div>
        <button
          type="button"
          onclick={runBatchAchievements}
          disabled={Boolean(achievementsTask) || isSubmittingBatch || gamesCount === 0}
          class="w-full bg-surface-card hover:bg-surface-cardHover disabled:opacity-50 text-neutral-200 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-surface-border transition-all cursor-pointer flex items-center justify-center space-x-1.5"
        >
          {#if achievementsTask}
            <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Running ({Math.round(achievementsTask.percent || 0)}%)...</span>
          {:else}
            <span>Download Missing</span>
          {/if}
        </button>
      </div>

      <!-- 2. Batch Artwork -->
      <div class="bg-surface-header border border-surface-border hover:border-surface-borderHighlight p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <span class="font-bold text-neutral-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>Artwork</span>
          </span>
          <p class="text-xs text-neutral-400 leading-relaxed">
            Resolve and cache missing posters, hero banners, and logos.
          </p>
          {#if artworkTask}
            <div class="mt-2 pt-2 border-t border-surface-border space-y-1.5 animate-fade-in">
              <div class="flex items-center justify-between text-2xs">
                <span class="text-neutral-200 font-semibold truncate flex items-center space-x-1.5">
                  <svg class="animate-spin w-3 h-3 text-white flex-shrink-0" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span class="truncate">{artworkTask.stage || artworkTask.details || 'Processing...'}</span>
                </span>
                <span class="font-mono text-neutral-300 font-bold ml-2 flex-shrink-0">{Math.round(artworkTask.percent || 0)}%</span>
              </div>
              <div class="w-full bg-surface-input rounded-full h-1.5 overflow-hidden border border-surface-border">
                <div class="bg-white h-1.5 rounded-full transition-all duration-300" style="width: {Math.max(2, Math.min(100, artworkTask.percent || 0))}%"></div>
              </div>
              <div class="flex items-center justify-between text-3xs">
                <span class="truncate text-neutral-500">{artworkTask.details || ''}</span>
                <button
                  type="button"
                  onclick={handleViewTasks}
                  class="text-neutral-300 hover:text-white font-bold flex items-center space-x-0.5 ml-2 flex-shrink-0 cursor-pointer"
                >
                  <span>View in Tasks</span>
                  <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
                </button>
              </div>
            </div>
          {/if}
        </div>
        <button
          type="button"
          onclick={runBatchArtwork}
          disabled={Boolean(artworkTask) || isSubmittingBatch || gamesCount === 0}
          class="w-full bg-surface-card hover:bg-surface-cardHover disabled:opacity-50 text-neutral-200 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-surface-border transition-all cursor-pointer flex items-center justify-center space-x-1.5"
        >
          {#if artworkTask}
            <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Running ({Math.round(artworkTask.percent || 0)}%)...</span>
          {:else}
            <span>Refresh Artwork</span>
          {/if}
        </button>
      </div>

      <!-- 3. Batch Apply DRM & Emulator -->
      <div class="bg-surface-header border border-surface-border hover:border-surface-borderHighlight p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <div class="flex items-center justify-between gap-1.5">
            <span class="font-bold text-neutral-100 text-xs flex items-center space-x-1.5 truncate">
              <svg class="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
              <span class="truncate">Apply Setup</span>
            </span>
            <span class="text-3xs font-mono bg-neutral-800 text-neutral-300 border border-neutral-700 px-1.5 py-0.5 rounded">
              {unpatchedGamesCount}
            </span>
          </div>
          <p class="text-xs text-neutral-400 leading-relaxed">
            Apply Goldberg emulator and unpacker to unpatched games.
          </p>
          {#if applyTask}
            <div class="mt-2 pt-2 border-t border-surface-border space-y-1.5 animate-fade-in">
              <div class="flex items-center justify-between text-2xs">
                <span class="text-neutral-200 font-semibold truncate flex items-center space-x-1.5">
                  <svg class="animate-spin w-3 h-3 text-white flex-shrink-0" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span class="truncate">{applyTask.stage || applyTask.details || 'Processing...'}</span>
                </span>
                <span class="font-mono text-neutral-300 font-bold ml-2 flex-shrink-0">{Math.round(applyTask.percent || 0)}%</span>
              </div>
              <div class="w-full bg-surface-input rounded-full h-1.5 overflow-hidden border border-surface-border">
                <div class="bg-white h-1.5 rounded-full transition-all duration-300" style="width: {Math.max(2, Math.min(100, applyTask.percent || 0))}%"></div>
              </div>
              <div class="flex items-center justify-between text-3xs">
                <span class="truncate text-neutral-500">{applyTask.details || ''}</span>
                <button
                  type="button"
                  onclick={handleViewTasks}
                  class="text-neutral-300 hover:text-white font-bold flex items-center space-x-0.5 ml-2 flex-shrink-0 cursor-pointer"
                >
                  <span>View in Tasks</span>
                  <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
                </button>
              </div>
            </div>
          {/if}
        </div>
        <button
          type="button"
          onclick={runBatchApply}
          disabled={Boolean(applyTask) || isSubmittingBatch || gamesCount === 0}
          class="w-full bg-surface-card hover:bg-surface-cardHover disabled:opacity-50 text-neutral-200 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-surface-border transition-all cursor-pointer flex items-center justify-center space-x-1.5"
        >
          {#if applyTask}
            <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Running ({Math.round(applyTask.percent || 0)}%)...</span>
          {:else}
            <span>Apply All</span>
          {/if}
        </button>
      </div>

      <!-- 4. Batch Normalize -->
      <div class="bg-surface-header border border-surface-border hover:border-surface-borderHighlight p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <span class="font-bold text-neutral-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Normalize Folders</span>
          </span>
          <p class="text-xs text-neutral-400 leading-relaxed">
            Standardize directory names to match official Steam titles.
          </p>
          {#if normalizeTask}
            <div class="mt-2 pt-2 border-t border-surface-border space-y-1.5 animate-fade-in">
              <div class="flex items-center justify-between text-2xs">
                <span class="text-neutral-200 font-semibold truncate flex items-center space-x-1.5">
                  <svg class="animate-spin w-3 h-3 text-white flex-shrink-0" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span class="truncate">{normalizeTask.stage || normalizeTask.details || 'Processing...'}</span>
                </span>
                <span class="font-mono text-neutral-300 font-bold ml-2 flex-shrink-0">{Math.round(normalizeTask.percent || 0)}%</span>
              </div>
              <div class="w-full bg-surface-input rounded-full h-1.5 overflow-hidden border border-surface-border">
                <div class="bg-white h-1.5 rounded-full transition-all duration-300" style="width: {Math.max(2, Math.min(100, normalizeTask.percent || 0))}%"></div>
              </div>
              <div class="flex items-center justify-between text-3xs">
                <span class="truncate text-neutral-500">{normalizeTask.details || ''}</span>
                <button
                  type="button"
                  onclick={handleViewTasks}
                  class="text-neutral-300 hover:text-white font-bold flex items-center space-x-0.5 ml-2 flex-shrink-0 cursor-pointer"
                >
                  <span>View in Tasks</span>
                  <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
                </button>
              </div>
            </div>
          {/if}
        </div>
        <button
          type="button"
          onclick={runBatchNormalize}
          disabled={Boolean(normalizeTask) || isSubmittingBatch || gamesCount === 0}
          class="w-full bg-surface-card hover:bg-surface-cardHover disabled:opacity-50 text-neutral-200 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-surface-border transition-all cursor-pointer flex items-center justify-center space-x-1.5"
        >
          {#if normalizeTask}
            <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Running ({Math.round(normalizeTask.percent || 0)}%)...</span>
          {:else}
            <span>Normalize</span>
          {/if}
        </button>
      </div>

      <!-- 5. Batch Verify Manifests -->
      <div class="bg-surface-header border border-surface-border hover:border-surface-borderHighlight p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors sm:col-span-2 lg:col-span-2">
        <div class="space-y-1">
          <span class="font-bold text-neutral-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            <span>Verify Manifests</span>
          </span>
          <p class="text-xs text-neutral-400 leading-relaxed">
            Verify SHA-1 file integrity against official depot manifests.
          </p>
          {#if verifyTask}
            <div class="mt-2 pt-2 border-t border-surface-border space-y-1.5 animate-fade-in">
              <div class="flex items-center justify-between text-2xs">
                <span class="text-neutral-200 font-semibold truncate flex items-center space-x-1.5">
                  <svg class="animate-spin w-3 h-3 text-white flex-shrink-0" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span class="truncate">{verifyTask.stage || verifyTask.details || 'Processing...'}</span>
                </span>
                <span class="font-mono text-neutral-300 font-bold ml-2 flex-shrink-0">{Math.round(verifyTask.percent || 0)}%</span>
              </div>
              <div class="w-full bg-surface-input rounded-full h-1.5 overflow-hidden border border-surface-border">
                <div class="bg-white h-1.5 rounded-full transition-all duration-300" style="width: {Math.max(2, Math.min(100, verifyTask.percent || 0))}%"></div>
              </div>
              <div class="flex items-center justify-between text-3xs">
                <span class="truncate text-neutral-500">{verifyTask.details || ''}</span>
                <button
                  type="button"
                  onclick={handleViewTasks}
                  class="text-neutral-300 hover:text-white font-bold flex items-center space-x-0.5 ml-2 flex-shrink-0 cursor-pointer"
                >
                  <span>View in Tasks</span>
                  <svg class="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
                </button>
              </div>
            </div>
          {/if}
        </div>
        <button
          type="button"
          onclick={runBatchVerify}
          disabled={Boolean(verifyTask) || isSubmittingBatch || gamesCount === 0}
          class="w-full bg-surface-card hover:bg-surface-cardHover disabled:opacity-50 text-neutral-200 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-surface-border transition-all cursor-pointer flex items-center justify-center space-x-1.5"
        >
          {#if verifyTask}
            <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
            </svg>
            <span>Running ({Math.round(verifyTask.percent || 0)}%)...</span>
          {:else}
            <span>Verify All Manifests</span>
          {/if}
        </button>
      </div>
    </div>
  </SettingSection>
</div>
