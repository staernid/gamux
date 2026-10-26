<script>
  import { AppAPI } from '../../../../api/app.js';
  import { formatBytes, formatETA } from '../../../../utils/format.js';
  import { extractTaskProgress, isActiveStatus } from '../../../../stores/tasks.js';

  /**
   * @typedef {Object} Props
   * @property {import('../../../../types/domain').Task | null} [activeGameTask]
   * @property {import('../../../../types/domain').Task | null} [activeVerifyTask]
   * @property {boolean} [isCheckingIntegrity]
   * @property {boolean} [isNormalizing]
   * @property {boolean} [isUpdating]
   * @property {string} [integrityError]
   * @property {import('../../../../types/domain').IntegrityReport | null} [integrityReport]
   * @property {(() => void | Promise<void>) | null} [onRunIntegrityCheck]
   * @property {(() => void | Promise<void>) | null} [onUpdateGame]
   * @property {(() => void | Promise<void>) | null} [onNormalizeDirectory]
   */

  /** @type {Props} */
  let {
    activeGameTask = null,
    activeVerifyTask = null,
    isCheckingIntegrity = false,
    isNormalizing = false,
    isUpdating = false,
    integrityError = '',
    integrityReport = null,
    onRunIntegrityCheck = () => {},
    onUpdateGame = () => {},
    onNormalizeDirectory = () => {}
  } = $props();

  let showDetails = $state(false);

  let verifyTask = $derived(activeVerifyTask || (activeGameTask?.type === 'verify' ? activeGameTask : null));
  let isVerifying = $derived(Boolean(isCheckingIntegrity || (verifyTask && isActiveStatus(verifyTask.status))));
  let progress = $derived(extractTaskProgress(verifyTask));

  async function handleCancelVerify() {
    if (verifyTask?.id) {
      try {
        await AppAPI.cancelTask(verifyTask.id);
      } catch (_) {}
    }
  }
</script>

