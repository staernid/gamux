<script>
  export let activeTab = 'general';
  export let showNewsTab = false;
  export let onSelectTab = (tab) => {};
</script>

<div class="px-5 border-b border-[#20202c] bg-[#0f0f15] flex space-x-6 text-xs font-bold">
  <button
    type="button"
    on:click={() => onSelectTab('general')}
    class="py-3 transition-colors {activeTab === 'general' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
  >
    General
  </button>
  <button
    type="button"
    on:click={() => onSelectTab('integrations')}
    class="py-3 transition-colors {activeTab === 'integrations' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
  >
    Integrations
  </button>
  <button
    type="button"
    on:click={() => onSelectTab('emu')}
    class="py-3 transition-colors {activeTab === 'emu' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
  >
    Emu
  </button>
  <button
    type="button"
    on:click={() => onSelectTab('integrity')}
    class="py-3 transition-colors {activeTab === 'integrity' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
  >
    Integrity
  </button>
  {#if showNewsTab}
    <button
      type="button"
      on:click={() => onSelectTab('news')}
      class="py-3 transition-colors {activeTab === 'news' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
    >
      Patch Notes
    </button>
  {/if}
</div>
