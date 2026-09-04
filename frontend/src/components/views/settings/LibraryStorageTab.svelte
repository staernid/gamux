<script>
  import { AppAPI } from '../../../api/app.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import StepperRow from '../../common/StepperRow.svelte';
  import GhostInput from '../../common/GhostInput.svelte';

  export let cfgDraft;

  let showAdvancedStorage = false;
  let pendingRemovePath = null;

  const retentionOptions = [
    { value: 1, label: '1 backup' },
    { value: 2, label: '2 backups' },
    { value: 3, label: '3 backups' },
    { value: 5, label: '5 backups (Recommended)' },
    { value: 10, label: '10 backups' },
    { value: 15, label: '15 backups' },
    { value: 20, label: '20 backups' },
    { value: 30, label: '30 backups' }
  ];

  async function handleAddLibraryPath() {
    try {
      const selected = await AppAPI.selectDirectory('Select Library Directory', '');
      if (selected) {
        const current = cfgDraft.library_paths || [];
        if (!current.includes(selected)) {
          cfgDraft.library_paths = [...current, selected];
          cfgDraft = { ...cfgDraft };
        }
      }
    } catch {
      // Handled
    }
  }

  function requestRemoveLibraryPath(p) {
    pendingRemovePath = p;
  }

  function cancelRemoveLibraryPath() {
    pendingRemovePath = null;
  }

  function confirmRemoveLibraryPath(p) {
    cfgDraft.library_paths = (cfgDraft.library_paths || []).filter(item => item !== p);
    cfgDraft = { ...cfgDraft };
    pendingRemovePath = null;
  }

  async function handleBrowseArtworkDir() {
    try {
      const selected = await AppAPI.selectDirectory('Select Artwork Directory', cfgDraft.artwork_dir || '');
      if (selected) {
        cfgDraft.artwork_dir = selected;
        cfgDraft = { ...cfgDraft };
      }
    } catch {
      // Handled
    }
  }

  async function handleBrowseBackupDir() {
    try {
      const selected = await AppAPI.selectDirectory('Select Backup Directory', cfgDraft.backup_dir || '');
      if (selected) {
        cfgDraft.backup_dir = selected;
        cfgDraft = { ...cfgDraft };
      }
    } catch {
      // Handled
    }
  }
</script>

<div class="space-y-3">
  <!-- Library Folders -->
  <SettingSection title="Library Directories">
    <svelte:fragment slot="action">
      <button
        type="button"
        on:click={handleAddLibraryPath}
        class="px-2.5 py-1 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-bold transition-colors flex items-center space-x-1 cursor-pointer"
      >
        <svg class="w-3.5 h-3.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>Add Folder</span>
      </button>
    </svelte:fragment>

    {#if (cfgDraft.library_paths || []).length === 0}
      <div class="p-3 bg-[#101013] border border-surface-border rounded-xl text-center">
        <p class="text-xs text-neutral-400">No library directories configured. Add a directory to start scanning.</p>
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
        {#each cfgDraft.library_paths as p}
          <div class="flex items-center justify-between p-2.5 bg-[#101013] border {pendingRemovePath === p ? 'border-rose-500/50 bg-rose-950/10' : 'border-surface-border'} rounded-lg transition-colors">
            <div class="flex items-center space-x-2 min-w-0 flex-1">
              <svg class="w-3.5 h-3.5 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
              <span class="font-mono text-xs text-neutral-200 truncate">{p}</span>
            </div>

            {#if pendingRemovePath === p}
              <!-- Two-Step Inline Confirmation State -->
              <div class="flex items-center space-x-1.5 flex-shrink-0 ml-2 animate-fade-in">
                <span class="text-2xs font-bold text-rose-400 mr-1">Remove?</span>
                <button
                  type="button"
                  on:click={() => confirmRemoveLibraryPath(p)}
                  class="px-2 py-0.5 rounded bg-rose-600 hover:bg-rose-500 text-white text-2xs font-black transition-colors cursor-pointer shadow-sm"
                  title="Confirm removal"
                >
                  Yes
                </button>
                <button
                  type="button"
                  on:click={cancelRemoveLibraryPath}
                  class="px-2 py-0.5 rounded bg-surface-card hover:bg-surface-cardHover text-neutral-300 text-2xs font-bold transition-colors cursor-pointer border border-surface-border"
                  title="Cancel removal"
                >
                  Cancel
                </button>
              </div>
            {:else}
              <button
                type="button"
                on:click={() => requestRemoveLibraryPath(p)}
                class="p-1 text-neutral-400 hover:text-rose-400 transition-colors cursor-pointer flex-shrink-0 ml-2"
                title="Remove library directory"
                aria-label={`Remove library path ${p}`}
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </SettingSection>

  <!-- Automation & Backups -->
  <SettingSection title="Automation & Backups">
    <div class="space-y-2.5">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
        <SeamlessSwitch
          label="Normalize Folders"
          description="Standardize folder names to official Steam game titles"
          bind:value={cfgDraft.normalize}
        />

        <SeamlessSwitch
          label="Auto-Backup Saves"
          description="Create a rolling save backup before launching any game"
          bind:value={cfgDraft.auto_backup_saves}
        />
      </div>

      {#if cfgDraft.auto_backup_saves}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
          <StepperRow
            label="Backup Retention Count"
            description="Number of rotating save snapshots to retain per game"
            options={retentionOptions}
            bind:value={cfgDraft.backup_retention_count}
          />
        </div>
      {/if}
    </div>

    <!-- Advanced Storage Accordion -->
    <div class="pt-3 mt-3 border-t border-surface-border">
      <button
        type="button"
        on:click={() => { showAdvancedStorage = !showAdvancedStorage; }}
        class="text-xs font-semibold text-neutral-400 hover:text-neutral-200 flex items-center space-x-1.5 cursor-pointer"
      >
        <svg class="w-3.5 h-3.5 transform transition-transform {showAdvancedStorage ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
        <span>Advanced Storage Locations</span>
      </button>

      {#if showAdvancedStorage}
        <div class="pt-3 grid grid-cols-1 md:grid-cols-2 gap-3 animate-fade-in">
          <GhostInput
            label="Artwork Directory"
            description="Local cache directory for covers, heroes, and logos"
            bind:value={cfgDraft.artwork_dir}
            placeholder="~/.local/share/gamux/art"
            mono={true}
          >
            <button
              slot="action"
              type="button"
              on:click={handleBrowseArtworkDir}
              class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
            >
              Browse
            </button>
          </GhostInput>

          <GhostInput
            label="Backup Directory"
            description="Local directory for automated game save snapshots"
            bind:value={cfgDraft.backup_dir}
            placeholder="~/.local/share/gamux/backups"
            mono={true}
          >
            <button
              slot="action"
              type="button"
              on:click={handleBrowseBackupDir}
              class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
            >
              Browse
            </button>
          </GhostInput>
        </div>
      {/if}
    </div>
  </SettingSection>
</div>
