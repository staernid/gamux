<script>
  import LibraryPathsCard from './cards/LibraryPathsCard.svelte';
  import StorageAutomationCard from './cards/StorageAutomationCard.svelte';
  import CredentialsCard from './cards/CredentialsCard.svelte';

  let { cfgDraft = $bindable() } = $props();
</script>

<div class="space-y-4">
  <!-- Bespoke Library Folders Manager -->
  <LibraryPathsCard bind:libraryPaths={cfgDraft.library_paths} />

  <!-- Bespoke Automation & Backups Card -->
  <StorageAutomationCard bind:cfgDraft />

  <!-- API Keys & Service Credentials Card -->
  <div id="credentials-card">
    <CredentialsCard bind:cfgDraft />
  </div>
</div>
