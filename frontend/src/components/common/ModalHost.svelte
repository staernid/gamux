<script>
  import { modalStore } from '../../stores/modal.js';
  import { selectedGame, refreshGame } from '../../stores/library.js';

  import SearchGridModal from '../stage/SearchGridModal.svelte';
  import ArtworkPickerModal from '../stage/ArtworkPickerModal.svelte';
  import MetadataLinkModal from '../stage/MetadataLinkModal.svelte';
  import GamePropertiesModal from '../stage/GamePropertiesModal.svelte';

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
  {#if $modalStore.type === 'search-grid'}
    <SearchGridModal
      {...$modalStore.props}
      on:close={handleClose}
    />
  {:else if $modalStore.type === 'artwork-picker'}
    <ArtworkPickerModal
      {...$modalStore.props}
      on:updated={handleUpdated}
      on:close={handleClose}
    />
  {:else if $modalStore.type === 'metadata-link'}
    <MetadataLinkModal
      {...$modalStore.props}
      on:updated={handleUpdated}
      on:close={handleClose}
    />
  {:else if $modalStore.type === 'game-properties'}
    <GamePropertiesModal
      {...$modalStore.props}
      on:updated={handleUpdated}
      on:close={handleClose}
    />
  {/if}
{/if}
