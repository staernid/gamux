<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { drawerDraft } from '../../stores/drawerDraft.js';
  import { configStore } from '../../stores/config.js';
  import DrawerGeneralTab from './drawer/DrawerGeneralTab.svelte';
  import DrawerIntegrationsTab from './drawer/DrawerIntegrationsTab.svelte';
  import DrawerEmuTab from './drawer/DrawerEmuTab.svelte';
  import DrawerSavesTab from './drawer/DrawerSavesTab.svelte';
  import DrawerLogsTab from './drawer/DrawerLogsTab.svelte';

  export let game = null;
  export let initialTab = 'general';

  const dispatch = createEventDispatcher();

  let activeTab = initialTab || 'general';
  let globalCfg = {};
  let isSaving = false;


  onMount(async () => {
    const cfg = await configStore.load();
    if (cfg) globalCfg = cfg;
    if (game) {
      drawerDraft.init(game, globalCfg);
    }
  });

  $: candidates = (game?.launchCandidates || [])
    .filter(c => {
      const exe = (c.executable || '').toLowerCase();
      return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
    })
    .map(c => ({
      name: c.name || c.executable || '',
      executable: c.executable || '',
      arguments: c.arguments || '',
      description: c.description || '',
    }));

  $: languageOptions = (game?.supportedLanguages || [
    'english', 'french', 'italian', 'german', 'spanish', 'japanese', 'schinese', 'russian', 'brazilian', 'koreana', 'polish', 'turkish'
  ]);

  $: if (game && globalCfg) {
    drawerDraft.init(game, globalCfg);
  }

  async function handleSave() {
    if (!game || isSaving) return;
    isSaving = true;
    try {
      const success = await drawerDraft.save(game);
      if (success) {
        dispatch('updated', { gameDir: game.gameDir });
        dispatch('close');
      }
    } finally {
      isSaving = false;
    }
  }

  function handleDiscard() {
    drawerDraft.discard();
    dispatch('close');
  }

  async function handleResetDefaults() {
    if (!game) return;
    const ok = await drawerDraft.resetDefaults(game);
    if (ok) {
      dispatch('updated', { gameDir: game.gameDir });
      dispatch('close');
    }
  }

  function handleRollback() {
    if (!game) return;
    drawerDraft.rollback(game);
    dispatch('updated', { gameDir: game.gameDir });
    dispatch('close');
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      handleDiscard();
    } else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSave();
    }
  }
</script>

