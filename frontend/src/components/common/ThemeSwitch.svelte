<script>
  import { themeStore } from '../../stores/theme.js';
  import { THEME_OPTIONS } from '../../constants/options.js';
  import { playSelect } from '../../utils/audio.js';
  import Icon from './Icon.svelte';

  let {
    label = 'Interface Theme',
    description = 'Appearance variant for the Gamux user interface',
    compact = false
  } = $props();

  let currentMode = $derived(themeStore.mode);
  let resolvedTheme = $derived(themeStore.resolved);

  function handleSelect(mode) {
    if (themeStore.mode === mode) return;
    playSelect();
    themeStore.setMode(mode);
  }
</script>

<div
  class="flex {compact ? 'flex-col sm:flex-row items-start sm:items-center' : 'items-center justify-between'} gap-4 p-3 rounded-xl bg-surface-card border border-surface-border transition-all"
  role="group"
  aria-labelledby="theme-switch-label"
>
  <!-- Label & Description -->
  <div class="space-y-0.5 min-w-0 flex-1">
    <div class="flex items-center space-x-2">
      <span id="theme-switch-label" class="text-xs font-semibold text-neutral-800 dark:text-neutral-200">
        {label}
      </span>
      {#if currentMode === 'system'}
        <span class="px-1.5 py-0.2 rounded-md bg-surface-pill border border-surface-border text-neutral-600 dark:text-neutral-300 text-3xs font-mono font-medium">
          Auto: {resolvedTheme === 'dark' ? 'Dark' : 'Light'}
        </span>
      {/if}
    </div>
    {#if description}
      <p class="text-2xs text-neutral-500 dark:text-neutral-400 leading-relaxed">
        {description}
      </p>
    {/if}
  </div>

  <!-- Segmented 3-Mode Pill Selector -->
  <div
    class="flex items-center p-1 rounded-xl bg-surface-base border border-surface-border shrink-0"
    role="radiogroup"
    aria-label="Theme selection"
  >
    {#each THEME_OPTIONS as opt}
      {@const isActive = currentMode === opt.value}
      <button
        type="button"
        role="radio"
        aria-checked={isActive}
        onclick={() => handleSelect(opt.value)}
        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all flex items-center space-x-1.5 cursor-pointer {isActive ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-950 font-bold shadow-sm' : 'text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white hover:bg-surface-cardHover'}"
        title="{opt.label} — {opt.desc}"
      >
        <Icon name={opt.icon} className="w-3.5 h-3.5 shrink-0" />
        <span>{opt.label}</span>
      </button>
    {/each}
  </div>
</div>
