<script>
  import { getInitials } from '../../utils/game.js';

  export let appId = '';
  export let name = '';
  export let className = 'w-full h-full object-cover';

  let imgFailed = false;

  $: capsuleUrl = (appId && appId !== '0' && appId !== 'Custom')
    ? `https://cdn.akamai.steamstatic.com/steam/apps/${appId}/header.jpg`
    : '';

  $: initials = getInitials(name);
</script>

{#if capsuleUrl && !imgFailed}
  <img
    src={capsuleUrl}
    alt={name}
    class={className}
    on:error={() => { imgFailed = true; }}
    loading="lazy"
  />
{:else}
  <div class="w-full h-full art-fallback-clean flex flex-col items-center justify-center p-4 text-center select-none">
    <div class="w-11 h-11 rounded-xl bg-[#283142] border border-[#3b475e] flex items-center justify-center text-white font-black text-sm shadow-md mb-2">
      {initials}
    </div>
    <span class="text-xs font-semibold text-slate-200 line-clamp-2 max-w-[85%] leading-tight">
      {name || 'Unknown Game'}
    </span>
  </div>
{/if}
