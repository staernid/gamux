<script>
  import { AppAPI } from '../../../api/app.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import GhostInput from '../../common/GhostInput.svelte';

  export let cfgDraft;

  async function handleBrowseArtworkDir() {
    const selected = await AppAPI.selectDirectory('Select Artwork Directory', cfgDraft.artwork_dir);
    if (selected) cfgDraft.artwork_dir = selected;
  }
</script>

<div class="space-y-3">
  <SettingSection
    title="Metadata & Artwork APIs"
    tooltip="Configure third-party API keys to enrich your library with game metadata, official news, and high-resolution artwork."
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
      />

      <GhostInput
        label="Hubcap Manifest API Key"
        description="Used to query Hubcap for game depot manifests and decryption keys during downloads"
        type="password"
        bind:value={cfgDraft.hubcap_api_key}
        placeholder="Hubcap API token"
        mono={true}
        status={cfgDraft.hubcap_api_key ? 'SET' : ''}
      />

      <GhostInput
        label="SteamGridDB API Key"
        description="Automated downloads for high-resolution vertical covers, banners, hero artwork, and clear logos"
        type="password"
        bind:value={cfgDraft.steamgriddb_api_key}
        placeholder="SteamGridDB API token"
        mono={true}
        status={cfgDraft.steamgriddb_api_key ? 'SET' : ''}
      />

      <GhostInput
        label="RAWG API Key"
        description="Provides fallback descriptions, release dates, genres, and metadata discovery for standalone titles"
        type="password"
        bind:value={cfgDraft.rawg_api_key}
        placeholder="RAWG API key"
        mono={true}
        status={cfgDraft.rawg_api_key ? 'SET' : ''}
      />

      <GhostInput
        label="Artwork Storage Directory"
        description="Local filesystem cache directory where downloaded covers, heroes, and logos are persisted"
        bind:value={cfgDraft.artwork_dir}
        placeholder="~/.local/share/gamux/art"
        mono={true}
      >
        <button
          slot="action"
          type="button"
          on:click={handleBrowseArtworkDir}
          class="px-3 py-1.5 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
        >
          Browse
        </button>
      </GhostInput>
    </div>
  </SettingSection>
</div>
