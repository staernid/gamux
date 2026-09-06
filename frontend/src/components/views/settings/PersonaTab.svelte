<script>
  import SettingSection from '../../common/SettingSection.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import { generateRandomSteamID as createRandomSteamID, isValidSteamID64 } from '../../../utils/steam.js';

  export let userCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  export let languagesList = [];
  export let countriesList = [];

  let customCountryCodeP1 = '';

  $: isCustomCountryP1 = !countriesList.some(c => c.value === userCfgDraft.ip_country) && userCfgDraft.ip_country !== '';
  $: isValidSteamIdP1 = isValidSteamID64(userCfgDraft.account_steamid);

  function generateRandomSteamID() {
    userCfgDraft = {
      ...userCfgDraft,
      account_steamid: createRandomSteamID()
    };
  }

  function handleCountrySelectP1(e) {
    const val = e.detail;
    if (val === 'CUSTOM') {
      if (!customCountryCodeP1) customCountryCodeP1 = 'US';
      userCfgDraft.ip_country = customCountryCodeP1.toUpperCase();
    } else {
      userCfgDraft.ip_country = val;
    }
    userCfgDraft = { ...userCfgDraft };
  }

  function handleCustomCountryInput(e) {
    customCountryCodeP1 = e.target.value.toUpperCase();
    userCfgDraft.ip_country = customCountryCodeP1;
    userCfgDraft = { ...userCfgDraft };
  }
</script>

<div class="space-y-6">
  <!-- Steam Persona Profile Card -->
  <SettingSection title="Steam & Persona Emulation">
    <div class="space-y-3">
      <p class="text-xs text-neutral-400">
        Local Steamworks persona configuration. Sets player nickname, emulated SteamID64, and language injected into Goldberg emulators.
      </p>

      <div class="space-y-2.5">
        <GhostInput
          label="Display Name"
          description="Persona nickname used in single player, host LAN sessions, and offline saves"
          bind:value={userCfgDraft.account_name}
          placeholder="GamuxPlayer"
        />

        <InheritSelect
          label="Emulation Language"
          description="Default language injected into Steam emulators and Goldberg Steamworks wrappers"
          bind:value={userCfgDraft.language}
          options={languagesList}
        />

        <div class="space-y-1">
          <GhostInput
            label="Steam ID (64-Bit)"
            description="Unique 17-digit identifier for local emulator save isolation and DLC entitlement"
            bind:value={userCfgDraft.account_steamid}
            placeholder="76561197960287930"
            mono={true}
          >
            <button
              slot="action"
              type="button"
              on:click={generateRandomSteamID}
              class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
              title="Generate a new randomized Steam ID"
            >
              <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span>Random</span>
            </button>
          </GhostInput>
          {#if userCfgDraft.account_steamid && !isValidSteamIdP1}
            <div class="flex items-center space-x-1.5 text-xs text-amber-400 font-mono px-1">
              <svg class="w-3.5 h-3.5 flex-shrink-0 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span>Warning: Standard SteamID64 format should be exactly 17 digits.</span>
            </div>
          {/if}
        </div>

        <div class="space-y-1.5">
          <InheritSelect
            label="Region Code (IP Country)"
            description="Two-letter ISO country code reported by Steamworks emulation (e.g. US, DE, JP)"
            value={isCustomCountryP1 ? 'CUSTOM' : userCfgDraft.ip_country}
            options={[...countriesList, { value: 'CUSTOM', label: 'Other (Custom 2-Letter ISO)...' }]}
            on:change={handleCountrySelectP1}
          />

          {#if isCustomCountryP1 || userCfgDraft.ip_country === 'CUSTOM'}
            <div class="p-2.5 bg-[#101013] border border-surface-border rounded-xl flex items-center justify-between gap-3 animate-fade-in">
              <span class="text-xs font-semibold text-neutral-300">Custom ISO Country Code:</span>
              <div class="flex items-center space-x-1.5">
                <input
                  type="text"
                  maxlength="2"
                  value={userCfgDraft.ip_country === 'CUSTOM' ? customCountryCodeP1 : userCfgDraft.ip_country}
                  on:input={handleCustomCountryInput}
                  placeholder="US"
                  class="w-16 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-2.5 py-1 text-xs text-white font-mono uppercase text-center outline-none"
                />
              </div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </SettingSection>
</div>
