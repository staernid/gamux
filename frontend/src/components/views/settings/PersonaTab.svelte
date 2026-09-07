<script>
  import SettingSection from '../../common/SettingSection.svelte';
  import SteamPersonaCard from './SteamPersonaCard.svelte';

  export let userCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  export let languagesList = [];
  export let countriesList = [];

  let importedAvatarUrl = '';
</script>

<div class="space-y-6">
  <!-- Steam Persona Profile Card -->
  <SettingSection title="Steam & Persona Emulation">
    <div class="space-y-3">
      <p class="text-xs text-neutral-400">
        Local Steamworks persona configuration. Sets player nickname, emulated SteamID64, and language injected into Goldberg emulators.
      </p>

      <SteamPersonaCard
        bind:accountName={userCfgDraft.account_name}
        bind:accountSteamId={userCfgDraft.account_steamid}
        bind:language={userCfgDraft.language}
        bind:ipCountry={userCfgDraft.ip_country}
        bind:avatarUrl={importedAvatarUrl}
        {languagesList}
        {countriesList}
        on:change={() => { userCfgDraft = { ...userCfgDraft }; }}
      />
    </div>
  </SettingSection>
</div>
