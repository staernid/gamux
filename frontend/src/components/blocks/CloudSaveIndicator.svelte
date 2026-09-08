<script>
  import { createEventDispatcher } from 'svelte';
  import { formatTimeAgo } from '../../utils/format.js';
  import Icon from '../common/Icon.svelte';

  export let backupCount = 0;
  export let lastBackupTime = 0; // unix timestamp or 0
  export let isSynced = true;
  export let className = '';
  export let onrestore = null;

  const dispatch = createEventDispatcher();

  $: formattedTime = lastBackupTime > 0 ? formatTimeAgo(lastBackupTime) : '';

  function handleRestore() {
    dispatch('restore');
    if (typeof onrestore === 'function') onrestore();
  }
</script>

<button
  type="button"
  on:click={handleRestore}
  class="inline-flex items-center space-x-2 px-3 py-1.5 rounded-xl bg-[#121214]/90 hover:bg-[#1c1c20] border border-[#27272a] hover:border-neutral-500 text-3xs font-mono transition-all shadow-sm cursor-pointer select-none {className}"
  data-testid="cloud-save-indicator"
  title="Savegame Backups & Snapshots"
>
  <Icon name="cloud" className="w-3.5 h-3.5 {isSynced ? 'text-neutral-400' : 'text-amber-400'}" />

  {#if backupCount > 0}
    <span class="text-neutral-300 font-bold">{backupCount} {backupCount === 1 ? 'Backup' : 'Backups'}</span>
    {#if formattedTime}
      <span class="text-neutral-500">·</span>
      <span class="text-neutral-400">{formattedTime}</span>
    {/if}
  {:else}
    <span class="text-neutral-400">No Backups</span>
  {/if}
</button>
