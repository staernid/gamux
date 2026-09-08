<script>
  export let logoUrl = '';
  export let localLogoUrl = '';
  export let title = '';
  export let className = '';
  export let maxHeight = 'max-h-24';
  export let maxWidth = 'max-w-[420px]';

  let imgFailed = false;

  $: activeUrl = (!imgFailed && (localLogoUrl || logoUrl)) || '';
  $: if (logoUrl || localLogoUrl) {
    imgFailed = false;
  }
</script>

<div class="flex items-center min-w-0 {className}" data-testid="game-logo">
  {#if activeUrl}
    <img
      src={activeUrl}
      alt={title || 'Game Logo'}
      class="object-contain object-left drop-shadow-2xl transition-opacity duration-300 {maxHeight} {maxWidth}"
      on:error={() => { imgFailed = true; }}
    />
  {:else if title}
    <h1 class="text-2xl sm:text-4xl font-black text-white tracking-tight drop-shadow-md select-text leading-tight">
      {title}
    </h1>
  {/if}
</div>