{#if game}
  <!-- Modal Backdrop -->
  <div
    class="fixed inset-0 bg-black/80 backdrop-blur-md z-50 flex items-center justify-center p-4 sm:p-6 animate-fade-in"
    on:click|self={handleDiscard}
    on:keydown={handleKeydown}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <!-- Modal Dialog Window -->
    <div class="bg-[#121218] border border-[#2c2c3e] w-full max-w-2xl rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[88vh] animate-slide-up">
      <!-- Modal Header -->
      <div class="px-5 py-4 border-b border-[#20202c] flex items-center justify-between bg-[#15151e]">
        <div class="flex items-center space-x-3 min-w-0">
          <div class="w-9 h-9 rounded-xl bg-sky-500/10 border border-sky-500/30 flex items-center justify-center text-sky-400 font-bold text-base shadow-inner">
            <svg class="w-5 h-5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </div>
          <div class="min-w-0">
            <h3 class="font-black text-white text-sm sm:text-base tracking-tight truncate">
              Game Properties & Compatibility
            </h3>
            <p class="text-[11px] text-slate-400 font-mono truncate">{game.name}</p>
          </div>
        </div>

        <button
          type="button"
          on:click={handleDiscard}
          class="flex items-center space-x-1.5 px-2 py-1 text-slate-400 hover:text-white rounded-xl hover:bg-slate-800/80 transition-colors"
          title="Discard and Close (Esc / B)"
        >
          <span class="text-[10px] text-slate-500 font-mono font-bold">Esc / B</span>
          <span>✕</span>
        </button>
      </div>

      <!-- Navigation Tabs -->
      <div class="flex items-center space-x-1 px-4 pt-3 border-b border-[#20202c] bg-[#111116] select-none text-xs font-bold overflow-x-auto custom-scrollbar">
        <button
          type="button"
          on:click={() => { activeTab = 'general'; }}
          class="px-3.5 py-2.5 rounded-t-xl transition-all flex items-center space-x-2 flex-shrink-0 {activeTab === 'general' ? 'bg-[#181822] text-sky-400 border-b-2 border-sky-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          <span>General</span>
        </button>

        <button
          type="button"
          on:click={() => { activeTab = 'integrations'; }}
          class="px-3.5 py-2.5 rounded-t-xl transition-all flex items-center space-x-2 flex-shrink-0 {activeTab === 'integrations' ? 'bg-[#181822] text-sky-400 border-b-2 border-sky-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <rect x="2" y="6" width="20" height="12" rx="4" stroke-width="2" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 12h4m-2-2v4m7-1h.01m3 2h.01" />
          </svg>
          <span>Runner</span>
        </button>

        <button
          type="button"
          on:click={() => { activeTab = 'emu'; }}
          class="px-3.5 py-2.5 rounded-t-xl transition-all flex items-center space-x-2 flex-shrink-0 {activeTab === 'emu' ? 'bg-[#181822] text-sky-400 border-b-2 border-sky-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
          <span>Emulator</span>
        </button>

        <button
          type="button"
          on:click={() => { activeTab = 'saves'; }}
          class="px-3.5 py-2.5 rounded-t-xl transition-all flex items-center space-x-2 flex-shrink-0 {activeTab === 'saves' ? 'bg-[#181822] text-emerald-400 border-b-2 border-emerald-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
          </svg>
          <span>Saves</span>
        </button>

        <button
          type="button"
          on:click={() => { activeTab = 'logs'; }}
          class="px-3.5 py-2.5 rounded-t-xl transition-all flex items-center space-x-2 flex-shrink-0 {activeTab === 'logs' ? 'bg-[#181822] text-sky-400 border-b-2 border-sky-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <span>Logs</span>
        </button>
      </div>

      <!-- Modal Body (Scrollable) -->
      <div class="p-4 sm:p-6 overflow-y-auto space-y-4 text-xs focus:outline-none flex-1 custom-scrollbar">
        {#if activeTab === 'general'}
          <DrawerGeneralTab
            {game}
            {candidates}
            {globalCfg}
          />
        {:else if activeTab === 'integrations'}
          <DrawerIntegrationsTab
            {game}
            {globalCfg}
          />
        {:else if activeTab === 'emu'}
          <DrawerEmuTab
            {game}
            {globalCfg}
            {languageOptions}
          />
        {:else if activeTab === 'saves'}
          <DrawerSavesTab
            {game}
          />
        {:else if activeTab === 'logs'}
          <DrawerLogsTab
            {game}
          />
        {/if}
      </div>


      <!-- Modal Footer with Explicit Save & Discard Buttons and Gamepad Hints -->
      <div class="px-5 py-3.5 bg-[#0f0f15] border-t border-[#20202c] flex items-center justify-between gap-3 select-none flex-shrink-0">
        <!-- Left Action: Reset / Rollback / Bumper Hint -->
        <div class="flex items-center space-x-2.5">
          <div class="hidden sm:flex items-center space-x-1.5 text-[10px] font-mono text-slate-500 mr-1">
            <span class="px-1.5 py-0.5 rounded bg-[#161622] border border-[#262638] text-slate-400 font-bold">LB / RB</span>
            <span>Tabs</span>
          </div>
          <button
            type="button"
            on:click={handleResetDefaults}
            class="text-[11px] font-bold text-slate-400 hover:text-slate-200 bg-[#171722] hover:bg-[#202030] border border-[#2b2b3c] px-3 py-1.5 rounded-xl transition-colors"
            title="Reset profile to auto-detected defaults"
          >
            Reset Defaults
          </button>
          {#if game.isPatched}
            <button
              type="button"
              on:click={handleRollback}
              class="text-[11px] font-bold text-rose-400 hover:text-rose-300 bg-rose-950/30 hover:bg-rose-900/50 border border-rose-800/40 px-3 py-1.5 rounded-xl transition-colors"
              title="Revert Goldberg Emulator and Steamless patches"
            >
              Rollback Patches
            </button>
          {/if}
        </div>

        <!-- Right Action: Discard / Save -->
        <div class="flex items-center space-x-2">
          <button
            type="button"
            on:click={handleDiscard}
            class="px-3.5 py-2 rounded-xl bg-transparent hover:bg-slate-800 text-slate-400 hover:text-white font-bold text-xs transition-colors flex items-center space-x-1.5"
          >
            <span class="text-[10px] text-slate-500 font-mono font-bold">B</span>
            <span>Cancel</span>
          </button>
          <button
            type="button"
            on:click={handleSave}
            disabled={isSaving}
            class="bg-gradient-to-r from-sky-500 to-cyan-400 hover:from-sky-400 hover:to-cyan-300 active:scale-95 text-slate-950 font-black px-4.5 py-2 rounded-xl transition text-xs shadow-lg shadow-sky-500/20 disabled:opacity-50 flex items-center space-x-1.5"
          >
            <span class="text-[10px] text-slate-950/70 font-mono font-black">X</span>
            <span>{isSaving ? 'Saving...' : 'Save Changes'}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
