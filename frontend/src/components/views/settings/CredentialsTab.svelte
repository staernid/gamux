<script>
  import CredentialsCard from './cards/CredentialsCard.svelte';

  let { cfgDraft = $bindable() } = $props();
</script>

<div class="space-y-4">
  <!-- Bespoke API Keys & Service Credentials Card -->
  <CredentialsCard bind:cfgDraft />
</div>
