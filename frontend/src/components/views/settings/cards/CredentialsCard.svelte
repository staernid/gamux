<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import GhostInput from '../../../common/GhostInput.svelte';
  import { openExternalUrl } from '../../../../utils/navigation.js';

  let { storageDraft = $bindable(), cfgDraft = $bindable() } = $props();
  let targetDraft = $derived(storageDraft || cfgDraft || {});
</script>

<SettingSection
  title="API Keys & Service Credentials"
  tooltip="Configure third-party API keys to enrich your library with game metadata, official news, community depot manifests, and high-resolution artwork."
>
  <div class="space-y-2.5">
    <GhostInput
      label="Steam Web API Key"
      description="Used to fetch official Steam depot manifests, DLC catalogs, community news, and store metadata"
      type="password"
      bind:value={targetDraft.steam_web_api_key}
      placeholder="32-char hex key (e.g. 7F8A1B2C...)"
      mono={true}
      status={targetDraft.steam_web_api_key ? 'SET' : ''}
    >
      {#snippet action()}
            <button
          
          type="button"
          onclick={() => openExternalUrl('https://steamcommunity.com/dev/apikey', { title: 'Steam Web API' })}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open Steam Community API key generation page"
        >
          <span>Get Key</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
          {/snippet}
    </GhostInput>

    <GhostInput
      label="Hubcap Manifest API Key"
      description="Used to query Hubcap for game depot manifests and decryption keys during downloads"
      type="password"
      bind:value={targetDraft.hubcap_api_key}
      placeholder="Hubcap API token"
      mono={true}
      status={targetDraft.hubcap_api_key ? 'SET' : ''}
    >
      {#snippet action()}
            <button
          
          type="button"
          onclick={() => openExternalUrl('https://github.com/mikeroyal/Steam-Guide', { title: 'Hubcap Documentation' })}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open Hubcap manifest documentation and access guide"
        >
          <span>Docs</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
          {/snippet}
    </GhostInput>

    <GhostInput
      label="SteamGridDB API Key"
      description="Automated downloads for high-resolution vertical covers, banners, hero artwork, and clear logos"
      type="password"
      bind:value={targetDraft.steamgriddb_api_key}
      placeholder="SteamGridDB API token"
      mono={true}
      status={targetDraft.steamgriddb_api_key ? 'SET' : ''}
    >
      {#snippet action()}
            <button
          
          type="button"
          onclick={() => openExternalUrl('https://www.steamgriddb.com/profile/preferences/api', { title: 'SteamGridDB API' })}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open SteamGridDB API key preferences page"
        >
          <span>Get Key</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
          {/snippet}
    </GhostInput>

    <GhostInput
      label="RAWG API Key"
      description="Provides fallback descriptions, release dates, genres, and metadata discovery for standalone titles"
      type="password"
      bind:value={targetDraft.rawg_api_key}
      placeholder="RAWG API key"
      mono={true}
      status={targetDraft.rawg_api_key ? 'SET' : ''}
    >
      {#snippet action()}
            <button
          
          type="button"
          onclick={() => openExternalUrl('https://rawg.io/apidocs', { title: 'RAWG API' })}
          class="px-2.5 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1 cursor-pointer flex-shrink-0"
          title="Open RAWG API documentation and key signup"
        >
          <span>Get Key</span>
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </button>
          {/snippet}
    </GhostInput>
  </div>
</SettingSection>
