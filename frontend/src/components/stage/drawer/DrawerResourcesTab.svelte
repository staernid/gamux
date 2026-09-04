<script>
  import { AppAPI } from '../../../api/app.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import { isSteamApp } from '../../../utils/game.js';

  export let game = null;

  $: name = game?.name || '';
  $: appId = game?.appId || '';
  $: gogId = game?.gogId || '';
  $: isSteam = isSteamApp(appId);
  $: isGOG = Boolean(gogId || (game?.store && game.store.toLowerCase().includes('gog')));

  // Computed URL endpoints
  $: pcgwUrl = isSteam
    ? `https://www.pcgamingwiki.com/api/appid.php?appid=${encodeURIComponent(appId)}`
    : isGOG && gogId
    ? `https://www.pcgamingwiki.com/api/gog.php?page=${encodeURIComponent(gogId)}`
    : `https://www.pcgamingwiki.com/w/index.php?search=${encodeURIComponent(name)}`;

  $: protonDbUrl = isSteam ? `https://www.protondb.com/app/${encodeURIComponent(appId)}` : `https://www.protondb.com/search?q=${encodeURIComponent(name)}`;
  $: steamDbUrl = isSteam ? `https://steamdb.info/app/${encodeURIComponent(appId)}/` : `https://steamdb.info/search/?a=app&q=${encodeURIComponent(name)}`;
  $: steamStoreUrl = isSteam ? `https://store.steampowered.com/app/${encodeURIComponent(appId)}` : null;
  $: steamGuidesUrl = isSteam ? `https://steamcommunity.com/app/${encodeURIComponent(appId)}/guides/` : null;
  $: steamDiscussionsUrl = isSteam ? `https://steamcommunity.com/app/${encodeURIComponent(appId)}/discussions/` : null;
  $: gogStoreUrl = `https://www.gog.com/en/games?query=${encodeURIComponent(name)}`;
  $: sgdbUrl = `https://www.steamgriddb.com/search/grids?term=${encodeURIComponent(name)}`;
  $: hltbUrl = `https://howlongtobeat.com/?q=${encodeURIComponent(name)}`;
  $: nexusModsUrl = `https://www.nexusmods.com/search/?gsearch=${encodeURIComponent(name)}`;
  $: wineHqUrl = `https://appdb.winehq.org/objectManager.php?sClass=application&sTitle=${encodeURIComponent(name)}`;

  async function openLink(url, title) {
    if (!url) return;
    try {
      if (typeof AppAPI.openURL === 'function') {
        await AppAPI.openURL(url);
      } else if (typeof window !== 'undefined' && window.runtime?.BrowserOpenURL) {
        window.runtime.BrowserOpenURL(url);
      } else {
        window.open(url, '_blank', 'noopener,noreferrer');
      }
      notificationStore.info('Opening Link', `Launched ${title} in web browser`);
    } catch (err) {
      notificationStore.error('Failed to open link', err?.message || String(err));
    }
  }

  async function copyLink(url, title) {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      notificationStore.info('Link Copied', `Copied ${title} URL to clipboard`);
    } catch (err) {
      notificationStore.error('Copy Failed', 'Could not copy URL to clipboard');
    }
  }
</script>

