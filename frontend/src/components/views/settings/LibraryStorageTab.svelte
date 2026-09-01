<script>
  import { AppAPI } from '../../../api/app.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import StepperRow from '../../common/StepperRow.svelte';
  import GhostInput from '../../common/GhostInput.svelte';

  export let cfgDraft;

  let showAdvancedStorage = false;

  async function handleAddLibraryPath() {
    try {
      const selected = await AppAPI.selectDirectory('Select Library Directory', '');
      if (selected) {
        const current = cfgDraft.library_paths || [];
        if (!current.includes(selected)) {
          cfgDraft.library_paths = [...current, selected];
        }
      }
    } catch {
      // Handled
    }
  }

  function handleRemoveLibraryPath(pathToRemove) {
    cfgDraft.library_paths = (cfgDraft.library_paths || []).filter(p => p !== pathToRemove);
  }
</script>

<div class="space-y-3">
  <!-- Library Folders -->
  <SettingSection title="Library Folders">
    <svelte:fragment slot="action">
      <button
        type="button"
        on:click={handleAddLibraryPath}
        class="px-2.5 py-1 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-bold transition-colors flex items-center space-x-1 cursor-pointer"
      >
        <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>Add Folder</span>
      </button>
    </svelte:fragment>

    {#if (cfgDraft.library_paths || []).length === 0}
      <div class="p-3 bg-[#0d0d14] border border-[#1e1e2c] rounded-xl text-center">
        <p class="text-xs text-slate-400">No library directories configured. Add a directory to start scanning.</p>
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
        {#each cfgDraft.library_paths as p}
          <div class="flex items-center justify-between p-2.5 bg-[#0d0d14] border border-[#1e1e2c] rounded-lg">
            <div class="flex items-center space-x-2 min-w-0">
              <svg class="w-3.5 h-3.5 text-emerald-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
              <span class="font-mono text-xs text-slate-200 truncate">{p}</span>
            </div>
            <button
              type="button"
              on:click={() => handleRemoveLibraryPath(p)}
              class="p-1 text-slate-400 hover:text-rose-400 transition-colors cursor-pointer flex-shrink-0 ml-2"
              title="Remove library directory"
              aria-label={`Remove library path ${p}`}
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </SettingSection>

  <!-- Automation & Save Backups -->
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
        <div class="pt-1">
          <StepperRow
            label="Backup Retention Count"
            min={1}
            max={30}
            bind:value={cfgDraft.backup_retention_count}
          />
        </div>
      {/if}
    </div>

    <!-- Advanced Storage Accordion -->
    <div class="pt-3 mt-3 border-t border-[#1a1a26]">
      <button
        type="button"
        on:click={() => { showAdvancedStorage = !showAdvancedStorage; }}
        class="text-xs font-semibold text-slate-400 hover:text-slate-200 flex items-center space-x-1.5 cursor-pointer"
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
            bind:value={cfgDraft.artwork_dir}
            placeholder="~/.local/share/gamux/art"
            mono={true}
          />

          <GhostInput
            label="Backup Directory"
            bind:value={cfgDraft.backup_dir}
            placeholder="~/.local/share/gamux/backups"
            mono={true}
          />
        </div>
      {/if}
    </div>
  </SettingSection>
</div>
