<script>
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
  <!-- 1. Active Profile Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="flex items-center justify-between">
      <div class="space-y-0.5">
        <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
          Active Profile
        </h3>
        <p class="text-xs text-neutral-400 leading-relaxed">
          Active seat identity and sandbox isolation level.
        </p>
      </div>

      {#if !active?.isMain}
        <button
          type="button"
          onclick={confirmDeleteProfile}
          class="px-3 py-1.5 text-2xs rounded-xl bg-surface-input hover:bg-red-500/10 hover:text-red-400 text-neutral-400 border border-surface-border hover:border-red-500/30 transition-all flex items-center space-x-1.5 cursor-pointer shrink-0"
        >
          <Icon name="trash" className="w-3.5 h-3.5" />
          <span>Delete Profile</span>
        </button>
      {/if}
    </div>

    <div class="flex items-center space-x-3.5 min-w-0 pt-1">
      <div class="w-12 h-12 rounded-full overflow-hidden shrink-0 bg-surface-input border border-surface-borderHighlight flex items-center justify-center text-sm font-bold uppercase text-white shadow-md">
        {#if active?.avatarUrl}
          <img src={active.avatarUrl} alt={active.name} class="w-full h-full object-cover" />
        {:else}
          <span>{(active?.name || 'GP').slice(0, 2).toUpperCase()}</span>
        {/if}
      </div>
      <div class="min-w-0">
        <div class="flex items-center space-x-2">
          <h4 class="text-sm font-bold text-white truncate">{active?.name || 'GamuxPlayer'}</h4>
          <span class="text-4xs px-2 py-0.5 rounded font-mono font-semibold uppercase tracking-wider {active?.isMain ? 'bg-neutral-800 text-neutral-300 border border-neutral-700' : 'bg-surface-input text-neutral-400 border border-surface-border'}">
            {active?.isMain ? 'Primary Host' : `Seat ${active?.index || 2}`}
          </span>
        </div>
        <p class="text-xs text-neutral-400 mt-0.5 font-mono">
          {active?.isMain ? 'Uses host environment and default prefix (~/Games/umu)' : 'Isolated sandbox with profile prefix (~/Games/umu)'}
        </p>
      </div>
    </div>
  </div>

  <!-- 2. Persona & Steamworks Emulation Card -->
  <div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
    <div class="space-y-0.5">
      <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
        Persona &amp; Steamworks Emulation
      </h3>
      <p class="text-xs text-neutral-400 leading-relaxed">
        Configure player identity, language, and emulated SteamID64 for this profile.
      </p>
    </div>

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
</div>
