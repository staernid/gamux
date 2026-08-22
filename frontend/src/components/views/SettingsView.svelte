<script>
  import { onMount } from 'svelte';
  import { activeView } from '../../stores/library.js';
  import { configStore } from '../../stores/config.js';
  import { AppAPI } from '../../api/app.js';

  let localCfg = {};
  let currentScale = 16;
  let activeTab = 'general'; // 'general' | 'paths' | 'tools'

  let tools = [];
  let isLoadingTools = false;

  onMount(async () => {
    const cfg = await configStore.load();
    if (cfg) localCfg = { ...cfg };
    configStore.uiScale.subscribe(s => { currentScale = s; })();
    loadTools();
  });

  async function loadTools() {
    isLoadingTools = true;
    try {
      tools = await AppAPI.getToolsStatus();
    } catch {
      tools = [];
    } finally {
      isLoadingTools = false;
    }
  }

  async function handleUpdateTool(toolKey) {
    try {
      await AppAPI.updateTool(toolKey);
      await loadTools();
    } catch (err) {
      alert(`Update failed: ${err}`);
    }
  }

  async function browseFolder(fieldKey, title) {
    const defaultVal = localCfg[fieldKey] || '';
    const selected = await AppAPI.selectDirectory(title, defaultVal);
    if (selected) {
      localCfg[fieldKey] = selected;
    }
  }

  async function handleSave() {
    configStore.setScale(currentScale);
    const ok = await configStore.save(localCfg);
    if (ok) {
      activeView.set('library');
    }
  }

  async function handleReset() {
    if (confirm('Reset all configuration settings to default values?')) {
      localCfg = {
        gbe_dir: '~/.local/share/gbe_fork',
        steamless_dir: '~/.local/share/steamless',
        lutris_dir: '~/.config/lutris/games',
        steam_userdata: '~/.local/share/Steam/userdata',
        steam_store_api: 'https://store.steampowered.com/api',
        steam_web_api: 'https://api.steampowered.com',
        github_api_url: 'https://api.github.com/repos/Detanup01/gbe_fork/releases/latest',
        steamless_github_api: 'https://api.github.com/repos/staernid/steamless-rs/releases/latest',
        gbe_mode: 'loader',
        platform: 'win64',
        runner: 'wine',
        lutris: true,
        steamless: true,
        achievements: true,
        normalize: true,
        enable_launch_notify: true,
      };
      currentScale = 16;
      configStore.setScale(16);
    }
  }
</script>

