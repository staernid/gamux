<script>
  import LibraryPathsCard from './cards/LibraryPathsCard.svelte';
  import StorageAutomationCard from './cards/StorageAutomationCard.svelte';
  import CredentialsCard from './cards/CredentialsCard.svelte';

  let { storageDraft = $bindable(), cfgDraft = $bindable() } = $props();
  let draft = $derived(storageDraft || cfgDraft || {});
</script>

<div class="space-y-4">
  <!-- Bespoke Library Folders Manager -->
  <LibraryPathsCard bind:libraryPaths={draft.library_paths} />

  <!-- Bespoke Automation & Backups Card -->
  <StorageAutomationCard bind:storageDraft bind:cfgDraft />

  <!-- API Keys & Service Credentials Card -->
  <div id="credentials-card">
    <CredentialsCard bind:storageDraft bind:cfgDraft />
  </div>
</div>
