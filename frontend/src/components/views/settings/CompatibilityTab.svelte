<script>
  import GamescopeCard from './cards/GamescopeCard.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';

  export let cfgDraft = {};
  export const runnersList = [];
</script>

<div class="space-y-4">
  <!-- Declarative Execution Environment & Compatibility -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Runtime & Execution Layer
      </h3>
    </div>
    <PropertySheet
      category={['Operational', 'Emulation']}
      excludeKeys={['normalize', 'transfer_speed_limit']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
  </div>

  <!-- Graphics Overlay -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Graphics Overlays
      </h3>
    </div>
    <PropertySheet
      includeKeys={['mangohud']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
  </div>

  <!-- Bespoke Gamescope Compositor & Resolution Controls -->
  <GamescopeCard bind:cfgDraft />
</div>
