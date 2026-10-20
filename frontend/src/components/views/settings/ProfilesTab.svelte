<script>
  import SettingSection from '../../common/SettingSection.svelte';
  import SteamPersonaCard from './SteamPersonaCard.svelte';
  import Icon from '../../common/Icon.svelte';
  import { profilesStore } from '../../../stores/profiles.js';
  import { modalStore } from '../../../stores/modal.js';

  let { languagesList = [], countriesList = [] } = $props();

  let active = $derived(profilesStore.activeProfile);

  function handlePersonaChange(e) {
    if (!active) return;
    const details = e.detail;
    const updated = {
      ...active,
      name: details.accountName,
      steamId: details.accountSteamId,
      language: details.language,
      ipCountry: details.ipCountry,
      avatarUrl: details.avatarUrl
    };
    profilesStore.save(updated);
  }

  function confirmDeleteProfile() {
    if (!active || active.isMain) return;
    const name = active.name || 'this profile';
    modalStore.open('confirm', {
      title: `Delete Profile "${name}"?`,
      message: `This will remove the profile configuration, isolated sandbox, and its Wine prefixes (~/Games/umu).`,
      confirmLabel: 'Delete Profile',
      cancelLabel: 'Keep Profile',
      danger: true,
      onConfirm: async () => {
        await profilesStore.delete(active.id);
      }
    });
  }
</script>

<div class="space-y-6">
  <!-- Profile Mode & Metadata Summary Banner -->
  <SettingSection title="Active Profile">
    <div class="p-4 rounded-2xl bg-surface-header/60 border border-surface-border flex items-center justify-between space-x-4">
      <div class="flex items-center space-x-3.5 min-w-0">
        <div class="w-12 h-12 rounded-full overflow-hidden flex-shrink-0 bg-surface-input border border-surface-borderHighlight flex items-center justify-center text-sm font-bold uppercase text-white shadow-md">
          {#if active?.avatarUrl}
            <img src={active.avatarUrl} alt={active.name} class="w-full h-full object-cover" />
          {:else}
            <span>{(active?.name || 'GP').slice(0, 2).toUpperCase()}</span>
          {/if}
        </div>
        <div class="min-w-0">
          <div class="flex items-center space-x-2">
            <h3 class="text-sm font-bold text-white truncate">{active?.name || 'GamuxPlayer'}</h3>
            <span class="text-4xs px-2 py-0.5 rounded font-mono font-semibold uppercase tracking-wider {active?.isMain ? 'bg-neutral-800 text-neutral-300 border border-neutral-700' : 'bg-surface-input text-neutral-400 border border-surface-border'}">
              {active?.isMain ? 'Primary Host' : `Seat ${active?.index || 2}`}
            </span>
          </div>
          <p class="text-xs text-neutral-400 mt-0.5 font-mono">
            {active?.isMain ? 'Uses host environment and default prefix (~/Games/umu)' : 'Isolated sandbox with profile prefix (~/Games/umu)'}
          </p>
        </div>
      </div>

      {#if !active?.isMain}
        <button
          type="button"
          onclick={confirmDeleteProfile}
          class="px-3 py-1.5 text-2xs rounded-xl bg-surface-input hover:bg-red-500/10 hover:text-red-400 text-neutral-400 border border-surface-border hover:border-red-500/30 transition-all flex items-center space-x-1.5 cursor-pointer"
        >
          <Icon name="trash" className="w-3.5 h-3.5" />
          <span>Delete Profile</span>
        </button>
      {/if}
    </div>
  </SettingSection>

  <!-- Steam Persona Emulation Settings -->
  <SettingSection title="Persona & Steamworks Emulation">
    <div class="space-y-3">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Configure player identity, language, and emulated SteamID64 for this profile.
      </p>

      {#if active}
        <SteamPersonaCard
          accountName={active.name}
          accountSteamId={active.steamId}
          language={active.language}
          ipCountry={active.ipCountry}
          avatarUrl={active.avatarUrl || ''}
          {languagesList}
          {countriesList}
          onchange={handlePersonaChange}
        />
      {/if}
    </div>
  </SettingSection>
</div>
