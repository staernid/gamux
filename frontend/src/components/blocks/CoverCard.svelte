<script>
  import { createEventDispatcher } from 'svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import { formatPlaytime, formatLastPlayed } from '../../utils/game.js';
  import { formatBytes } from '../../utils/format.js';

  export let appId = '';
  export let name = '';
  export let coverUrl = '';
  export let localCoverUrl = '';
  export let platform = 'win64';
  export let isRunning = false;
  export let isReady = true;
  export let isStub = false;
  export let playtimeSeconds = 0;
  export let lastPlayed = 0;
  export let diskSizeBytes = 0;
  export let selected = false;
  export let size = 'md'; // 'sm' | 'md' | 'lg'
  export let className = '';
  export let onclick = null;

  const dispatch = createEventDispatcher();

  function handleClick() {
    dispatch('click');
    if (typeof onclick === 'function') onclick();
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleClick();
    }
  }

  $: sizeClasses = (() => {
    switch (size) {
      case 'sm': return 'w-28 sm:w-36';
      case 'lg': return 'w-44 sm:w-56';
      case 'full': return 'w-full';
      default: return 'w-36 sm:w-44';
    }
  })();
</script>

<button
  type="button"
  on:click={handleClick}
  on:keydown={handleKeydown}
  class="group cursor-pointer flex flex-col items-center text-left transition-all duration-200 transform hover:-translate-y-1.5 focus:outline-none select-none bg-transparent border-0 p-0 {sizeClasses} {className}"
  data-testid="cover-card"
>
  <!-- Poster Art Container -->
  <div
    class="relative w-full aspect-[2/3] rounded-2xl overflow-hidden shadow-lg group-hover:shadow-2xl border transition-all duration-200 {selected ? 'border-white ring-2 ring-white/50 shadow-white/10' : 'border-[#27272a] group-hover:border-neutral-400'}"
  >
    <PosterArt
      {appId}
      {name}
      posterUrl={coverUrl}
      {localCoverUrl}
      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
    />

    <!-- Top Left Badge: Platform / Stub -->
    <div
      class="absolute top-2 left-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider text-neutral-200 bg-neutral-900/85 border border-neutral-700/80 shadow-md backdrop-blur-md"
    >
      {isStub ? 'STUB' : (platform || 'WIN')}
    </div>

    <!-- Top Right Status Signifier -->
    {#if isRunning}
      <div
        class="absolute top-2 right-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider bg-emerald-500/25 text-emerald-300 border border-emerald-500/50 shadow-md backdrop-blur-md flex items-center space-x-1 animate-pulse"
      >
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
        <span>RUNNING</span>
      </div>
    {:else if !isReady && !isStub}
      <div
        class="absolute top-2 right-2 px-1.5 py-0.5 rounded text-4xs font-mono font-bold uppercase tracking-wider bg-amber-500/25 text-amber-300 border border-amber-500/40 shadow-md backdrop-blur-md flex items-center space-x-1"
        title="Setup Required"
      >
        <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
        <span>SETUP</span>
      </div>
    {/if}

    <!-- Bottom Playtime Gradient Overlay -->
    {#if playtimeSeconds > 0}
      <div class="absolute inset-x-0 bottom-0 pt-6 pb-2 px-2.5 bg-gradient-to-t from-black/90 via-black/50 to-transparent flex items-end justify-between text-4xs font-mono text-neutral-300 pointer-events-none">
        <span>{formatPlaytime(playtimeSeconds)}</span>
        {#if lastPlayed > 0}
          <span class="text-neutral-400">{formatLastPlayed(lastPlayed)}</span>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Title & Metadata Caption -->
  <div class="w-full mt-2 space-y-0.5 min-w-0">
    <h4 class="text-xs font-bold text-neutral-200 group-hover:text-white truncate transition-colors" title={name}>
      {name}
    </h4>
    {#if diskSizeBytes > 0}
      <p class="text-4xs font-mono text-neutral-500 truncate">
        {formatBytes(diskSizeBytes)}
      </p>
    {/if}
  </div>
</button>
