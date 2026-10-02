<script>
  import { modalStore } from '../../stores/modal.js';
  import { selectedGame, refreshGame } from '../../stores/library.js';

  import CatalogModal from '../stage/CatalogModal.svelte';
  import MetadataArtworkModal from '../stage/MetadataArtworkModal.svelte';
  import ConfirmModal from './ConfirmModal.svelte';

  async function handleUpdated(e) {
    const gameDir = $modalStore?.props?.game?.gameDir || $selectedGame?.gameDir;
    if (gameDir) {
      try {
        await refreshGame(gameDir);
      } catch (err) {
        console.warn('Modal update refresh notice:', err);
      }
    }
  }

  function handleClose() {
    modalStore.close();
  }
</script>

{#if $modalStore}
  {#if $modalStore.type === 'catalog'}
    <CatalogModal
      {...$modalStore.props}
      onclose={handleClose}
    />
  {:else if $modalStore.type === 'metadata-artwork'}
    <MetadataArtworkModal
      {...$modalStore.props}
      onupdated={handleUpdated}
      onclose={handleClose}
    />
  {:else if $modalStore.type === 'artwork-picker'}
    <MetadataArtworkModal
      initialTab="artwork"
      {...$modalStore.props}
      onupdated={handleUpdated}
      onclose={handleClose}
    />
  {:else if $modalStore.type === 'metadata-link'}
    <MetadataArtworkModal
      initialTab="metadata"
      {...$modalStore.props}
      onupdated={handleUpdated}
      onclose={handleClose}
    />
  {:else if $modalStore.type === 'confirm'}
    <ConfirmModal
      {...$modalStore.props}
      onclose={handleClose}
    />
  {/if}
{/if}
