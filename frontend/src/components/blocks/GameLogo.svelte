<script>
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

  let failedUrl = $state('');

  let candidateUrl = $derived(localLogoUrl || logoUrl || '');
  let activeUrl = $derived((candidateUrl && candidateUrl !== failedUrl) ? candidateUrl : '');
</script>

<div class="flex items-center min-w-0 {className}" data-testid="game-logo">
  {#if activeUrl}
    <img
      src={activeUrl}
      alt={title || 'Game Logo'}
      class="object-contain object-left drop-shadow-2xl transition-opacity duration-300 {maxHeight} {maxWidth}"
      onerror={() => { failedUrl = candidateUrl; }}
    />
  {:else if title}
    <h1 class="text-2xl sm:text-4xl font-black text-white tracking-tight drop-shadow-md select-text leading-tight">
      {title}
    </h1>
  {/if}
</div>
