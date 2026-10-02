<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { notificationStore } from '../../../../stores/notifications.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null } = $props();

  let logContent = $state('');
  let isLoading = $state(false);
  let error = $state('');
  let logContainerRef = $state();
  let copied = $state(false);
  let pollInterval = null;

  let isRunning = $derived(game?.isRunning || false);

  async function fetchLog(scroll = false) {
    if (!game?.gameDir) return;
    try {
      const data = await AppAPI.getGameLog(game.gameDir);
      logContent = data || '';
      error = '';
      if (scroll) {
        await tick();
        scrollToBottom();
      }
    } catch (err) {
      if (!logContent) {
        error = err?.message || String(err);
      }
    }
  }

  function scrollToBottom() {
    if (logContainerRef) {
      logContainerRef.scrollTop = logContainerRef.scrollHeight;
    }
  }

  async function handleRefresh() {
    isLoading = true;
    await fetchLog(true);
    isLoading = false;
  }

  async function handleCopy() {
    if (!logContent) return;
    try {
      await navigator.clipboard.writeText(logContent);
      copied = true;
      notificationStore.info('Log Copied', 'Execution log copied to clipboard');
      setTimeout(() => { copied = false; }, 2000);
    } catch {
      notificationStore.error('Copy Failed', 'Could not copy log text');
    }
  }

  onMount(async () => {
    isLoading = true;
    await fetchLog(true);
    isLoading = false;

    // If game is currently running, poll for live updates every 2 seconds
    pollInterval = setInterval(() => {
      if (isRunning) {
        fetchLog(false);
      }
    }, 2000);
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
  });
</script>

<div class="space-y-3">
  <!-- Header Bar -->
  <div class="flex items-center justify-between flex-wrap gap-2 text-xs">
    <div class="flex items-center space-x-2.5">
      {#if isRunning}
        <span class="flex items-center space-x-1.5 px-2.5 py-1 rounded-lg bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 font-bold font-mono text-2xs animate-pulse">
          <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
          <span>Live Log (Running)</span>
        </span>
      {:else if logContent}
        <span class="flex items-center space-x-1.5 px-2.5 py-1 rounded-lg bg-[#1a1a24] border border-[#2a2a3a] text-slate-300 font-mono text-2xs">
          <span class="w-2 h-2 rounded-full bg-slate-500"></span>
          <span>Last Run Log</span>
        </span>
      {:else}
        <span class="text-slate-400 text-xs">No active log</span>
      {/if}
    </div>

    <!-- Actions: Copy & Refresh -->
    <div class="flex items-center space-x-2">
      <button
        type="button"
        onclick={handleRefresh}
        disabled={isLoading}
        class="bg-[#1a1a26] hover:bg-[#242436] text-slate-300 hover:text-white border border-[#2c2c3e] px-3 py-1.5 rounded-xl font-bold transition disabled:opacity-50 flex items-center space-x-1.5 cursor-pointer"
        title="Reload log from disk"
      >
        <svg class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        <span>{isLoading ? 'Loading...' : 'Refresh'}</span>
      </button>

      <button
        type="button"
        onclick={handleCopy}
        disabled={!logContent}
        class="bg-white hover:bg-neutral-200 text-black font-black px-3.5 py-1.5 rounded-xl transition disabled:opacity-40 flex items-center space-x-1.5 cursor-pointer shadow-sm"
        title="Copy full log to clipboard"
      >
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
        </svg>
        <span>{copied ? 'Copied!' : 'Copy Log'}</span>
      </button>
    </div>
  </div>

  <!-- Monospace Log Container -->
  {#if error && !logContent}
    <div class="p-6 rounded-2xl bg-[#0e0e14] border border-[#202030] text-center space-y-2 font-mono text-xs">
      <p class="text-slate-400 font-sans">{error}</p>
      <p class="text-2xs text-slate-500 font-sans">
        Launch the game to generate real-time stdout and stderr execution traces.
      </p>
    </div>
  {:else if logContent}
    <div
      bind:this={logContainerRef}
      class="w-full bg-[#060609] border border-[#1c1c28] rounded-2xl p-4 font-mono text-2xs leading-relaxed text-slate-300 max-h-96 sm:max-h-[28rem] overflow-y-auto overflow-x-auto whitespace-pre custom-scrollbar select-text shadow-inner"
    >
      {logContent}
    </div>
  {:else}
    <div class="p-8 rounded-2xl bg-[#0e0e14] border border-dashed border-[#242436] text-center space-y-2 text-xs">
      <p class="text-slate-300 font-bold">No Execution Logs Recorded</p>
      <p class="text-slate-500 font-mono text-2xs">
        Outputs from UMU, Wine, Proton, and native executables will appear here automatically when launched.
      </p>
    </div>
  {/if}
</div>
