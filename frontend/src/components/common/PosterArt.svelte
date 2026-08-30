<script>
  import { getInitials } from '../../utils/game.js';

  export let appId = '';
  export let name = '';
  export let posterUrl = '';
  export let localCoverUrl = '';
  export let className = 'w-full h-full object-cover';

  $: isForcedNone = (posterUrl === 'none' || localCoverUrl === 'none');

  $: candidateUrls = (() => {
    if (isForcedNone) return [];
    const urls = [];
    if (localCoverUrl && localCoverUrl !== 'none') urls.push(localCoverUrl);
    if (posterUrl && posterUrl !== 'none' && posterUrl !== localCoverUrl) urls.push(posterUrl);
    if (appId && appId !== '0' && appId !== 'Custom' && /^\d+$/.test(String(appId).trim())) {
      urls.push(`https://cdn.cloudflare.steamstatic.com/steam/apps/${appId}/library_600x900.jpg`);
      urls.push(`https://cdn.cloudflare.steamstatic.com/steam/apps/${appId}/capsule_616x353.jpg`);
      urls.push(`https://cdn.cloudflare.steamstatic.com/steam/apps/${appId}/header.jpg`);
    }
    return urls;
  })();

  let urlIndex = 0;
  let imgFailed = false;
  let candidateKey = '';

  $: {
    const key = `${isForcedNone}|${localCoverUrl}|${posterUrl}|${appId}`;
    if (key !== candidateKey) {
      candidateKey = key;
      urlIndex = 0;
      imgFailed = false;
    }
  }

  $: activeUrl = candidateUrls[urlIndex] || '';

  function handleImgError() {
    if (urlIndex + 1 < candidateUrls.length) {
      urlIndex += 1;
    } else {
      imgFailed = true;
    }
  }

  function handleImgLoad() {
    imgFailed = false;
  }

  $: initials = getInitials(name);
</script>

{#if activeUrl && !imgFailed}
  <img
    src={activeUrl}
    alt={name}
    class={className}
    on:error={handleImgError}
    on:load={handleImgLoad}
    decoding="async"
    draggable="false"
  />
{:else}
  <div class="w-full h-full art-fallback-clean flex flex-col items-center justify-center p-4 text-center select-none bg-[#161622]">
    <div class="w-12 h-12 rounded-2xl bg-[#252538] border border-[#3b475e] flex items-center justify-center text-white font-black text-sm shadow-lg mb-2">
      {initials}
    </div>
    <span class="text-xs font-semibold text-slate-200 line-clamp-2 max-w-[85%] leading-tight">
      {name || 'Unknown Game'}
    </span>
  </div>
{/if}
