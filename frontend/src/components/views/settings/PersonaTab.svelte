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

  let customCountryCode = '';

  $: isCustomCountry = !countriesList.some(c => c.value === userCfgDraft.ip_country) && userCfgDraft.ip_country !== '';
  $: isValidSteamId = /^\d{17}$/.test((userCfgDraft.account_steamid || '').trim());

  function generateRandomSteamID() {
    const base = BigInt('76561197960265728');
    const randomOffset = BigInt(Math.floor(Math.random() * 1000000000));
    userCfgDraft = {
      ...userCfgDraft,
      account_steamid: (base + randomOffset).toString()
    };
  }

  function handleCountrySelect(e) {
    const val = e.detail;
    if (val === 'CUSTOM') {
      if (!customCountryCode) customCountryCode = 'US';
      userCfgDraft.ip_country = customCountryCode.toUpperCase();
    } else {
      userCfgDraft.ip_country = val;
    }
    userCfgDraft = { ...userCfgDraft };
  }

  function handleCustomCountryInput(e) {
    const code = e.target.value.toUpperCase().replace(/[^A-Z]/g, '').slice(0, 2);
    customCountryCode = code;
    userCfgDraft.ip_country = code;
    userCfgDraft = { ...userCfgDraft };
  }
</script>

<div class="space-y-3">
  <SettingSection title="Steam & Persona Emulation">
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

      <div class="space-y-1">
        <GhostInput
          label="Steam ID (64-Bit)"
          description="Unique 17-digit numerical identifier for local emulator save isolation and DLC entitlement"
          bind:value={userCfgDraft.account_steamid}
          placeholder="76561197960287930"
          mono={true}
        >
          <button
            slot="action"
            type="button"
            on:click={generateRandomSteamID}
            class="px-3 py-1.5 bg-[#18181b] hover:bg-[#222225] border border-[#27272a] text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
            title="Generate a new randomized Steam ID"
          >
            <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Random</span>
          </button>
        </GhostInput>
        {#if userCfgDraft.account_steamid && !isValidSteamId}
          <p class="text-xs text-amber-400 font-mono px-1">
            ⚠️ Warning: Standard SteamID64 format should be exactly 17 digits.
          </p>
        {/if}
      </div>

      <div class="space-y-1.5">
        <InheritSelect
          label="Region Code (IP Country)"
          description="Two-letter ISO country code reported by Steamworks emulation (e.g. US, DE, JP)"
          value={isCustomCountry ? 'CUSTOM' : userCfgDraft.ip_country}
          options={[...countriesList, { value: 'CUSTOM', label: 'Other (Custom 2-Letter ISO)...' }]}
          on:change={handleCountrySelect}
        />

        {#if isCustomCountry || userCfgDraft.ip_country === 'CUSTOM'}
          <div class="p-2.5 bg-[#101013] border border-[#27272a] rounded-xl flex items-center justify-between gap-3 animate-fade-in">
            <span class="text-xs font-semibold text-neutral-300">Custom ISO Country Code:</span>
            <div class="flex items-center space-x-1.5">
              <input
                type="text"
                maxlength="2"
                value={userCfgDraft.ip_country === 'CUSTOM' ? customCountryCode : userCfgDraft.ip_country}
                on:input={handleCustomCountryInput}
                placeholder="US"
                class="w-16 bg-[#141417] border border-[#27272a] focus:border-neutral-400 rounded-lg px-2.5 py-1 text-xs text-white font-mono uppercase text-center outline-none"
              />
            </div>
          </div>
        {/if}
      </div>
    </div>
  </SettingSection>
</div>
