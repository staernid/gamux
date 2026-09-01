<script>
  import { notificationStore } from '../../../stores/notifications.js';
  import SettingSection from '../../common/SettingSection.svelte';

  let logFilter = 'all'; // 'all' | 'error' | 'warning' | 'info' | 'success'
  const notifItems = notificationStore.items;

  $: filteredLogs = ($notifItems || []).filter(item => {
    if (logFilter === 'all') return true;
    return item.type === logFilter;
  });

  function copyLogsToClipboard() {
    const text = ($notifItems || [])
      .map(i => `[${new Date(i.timestamp).toISOString()}] [${(i.type || 'INFO').toUpperCase()}] ${i.title}${i.message ? ': ' + i.message : ''}${i.details ? ' (' + i.details + ')' : ''}`)
      .join('\n');
    navigator.clipboard?.writeText(text);
  }

  function clearAllLogs() {
    notificationStore.items.clearAll();
  }
</script>

<div class="space-y-3">
  <SettingSection
    title="System Logs & Activity Stream"
    tooltip="Real-time telemetry, notifications, and event activity log stream."
  >
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2.5 pb-2 border-b border-[#1c1c28]">
      <div class="flex items-center space-x-2">
        <span class="text-[10px] font-mono bg-[#0c0c10] text-slate-400 px-2 py-0.5 rounded-full border border-[#20202c]">
          {($notifItems || []).length} entries
        </span>
      </div>

      <div class="flex items-center space-x-2 flex-wrap gap-y-1">
        <!-- Filter Chips -->
        <div class="flex items-center space-x-1 bg-[#0c0c10] p-0.5 rounded-lg border border-[#242434] text-[11px] font-mono">
          {#each ['all', 'error', 'warning', 'info', 'success'] as f}
            <button
              type="button"
              on:click={() => { logFilter = f; }}
              class="px-2 py-0.5 rounded capitalize transition-colors cursor-pointer {logFilter === f ? 'bg-[#222232] text-white font-bold' : 'text-slate-500 hover:text-slate-300'}"
            >
              {f}
            </button>
          {/each}
        </div>

        {#if ($notifItems || []).length > 0}
          <button
            type="button"
            on:click={copyLogsToClipboard}
            class="text-[11px] font-bold bg-[#181824] hover:bg-[#222232] text-slate-300 px-2.5 py-1 rounded-lg border border-[#2c2c3e] transition-colors flex items-center space-x-1.5 cursor-pointer"
            title="Copy log to clipboard"
          >
            <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <span>Copy</span>
          </button>
          <button
            type="button"
            on:click={clearAllLogs}
            class="text-[11px] font-bold bg-[#181824] hover:bg-rose-950/40 text-slate-400 hover:text-rose-300 px-2.5 py-1 rounded-lg border border-[#2c2c3e] hover:border-rose-900/50 transition-colors cursor-pointer"
          >
            Clear
          </button>
        {/if}
      </div>
    </div>

    <!-- Monospace Terminal Log Stream -->
    <div class="bg-[#08080c] border border-[#1e1e2c] rounded-xl p-3 sm:p-3.5 max-h-[440px] overflow-y-auto space-y-1.5 font-mono text-xs custom-scrollbar">
      {#if filteredLogs.length === 0}
        <div class="py-10 text-center text-slate-500 text-xs">
          <span>No log entries recorded.</span>
        </div>
      {:else}
        {#each filteredLogs as item (item.id || item.timestamp)}
          <div class="flex items-start space-x-2 py-1 border-b border-[#14141d] last:border-0 hover:bg-[#0e0e14] px-1.5 rounded transition-colors">
            <span class="text-slate-600 text-[10px] select-none pt-0.5 whitespace-nowrap">
              {new Date(item.timestamp).toLocaleTimeString()}
            </span>

            <!-- Level Badge -->
            <span class="px-1.5 py-0.2 rounded text-[9px] font-black uppercase tracking-wider flex-shrink-0 {item.type === 'error' ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30' : item.type === 'warning' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30' : item.type === 'success' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' : 'bg-sky-500/20 text-sky-300 border border-sky-500/30'}">
              {item.type || 'INFO'}
            </span>

            <!-- Content -->
            <div class="min-w-0 flex-1 space-y-0.5">
              <div class="flex items-center space-x-1.5 flex-wrap">
                <strong class="text-slate-200 font-bold text-xs">{item.title}</strong>
                {#if item.gameName}
                  <span class="text-[10px] text-cyan-400 bg-cyan-950/40 px-1.5 py-0.2 rounded border border-cyan-900/50">
                    {item.gameName}
                  </span>
                {/if}
              </div>
              {#if item.message}
                <p class="text-slate-400 text-[11px] break-words">{item.message}</p>
              {/if}
              {#if item.details}
                <p class="text-slate-500 text-[10px] break-words">{item.details}</p>
              {/if}
            </div>
          </div>
        {/each}
      {/if}
    </div>
  </SettingSection>
</div>
