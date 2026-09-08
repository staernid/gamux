<script>
  import SettingSection from '../../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../../common/SeamlessSwitch.svelte';
  import StepperRow from '../../../common/StepperRow.svelte';
  import GhostInput from '../../../common/GhostInput.svelte';
  import { selectDirectoryDialog } from '../../../../utils/filesystem.js';

  export let cfgDraft;

  let showAdvancedStorage = false;

  const retentionOptions = [
    { value: 1, label: '1 Snapshot' },
    { value: 3, label: '3 Snapshots' },
    { value: 5, label: '5 Snapshots (Default)' },
    { value: 10, label: '10 Snapshots' },
    { value: 20, label: '20 Snapshots' }
  ];

  async function handleBrowseArtworkDir() {
    const selected = await selectDirectoryDialog('Select Artwork Cache Directory', cfgDraft.artwork_dir || '');
    if (selected) {
      cfgDraft.artwork_dir = selected;
      cfgDraft = { ...cfgDraft };
    }
  }

  async function handleBrowseBackupDir() {
    const selected = await selectDirectoryDialog('Select Save Backup Directory', cfgDraft.backup_dir || '');
    if (selected) {
      cfgDraft.backup_dir = selected;
      cfgDraft = { ...cfgDraft };
    }
  }
</script>

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
