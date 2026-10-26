<script>
    import Icon from '../common/Icon.svelte';


  /**
   * @typedef {Object} Props
   * @property {boolean} [isInstalled]
   * @property {boolean} [isReady]
   * @property {boolean} [isRunning]
   * @property {boolean} [isTaskActive]
   * @property {string} [taskStatus] - 'downloading' | 'running' | 'queued' | 'pending'
   * @property {number} [taskProgress] - 0 - 100
   * @property {string} [taskLabel]
   * @property {string} [primaryLabel]
   * @property {number} [pid]
   * @property {string} [className]
   * @property {(() => void) | null} [onplay]
   * @property {(() => void) | null} [onstop]
   * @property {(() => void) | null} [onsetup]
   * @property {(() => void) | null} [oninstall]
   * @property {(() => void) | null} [onconfigure]
   * @property {(() => void) | null} [ontaskClick]
   */

  /** @type {Props} */
  let {
    isInstalled = true,
    isReady = true,
    isRunning = false,
    isTaskActive = false,
    taskStatus = '',
    taskProgress = 0,
    taskLabel = '',
    primaryLabel = '',
    pid = 0,
    className = '',
    onplay = null,
    onstop = null,
    onsetup = null,
    oninstall = null,
    onconfigure = null,
    ontaskClick = null
  } = $props();

  let computedLabel = $derived((() => {
    if (primaryLabel) return primaryLabel;
    if (isRunning) return pid ? `Stop Process (#${pid})` : 'Stop Process';
    if (isTaskActive) {
      if (taskLabel) return `${taskLabel} (${Math.round(taskProgress || 0)}%)`;
      if (taskStatus === 'queued') return 'Queued in Tasks';
      return `Processing (${Math.round(taskProgress || 0)}%)`;
    }
    if (!isInstalled) return 'Install Game';
    if (!isReady) return 'Setup & Apply DRM Fix';
    return 'Play Now';
  })());

  function handlePrimaryClick() {
    if (isRunning) {
      onstop?.();
    } else if (isTaskActive) {
      ontaskClick?.();
    } else if (!isInstalled) {
      oninstall?.();
    } else if (!isReady) {
      onsetup?.();
    } else {
      onplay?.();
    }
  }

  function handleConfigureClick() {
    onconfigure?.();
  }
</script>

<div class="inline-flex items-center gap-2 select-none {className}" data-testid="play-action-cluster">
  <!-- Dominant Hero Action Button -->
  <button
    type="button"
    onclick={handlePrimaryClick}
    class="relative group px-6 py-3 rounded-2xl font-black text-xs sm:text-sm tracking-wide transition-all duration-200 flex items-center space-x-2.5 shadow-xl cursor-pointer overflow-hidden {isRunning ? 'bg-rose-500/20 hover:bg-rose-600 text-rose-300 hover:text-white border border-rose-500/40 hover:border-rose-600' : isTaskActive ? 'bg-neutral-800 text-neutral-200 border border-neutral-700' : (!isReady && isInstalled) ? 'bg-amber-500 hover:bg-amber-400 text-neutral-950 shadow-amber-500/10' : 'bg-neutral-100 hover:bg-white text-neutral-950 hover:shadow-2xl active:scale-[0.98]'}"
  >
    <!-- Live Progress Bar Fill when Task Active -->
    {#if isTaskActive && taskProgress > 0}
      <div
        class="absolute inset-y-0 left-0 bg-neutral-700/80 transition-all duration-300 pointer-events-none"
        style="width: {taskProgress}%"
      ></div>
    {/if}

    <div class="relative z-10 flex items-center space-x-2">
      {#if isRunning}
        <Icon name="stop" className="w-3.5 h-3.5 text-rose-400 group-hover:text-white transition-colors" />
      {:else if isTaskActive}
        <Icon name="spinner" className="w-4 h-4 text-neutral-300" />
      {:else if !isReady && isInstalled}
        <Icon name="lightning" className="w-4 h-4 text-neutral-950" />
      {:else}
        <Icon name="play" className="w-4 h-4 text-neutral-950" />
      {/if}

      <span>{computedLabel}</span>
    </div>
  </button>

  <!-- Secondary Tuning Gear Trigger -->
  <button
    type="button"
    onclick={handleConfigureClick}
    class="p-3 rounded-2xl bg-surface-header/90 hover:bg-surface-cardHover border border-surface-border hover:border-neutral-500 text-neutral-400 hover:text-white transition-all shadow-md cursor-pointer flex items-center justify-center"
    title="Tuning & Compatibility Options"
    aria-label="Tuning & Compatibility Options"
  >
    <Icon name="settings" className="w-4 h-4 text-neutral-400 hover:text-white transition-colors" />
  </button>
</div>
