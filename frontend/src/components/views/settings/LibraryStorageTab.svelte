<script>
  import LibraryPathsCard from './cards/LibraryPathsCard.svelte';
  import StorageAutomationCard from './cards/StorageAutomationCard.svelte';

  export let cfgDraft;
</script>

<div class="space-y-4">
  <!-- Bespoke Library Folders Manager -->
  <LibraryPathsCard bind:libraryPaths={cfgDraft.library_paths} />

  <!-- Bespoke Automation & Backups Card -->
  <StorageAutomationCard bind:cfgDraft />
</div>
