<script>
  import CoopTab from './CoopTab.svelte';
  import ToolsMaintenanceTab from './ToolsMaintenanceTab.svelte';
  import LogsTab from './LogsTab.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';
  import { games } from '../../../stores/library.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import { AppAPI } from '../../../api/app.js';

  /**
   * @typedef {Object} Props
   * @property {any} cfgDraft
   * @property {string} [advancedSubtab]
   * @property {Array} [toolStatuses]
   * @property {boolean} [isCheckingTools]
   * @property {() => Promise<void>} [onloadToolsStatus]
   */

  /** @type {Props} */
  let {
    cfgDraft = $bindable(),
    advancedSubtab = $bindable('coop'),
    toolStatuses = [],
    isCheckingTools = false,
    onloadToolsStatus = undefined
  } = $props();

  let updatingTools = $state({});
  let isSubmittingBatch = $state(false);

  async function handleUpdateTool(e) {
    const { key, name } = e.detail;
    if (!key) return;
    updatingTools[key] = true;
    updatingTools = { ...updatingTools };

    try {
      const taskId = await AppAPI.updateTool(key);
      notificationStore.items.add({
        type: 'success',
        title: 'Tool Update Started',
        message: `${name || key} installation/update started in background.`,
        actionLabel: 'View in Tasks',
        actionView: 'tasks',
        actionTaskId: taskId
      });
      await onloadToolsStatus?.();
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Update Failed',
        message: `Failed to update ${name || key}: ${err?.message || String(err)}`
      });
    } finally {
      updatingTools[key] = false;
      updatingTools = { ...updatingTools };
    }
  }

  async function handleBatchAchievements() {
    isSubmittingBatch = true;
    try {
      const taskId = await AppAPI.submitBatchLibraryOp('achievements');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Achievements Started',
        message: 'Downloading achievement schemas and assets for your library.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchArtwork() {
    isSubmittingBatch = true;
    try {
      const taskId = await AppAPI.submitBatchLibraryOp('artwork');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Artwork Started',
        message: 'Fetching missing game artwork and covers from providers.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchApply() {
    isSubmittingBatch = true;
    try {
      const taskId = await AppAPI.submitBatchLibraryOp('apply');
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Setup Started',
        message: 'Applying Goldberg emulators and Steamless unpackers to unpatched games.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchNormalize() {
    isSubmittingBatch = true;
    try {
      const taskId = await AppAPI.submitBatchLibraryOp('normalize');
      notificationStore.items.add({
        type: 'info',
        title: 'Normalization Started',
        message: 'Standardizing game directory folder names to official Steam titles.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchVerify() {
    isSubmittingBatch = true;
    try {
      const taskId = await AppAPI.submitBatchLibraryOp('verify');
      notificationStore.items.add({
        type: 'info',
        title: 'Verification Started',
        message: 'Verifying SHA-1 file integrity against depot manifests.',
        actionLabel: 'View Task in Queue',
        actionView: 'tasks',
        actionTaskId: taskId
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Batch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }
</script>

<div class="space-y-4">
  <!-- Advanced Segmented Sub-Navigation -->
  <div class="flex items-center space-x-1 p-1 bg-surface-header/80 border border-surface-border rounded-xl text-xs overflow-x-auto" role="tablist">
    <!-- Co-Op Subtab -->
    <button
      type="button"
      role="tab"
      aria-selected={advancedSubtab === 'coop'}
      onclick={() => { advancedSubtab = 'coop'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'coop' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
      </svg>
      <span>Co-Op Multiseat</span>
    </button>

    <!-- Maintenance Subtab -->
    <button
      type="button"
      role="tab"
      aria-selected={advancedSubtab === 'maintenance'}
      onclick={() => { advancedSubtab = 'maintenance'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'maintenance' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z" />
      </svg>
      <span>Maintenance</span>
    </button>

    <!-- Logs Subtab -->
    <button
      type="button"
      role="tab"
      aria-selected={advancedSubtab === 'logs'}
      onclick={() => { advancedSubtab = 'logs'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'logs' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <span>Logs</span>
    </button>

    <!-- Schema Subtab -->
    <button
      type="button"
      role="tab"
      aria-selected={advancedSubtab === 'schema'}
      onclick={() => { advancedSubtab = 'schema'; }}
      class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {advancedSubtab === 'schema' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200'}"
      title="Dynamic declarative settings schema"
    >
      <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <span>All Options</span>
    </button>
  </div>

  <!-- Advanced Subtab Content -->
  {#if advancedSubtab === 'coop'}
    <CoopTab bind:cfgDraft />
  {:else if advancedSubtab === 'maintenance'}
    <ToolsMaintenanceTab
      {toolStatuses}
      {isCheckingTools}
      {updatingTools}
      {isSubmittingBatch}
      gamesCount={($games || []).length}
      unpatchedGamesCount={($games || []).filter(g => !g.is_patched && !g.isPatched).length}
      oncheckAll={onloadToolsStatus}
      onupdateTool={handleUpdateTool}
      onbatchAchievements={handleBatchAchievements}
      onbatchArtwork={handleBatchArtwork}
      onbatchApply={handleBatchApply}
      onbatchNormalize={handleBatchNormalize}
      onbatchVerify={handleBatchVerify}
    />
  {:else if advancedSubtab === 'logs'}
    <LogsTab />
  {:else if advancedSubtab === 'schema'}
    <div class="bg-surface-card border border-surface-border rounded-2xl p-6 shadow-xl space-y-4">
      <div class="border-b border-surface-border pb-3">
        <h3 class="text-sm font-bold text-white">Declarative Configuration Engine</h3>
        <p class="text-xs text-neutral-400 mt-0.5 leading-relaxed">All settings defined in the Go backend schema are automatically discovered and rendered here.</p>
      </div>
      <PropertySheet
        bind:values={cfgDraft}
        scope="global"
        groupByCategory={true}
      />
    </div>
  {/if}
</div>
