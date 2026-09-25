<script>
  import { createEventDispatcher } from 'svelte';
  import ModalShell from '../common/ModalShell.svelte';
  import MetadataLinker from '../domain/MetadataLinker.svelte';
  import ArtworkPicker from '../domain/ArtworkPicker.svelte';

  export let game = null;
  export let initialTab = 'metadata'; // 'metadata' | 'artwork' | 'cover' | 'banner' | 'hero' | 'logo'

  const dispatch = createEventDispatcher();

  const isArtworkInitial = initialTab === 'cover' || initialTab === 'banner' || initialTab === 'hero' || initialTab === 'logo' || initialTab === 'artwork';
  let activeTab = isArtworkInitial ? 'artwork' : 'metadata';
  let assetType = (initialTab === 'cover' || initialTab === 'banner' || initialTab === 'hero' || initialTab === 'logo') ? initialTab : 'cover';

  $: gameName = game?.name || '';
  $: appId = game?.appId || '';

  function handleUpdated(e) {
    const updated = e.detail;
    if (updated) {
      game = { ...game, ...updated };
    }
    dispatch('updated', updated);
  }
</script>

<ModalShell
  title={gameName}
  badge="Game Identity"
  subtitle="Configure store bindings, AppID detection, and visual box art."
  maxWidth="max-w-4xl"
  on:close={() => dispatch('close')}
>
  <!-- Primary Tab Selector (Store Link vs Artwork Gallery) -->
  <div class="max-w-4xl w-full mx-auto mb-4 flex items-center justify-between border-b border-surface-border pb-3 flex-shrink-0 gap-4">
    <div class="flex items-center space-x-2">
      <button
        type="button"
        on:click={() => { activeTab = 'metadata'; }}
        class="px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeTab === 'metadata' ? 'bg-white text-neutral-950 shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white'}"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        <span>1. Store & Metadata Link</span>
      </button>

      <button
        type="button"
        on:click={() => { activeTab = 'artwork'; }}
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
        on:updated={handleUpdated}
      />
    {:else if activeTab === 'artwork'}
      <ArtworkPicker
        {game}
        bind:assetType
        on:updated={handleUpdated}
      />
    {/if}
  </div>
</ModalShell>
