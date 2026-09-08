<script>
  import CredentialsCard from './cards/CredentialsCard.svelte';

  export let cfgDraft;
</script>

<div class="space-y-4">
  <!-- Bespoke API Keys & Service Credentials Card -->
  <CredentialsCard bind:cfgDraft />
</div>
