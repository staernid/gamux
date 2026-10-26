<script>
  /**
   * @typedef {Object} Props
   * @property {number} [totalDepotsCount]
   * @property {number} [languageDepotsCount]
   * @property {number} [optionalDepotsCount]
   * @property {number} [soundtrackDepotsCount]
   * @property {number} [dlcDepotsCount]
   * @property {number} [coreDepotsCount]
   * @property {string} [selectedCategoryFilter]
   * @property {string[]} [availableLanguages]
   * @property {string} [configuredLanguage]
   * @property {(filter: string) => void} [onselectFilter]
   * @property {(lang: string) => void} [onchangeLanguage]
   */

  /** @type {Props} */
  let {
    totalDepotsCount = 0,
    languageDepotsCount = 0,
    optionalDepotsCount = 0,
    soundtrackDepotsCount = 0,
    dlcDepotsCount = 0,
    coreDepotsCount = 0,
    selectedCategoryFilter = 'all',
    availableLanguages = [],
    configuredLanguage = '',
    onselectFilter = undefined,
    onchangeLanguage = undefined,
  } = $props();
</script>

<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 pb-1">
  <div class="flex items-center space-x-1.5 overflow-x-auto py-0.5">
    <button
      type="button"
      onclick={() => onselectFilter?.('all')}
      class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all cursor-pointer {selectedCategoryFilter === 'all' ? 'bg-neutral-200 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
    >
      All ({totalDepotsCount})
    </button>

    {#if languageDepotsCount > 0}
      <button
        type="button"
        onclick={() => onselectFilter?.('languages')}
        class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'languages' ? 'bg-blue-500 text-white font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
      >
        <span>🌐 Languages</span>
        <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{languageDepotsCount}</span>
      </button>
    {/if}

    {#if optionalDepotsCount > 0}
      <button
        type="button"
        onclick={() => onselectFilter?.('optional')}
        class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'optional' ? 'bg-amber-500 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
      >
        <span>🎨 High-Res / Optional</span>
        <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{optionalDepotsCount}</span>
      </button>
    {/if}

    {#if soundtrackDepotsCount > 0}
      <button
        type="button"
        onclick={() => onselectFilter?.('soundtrack')}
        class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'soundtrack' ? 'bg-purple-500 text-white font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
      >
        <span>🎵 Soundtracks</span>
        <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{soundtrackDepotsCount}</span>
      </button>
    {/if}

    {#if dlcDepotsCount > 0}
      <button
        type="button"
        onclick={() => onselectFilter?.('dlc')}
        class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'dlc' ? 'bg-emerald-500 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
      >
        <span>📦 DLCs</span>
        <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{dlcDepotsCount}</span>
      </button>
    {/if}

    {#if coreDepotsCount > 0}
      <button
        type="button"
        onclick={() => onselectFilter?.('core')}
        class="px-2.5 py-1 rounded-lg text-2xs font-bold transition-all flex items-center space-x-1 cursor-pointer {selectedCategoryFilter === 'core' ? 'bg-neutral-200 text-neutral-950 font-bold' : 'bg-surface-header hover:bg-surface-cardHover text-neutral-400 border border-surface-border'}"
      >
        <span>Core Content</span>
        <span class="px-1 py-0.2 rounded-full text-3xs bg-black/30">{coreDepotsCount}</span>
      </button>
    {/if}
  </div>

  {#if availableLanguages.length > 0}
    <div class="flex items-center space-x-2 text-2xs text-neutral-400 flex-shrink-0">
      <span class="font-bold">Voice Pack:</span>
      <select
        value={configuredLanguage}
        onchange={(e) => onchangeLanguage?.(e.target.value)}
        class="bg-surface-card border border-surface-border text-neutral-200 rounded-lg px-2.5 py-1 text-2xs outline-none cursor-pointer hover:border-neutral-500 transition-colors"
      >
        {#each availableLanguages as lang}
          <option value={lang}>{lang.charAt(0).toUpperCase() + lang.slice(1)}</option>
        {/each}
      </select>
    </div>
  {/if}
</div>
