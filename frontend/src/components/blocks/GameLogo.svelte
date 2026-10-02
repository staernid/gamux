<script>
  import { run } from 'svelte/legacy';

  /**
   * @typedef {Object} Props
   * @property {string} [logoUrl]
   * @property {string} [localLogoUrl]
   * @property {string} [title]
   * @property {string} [className]
   * @property {string} [maxHeight]
   * @property {string} [maxWidth]
   */

  /** @type {Props} */
  let {
    logoUrl = '',
    localLogoUrl = '',
    title = '',
    className = '',
    maxHeight = 'max-h-24',
    maxWidth = 'max-w-[420px]'
  } = $props();

  let imgFailed = $state(false);

  let activeUrl = $derived((!imgFailed && (localLogoUrl || logoUrl)) || '');
  run(() => {
    if (logoUrl || localLogoUrl) {
      imgFailed = false;
    }
  });
</script>

<div class="flex items-center min-w-0 {className}" data-testid="game-logo">
  {#if activeUrl}
    <img
      src={activeUrl}
      alt={title || 'Game Logo'}
      class="object-contain object-left drop-shadow-2xl transition-opacity duration-300 {maxHeight} {maxWidth}"
      onerror={() => { imgFailed = true; }}
    />
  {:else if title}
    <h1 class="text-2xl sm:text-4xl font-black text-white tracking-tight drop-shadow-md select-text leading-tight">
      {title}
    </h1>
  {/if}
</div>
