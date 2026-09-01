<script>
  import { createEventDispatcher } from 'svelte';
  import DrawerDLCTab from '../../stage/drawer/DrawerDLCTab.svelte';
  import DrawerIntegrityTab from '../../stage/drawer/DrawerIntegrityTab.svelte';
  import DrawerLogsTab from '../../stage/drawer/DrawerLogsTab.svelte';

  export let game = null;
  export let isGameRunning = false;
  export let isTaskActive = false;
  export let isCheckingIntegrity = false;
  export let isNormalizing = false;
  export let integrityError = '';
  export let integrityReport = null;

  const dispatch = createEventDispatcher();

  let activeFilesSubTab = 'depots'; // 'depots' | 'integrity' | 'logs'

  function handleRunIntegrityCheck() {
    dispatch('runIntegrityCheck');
  }

  function handleUpdateGame() {
    dispatch('updateGame');
  }

  function handleNormalizeDirectory() {
    dispatch('normalizeDirectory');
  }
</script>

<div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl space-y-6">
  <!-- Files Sub-Pills -->
  <div class="flex items-center space-x-2 bg-[#0c0c10] p-1.5 rounded-xl border border-[#20202e] flex-wrap gap-y-1">
    <button
      type="button"
      on:click={() => { activeFilesSubTab = 'depots'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'depots' ? 'bg-[#202030] text-purple-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
      </svg>
      <span>DLC & Depots</span>
    </button>

    <button
      type="button"
      on:click={() => { activeFilesSubTab = 'integrity'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'integrity' ? 'bg-[#202030] text-cyan-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
      <span>Integrity & Repair</span>
    </button>

    <button
      type="button"
      on:click={() => { activeFilesSubTab = 'logs'; }}
      class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeFilesSubTab === 'logs' ? 'bg-[#202030] text-slate-200 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
    >
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <span>Execution Logs</span>
      {#if isGameRunning}
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Game currently executing"></span>
      {/if}
    </button>
  </div>

  <!-- Files Sub-View Content -->
  {#if activeFilesSubTab === 'depots'}
    <DrawerDLCTab {game} />
  {:else if activeFilesSubTab === 'integrity'}
    <DrawerIntegrityTab
      {isCheckingIntegrity}
      {isNormalizing}
      isUpdating={isTaskActive}
      {integrityError}
      {integrityReport}
      onRunIntegrityCheck={handleRunIntegrityCheck}
      onUpdateGame={handleUpdateGame}
      onNormalizeDirectory={handleNormalizeDirectory}
    />
  {:else if activeFilesSubTab === 'logs'}
    <DrawerLogsTab {game} />
  {/if}
</div>
