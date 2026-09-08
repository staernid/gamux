<script>
  export let description = '';
  export let developers = [];
  export let publishers = [];
  export let releaseDate = '';
  export let buildId = '';
  export let clampLines = null; // e.g. 3 or null for natural wrap
  export let className = '';

  function cleanText(raw) {
    if (!raw) return '';
    return raw
      .replace(/<[^>]*>/g, '')
      .replace(/&quot;/g, '"')
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&#39;/g, "'")
      .trim();
  }

  $: cleanDescription = cleanText(description);
  $: devList = Array.isArray(developers) ? developers.filter(Boolean) : (developers ? [developers] : []);
  $: pubList = Array.isArray(publishers) ? publishers.filter(Boolean) : (publishers ? [publishers] : []);
  $: showPub = pubList.length > 0 && pubList.join(',') !== devList.join(',');
</script>

<div class="space-y-3 select-text {className}" data-testid="game-synopsis">
  {#if cleanDescription}
    <p
      class="text-xs sm:text-sm text-neutral-300 leading-relaxed font-normal {clampLines ? `line-clamp-${clampLines}` : ''}"
    >
      {cleanDescription}
    </p>
  {/if}

  {#if devList.length > 0 || showPub || releaseDate || (buildId && buildId !== '0')}
    <div class="flex flex-wrap items-center gap-x-4 gap-y-1.5 pt-2 text-2xs sm:text-xs text-neutral-400 font-mono">
      {#if devList.length > 0}
        <div class="flex items-center space-x-1.5">
          <span class="text-neutral-500">Developer:</span>
          <span class="text-neutral-200 font-medium">{devList.join(', ')}</span>
        </div>
      {/if}

      {#if showPub}
        <div class="flex items-center space-x-1.5">
          <span class="text-neutral-500">Publisher:</span>
          <span class="text-neutral-200 font-medium">{pubList.join(', ')}</span>
        </div>
      {/if}

      {#if releaseDate}
        <div class="flex items-center space-x-1.5">
          <span class="text-neutral-500">Released:</span>
          <span class="text-neutral-200 font-medium">{releaseDate}</span>
        </div>
      {/if}

      {#if buildId && buildId !== '0'}
        <div class="flex items-center space-x-1.5">
          <span class="text-neutral-500">Build:</span>
          <span class="text-neutral-300">{buildId}</span>
        </div>
      {/if}
    </div>
  {/if}
</div>
