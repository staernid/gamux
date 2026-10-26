<script>
  /**
   * @typedef {Object} Props
   * @property {string} [activeTab]
   * @property {number} [overrideCount]
   * @property {boolean} [isGameRunning]
   * @property {number} [outdatedModsCount]
   * @property {(tab: string) => void} [onselectTab]
   */

  /** @type {Props} */
  let {
    activeTab = 'overview',
    overrideCount = 0,
    isGameRunning = false,
    outdatedModsCount = 0,
    onselectTab = undefined
  } = $props();
</script>

<div class="flex items-center space-x-2 border-b border-surface-border pb-2 text-xs font-bold select-none overflow-x-auto custom-scrollbar">
  <!-- 1. Overview Tab -->
  <button
    type="button"
    onclick={() => onselectTab?.('overview')}
    class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 shrink-0 cursor-pointer {activeTab === 'overview' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
    <span>Overview</span>
  </button>

  <!-- 2. Configuration & Overrides Tab (Diff Lens) -->
  <button
    type="button"
    onclick={() => onselectTab?.('config')}
    class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 shrink-0 cursor-pointer {activeTab === 'config' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
    </svg>
    <span>Configuration</span>
    {#if overrideCount > 0}
      <span class="px-1.5 py-0.2 rounded-full bg-neutral-800 text-neutral-200 text-3xs font-mono border border-neutral-600">
        {overrideCount}
      </span>
    {/if}
  </button>

  <!-- 3. Files & Depots Tab -->
  <button
    type="button"
    onclick={() => onselectTab?.('files')}
    class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 shrink-0 cursor-pointer {activeTab === 'files' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
    <span>Files & Depots</span>
    {#if isGameRunning}
      <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Execution log active"></span>
    {/if}
  </button>

  <!-- 4. Mods Tab -->
  <button
    type="button"
    onclick={() => onselectTab?.('mods')}
    class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 shrink-0 cursor-pointer {activeTab === 'mods' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
    </svg>
    <span>Mods</span>
    {#if outdatedModsCount > 0}
      <span class="w-2 h-2 rounded-full bg-amber-400 animate-pulse" title="{outdatedModsCount} updates available"></span>
    {/if}
  </button>

  <!-- 5. Saves Tab -->
  <button
    type="button"
    onclick={() => onselectTab?.('saves')}
    class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 shrink-0 cursor-pointer {activeTab === 'saves' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <svg class="w-4 h-4 opacity-80 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
    </svg>
    <span>Saves</span>
  </button>
</div>
