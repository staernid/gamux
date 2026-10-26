<script>
  /**
   * @typedef {Object} Props
   * @property {string} [activeTab]
   * @property {(tab: string) => void} [onselectTab]
   * @property {(e: KeyboardEvent, index: number) => void} [ontabKeyDown]
   */

  /** @type {Props} */
  let {
    activeTab = 'general',
    onselectTab = undefined,
    ontabKeyDown = undefined
  } = $props();
</script>

<div class="flex items-center space-x-1.5 overflow-x-auto pb-1 text-xs" role="tablist">
  <!-- 0. General Tab -->
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'general'}
    onclick={() => onselectTab?.('general')}
    onkeydown={(e) => ontabKeyDown?.(e, 0)}
    class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 shrink-0 cursor-pointer {activeTab === 'general' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">1</span>
    <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
    </svg>
    <span>General</span>
  </button>

  <!-- 1. Storage Tab (Library & Backups & Credentials) -->
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'storage' || activeTab === 'library'}
    onclick={() => onselectTab?.('storage')}
    onkeydown={(e) => ontabKeyDown?.(e, 1)}
    class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 shrink-0 cursor-pointer {activeTab === 'storage' || activeTab === 'library' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">2</span>
    <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
    </svg>
    <span>Storage &amp; Library</span>
  </button>

  <!-- 2. Compatibility Tab (System Baseline) -->
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'compatibility'}
    onclick={() => onselectTab?.('compatibility')}
    onkeydown={(e) => ontabKeyDown?.(e, 2)}
    class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 shrink-0 cursor-pointer {activeTab === 'compatibility' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">3</span>
    <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
    <span>Compatibility</span>
  </button>

  <!-- 3. Profiles Tab -->
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'profiles' || activeTab === 'persona'}
    onclick={() => onselectTab?.('profiles')}
    onkeydown={(e) => ontabKeyDown?.(e, 3)}
    class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 shrink-0 cursor-pointer {activeTab === 'profiles' || activeTab === 'persona' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">4</span>
    <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
    </svg>
    <span>Profiles</span>
  </button>

  <!-- 4. Advanced Tab -->
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'advanced'}
    onclick={() => onselectTab?.('advanced')}
    onkeydown={(e) => ontabKeyDown?.(e, 4)}
    class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5 shrink-0 cursor-pointer {activeTab === 'advanced' ? 'bg-surface-cardHover text-white shadow-sm border border-surface-borderHighlight' : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-input'}"
  >
    <span class="text-4xs font-mono px-1 py-0.2 rounded bg-surface-input border border-surface-border text-neutral-400">5</span>
    <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" />
    </svg>
    <span>Advanced</span>
  </button>
</div>
