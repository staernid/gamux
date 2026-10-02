<script>
  import AudioSettingsCard from './cards/AudioSettingsCard.svelte';
  import NetworkDownloadsCard from './cards/NetworkDownloadsCard.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';

  let { cfgDraft = $bindable() } = $props();
</script>

<div class="space-y-4">
  <!-- Bespoke Audio & UI Scale Settings -->
  <AudioSettingsCard />

  <!-- Network & Downloads Card -->
  <NetworkDownloadsCard bind:cfgDraft />

  <!-- Declarative Lifecycle Settings -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Process Launch & Window Lifecycle
      </h3>
    </div>
    <PropertySheet
      category="Lifecycle"
      excludeKeys={['enable_launch_notify']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
  </div>
</div>
