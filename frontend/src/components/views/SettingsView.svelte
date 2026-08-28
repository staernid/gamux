<script>
  import { onMount } from 'svelte';
  import { activeView, libraryPaths, addLibraryPath, removeLibraryPath, scanLibrary, games, batchApplyCleanGames } from '../../stores/library.js';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import { isSfxEnabled, setSfxEnabled } from '../../utils/audio.js';

  let localCfg = {};
  let currentScale = 18;
  let sfxOn = isSfxEnabled();
  let activeTab = 'system'; // 'system' | 'emu' | 'paths' | 'integrations' | 'tools' | 'logs'

  let logFilter = 'all'; // 'all' | 'error' | 'warning' | 'info' | 'success'
  const notifItems = notificationStore.items;

  $: {
    setSfxEnabled(sfxOn);
  }

  $: filteredLogs = ($notifItems || []).filter(item => {
    if (logFilter === 'all') return true;
    return item.type === logFilter;
  });

  function copyLogsToClipboard() {
    if (!$notifItems || $notifItems.length === 0) return;
    const text = $notifItems.map(item => {
      const time = new Date(item.timestamp).toISOString();
      const level = (item.type || 'info').toUpperCase().padEnd(7);
      const title = item.title ? `[${item.title}] ` : '';
      const game = item.gameName ? `(${item.gameName}) ` : '';
      const details = item.details ? ` - ${item.details}` : '';
      return `${time} ${level} ${title}${game}${item.message || ''}${details}`;
    }).join('\n');

    navigator.clipboard?.writeText(text);
    notificationStore.items.add({
      type: 'success',
      title: 'Logs Copied',
      message: `Copied ${$notifItems.length} log lines to clipboard.`
    });
  }

  let gbeStatus = null;
  let toolStatuses = [];
  let isCheckingTools = false;
  let updatingTools = {};
  let expandedTools = {};
  let isGbeExpanded = false;
  let globalEnvList = [];
  let gseUserCfg = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  onMount(async () => {
    const [cfg, gseCfg] = await Promise.all([
      configStore.load(),
      AppAPI.getGSEUserConfig().catch(() => null)
    ]);
    if (cfg) {
      localCfg = { ...cfg };
      globalEnvList = Object.entries(cfg.global_env || {}).map(([k, v]) => ({ key: k, value: v }));
    }
    if (gseCfg) {
      gseUserCfg = { ...gseCfg };
    }
    configStore.uiScale.subscribe(s => { currentScale = s; })();
    
    const urlParams = new URLSearchParams(window.location.search);
    const tabParam = urlParams.get('tab');
    if (tabParam && ['system', 'emu', 'paths', 'integrations', 'tools', 'logs'].includes(tabParam)) {
      activeTab = tabParam;
    }

    loadToolsStatus();
  });

  function addGlobalEnv() {
    globalEnvList = [...globalEnvList, { key: '', value: '' }];
  }

  function removeGlobalEnv(idx) {
    globalEnvList = globalEnvList.filter((_, i) => i !== idx);
  }

  async function loadToolsStatus() {
    isCheckingTools = true;
    try {
      const statuses = await AppAPI.getToolsStatus();
      toolStatuses = statuses || [];
      gbeStatus = toolStatuses?.find(s => s.key === 'gbe') || toolStatuses?.[0] || null;
    } catch {
      toolStatuses = [];
      gbeStatus = null;
    } finally {
      isCheckingTools = false;
    }
  }

  async function handleUpdateTool(key, displayName) {
    updatingTools = { ...updatingTools, [key]: true };
    try {
      await AppAPI.updateTool(key);
      notificationStore.items.add({
        type: 'info',
        title: 'Download Queued',
        message: `${displayName || key} update queued in Downloads.`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Tool Update Failed',
        message: err?.message || String(err)
      });
    } finally {
      updatingTools = { ...updatingTools, [key]: false };
    }
  }

  let isSubmittingBatch = false;

  async function handleBatchAchievements() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchAchievements();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Achievements Task Queued',
        message: 'Downloading missing achievements and icons in background. Track progress in Tasks.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchArtwork() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchArtwork();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Artwork Task Queued',
        message: 'Resolving and caching missing posters, heroes, and banners in background.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchNormalize() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchNormalize();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Directory Normalization Queued',
        message: 'Standardizing directory names across all library games.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleBatchVerify() {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchVerify();
      notificationStore.items.add({
        type: 'info',
        title: 'Batch Integrity Verification Queued',
        message: 'Verifying depot file integrity for all manifest-backed games.'
      });
      activeView.set('tasks');
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Queue Task',
        message: err?.message || String(err)
      });
    } finally {
      isSubmittingBatch = false;
    }
  }

  async function handleAddLibrary() {
    const selected = await AppAPI.selectDirectory('Select Steam Games Library Folder');
    if (selected) {
      addLibraryPath(selected);
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
    localCfg.library_paths = $libraryPaths;
    
    // Save Global Environment Variables
    const envObj = {};
    for (const item of globalEnvList) {
      const k = (item.key || '').trim();
      if (k) {
        envObj[k] = item.value || '';
      }
    }
    localCfg.global_env = envObj;

    const [ok] = await Promise.all([
      configStore.save(localCfg),
      AppAPI.saveGSEUserConfig(gseUserCfg).catch(err => {
        console.error('Failed to save GSE user config:', err);
        return false;
      })
    ]);
    if (ok) {
      notificationStore.items.add({
        type: 'success',
        title: 'Settings Saved',
        message: 'System and Goldberg configuration updated successfully.'
      });
      activeView.set('library');
    }
  }

  async function handleReset() {
    if (confirm('Reset all configuration settings to default values?')) {
      localCfg = {
        gbe_dir: '~/.local/share/gbe_fork',
        steamless_dir: '~/.local/share/steamless',
        steam_userdata: '~/.local/share/Steam/userdata',
        operating_mode: 'loader',
        default_runner: 'umu',
        steamless: true,
        achievements: true,
        normalize: true,
        mangohud: false,
        wine_fullscreen_fsr: false,
        proton_nvapi: false,
        enable_launch_notify: true,
        steam_web_api_key: '',
        steamgriddb_api_key: '',
        hubcap_api_key: '',
      };
      gseUserCfg = {
        account_name: 'GamuxPlayer',
        account_steamid: '76561197960287930',
        language: 'english',
        ip_country: 'US'
      };
      currentScale = 16;
      configStore.setScale(16);
    }
  }
</script>

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-6 sm:px-10 pb-10 max-w-3xl mx-auto w-full space-y-6 text-xs sm:text-sm animate-fade-in">
  <!-- Header -->
  <div class="border-b border-[#20202c] pb-4 flex items-center justify-between">
    <div>
      <h2 class="text-xl font-black text-white">System Settings</h2>
      <p class="text-xs text-slate-400 mt-0.5 font-mono">~/.config/gamux/config.json</p>
    </div>
  </div>

  <!-- Settings Panel -->
  <div class="bg-[#121218] border border-[#242434] rounded-2xl overflow-hidden shadow-xl">
    <!-- 5 Tabs: System, Emu, Paths, Integrations, Tools -->
    <!-- Tab Bar -->
    <div class="border-b border-[#20202c] bg-[#0f0f15] px-6 flex space-x-6 text-xs font-bold overflow-x-auto">
      <button
        type="button"
        on:click={() => { activeTab = 'system'; }}
        class="py-3 transition-colors flex items-center space-x-2 {activeTab === 'system' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
        </svg>
        <span>System</span>
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'emu'; }}
        class="py-3 transition-colors flex items-center space-x-2 {activeTab === 'emu' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
        <span>DRM</span>
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'paths'; }}
        class="py-3 transition-colors flex items-center space-x-2 {activeTab === 'paths' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <span>Paths</span>
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'integrations'; }}
        class="py-3 transition-colors flex items-center space-x-2 {activeTab === 'integrations' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <rect x="2" y="6" width="20" height="12" rx="4" stroke-width="2" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 12h4m-2-2v4m7-1h.01m3 2h.01" />
        </svg>
        <span>Integrations</span>
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'tools'; }}
        class="py-3 transition-colors flex items-center space-x-2 {activeTab === 'tools' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
        <span>Tools</span>
      </button>
      <button
        type="button"
        on:click={() => { activeTab = 'logs'; }}
        class="py-3 transition-colors flex items-center space-x-2 {activeTab === 'logs' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
      >
        <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>Logs</span>
      </button>
    </div>

    <!-- Body -->
    <div class="p-6 space-y-4">
      {#if activeTab === 'system'}
        <!-- SYSTEM TAB -->
        <div class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3.5">
            <div>
              <label for="setUiScale" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Display Scale
              </label>
              <select id="setUiScale" bind:value={currentScale} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value={16}>Compact (16px)</option>
                <option value={18}>Standard (18px)</option>
                <option value={20}>Large (20px)</option>
                <option value={22}>Handheld / Steam Deck (22px)</option>
                <option value={24}>Extra Large / 4K TV (24px)</option>
              </select>
            </div>
            <div>
              <label for="setSfx" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Tactile Audio SFX
              </label>
              <select id="setSfx" bind:value={sfxOn} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value={true}>Enabled (Synthesized)</option>
                <option value={false}>Disabled (Muted)</option>
              </select>
            </div>
            <div>
              <label for="setLaunchNotify" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Launch Notifications
              </label>
              <select id="setLaunchNotify" bind:value={localCfg.enable_launch_notify} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value={true}>Enabled</option>
                <option value={false}>Disabled</option>
              </select>
            </div>
          </div>

          <!-- Global Environment Variables -->
          <div class="space-y-2 pt-3 border-t border-[#20202c]">
            <div class="flex items-center justify-between">
              <div>
                <span class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider">
                  Global Environment Variables
                </span>
                <span class="text-[10px] text-slate-500">
                  Applied to all games launched via Standalone runner. Can be overridden per-game.
                </span>
              </div>
              <button
                type="button"
                on:click={addGlobalEnv}
                class="bg-[#20202c] hover:bg-[#2c2c3c] text-white text-xs font-bold px-3 py-1 rounded-lg border border-[#343446] transition-colors"
              >
                + Add Variable
              </button>
            </div>

            {#if globalEnvList.length > 0}
              <div class="space-y-2 max-h-48 overflow-y-auto pr-1">
                {#each globalEnvList as item, idx}
                  <div class="flex items-center space-x-2">
                    <input
                      type="text"
                      bind:value={item.key}
                      placeholder="KEY (e.g. MANGOHUD)"
                      class="w-1/2 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-1.5 text-slate-100 font-mono text-xs outline-none focus:border-white"
                    />
                    <input
                      type="text"
                      bind:value={item.value}
                      placeholder="VALUE (e.g. 1)"
                      class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-1.5 text-slate-100 font-mono text-xs outline-none focus:border-white"
                    />
                    <button
                      type="button"
                      on:click={() => removeGlobalEnv(idx)}
                      class="p-1.5 rounded-lg bg-[#20202c] hover:bg-rose-950/40 text-slate-400 hover:text-rose-400 border border-[#303042] transition-colors"
                      title="Remove Variable"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="p-3 rounded-xl bg-[#0b0b10] border border-[#20202e] text-center text-slate-500 text-xs">
                No global environment variables set. Click <strong>+ Add Variable</strong> to configure.
              </div>
            {/if}
          </div>

          <!-- Toggles -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-3 border-t border-[#20202c]">
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.normalize} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Normalize game folder names automatically</span>
            </label>
          </div>
        </div>

      {:else if activeTab === 'emu'}
        <!-- EMU TAB -->
        <div class="space-y-4">
          <!-- GBE Fork Release Card on Top -->
          <div class="bg-[#0b0b10] border border-[#262638] rounded-xl p-4 shadow-inner space-y-2">
            <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
              <button
                type="button"
                class="text-left space-y-1 select-none group"
                on:click={() => { isGbeExpanded = !isGbeExpanded; }}
              >
                <div class="flex items-center space-x-2">
                  <strong class="text-white font-bold text-sm group-hover:text-cyan-300 transition-colors">gbe_fork</strong>
                  {#if gbeStatus?.has_update}
                    <span class="text-[10px] bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 px-2 py-0.5 rounded-full font-bold">
                      Update Available
                    </span>
                  {:else if isCheckingTools}
                    <span class="text-[10px] text-slate-400 font-mono animate-pulse">Checking...</span>
                  {:else}
                    <span class="text-[10px] bg-slate-800 text-slate-300 border border-slate-700 px-2 py-0.5 rounded-full font-bold">
                      Up to date
                    </span>
                  {/if}
                  <svg class="w-3.5 h-3.5 text-slate-500 transform transition-transform {isGbeExpanded ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                  </svg>
                </div>
                <p class="text-slate-400 font-mono text-[11px]">
                  Installed: <span class="text-slate-200">{gbeStatus?.installed_tag || 'N/A'}</span>
                  {#if gbeStatus?.latest_tag}
                    · Latest: <span class="text-emerald-400">{gbeStatus.latest_tag}</span>
                  {/if}
                </p>
              </button>

              <div class="flex items-center space-x-2 w-full sm:w-auto justify-end">
                <button
                  type="button"
                  on:click={loadToolsStatus}
                  disabled={isCheckingTools || updatingTools['gbe']}
                  class="bg-[#181824] hover:bg-[#222232] text-slate-300 font-bold text-xs px-3 py-1.5 rounded-xl border border-[#2c2c3e] transition-colors disabled:opacity-50"
                >
                  {isCheckingTools ? 'Checking...' : 'Check'}
                </button>
                <button
                  type="button"
                  on:click={() => handleUpdateTool('gbe', 'Goldberg Steam Emulator')}
                  disabled={updatingTools['gbe']}
                  class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-4 py-1.5 rounded-xl transition-colors shadow-sm disabled:opacity-50"
                >
                  {updatingTools['gbe'] ? 'Updating...' : gbeStatus?.has_update ? 'Update to Latest' : 'Reinstall'}
                </button>
              </div>
            </div>

            <!-- Expandable Release Notes -->
            {#if isGbeExpanded}
              <div class="pt-3 border-t border-[#202030] text-xs text-slate-300 space-y-2 animate-fade-in">
                <div class="flex items-center justify-between text-[11px] text-slate-400 font-mono">
                  <span class="font-bold text-slate-200">{gbeStatus?.release_name || gbeStatus?.latest_tag || 'Release Details'}</span>
                  {#if gbeStatus?.published_at}
                    <span>{new Date(gbeStatus.published_at).toLocaleDateString()}</span>
                  {/if}
                </div>
                {#if gbeStatus?.release_notes}
                  <div class="bg-[#08080c] p-3 rounded-lg border border-[#1e1e2c] font-mono text-[11px] whitespace-pre-wrap max-h-48 overflow-y-auto leading-relaxed text-slate-300">
                    {gbeStatus.release_notes}
                  </div>
                {:else}
                  <div class="text-slate-500 italic text-[11px]">No release notes available.</div>
                {/if}
                {#if gbeStatus?.release_url}
                  <div>
                    <a href={gbeStatus.release_url} target="_blank" rel="noopener noreferrer" class="text-cyan-400 hover:underline text-[11px] font-bold">
                      View on GitHub ↗
                    </a>
                  </div>
                {/if}
              </div>
            {/if}
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <div>
              <label for="setOperatingMode" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Default Operating Mode
              </label>
              <select id="setOperatingMode" bind:value={localCfg.operating_mode} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value="loader">Loader (Non-destructive, Recommended)</option>
                <option value="portable">Portable (Direct DLL/SO Replacement)</option>
                <option value="standalone">Standalone (Direct Play)</option>
                <option value="none">None (Disabled / DRM-free)</option>
              </select>
            </div>

            <div>
              <label for="setLanguage" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Default Language
              </label>
              <select id="setLanguage" bind:value={localCfg.language} class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none capitalize">
                <option value="auto">Auto (System Locale / Game Default)</option>
                <option value="english">English</option>
                <option value="german">German</option>
                <option value="french">French</option>
                <option value="spanish">Spanish</option>
                <option value="japanese">Japanese</option>
                <option value="schinese">Simplified Chinese</option>
                <option value="tchinese">Traditional Chinese</option>
                <option value="russian">Russian</option>
                <option value="brazilian">Portuguese (Brazil)</option>
                <option value="koreana">Korean</option>
                <option value="polish">Polish</option>
                <option value="italian">Italian</option>
                <option value="turkish">Turkish</option>
              </select>
            </div>
          </div>

          <!-- Persona Profile -->
          <div class="space-y-3 pt-3 border-t border-[#20202c]">
            <div class="flex items-center justify-between">
              <div>
                <span class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider">
                  Global Steam Persona
                </span>
                <span class="text-[10px] text-slate-500 font-mono">
                  ~/.local/share/GSE Saves/settings/configs.user.ini
                </span>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3.5">
              <div>
                <label for="setAccountName" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                  Account Name
                </label>
                <input
                  id="setAccountName"
                  type="text"
                  bind:value={gseUserCfg.account_name}
                  placeholder="GamuxPlayer"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none"
                />
              </div>
              <div>
                <label for="setSteamId" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                  SteamID (64-bit)
                </label>
                <input
                  id="setSteamId"
                  type="text"
                  bind:value={gseUserCfg.account_steamid}
                  placeholder="76561197960287930"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono"
                />
              </div>
              <div>
                <label for="setIpCountry" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                  Country Code
                </label>
                <input
                  id="setIpCountry"
                  type="text"
                  maxlength="2"
                  bind:value={gseUserCfg.ip_country}
                  placeholder="US"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono uppercase"
                />
              </div>
            </div>
          </div>

          <!-- Toggles -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-3 border-t border-[#20202c]">
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.steamless} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Unpack SteamStub DRM automatically</span>
            </label>
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.achievements} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Fetch achievements & DLC metadata</span>
            </label>
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.disable_overlay} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Disable Steam overlay</span>
            </label>
            <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" bind:checked={localCfg.offline_mode} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
              <span>Force offline LAN mode</span>
            </label>
          </div>
        </div>

      {:else if activeTab === 'paths'}
        <!-- PATHS TAB -->
        <div class="space-y-5">
          <!-- Game Libraries Section -->
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <strong class="text-white font-bold block text-sm">Game Libraries</strong>
                <p class="text-[11px] text-slate-400">Folders containing installed Steam games (Internal storage, SD cards, external drives).</p>
              </div>
              <button
                type="button"
                on:click={handleAddLibrary}
                class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-3.5 py-1.5 rounded-xl transition-colors shadow-sm"
              >
                + Add Library
              </button>
            </div>

            <div class="space-y-2">
              {#if $libraryPaths.length > 0}
                {#each $libraryPaths as libPath}
                  <div class="bg-[#0b0b10] border border-[#242436] p-3 rounded-xl flex items-center justify-between gap-3">
                    <div class="flex items-center space-x-2.5 min-w-0">
                      <svg class="w-4 h-4 text-slate-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                      </svg>
                      <span class="font-mono text-xs text-slate-200 truncate">{libPath}</span>
                    </div>
                    <div class="flex items-center space-x-2 flex-shrink-0">
                      <button
                        type="button"
                        on:click={() => scanLibrary(libPath)}
                        class="text-xs text-slate-400 hover:text-white px-2 py-1 bg-[#181824] rounded-lg border border-[#2c2c3e]"
                        title="Scan this library folder"
                      >
                        Scan
                      </button>
                      <button
                        type="button"
                        on:click={() => removeLibraryPath(libPath)}
                        class="text-xs text-rose-400 hover:text-rose-300 px-2 py-1 bg-[#201018] rounded-lg border border-[#3e202c]"
                        title="Remove library folder"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                {/each}
              {:else}
                <div class="p-4 rounded-xl border border-dashed border-[#282838] text-center text-slate-500 font-mono text-xs">
                  No library directories configured. Click "+ Add Library" to add a games folder.
                </div>
              {/if}
            </div>
          </div>

          <!-- System Binaries & Helper Paths -->
          <div class="space-y-3 pt-3 border-t border-[#20202c]">
            <strong class="text-white font-bold block text-sm">System Helper Paths</strong>

            <div>
              <label for="setGbeDir" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Goldberg Emulator Path
              </label>
              <div class="flex space-x-2">
                <input id="setGbeDir" type="text" bind:value={localCfg.gbe_dir} class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
                <button type="button" on:click={() => browseFolder('gbe_dir', 'Select Goldberg Directory')} class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244] text-xs">Browse</button>
              </div>
            </div>

            <div>
              <label for="setSteamlessDir" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Steamless Binary Path
              </label>
              <div class="flex space-x-2">
                <input id="setSteamlessDir" type="text" bind:value={localCfg.steamless_dir} class="flex-1 bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
                <button type="button" on:click={() => browseFolder('steamless_dir', 'Select Steamless Directory')} class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323244] text-xs">Browse</button>
              </div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'integrations'}
        <!-- INTEGRATIONS TAB -->
        <div class="space-y-4">
          <!-- Primary Runner Configuration -->
          <div class="bg-[#09090d] p-3.5 rounded-xl border border-[#202030]">
            <div>
              <label for="setDefaultRunner" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Default Compatibility Runner
              </label>
              <select id="setDefaultRunner" bind:value={localCfg.default_runner} class="w-full bg-[#121218] border border-[#2c2c3e] rounded-xl px-3 py-2 text-slate-100 outline-none">
                <option value="umu">UMU Launcher (umu-run - Recommended)</option>
                <option value="wine">Wine Direct (wine)</option>
                <option value="proton">Proton Direct (proton)</option>
                <option value="linux">Linux Native Executable</option>
              </select>
              <p class="text-[10px] text-slate-400 mt-1">
                Compatibility runner used for launching Windows games directly on Linux.
              </p>
            </div>
          </div>

          <!-- Compatibility & Prefixes -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <div>
              <label for="setWinePrefix" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Default Wine Prefix
              </label>
              <input
                id="setWinePrefix"
                type="text"
                bind:value={localCfg.wine_prefix}
                placeholder="~/.wine (or leave blank for runner default)"
                class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
              />
            </div>
            <div>
              <label for="setProtonPath" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
                Custom Proton Path
              </label>
              <input
                id="setProtonPath"
                type="text"
                bind:value={localCfg.proton_path}
                placeholder="/path/to/Proton (optional)"
                class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
              />
            </div>
          </div>

          <!-- Gaming Environment & Overlays -->
          <div class="pt-3 border-t border-[#20202c]">
            <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-2">
              Standalone Gaming Overlays & Performance
            </span>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={localCfg.mangohud} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>MangoHud Overlay</span>
              </label>
              <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={localCfg.wine_fullscreen_fsr} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>Wine Fullscreen FSR</span>
              </label>
              <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={localCfg.proton_nvapi} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>Proton NVAPI (DLSS)</span>
              </label>
            </div>
          </div>

          <!-- API Keys & Online Services -->
          <div class="pt-3 border-t border-[#20202c]">
            <span class="flex items-center space-x-1.5 text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-2">
              <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
              </svg>
              <span>API Keys & External Services</span>
            </span>
            <div class="bg-[#09090d] p-3.5 rounded-xl border border-[#202030] space-y-4">
              <!-- Steam Web API Key -->
              <div>
                <div class="flex items-center justify-between mb-1">
                  <label for="setSteamWebKey" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider">
                    Steam Web API Key
                  </label>
                  <a
                    href="https://steamcommunity.com/dev/apikey"
                    target="_blank"
                    rel="noreferrer"
                    class="text-[10px] text-blue-400 hover:text-blue-300 underline"
                  >
                    Get free API key ↗
                  </a>
                </div>
                <input
                  id="setSteamWebKey"
                  type="password"
                  bind:value={localCfg.steam_web_api_key}
                  placeholder="Enter 32-character Steam Web API key"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
                <p class="text-[10px] text-slate-400 mt-1">
                  Used for fetching detailed Steam store metadata, official achievement schemas, and game changelogs.
                </p>
              </div>

              <!-- SteamGridDB API Key -->
              <div class="pt-3 border-t border-[#181824]">
                <div class="flex items-center justify-between mb-1">
                  <label for="setSGDBKey" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider">
                    SteamGridDB API Key
                  </label>
                  <a
                    href="https://www.steamgriddb.com/profile/preferences/api"
                    target="_blank"
                    rel="noreferrer"
                    class="text-[10px] text-blue-400 hover:text-blue-300 underline"
                  >
                    Get free API key ↗
                  </a>
                </div>
                <input
                  id="setSGDBKey"
                  type="password"
                  bind:value={localCfg.steamgriddb_api_key}
                  placeholder="Enter SteamGridDB API key for cross-store covers & heroes"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
                <p class="text-[10px] text-slate-400 mt-1">
                  Provides high-res vertical posters, hero banners, and transparent logos for Steam, GOG, Epic, and Standalone games. Free keyless fallbacks (Steam CDN, GOG) will be used automatically when this is blank.
                </p>
              </div>

              <!-- Hubcap Depot API Key -->
              <div class="pt-3 border-t border-[#181824]">
                <div class="flex items-center justify-between mb-1">
                  <label for="setHubcapKey" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider">
                    Hubcap Depot API Key
                  </label>
                </div>
                <input
                  id="setHubcapKey"
                  type="password"
                  bind:value={localCfg.hubcap_api_key}
                  placeholder="Enter Hubcap API key (optional)"
                  class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
                <p class="text-[10px] text-slate-400 mt-1">
                  Used for unlocking and fetching community depot manifests, depot keys, and decryption metadata.
                </p>
              </div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'tools'}
        <!-- TOOLS TAB -->
        <div class="space-y-4">
          <!-- Toolchain Management -->
          <div class="bg-[#0b0b10] border border-[#242436] rounded-xl p-4 space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <strong class="text-white font-bold text-sm flex items-center space-x-2">
                  <svg class="w-4 h-4 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                  <span>Helper Tools & Toolchain</span>
                </strong>
                <p class="text-[11px] text-slate-400">
                  External tools for Steam emulation, DRM removal, and Proton execution.
                </p>
              </div>
              <button
                type="button"
                on:click={loadToolsStatus}
                disabled={isCheckingTools}
                class="bg-[#181824] hover:bg-[#222232] text-slate-300 font-bold text-xs px-3 py-1.5 rounded-xl border border-[#2c2c3e] transition-colors disabled:opacity-50"
              >
                {isCheckingTools ? 'Checking...' : 'Check All'}
              </button>
            </div>

            <div class="space-y-2.5 pt-1">
              {#each toolStatuses as tool}
                <div class="bg-[#08080c] border border-[#1e1e2c] p-3 rounded-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
                  <div class="space-y-0.5">
                    <div class="flex items-center space-x-2">
                      <span class="font-bold text-slate-100 text-xs">{tool.name}</span>
                      {#if tool.has_update}
                        <span class="text-[10px] bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 px-2 py-0.5 rounded-full font-bold">
                          Update Available
                        </span>
                      {:else}
                        <span class="text-[10px] bg-slate-800 text-slate-400 border border-slate-700 px-2 py-0.5 rounded-full font-medium">
                          {tool.installed_tag || 'Installed'}
                        </span>
                      {/if}
                    </div>
                    <div class="text-[11px] font-mono text-slate-400">
                      Installed: <span class="text-slate-200">{tool.installed_tag || 'N/A'}</span>
                      {#if tool.latest_tag}
                        · Latest: <span class="text-emerald-400">{tool.latest_tag}</span>
                      {/if}
                    </div>
                  </div>

                  <div class="flex items-center space-x-2">
                    {#if tool.release_url}
                      <a href={tool.release_url} target="_blank" rel="noopener noreferrer" class="text-slate-400 hover:text-cyan-300 text-xs px-2.5 py-1.5 rounded-lg border border-[#262638] bg-[#101018]">
                        GitHub ↗
                      </a>
                    {/if}
                    <button
                      type="button"
                      on:click={() => handleUpdateTool(tool.key, tool.name)}
                      disabled={updatingTools[tool.key]}
                      class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-3.5 py-1.5 rounded-xl transition-colors shadow-sm disabled:opacity-50"
                    >
                      {updatingTools[tool.key] ? 'Updating...' : tool.has_update ? 'Update' : 'Reinstall'}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Batch Library Maintenance Operations -->
          <div class="bg-[#0b0b10] border border-[#242436] rounded-xl p-4 space-y-4">
            <div>
              <strong class="text-white font-bold text-sm flex items-center space-x-2">
                <svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                <span>Batch Maintenance Operations</span>
              </strong>
              <p class="text-[11px] text-slate-400">
                Automated multi-game library operations executed in the background.
              </p>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1">
              <!-- 1. Batch Achievements -->
              <div class="bg-[#08080c] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                      <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                      </svg>
                      <span>Missing Achievements</span>
                    </span>
                  </div>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Download official Steam achievement schemas and icon artwork to steam_settings for all library titles.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={handleBatchAchievements}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-sky-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-sky-900/40 hover:border-sky-700/60 transition-all flex items-center justify-center space-x-1.5 active:scale-98"
                >
                  <span>Download Achievements</span>
                </button>
              </div>

              <!-- 2. Batch Artwork -->
              <div class="bg-[#08080c] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                      <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                      <span>Refresh Artwork</span>
                    </span>
                  </div>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Query SteamGridDB and Steam CDN to resolve and cache vertical posters, hero banners, and logos.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={handleBatchArtwork}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-purple-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-purple-900/40 hover:border-purple-700/60 transition-all flex items-center justify-center space-x-1.5 active:scale-98"
                >
                  <span>Refresh All Artwork</span>
                </button>
              </div>

              <!-- 3. Batch Apply GBE & Steamless -->
              <div class="bg-[#08080c] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                      <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                      </svg>
                      <span>Apply DRM & Emulator</span>
                    </span>
                    <span class="text-[10px] font-mono bg-[#161622] text-amber-300 border border-amber-500/30 px-2 py-0.5 rounded-md">
                      {$games.filter(g => !g.isPatched).length} unpatched
                    </span>
                  </div>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Unpack SteamStub DRM and generate Goldberg emulator configuration on unpatched games.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={batchApplyCleanGames}
                  disabled={$games.filter(g => !g.isPatched).length === 0}
                  class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-amber-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-amber-900/40 hover:border-amber-700/60 transition-all flex items-center justify-center space-x-1.5 active:scale-98"
                >
                  <span>Apply Setup ({$games.filter(g => !g.isPatched).length})</span>
                </button>
              </div>

              <!-- 4. Batch Normalize Directories -->
              <div class="bg-[#08080c] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                      <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                      </svg>
                      <span>Normalize Directory Names</span>
                    </span>
                  </div>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Standardize scene release and archive directory names (e.g. Game.Name.v1.0-P2P &rarr; Game Name).
                  </p>
                </div>
                <button
                  type="button"
                  on:click={handleBatchNormalize}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-emerald-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-emerald-900/40 hover:border-emerald-700/60 transition-all flex items-center justify-center space-x-1.5 active:scale-98"
                >
                  <span>Normalize Directories</span>
                </button>
              </div>

              <!-- 5. Batch Verify Integrity -->
              <div class="bg-[#08080c] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors md:col-span-2">
                <div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                      </svg>
                      <span>Verify All Depot Manifests</span>
                    </span>
                  </div>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Verify SHA-1 file integrity against official Steam depot manifests for all games that include manifest data.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={handleBatchVerify}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-cyan-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-cyan-900/40 hover:border-cyan-700/60 transition-all flex items-center justify-center space-x-1.5 active:scale-98"
                >
                  <span>Verify All Game Manifests</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      {:else if activeTab === 'logs'}
        <!-- LOGS & DIAGNOSTICS TAB -->
        <div class="space-y-4">
          <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-2 border-b border-[#20202c]">
            <div class="flex items-center space-x-2">
              <span class="font-bold text-white text-xs uppercase tracking-wider">System Logs & Activity</span>
              <span class="text-[10px] font-mono bg-[#08080c] text-slate-400 px-2 py-0.5 rounded-full border border-[#20202c]">
                {$notifItems.length} entries
              </span>
            </div>

            <div class="flex items-center space-x-2 flex-wrap gap-y-1">
              <!-- Filter Chips -->
              <div class="flex items-center space-x-1 bg-[#0c0c10] p-1 rounded-xl border border-[#242434] text-[11px] font-mono">
                {#each ['all', 'error', 'warning', 'info', 'success'] as f}
                  <button
                    type="button"
                    on:click={() => { logFilter = f; }}
                    class="px-2 py-0.5 rounded-lg capitalize transition-colors {logFilter === f ? 'bg-[#222232] text-white font-bold' : 'text-slate-500 hover:text-slate-300'}"
                  >
                    {f}
                  </button>
                {/each}
              </div>

              {#if $notifItems.length > 0}
                <button
                  type="button"
                  on:click={copyLogsToClipboard}
                  class="text-[11px] font-bold bg-[#181824] hover:bg-[#222232] text-slate-300 px-2.5 py-1 rounded-xl border border-[#2c2c3e] transition-colors flex items-center space-x-1.5 cursor-pointer"
                  title="Copy log to clipboard"
                >
                  <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  <span>Copy</span>
                </button>
                <button
                  type="button"
                  on:click={() => notificationStore.items.clearAll()}
                  class="text-[11px] font-bold bg-[#181824] hover:bg-rose-950/40 text-slate-400 hover:text-rose-300 px-2.5 py-1 rounded-xl border border-[#2c2c3e] hover:border-rose-900/50 transition-colors"
                >
                  Clear
                </button>
              {/if}
            </div>
          </div>

          <!-- Monospace Terminal Log Stream -->
          <div class="bg-[#08080c] border border-[#1e1e2c] rounded-xl p-3 sm:p-4 max-h-[440px] overflow-y-auto space-y-2 font-mono text-xs custom-scrollbar">
            {#if filteredLogs.length === 0}
              <div class="py-12 text-center text-slate-500 text-xs">
                <span>No log entries recorded.</span>
              </div>
            {:else}
              {#each filteredLogs as item (item.id || item.timestamp)}
                <div class="flex items-start space-x-2.5 py-1 border-b border-[#14141d] last:border-0 hover:bg-[#0e0e14] px-1.5 rounded transition-colors">
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
                      <strong class="text-slate-200 font-bold">{item.title}</strong>
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
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between p-4 border-t border-[#20202c] bg-[#0f0f15]">
      <button type="button" on:click={handleReset} class="text-slate-400 hover:text-slate-200 text-xs font-bold px-2 py-1">
        Reset Defaults
      </button>
      <button
        type="button"
        on:click={handleSave}
        class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-xs px-6 py-2 rounded-xl transition-colors shadow-sm"
      >
        Save Settings
      </button>
    </div>
  </div>
</div>
