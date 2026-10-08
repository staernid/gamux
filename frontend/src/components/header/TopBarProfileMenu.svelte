<script>
  import { profilesStore } from '../../stores/profiles.js';
  import { activeView } from '../../stores/library.js';
  import Icon from '../common/Icon.svelte';

  let { onNavigateToProfilesSettings = null } = $props();

  let isOpen = $state(false);
  let isAdding = $state(false);
  let newProfileName = $state('');
  let containerRef = $state();

  let active = $derived(profilesStore.activeProfile);
  let profileList = $derived(profilesStore.profiles);

  function getInitials(name) {
    if (!name) return 'GP';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }

  function toggleMenu() {
    isOpen = !isOpen;
    if (!isOpen) {
      isAdding = false;
      newProfileName = '';
    }
  }

  function handleSelectProfile(p) {
    if (p.id !== active?.id) {
      profilesStore.setActive(p.id);
    }
    isOpen = false;
  }

  async function handleCreateProfile() {
    const trimmed = newProfileName.trim();
    if (!trimmed) return;
    await profilesStore.create(trimmed);
    newProfileName = '';
    isAdding = false;
    isOpen = false;
  }

  function handleOpenSettings() {
    isOpen = false;
    if (typeof onNavigateToProfilesSettings === 'function') {
      onNavigateToProfilesSettings();
    } else {
      $activeView = 'settings';
    }
  }

  function handleClickOutside(e) {
    if (isOpen && containerRef && !containerRef.contains(e.target)) {
      isOpen = false;
      isAdding = false;
      newProfileName = '';
    }
  }

  $effect(() => {
    document.addEventListener('click', handleClickOutside);
    return () => {
      document.removeEventListener('click', handleClickOutside);
    };
  });
</script>

<div class="relative pointer-events-auto" bind:this={containerRef}>
  <!-- Avatar Trigger Button -->
  <button
    type="button"
    onclick={toggleMenu}
    class="w-9 h-9 rounded-full border border-surface-border hover:border-surface-borderHighlight overflow-hidden flex items-center justify-center bg-surface-header hover:bg-surface-cardHover shadow-lg transition-all cursor-pointer focus:outline-none focus:ring-1 focus:ring-neutral-400 group"
    title={active?.name ? `Active Profile: ${active.name}` : 'Player Profiles'}
    aria-label="Player Profiles"
    aria-expanded={isOpen}
  >
    {#if active?.avatar_url}
      <img src={active.avatar_url} alt={active.name} class="w-full h-full object-cover" />
    {:else}
      <div class="w-full h-full flex items-center justify-center bg-surface-input text-neutral-300 font-bold text-2xs uppercase tracking-wider group-hover:text-white transition-colors">
        {getInitials(active?.name)}
      </div>
    {/if}
  </button>

  <!-- Flyout Menu -->
  {#if isOpen}
    <div
      class="absolute top-full right-0 mt-2 w-64 bg-surface-header border border-surface-border rounded-2xl shadow-2xl p-2 z-50 animate-fade-in flex flex-col space-y-1"
      role="menu"
      tabindex="-1"
    >
      <!-- Header / Current Active -->
      <div class="px-2.5 py-1.5 border-b border-surface-border/60 mb-1">
        <span class="text-3xs uppercase tracking-wider font-semibold text-neutral-400">Active Profile</span>
        <div class="flex items-center justify-between mt-0.5">
          <span class="text-xs font-bold text-white truncate">{active?.name || 'GamuxPlayer'}</span>
          <span class="text-4xs px-1.5 py-0.5 rounded font-mono bg-surface-input text-neutral-400 border border-surface-border">
            {active?.is_main ? 'Host' : `Seat ${active?.index || 2}`}
          </span>
        </div>
      </div>

      <!-- Profile List -->
      <div class="max-h-48 overflow-y-auto space-y-0.5 pr-0.5">
        {#each profileList as p (p.id)}
          <button
            type="button"
            onclick={() => handleSelectProfile(p)}
            class="w-full flex items-center justify-between px-2.5 py-1.5 rounded-xl transition-all text-left cursor-pointer {p.id === active?.id ? 'bg-surface-cardHover text-white border border-surface-borderHighlight' : 'text-neutral-300 hover:text-white hover:bg-surface-input'}"
          >
            <div class="flex items-center space-x-2 min-w-0">
              <div class="w-5 h-5 rounded-full overflow-hidden flex-shrink-0 bg-surface-input border border-surface-border flex items-center justify-center text-4xs font-bold uppercase text-neutral-300">
                {#if p.avatar_url}
                  <img src={p.avatar_url} alt={p.name} class="w-full h-full object-cover" />
                {:else}
                  {getInitials(p.name)}
                {/if}
              </div>
              <div class="min-w-0 flex flex-col">
                <span class="text-xs truncate font-medium">{p.name}</span>
                <span class="text-4xs text-neutral-400 font-mono truncate">
                  {p.is_main ? '~/Games/umu (Host)' : `~/Games/umu (Seat ${p.index || 2})`}
                </span>
              </div>
            </div>

            {#if p.id === active?.id}
              <Icon name="check" className="w-3.5 h-3.5 text-white flex-shrink-0 ml-1.5" />
            {/if}
          </button>
        {/each}
      </div>

      <!-- Add Profile Section -->
      <div class="border-t border-surface-border/60 pt-1 mt-1">
        {#if isAdding}
          <div class="p-1 space-y-1.5">
            <input
              type="text"
              bind:value={newProfileName}
              placeholder="Profile Name"
              class="w-full px-2 py-1 text-xs rounded-lg bg-surface-input border border-surface-border text-white placeholder-neutral-500 focus:outline-none focus:border-surface-borderHighlight font-medium"
              onkeydown={(e) => { if (e.key === 'Enter') handleCreateProfile(); else if (e.key === 'Escape') isAdding = false; }}
            />
            <div class="flex items-center justify-end space-x-1.5">
              <button
                type="button"
                onclick={() => { isAdding = false; newProfileName = ''; }}
                class="px-2 py-0.5 text-2xs text-neutral-400 hover:text-white transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="button"
                onclick={handleCreateProfile}
                disabled={!newProfileName.trim()}
                class="px-2.5 py-0.5 text-2xs rounded bg-white text-neutral-950 font-bold hover:bg-neutral-200 disabled:opacity-40 transition-all cursor-pointer"
              >
                Add
              </button>
            </div>
          </div>
        {:else}
          <button
            type="button"
            onclick={() => { isAdding = true; }}
            class="w-full flex items-center space-x-2 px-2.5 py-1.5 rounded-xl text-neutral-300 hover:text-white hover:bg-surface-input transition-all cursor-pointer text-xs"
          >
            <Icon name="plus" className="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" />
            <span>Add New Profile</span>
          </button>
        {/if}
      </div>

      <!-- Settings Link -->
      <div class="border-t border-surface-border/60 pt-1">
        <button
          type="button"
          onclick={handleOpenSettings}
          class="w-full flex items-center space-x-2 px-2.5 py-1.5 rounded-xl text-neutral-400 hover:text-white hover:bg-surface-input transition-all cursor-pointer text-2xs"
        >
          <Icon name="settings" className="w-3.5 h-3.5 flex-shrink-0" />
          <span>Profile Settings</span>
        </button>
      </div>
    </div>
  {/if}
</div>
