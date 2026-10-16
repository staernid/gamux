<script>
  import ModalShell from '../common/ModalShell.svelte';
  import MetadataLinker from '../domain/MetadataLinker.svelte';
  import ArtworkPicker from '../domain/ArtworkPicker.svelte';
  import { drawerDraft } from '../../stores/drawerDraft.js';
  import { updateGameInStore } from '../../stores/library.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   * @property {string} [initialTab] - 'metadata' | 'artwork' | 'cover' | 'banner' | 'hero' | 'logo'
   */

  /** @type {Props} */
  let { game = $bindable(null), initialTab = 'metadata' ,
    onupdated,
    onclose
  } = $props();

  // svelte-ignore state_referenced_locally
  const isArtworkInitial = initialTab === 'cover' || initialTab === 'banner' || initialTab === 'hero' || initialTab === 'logo' || initialTab === 'artwork';
  // svelte-ignore state_referenced_locally
  let activeTab = $state(isArtworkInitial ? 'artwork' : 'metadata');
  // svelte-ignore state_referenced_locally
  let assetType = $state((initialTab === 'cover' || initialTab === 'banner' || initialTab === 'hero' || initialTab === 'logo') ? initialTab : 'cover');

  let gameName = $derived(game?.name || '');
  let appId = $derived(game?.appId || '');

  function handleUpdated(e) {
    const updated = e?.detail ?? e;
    if (updated) {
      game = { ...game, ...updated };
      updateGameInStore(updated);
      drawerDraft.syncIdentity(updated);
    }
    onupdated?.(updated);
  }
</script>

<ModalShell
  title={gameName}
  badge="Game Identity"
  subtitle="Configure store bindings, AppID detection, and visual box art."
  maxWidth="max-w-4xl"
  onclose={() => onclose?.()}
>
  <!-- Primary Tab Selector (Store Link vs Artwork Gallery) -->
  <div class="max-w-4xl w-full mx-auto mb-4 flex items-center justify-between border-b border-surface-border pb-3 flex-shrink-0 gap-4">
    <div class="flex items-center space-x-2">
      <button
        type="button"
        onclick={() => { activeTab = 'metadata'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'metadata' ? 'bg-white text-neutral-950 shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        <span>1. Store & Metadata Link</span>
      </button>

      <button
        type="button"
        onclick={() => { activeTab = 'artwork'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'artwork' ? 'bg-white text-neutral-950 shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>2. Artwork Gallery & Custom Art</span>
      </button>
    </div>

    <!-- Active Source Indicator -->
    <div class="hidden sm:flex items-center space-x-2 text-2xs font-mono text-neutral-400 bg-surface-input border border-surface-border px-3 py-1.5 rounded-xl">
      <span class="text-neutral-500">Source:</span>
      <span class="text-neutral-200 font-bold uppercase">{game?.source || game?.store || 'Steam'}</span>
      {#if appId && appId !== '0'}
        <span class="text-neutral-300">#{appId}</span>
      {/if}
    </div>
  </div>

  <!-- Content Container -->
  <div class="flex-1 max-w-4xl w-full mx-auto overflow-y-auto pr-1 custom-scrollbar space-y-4">
    {#if activeTab === 'metadata'}
      <MetadataLinker
        {game}
        onupdated={handleUpdated}
        onlinked={(detail) => { if (detail?.game) handleUpdated(detail.game); }}
        onunlinked={(detail) => { if (detail?.game || detail) handleUpdated(detail?.game ?? detail); }}
      />
    {:else if activeTab === 'artwork'}
      <ArtworkPicker
        {game}
        bind:assetType
        onupdated={handleUpdated}
      />
    {/if}
  </div>
</ModalShell>
