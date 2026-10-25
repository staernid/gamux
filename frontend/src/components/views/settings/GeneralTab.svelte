<script>
  import AudioSettingsCard from './cards/AudioSettingsCard.svelte';
  import NetworkDownloadsCard from './cards/NetworkDownloadsCard.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';

  let { generalDraft = $bindable(), cfgDraft = $bindable() } = $props();
  let draft = $derived(generalDraft || cfgDraft);
</script>

<div class="space-y-4">
  <!-- Bespoke Audio & UI Scale Settings -->
  <AudioSettingsCard />

  <!-- Network & Downloads Card -->
  <NetworkDownloadsCard bind:generalDraft bind:cfgDraft />

  <!-- Declarative Lifecycle Settings Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="space-y-0.5">
      <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
        Process Launch &amp; Window Lifecycle
      </h3>
      <p class="text-xs text-neutral-400 leading-relaxed">
        Behavior for game process launch, window focus, and system tray minimization.
      </p>
    </div>
    <PropertySheet
      category="Lifecycle"
      excludeKeys={['enable_launch_notify']}
      groupByCategory={false}
      bind:values={draft}
    />
  </div>
</div>
