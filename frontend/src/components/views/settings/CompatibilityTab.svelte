<script>
  import GamescopeCard from './cards/GamescopeCard.svelte';
  import PropertySheet from '../../schema/PropertySheet.svelte';

  export let cfgDraft = {};
  export const runnersList = [];
</script>

<div class="space-y-6">
  <!-- System Baseline Inheritance Callout Banner -->
  <div class="bg-surface-header/70 border border-surface-border rounded-2xl p-4 shadow-sm flex items-start space-x-3.5">
    <div class="w-9 h-9 rounded-xl bg-surface-input border border-surface-borderHighlight flex items-center justify-center flex-shrink-0 text-neutral-300 mt-0.5">
      <svg class="w-4 h-4 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
      </svg>
    </div>
    <div class="space-y-0.5 min-w-0">
      <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
        Global Runtime Baseline
      </h3>
      <p class="text-xs text-neutral-400 leading-relaxed">
        These settings define the default runtime environment, Proton/Wine runner, overlays, and Gamescope configuration inherited by all games in your library. Individual games can override any of these options in their Game Hub configuration.
      </p>
    </div>
  </div>

  <!-- 1. Linux & Wine Runtime Layer -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Runtime &amp; Execution Layer
      </h3>
    </div>
    <p class="text-xs text-neutral-400 leading-relaxed">
      Proton and Wine execution environment for running Windows executables on Linux.
    </p>
    <PropertySheet
      includeKeys={['default_runner', 'wine_prefix']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
  </div>

  <!-- 2. Graphics & Compositor -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Graphics &amp; Compositor
      </h3>
    </div>
    <PropertySheet
      includeKeys={['mangohud']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
    <GamescopeCard bind:cfgDraft />
  </div>

  <!-- 3. Steam & Goldberg Emulation (GBE Fork) -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Steam &amp; Goldberg Emulation
      </h3>
    </div>
    <p class="text-xs text-neutral-400 leading-relaxed">
      Default Steamworks integration and Goldberg emulator (<code class="font-mono text-neutral-300">gbe_fork</code>) settings applied to Windows Steam titles.
    </p>
    <PropertySheet
      includeKeys={['operating_mode', 'steamless', 'achievements', 'language', 'disable_overlay', 'offline_mode']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
  </div>

  <!-- 4. Process Environment (Global Variables) -->
  <div class="space-y-3">
    <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
      <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
        Process Environment
      </h3>
    </div>
    <PropertySheet
      includeKeys={['global_env']}
      groupByCategory={false}
      bind:values={cfgDraft}
    />
  </div>
</div>
