<script>
    import GhostInput from '../../common/GhostInput.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import { AppAPI } from '../../../api/app.js';
  import { generateRandomSteamID as createRandomSteamID, isValidSteamID64 } from '../../../utils/steam.js';








  /**
   * @typedef {Object} Props
   * @property {string} [accountName]
   * @property {string} [accountSteamId]
   * @property {string} [language]
   * @property {string} [ipCountry]
   * @property {string} [avatarUrl]
   * @property {{ value: string, label: string }[]} [languagesList]
   * @property {{ value: string, label: string }[]} [countriesList]
   * @property {string} [collisionSteamId]
   * @property {string} [nameLabel]
   * @property {string} [nameDescription]
   * @property {string} [namePlaceholder]
   * @property {string} [langLabel]
   * @property {string} [langDescription]
   * @property {string} [steamidLabel]
   * @property {string} [steamidDescription]
   * @property {string} [steamidPlaceholder]
   * @property {string} [countryLabel]
   * @property {string} [countryDescription]
   * @property {string} [syncTitle]
   * @property {string} [syncSubtitle]
   * @property {string} [syncPlaceholder]
   * @property {boolean} [showSync]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    accountName = $bindable('GamuxPlayer'),
    accountSteamId = $bindable('76561197960287930'),
    language = $bindable('english'),
    ipCountry = $bindable('US'),
    avatarUrl = $bindable(''),
    languagesList = [],
    countriesList = [],
    collisionSteamId = '',
    nameLabel = 'Display Name',
    nameDescription = 'Persona nickname used in single player, host LAN sessions, and offline saves',
    namePlaceholder = 'GamuxPlayer',
    langLabel = 'Emulation Language',
    langDescription = 'Default language injected into Steam emulators and Goldberg Steamworks wrappers',
    steamidLabel = 'Steam ID (64-Bit)',
    steamidDescription = 'Unique 17-digit identifier for local emulator save isolation and DLC entitlement',
    steamidPlaceholder = '76561197960287930',
    countryLabel = 'Region Code (IP Country)',
    countryDescription = 'Two-letter ISO country code reported by Steamworks emulation (e.g. US, DE, JP)',
    syncTitle = 'Auto-Sync from Steam Profile',
    syncSubtitle = 'Enter custom URL, vanity name, or SteamID64 to sync persona and avatar',
    syncPlaceholder = 'e.g. gaben or https://steamcommunity.com/id/...',
    showSync = true,
    children
  ,
    onchange
  } = $props();

  let customCountryCode = $state('');
  let steamProfileInput = $state('');
  let isResolvingProfile = $state(false);
  let profileError = $state('');
  let profileSuccess = $state('');

  let isCustomCountry = $derived(!countriesList.some(c => c.value === ipCountry) && ipCountry !== '');
  let isValidSteamId = $derived(isValidSteamID64(accountSteamId));
  let hasCollision = $derived(Boolean(collisionSteamId && accountSteamId && collisionSteamId === accountSteamId));

  function handleRandomSteamID() {
    accountSteamId = createRandomSteamID();
    onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl });
  }

  async function handleImportSteamProfile() {
    if (!steamProfileInput.trim()) return;
    isResolvingProfile = true;
    profileError = '';
    profileSuccess = '';
    try {
      const profile = await AppAPI.resolveSteamProfile(steamProfileInput.trim());
      if (profile) {
        if (profile.persona_name) {
          accountName = profile.persona_name;
        }
        if (profile.steam_id) {
          accountSteamId = profile.steam_id;
        }
        if (profile.country_code && profile.country_code.length === 2) {
          ipCountry = profile.country_code.toUpperCase();
        }
        if (profile.avatar_url) {
          avatarUrl = profile.avatar_url;
        }
        profileSuccess = `Imported profile: ${profile.persona_name} (${profile.steam_id})`;
        onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl });
      }
    } catch (err) {
      profileError = err?.message || 'Failed to resolve Steam profile.';
    } finally {
      isResolvingProfile = false;
    }
  }

  function handleCountrySelect(e) {
    const val = e.detail;
    if (val === 'CUSTOM') {
      if (!customCountryCode) customCountryCode = 'US';
      ipCountry = customCountryCode.toUpperCase();
    } else {
      ipCountry = val;
    }
    onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl });
  }

  function handleCustomCountryInput(e) {
    customCountryCode = e.target.value.toUpperCase();
    ipCountry = customCountryCode;
    onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl });
  }
</script>

<div class="space-y-3">
  <!-- 1-Click Steam Profile Sync -->
  {#if showSync}
    <div class="p-3 bg-surface-header border border-surface-border rounded-xl space-y-2.5">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center space-x-2.5 min-w-0">
          {#if avatarUrl}
            <img src={avatarUrl} alt="Avatar" class="w-7 h-7 rounded-full border border-neutral-700 object-cover shrink-0" />
          {:else}
            <div class="w-7 h-7 rounded-full bg-neutral-800 border border-neutral-700 flex items-center justify-center shrink-0 text-neutral-400">
              <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
              </svg>
            </div>
          {/if}
          <div class="min-w-0">
            <span class="text-xs font-semibold text-neutral-200 block">{syncTitle}</span>
            <span class="text-3xs text-neutral-400 leading-relaxed block">{syncSubtitle}</span>
          </div>
        </div>
      </div>

      <div class="flex items-center space-x-2">
        <input
          type="text"
          bind:value={steamProfileInput}
          placeholder={syncPlaceholder}
          onkeydown={(e) => { if (e.key === 'Enter') handleImportSteamProfile(); }}
          class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-3 py-1.5 text-xs text-white placeholder-neutral-500 font-mono outline-none"
        />
        <button
          type="button"
          onclick={handleImportSteamProfile}
          disabled={isResolvingProfile || !steamProfileInput.trim()}
          class="px-3.5 py-1.5 bg-neutral-800 hover:bg-neutral-700 disabled:opacity-50 disabled:cursor-not-allowed border border-neutral-600 text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer shrink-0"
        >
          {#if isResolvingProfile}
            <div class="w-3.5 h-3.5 border-2 border-neutral-400/30 border-t-white rounded-full animate-spin"></div>
            <span>Importing...</span>
          {:else}
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>
            <span>Import</span>
          {/if}
        </button>
      </div>

      {#if profileError}
        <div class="text-2xs text-rose-400 font-mono px-1">
          {profileError}
        </div>
      {/if}
      {#if profileSuccess}
        <div class="text-2xs text-emerald-400 font-mono px-1">
          ✓ {profileSuccess}
        </div>
      {/if}
    </div>
  {/if}

  <div class="space-y-2.5">
    <GhostInput
      label={nameLabel}
      description={nameDescription}
      bind:value={accountName}
      placeholder={namePlaceholder}
      oninput={() => onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl })}
    />

    <InheritSelect
      label={langLabel}
      description={langDescription}
      bind:value={language}
      options={languagesList}
      onchange={(val) => {
        language = val;
        onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl });
      }}
    />

    <div class="space-y-1">
      <GhostInput
        label={steamidLabel}
        description={steamidDescription}
        bind:value={accountSteamId}
        placeholder={steamidPlaceholder}
        mono={true}
        oninput={() => onchange?.({ accountName, accountSteamId, language, ipCountry, avatarUrl })}
      >
        {#snippet action()}
                <button
            
            type="button"
            onclick={handleRandomSteamID}
            class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer shrink-0"
            title="Generate a new randomized Steam ID"
          >
            <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Random</span>
          </button>
              {/snippet}
      </GhostInput>

      {#if accountSteamId && !isValidSteamId}
        <div class="flex items-center space-x-1.5 text-xs text-amber-400 font-mono px-1">
          <svg class="w-3.5 h-3.5 shrink-0 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <span>Warning: Standard SteamID64 format should be exactly 17 digits.</span>
        </div>
      {/if}

      {#if hasCollision}
        <div class="flex items-center space-x-1.5 text-xs text-rose-400 font-mono px-1">
          <svg class="w-3.5 h-3.5 shrink-0 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Collision: Player 1 and Player 2 have the same SteamID. Click "Random" to generate a unique ID.</span>
        </div>
      {/if}
    </div>

    <div class="space-y-1.5">
      <InheritSelect
        label={countryLabel}
        description={countryDescription}
        value={isCustomCountry ? 'CUSTOM' : ipCountry}
        options={[...countriesList, { value: 'CUSTOM', label: 'Other (Custom 2-Letter ISO)...' }]}
        onchange={handleCountrySelect}
      />

      {#if isCustomCountry || ipCountry === 'CUSTOM'}
        <div class="p-2.5 bg-surface-input border border-surface-border rounded-xl flex items-center justify-between gap-3 animate-fade-in">
          <span class="text-xs font-semibold text-neutral-300">Custom ISO Country Code:</span>
          <div class="flex items-center space-x-1.5">
            <input
              type="text"
              maxlength="2"
              value={ipCountry === 'CUSTOM' ? customCountryCode : ipCountry}
              oninput={handleCustomCountryInput}
              placeholder="US"
              class="w-16 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-2.5 py-1 text-xs text-white font-mono uppercase text-center outline-none"
            />
          </div>
        </div>
      {/if}
    </div>

    {@render children?.()}
  </div>
</div>
