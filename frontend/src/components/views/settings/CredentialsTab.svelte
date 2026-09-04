<script>
  import { AppAPI } from '../../../api/app.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import GhostInput from '../../common/GhostInput.svelte';

  export let cfgDraft;

  function openExternal(url) {
    if (!url) return;
    try {
      AppAPI.openURL(url);
    } catch {
      window.open(url, '_blank');
    }
  }
</script>

<div class="space-y-3">
  <SettingSection
    title="API Keys & Service Credentials"
    tooltip="Configure third-party API keys to enrich your library with game metadata, official news, community depot manifests, and high-resolution artwork."
  >
    <div class="space-y-2.5">
      <GhostInput
        label="Steam Web API Key"
        description="Used to fetch official Steam depot manifests, DLC catalogs, community news, and store metadata"
        type="password"
        bind:value={cfgDraft.steam_web_api_key}
        placeholder="32-char hex key (e.g. 7F8A1B2C...)"
        mono={true}
        status={cfgDraft.steam_web_api_key ? 'SET' : ''}
      >
        <button
          slot="action"
          type="button"
          on:click={() => openExternal('https://steamcommunity.com/dev/apikey')}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open Steam Community API key generation page"
        >
          <span>Get Key</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
      </GhostInput>

      <GhostInput
        label="Hubcap Manifest API Key"
        description="Used to query Hubcap for game depot manifests and decryption keys during downloads"
        type="password"
        bind:value={cfgDraft.hubcap_api_key}
        placeholder="Hubcap API token"
        mono={true}
        status={cfgDraft.hubcap_api_key ? 'SET' : ''}
      >
        <button
          slot="action"
          type="button"
          on:click={() => openExternal('https://github.com/mikeroyal/Steam-Guide')}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open Hubcap manifest documentation and access guide"
        >
          <span>Docs</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
      </GhostInput>

      <GhostInput
        label="SteamGridDB API Key"
        description="Automated downloads for high-resolution vertical covers, banners, hero artwork, and clear logos"
        type="password"
        bind:value={cfgDraft.steamgriddb_api_key}
        placeholder="SteamGridDB API token"
        mono={true}
        status={cfgDraft.steamgriddb_api_key ? 'SET' : ''}
      >
        <button
          slot="action"
          type="button"
          on:click={() => openExternal('https://www.steamgriddb.com/profile/preferences/api')}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open SteamGridDB API key preferences page"
        >
          <span>Get Key</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
      </GhostInput>

      <GhostInput
        label="RAWG API Key"
        description="Provides fallback descriptions, release dates, genres, and metadata discovery for standalone titles"
        type="password"
        bind:value={cfgDraft.rawg_api_key}
        placeholder="RAWG API key"
        mono={true}
        status={cfgDraft.rawg_api_key ? 'SET' : ''}
      >
        <button
          slot="action"
          type="button"
          on:click={() => openExternal('https://rawg.io/apidocs')}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open RAWG API documentation and key signup"
        >
          <span>Get Key</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
      </GhostInput>
    </div>
  </SettingSection>
</div>
