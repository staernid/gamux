<script>
  import SettingSection from '../../common/SettingSection.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';

  export let userCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };
  export let languagesList = [];
  export let countriesList = [];

  function generateRandomSteamID() {
    const base = BigInt('76561197960265728');
    const randomOffset = BigInt(Math.floor(Math.random() * 1000000000));
    userCfgDraft.account_steamid = (base + randomOffset).toString();
  }
</script>

<div class="space-y-3">
  <SettingSection title="Virtual Steam Persona">
    <div class="space-y-2.5">
      <GhostInput
        label="Display Name"
        description="Persona nickname used in local emulated multiplayer and offline save profiles"
        bind:value={userCfgDraft.account_name}
        placeholder="GamuxPlayer"
      />

      <InheritSelect
        label="Emulation Language"
        description="Default language injected into Steam emulators and Goldberg Steamworks wrappers"
        bind:value={userCfgDraft.language}
        options={languagesList}
      />

      <GhostInput
        label="Steam ID (64-Bit)"
        description="Unique 64-bit numerical identifier for local emulator save isolation and DLC entitlement"
        bind:value={userCfgDraft.account_steamid}
        placeholder="76561197960287930"
        mono={true}
      >
        <button
          slot="action"
          type="button"
          on:click={generateRandomSteamID}
          class="px-3 py-1.5 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
          title="Generate a new randomized Steam ID"
        >
          <svg class="w-3.5 h-3.5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Random</span>
        </button>
      </GhostInput>

      <InheritSelect
        label="Region Code (IP Country)"
        description="Two-letter ISO country code reported by Steamworks emulation (e.g. US, DE, JP)"
        bind:value={userCfgDraft.ip_country}
        options={countriesList}
      />
    </div>
  </SettingSection>
</div>