<div class="space-y-6 animate-fade-in text-slate-200">
  <!-- Header Info Banner -->
  <div class="bg-[#0e0e14] border border-[#1e1e2c] rounded-2xl p-5 shadow-xl flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div class="flex items-center space-x-3">
      <div class="p-2.5 rounded-xl bg-cyan-500/10 border border-cyan-500/20 text-cyan-400">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
        </svg>
      </div>
      <div>
        <h3 class="font-bold text-white text-base">Game Resources & Community Hub</h3>
        <p class="text-xs text-slate-400">External databases, compatibility reports, mod repositories, and wiki articles</p>
      </div>
    </div>

    <!-- Quick PCGW CTA Pill -->
    <button
      type="button"
      on:click={() => openLink(pcgwUrl, 'PCGamingWiki')}
      class="px-4 py-2 rounded-xl bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white text-xs font-bold transition-all shadow-lg flex items-center space-x-2 flex-shrink-0 cursor-pointer active:scale-95"
    >
      <svg class="w-3.5 h-3.5 opacity-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
      </svg>
      <span>Open PCGamingWiki</span>
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
      </svg>
    </button>
  </div>

  <!-- Category 1: Technical & Compatibility -->
  <div class="space-y-3">
    <h4 class="text-xs font-bold uppercase tracking-wider text-cyan-400 flex items-center space-x-2">
      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      <span>Technical & Compatibility Databases</span>
    </h4>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3.5 text-xs">
      <!-- PCGamingWiki Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-cyan-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-cyan-300 transition-colors">PCGamingWiki</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-cyan-500/10 text-cyan-300 border border-cyan-500/20">Wiki / Fixes</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Essential configuration fixes, save game file paths, FOV calculators, and technical issues.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(pcgwUrl, 'PCGamingWiki')}
            class="flex-1 bg-[#1a1a28] hover:bg-cyan-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>Open Wiki</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
          <button
            type="button"
            on:click={() => copyLink(pcgwUrl, 'PCGamingWiki')}
            title="Copy URL"
            class="p-1.5 bg-[#1a1a28] hover:bg-[#252538] text-slate-400 hover:text-white rounded-xl transition cursor-pointer"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
        </div>
      </div>

      <!-- ProtonDB Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-purple-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-purple-300 transition-colors">ProtonDB</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-purple-500/10 text-purple-300 border border-purple-500/20">Linux Gaming</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Community Linux compatibility reports, Proton GE recommendations, and launch parameter tweaks.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(protonDbUrl, 'ProtonDB')}
            class="flex-1 bg-[#1a1a28] hover:bg-purple-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>View Reports</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
          <button
            type="button"
            on:click={() => copyLink(protonDbUrl, 'ProtonDB')}
            title="Copy URL"
            class="p-1.5 bg-[#1a1a28] hover:bg-[#252538] text-slate-400 hover:text-white rounded-xl transition cursor-pointer"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
        </div>
      </div>

      <!-- SteamDB Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-amber-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-amber-300 transition-colors">SteamDB</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-amber-500/10 text-amber-300 border border-amber-500/20">Depots & Builds</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Depot manifests, update histories, launch configurations, and technical database listings.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(steamDbUrl, 'SteamDB')}
            class="flex-1 bg-[#1a1a28] hover:bg-amber-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>Explore SteamDB</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
          <button
            type="button"
            on:click={() => copyLink(steamDbUrl, 'SteamDB')}
            title="Copy URL"
            class="p-1.5 bg-[#1a1a28] hover:bg-[#252538] text-slate-400 hover:text-white rounded-xl transition cursor-pointer"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
        </div>
      </div>

      <!-- WineHQ AppDB Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-emerald-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-emerald-300 transition-colors">WineHQ AppDB</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-emerald-500/10 text-emerald-300 border border-emerald-500/20">Wine Staging</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Legacy Windows compatibility reports, DLL overrides, and standalone Wine execution tests.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(wineHqUrl, 'WineHQ AppDB')}
            class="flex-1 bg-[#1a1a28] hover:bg-emerald-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>Search AppDB</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
          <button
            type="button"
            on:click={() => copyLink(wineHqUrl, 'WineHQ AppDB')}
            title="Copy URL"
            class="p-1.5 bg-[#1a1a28] hover:bg-[#252538] text-slate-400 hover:text-white rounded-xl transition cursor-pointer"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Category 2: Store & Community -->
  <div class="space-y-3">
    <h4 class="text-xs font-bold uppercase tracking-wider text-sky-400 flex items-center space-x-2">
      <svg class="w-3.5 h-3.5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
      </svg>
      <span>Storefront & Community Centers</span>
    </h4>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3.5 text-xs">
      <!-- Steam Store / GOG Store Card -->
      {#if isSteam && steamStoreUrl}
        <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-sky-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="font-bold text-white text-sm group-hover:text-sky-300 transition-colors">Steam Store</span>
              <span class="px-2 py-0.5 rounded text-3xs font-mono bg-sky-500/10 text-sky-300 border border-sky-500/20">Official Store</span>
            </div>
            <p class="text-slate-400 text-xs line-clamp-2">
              Official store page, DLC packages, system specs, trailers, and developer updates.
            </p>
          </div>
          <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
            <button
              type="button"
              on:click={() => openLink(steamStoreUrl, 'Steam Store')}
              class="flex-1 bg-[#1a1a28] hover:bg-sky-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
            >
              <span>View Store Page</span>
              <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            </button>
          </div>
        </div>
      {:else}
        <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-purple-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="font-bold text-white text-sm group-hover:text-purple-300 transition-colors">GOG Store</span>
              <span class="px-2 py-0.5 rounded text-3xs font-mono bg-purple-500/10 text-purple-300 border border-purple-500/20">DRM-Free Store</span>
            </div>
            <p class="text-slate-400 text-xs line-clamp-2">
              GOG.com catalog page, soundtrack downloads, and bonus goodies.
            </p>
          </div>
          <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
            <button
              type="button"
              on:click={() => openLink(gogStoreUrl, 'GOG Store')}
              class="flex-1 bg-[#1a1a28] hover:bg-purple-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
            >
              <span>Search GOG</span>
              <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            </button>
          </div>
        </div>
      {/if}

      <!-- Steam Guides Card -->
      {#if isSteam && steamGuidesUrl}
        <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-blue-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="font-bold text-white text-sm group-hover:text-blue-300 transition-colors">Steam Guides</span>
              <span class="px-2 py-0.5 rounded text-3xs font-mono bg-blue-500/10 text-blue-300 border border-blue-500/20">Community</span>
            </div>
            <p class="text-slate-400 text-xs line-clamp-2">
              Community walkthroughs, achievement guides, secret locations, and optimization tips.
            </p>
          </div>
          <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
            <button
              type="button"
              on:click={() => openLink(steamGuidesUrl, 'Steam Guides')}
              class="flex-1 bg-[#1a1a28] hover:bg-blue-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
            >
              <span>Open Guides</span>
              <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            </button>
          </div>
        </div>
      {/if}

      <!-- Steam Community Discussions Card -->
      {#if isSteam && steamDiscussionsUrl}
        <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-indigo-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="font-bold text-white text-sm group-hover:text-indigo-300 transition-colors">Discussions</span>
              <span class="px-2 py-0.5 rounded text-3xs font-mono bg-indigo-500/10 text-indigo-300 border border-indigo-500/20">Forums</span>
            </div>
            <p class="text-slate-400 text-xs line-clamp-2">
              Official community board for gameplay questions, technical issues, and player feedback.
            </p>
          </div>
          <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
            <button
              type="button"
              on:click={() => openLink(steamDiscussionsUrl, 'Steam Discussions')}
              class="flex-1 bg-[#1a1a28] hover:bg-indigo-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
            >
              <span>Open Discussions</span>
              <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <!-- Category 3: Media & Customization -->
  <div class="space-y-3">
    <h4 class="text-xs font-bold uppercase tracking-wider text-pink-400 flex items-center space-x-2">
      <svg class="w-3.5 h-3.5 text-pink-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
      </svg>
      <span>Media, Mods & Playtime Databases</span>
    </h4>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3.5 text-xs">
      <!-- SteamGridDB Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-pink-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-pink-300 transition-colors">SteamGridDB</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-pink-500/10 text-pink-300 border border-pink-500/20">Artwork</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Community custom poster covers, wide banners, hero backdrops, and transparent logos.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(sgdbUrl, 'SteamGridDB')}
            class="flex-1 bg-[#1a1a28] hover:bg-pink-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>Search Grids</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
        </div>
      </div>

      <!-- HowLongToBeat Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-teal-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-teal-300 transition-colors">HowLongToBeat</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-teal-500/10 text-teal-300 border border-teal-500/20">Playtime</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Crowdsourced playthrough completion times for main story, extras, and 100% completion.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(hltbUrl, 'HowLongToBeat')}
            class="flex-1 bg-[#1a1a28] hover:bg-teal-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>Check Playtime</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Nexus Mods Card -->
      <div class="bg-[#0e0e14] hover:bg-[#151520] border border-[#1e1e2c] hover:border-orange-500/40 p-4 rounded-2xl transition shadow-lg flex flex-col justify-between space-y-3 group">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-sm group-hover:text-orange-300 transition-colors">Nexus Mods</span>
            <span class="px-2 py-0.5 rounded text-3xs font-mono bg-orange-500/10 text-orange-300 border border-orange-500/20">Modifications</span>
          </div>
          <p class="text-slate-400 text-xs line-clamp-2">
            Community game modifications, custom texture packs, shaders, UI tweaks, and loaders.
          </p>
        </div>
        <div class="flex items-center space-x-2 pt-2 border-t border-[#1a1a26]">
          <button
            type="button"
            on:click={() => openLink(nexusModsUrl, 'Nexus Mods')}
            class="flex-1 bg-[#1a1a28] hover:bg-orange-600 text-slate-200 hover:text-white py-1.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-1.5 cursor-pointer"
          >
            <span>Search Mods</span>
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</div>
