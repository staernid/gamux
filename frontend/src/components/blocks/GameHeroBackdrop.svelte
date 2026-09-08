<script>
  export let heroUrl = '';
  export let localHeroUrl = '';
  export let glowColor = 'rgba(255, 255, 255, 0.05)';
  export let accentColor = 'rgba(255, 255, 255, 0.03)';
  export let dimLevel = 0.5; // 0.0 (bright) to 1.0 (black)
  export let className = 'w-full h-full';

  let imgFailed = false;

  $: activeUrl = (!imgFailed && (localHeroUrl || heroUrl)) || '';
  $: if (heroUrl || localHeroUrl) {
    imgFailed = false;
  }
</script>

<div class="relative overflow-hidden select-none {className}" data-testid="game-hero-backdrop">
  <!-- Backdrop Image Layer -->
  {#if activeUrl}
    <img
      src={activeUrl}
      alt="Hero Backdrop"
      class="absolute inset-0 w-full h-full object-cover object-center filter transition-all duration-700 ease-out"
      style="filter: brightness({1 - dimLevel * 0.5});"
      on:error={() => { imgFailed = true; }}
    />
  {:else}
    <!-- Dark fallback ambient surface -->
    <div class="absolute inset-0 bg-[#09090b]"></div>
  {/if}

  <!-- Ambient Radial Color Glow (Top Left) -->
  <div
    class="absolute -top-24 -left-24 w-[600px] h-[350px] rounded-full opacity-25 filter blur-3xl pointer-events-none transition-colors duration-700"
    style="background-color: {glowColor};"
  ></div>

  <!-- Ambient Secondary Glow (Bottom Right) -->
  <div
    class="absolute -bottom-16 -right-16 w-[450px] h-[280px] rounded-full opacity-15 filter blur-3xl pointer-events-none transition-colors duration-700"
    style="background-color: {accentColor};"
  ></div>

  <!-- Deep Atmospheric Vignette & Contrast Gradients -->
  <div class="absolute inset-0 bg-gradient-to-t from-[#09090b] via-[#09090b]/60 to-transparent pointer-events-none"></div>
  <div class="absolute inset-0 bg-gradient-to-r from-[#09090b]/80 via-transparent to-[#09090b]/60 pointer-events-none"></div>
  <div class="absolute inset-x-0 top-0 h-28 bg-gradient-to-b from-[#09090b]/60 to-transparent pointer-events-none"></div>

  <!-- Foreground Slot Content (if any) -->
  <div class="relative z-10 w-full h-full">
    <slot />
  </div>
</div>
