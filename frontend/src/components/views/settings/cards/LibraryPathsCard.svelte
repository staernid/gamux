<script>
  import { createEventDispatcher } from 'svelte';
  import { selectDirectoryDialog } from '../../../../utils/filesystem.js';
  import SettingSection from '../../../common/SettingSection.svelte';

  export let libraryPaths = [];

  const dispatch = createEventDispatcher();
  let pendingRemovePath = null;

  async function handleAddLibraryPath() {
    const selected = await selectDirectoryDialog('Select Library Directory', '');
    if (selected) {
      const current = libraryPaths || [];
      if (!current.includes(selected)) {
        libraryPaths = [...current, selected];
        dispatch('change', libraryPaths);
      }
    }
  }

  function requestRemoveLibraryPath(p) {
    pendingRemovePath = p;
  }

  function cancelRemoveLibraryPath() {
    pendingRemovePath = null;
  }

  function confirmRemoveLibraryPath(p) {
    libraryPaths = (libraryPaths || []).filter(item => item !== p);
    pendingRemovePath = null;
    dispatch('change', libraryPaths);
  }
</script>

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

  {#if (libraryPaths || []).length === 0}
    <div class="p-3 bg-[#101013] border border-surface-border rounded-xl text-center">
      <p class="text-xs text-neutral-400">No library directories configured. Add a directory to start scanning.</p>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
      {#each libraryPaths as p}
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