<div class="space-y-3">
  <p class="text-neutral-400 text-xs">
    Verify local game files against official SteamPipe depot manifests.
  </p>

  <div class="flex items-center space-x-2 flex-wrap gap-y-2">
    <button
      type="button"
      onclick={onRunIntegrityCheck}
      disabled={isVerifying || isUpdating || isNormalizing}
      class="bg-white hover:bg-neutral-200 text-black font-black px-4 py-2 rounded-xl transition-colors text-xs disabled:opacity-50 flex items-center space-x-1.5 cursor-pointer disabled:cursor-not-allowed shadow-sm"
    >
      {#if isVerifying}
        <svg class="animate-spin w-3 h-3 text-black" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
        </svg>
        <span>Verifying Files...</span>
      {:else}
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>Verify Files</span>
      {/if}
    </button>

    <button
      type="button"
      onclick={onUpdateGame}
      disabled={isUpdating || isVerifying}
      class="bg-surface-header hover:bg-surface-cardHover text-neutral-200 border border-surface-border font-semibold px-4 py-2 rounded-xl transition-colors text-xs disabled:opacity-50 flex items-center space-x-1.5 cursor-pointer disabled:cursor-not-allowed"
      title="Check and download depot updates"
    >
      {#if isUpdating}
        <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
        </svg>
        <span>Updating Depots...</span>
      {:else}
        <span>Update Depots</span>
      {/if}
    </button>

    <button
      type="button"
      onclick={onNormalizeDirectory}
      disabled={isNormalizing}
      class="bg-surface-header hover:bg-surface-cardHover text-neutral-400 hover:text-neutral-200 border border-surface-border font-semibold px-3 py-2 rounded-xl transition-colors text-xs disabled:opacity-50 cursor-pointer"
      title="Rename game directory to Steam naming conventions"
    >
      {isNormalizing ? 'Renaming...' : 'Normalize Folder'}
    </button>
  </div>

  {#if isVerifying && verifyTask}
    <div class="bg-surface-header border border-surface-border p-4 rounded-xl space-y-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-2">
          <svg class="animate-spin w-3.5 h-3.5 text-neutral-200" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span class="text-xs font-bold text-white">Verifying File Integrity...</span>
        </div>
        <div class="flex items-center space-x-3">
          <span class="font-mono text-xs font-bold text-neutral-200">{progress.percent.toFixed(1)}%</span>
          <button
            type="button"
            onclick={handleCancelVerify}
            class="text-neutral-400 hover:text-rose-400 text-2xs font-mono transition-colors cursor-pointer"
          >
            Cancel
          </button>
        </div>
      </div>

      <!-- Progress Bar -->
      <div class="w-full bg-neutral-800 rounded-full h-1.5 overflow-hidden">
        <div
          class="bg-white h-full transition-all duration-150 rounded-full"
          style="width: {progress.percent}%"
        ></div>
      </div>

      <!-- Current File and Speed/ETA Telemetry -->
      <div class="flex items-center justify-between text-2xs font-mono text-neutral-400 gap-2 flex-wrap">
        <div class="truncate max-w-[60%] text-neutral-300" title={verifyTask.details || ''}>
          <span class="text-neutral-500">Scanning:</span> {verifyTask.details || 'Inspecting files...'}
        </div>
        <div class="flex items-center space-x-3 shrink-0">
          {#if progress.speedBps > 0}
            <span class="text-neutral-300">{formatBytes(progress.speedBps)}/s</span>
          {/if}
          {#if progress.etaSeconds > 0}
            <span>ETA {formatETA(progress.etaSeconds)}</span>
          {/if}
          {#if progress.totalBytes > 0}
            <span>{formatBytes(progress.currentBytes)} / {formatBytes(progress.totalBytes)}</span>
          {:else if progress.totalItems > 0}
            <span>{progress.currentItems} / {progress.totalItems} files</span>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  {#if integrityError}
    <div class="p-3.5 rounded-xl bg-[#181214] border border-rose-900/50 text-rose-300 text-xs space-y-1">
      <strong class="block font-bold">Verification Notice</strong>
      <p>{integrityError}</p>
    </div>
  {/if}

  {#if integrityReport}
    {@const total = integrityReport.official_count ?? integrityReport.OfficialCount ?? 0}
    {@const valid = integrityReport.intact_count ?? integrityReport.IntactCount ?? 0}
    {@const modified = integrityReport.modified_files ?? integrityReport.ModifiedFiles ?? []}
    {@const missing = integrityReport.missing_files ?? integrityReport.MissingFiles ?? []}
    {@const untracked = integrityReport.untracked_files ?? integrityReport.UntrackedFiles ?? []}
    {@const damagedCount = modified.length + missing.length}

    <div class="bg-surface-header border border-surface-border p-4 rounded-xl space-y-3 text-xs">
      {#if damagedCount === 0}
        <div class="flex items-center justify-between text-emerald-400">
          <div class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-emerald-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span class="font-medium text-neutral-200">All {total} official game files are intact and verified</span>
          </div>
          <span class="text-2xs font-mono uppercase tracking-wider text-emerald-400/80 bg-emerald-950/40 border border-emerald-800/40 px-2 py-0.5 rounded-md">Verified</span>
        </div>
      {:else}
        <div class="space-y-3">
          <div class="flex items-center justify-between text-rose-400">
            <div class="flex items-center space-x-2">
              <svg class="w-4 h-4 text-rose-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span class="font-medium text-rose-200">{damagedCount} {damagedCount === 1 ? 'file needs' : 'files need'} repair</span>
            </div>
            <button
              type="button"
              onclick={onUpdateGame}
              disabled={isUpdating}
              class="bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs px-3 py-1.5 rounded-lg transition-colors shadow-sm cursor-pointer flex items-center space-x-1.5 disabled:opacity-50"
            >
              {#if isUpdating}
                <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Repairing...</span>
              {:else}
                <span>Repair Game Files</span>
              {/if}
            </button>
          </div>

          <div class="grid grid-cols-3 gap-2 text-2xs pt-1 border-t border-surface-border">
            <div class="text-rose-400/90 font-mono">
              <span class="text-neutral-400">Corrupted:</span> {modified.length}
            </div>
            <div class="text-rose-400/90 font-mono">
              <span class="text-neutral-400">Missing:</span> {missing.length}
            </div>
            <div class="text-emerald-400/90 font-mono">
              <span class="text-neutral-400">Intact:</span> {valid} / {total}
            </div>
          </div>

          <div>
            <button
              type="button"
              onclick={() => showDetails = !showDetails}
              class="text-neutral-400 hover:text-neutral-300 text-2xs font-mono underline cursor-pointer"
            >
              {showDetails ? 'Hide damaged files list' : `Show ${damagedCount} damaged ${damagedCount === 1 ? 'file' : 'files'}`}
            </button>

            {#if showDetails}
              <div class="mt-2 p-2 rounded-lg bg-black/40 border border-surface-border max-h-40 overflow-y-auto space-y-1 font-mono text-2xs text-neutral-300">
                {#each modified as file}
                  <div class="text-rose-400 flex items-center space-x-1.5 truncate">
                    <span class="text-rose-500/70 font-bold shrink-0">[BAD]</span>
                    <span class="truncate" title={file}>{file}</span>
                  </div>
                {/each}
                {#each missing as file}
                  <div class="text-amber-400 flex items-center space-x-1.5 truncate">
                    <span class="text-amber-500/70 font-bold shrink-0">[MISSING]</span>
                    <span class="truncate" title={file}>{file}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {/if}

      {#if untracked.length > 0}
        <div class="pt-2 border-t border-surface-border flex items-center justify-between text-2xs text-neutral-400">
          <span>Extra / Untracked files (mods or logs):</span>
          <span class="font-mono text-neutral-300">{untracked.length}</span>
        </div>
      {/if}
    </div>
  {/if}
</div>