<div class="flex-1 overflow-y-auto p-6 sm:p-10 max-w-3xl mx-auto w-full space-y-6 text-xs sm:text-sm animate-fade-in">
  <!-- Header -->
  <div class="border-b border-[#20202c] pb-4 flex items-center justify-between">
    <div>
      <h2 class="text-xl font-black text-white">System Settings</h2>
      <p class="text-xs text-slate-400 mt-0.5 font-mono">~/.config/gamux/config.json</p>
    </div>

    <button
      type="button"
      on:click={() => { $activeView = 'library'; }}
      class="text-xs font-bold text-slate-400 hover:text-white px-3 py-1.5 rounded-lg bg-[#1a1a24] border border-[#2a2a38]"
    >
      ✕ Shelf
    </button>
  </div>

  <!-- Settings Panel -->
  <div class="bg-[#121218] border border-[#242434] rounded-2xl overflow-hidden shadow-xl">
    <!-- Sub-tabs -->
    <div class="border-b border-[#20202c] bg-[#0f0f15] px-6 flex space-x-6 text-xs font-bold">
      <button
        type="button"
        on:click={() => { activeTab = 'general'; }}
        class="py-3 transition-colors {activeTab === 'general' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        Defaults & UI
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'paths'; }}
        class="py-3 transition-colors {activeTab === 'paths' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        System Paths
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'tools'; }}
        class="py-3 transition-colors {activeTab === 'tools' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        Tool Releases (GBE & Steamless)
      </button>
    </div>

    <!-- Body -->
    <div class="p-6 space-y-4">
      {#if activeTab === 'general'}
        <div class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <div>
              <label for="setGbeMode" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Default Goldberg Mode
              </label>
              <select id="setGbeMode" bind:value={localCfg.gbe_mode} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value="loader">Loader Mode (ColdClientLoader)</option>
                <option value="portable">Portable Mode (Direct DLL)</option>
              </select>
            </div>
            <div>
              <label for="setUiScale" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Display Scale
              </label>
              <select id="setUiScale" bind:value={currentScale} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value={14}>Compact (14px)</option>
                <option value={16}>Standard (16px)</option>
                <option value={18}>Large (18px)</option>
                <option value={20}>Extra Large / Handheld (20px)</option>
              </select>
            </div>
          </div>

          <!-- Toggles -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-3 border-t border-[#20202c]">
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.lutris} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Enable Lutris registration by default</span>
            </label>
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.steamless} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Unpack SteamStub DRM automatically</span>
            </label>
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.achievements} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Fetch achievements & DLC metadata</span>
            </label>
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.normalize} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Normalize folder names</span>
            </label>
          </div>
        </div>

      {:else if activeTab === 'paths'}
        <div class="space-y-3">
          <div>
            <label for="setGbeDir" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
              Goldberg Emulator Path
            </label>
            <div class="flex space-x-2">
              <input id="setGbeDir" type="text" bind:value={localCfg.gbe_dir} class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
              <button type="button" on:click={() => browseFolder('gbe_dir', 'Select Goldberg Directory')} class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244]">Browse</button>
            </div>
          </div>
          <div>
            <label for="setSteamlessDir" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
              Steamless Binary Path
            </label>
            <div class="flex space-x-2">
              <input id="setSteamlessDir" type="text" bind:value={localCfg.steamless_dir} class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
              <button type="button" on:click={() => browseFolder('steamless_dir', 'Select Steamless Directory')} class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244]">Browse</button>
            </div>
          </div>
          <div>
            <label for="setLutrisDir" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
              Lutris Configs Path
            </label>
            <div class="flex space-x-2">
              <input id="setLutrisDir" type="text" bind:value={localCfg.lutris_dir} class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
              <button type="button" on:click={() => browseFolder('lutris_dir', 'Select Lutris Directory')} class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244]">Browse</button>
            </div>
          </div>
        </div>

      {:else if activeTab === 'tools'}
        <div class="space-y-3">
          {#if isLoadingTools}
            <div class="py-8 text-center text-slate-400 font-mono">Checking GitHub release versions...</div>
          {:else if tools.length > 0}
            {#each tools as tool}
              <div class="bg-[#0c0c10] border border-[#222230] p-3.5 rounded-xl flex items-center justify-between">
                <div>
                  <strong class="text-white font-bold block">{tool.name}</strong>
                  <span class="text-slate-400 font-mono text-[11px]">
                    Installed: <span class="text-slate-200">{tool.installed_tag || 'N/A'}</span> · Latest: <span class="text-slate-200">{tool.latest_tag || 'N/A'}</span>
                  </span>
                </div>
                <button
                  type="button"
                  on:click={() => handleUpdateTool(tool.key)}
                  class="bg-[#20202c] hover:bg-[#2a2a3a] text-white font-bold text-xs px-3.5 py-1.5 rounded-lg border border-[#323244]"
                >
                  {tool.has_update ? 'Update' : 'Reinstall'}
                </button>
              </div>
            {/each}
          {:else}
            <div class="py-6 text-center text-slate-500 font-mono">No tools reported.</div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between p-4 border-t border-[#20202c] bg-[#0f0f15]">
      <button type="button" on:click={handleReset} class="text-slate-400 hover:text-slate-200 text-xs font-bold px-2 py-1">
        Reset Defaults
      </button>
      <div class="flex space-x-2">
        <button type="button" on:click={() => { $activeView = 'library'; }} class="text-slate-400 hover:text-white text-xs font-bold px-3 py-1.5">
          Cancel
        </button>
        <button
          type="button"
          on:click={handleSave}
          class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-xs px-5 py-2 rounded-xl transition-colors shadow-sm"
        >
          Save Settings
        </button>
      </div>
    </div>
  </div>
</div>
