<script>
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { activeView, libraryPaths, addLibraryPath, removeLibraryPath, scanLibrary, games, batchApplyCleanGames } from '../../stores/library.js';
  import { configStore } from '../../stores/config.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { AppAPI } from '../../api/app.js';
  import { isSfxEnabled, setSfxEnabled } from '../../utils/audio.js';
  import SwitcherRow from '../common/SwitcherRow.svelte';
  import SettingSection from '../common/SettingSection.svelte';
  import SettingRow from '../common/SettingRow.svelte';

  let localCfg = {};
  let currentScale = 18;
  let sfxOn = isSfxEnabled();
  let activeTab = 'system'; // 'system' | 'runner' | 'emu' | 'paths' | 'integrations' | 'tools' | 'logs'
  let searchQuery = '';

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
    currentScale = get(configStore.uiScale) || 18;
    
    const urlParams = new URLSearchParams(window.location.search);
    const tabParam = urlParams.get('tab');
    if (tabParam && ['system', 'runner', 'emu', 'paths', 'integrations', 'tools', 'logs'].includes(tabParam)) {
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

  async function handleBatchOp(kind, notifTitle, notifMessage) {
    isSubmittingBatch = true;
    try {
      await AppAPI.submitBatchLibraryOp(kind);
      notificationStore.items.add({
        type: 'info',
        title: notifTitle,
        message: notifMessage
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
    const selected = await AppAPI.selectDirectory('Select Steam Games Library Folder', '');
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

  const navCategories = [
    { id: 'system', label: 'System & Interface', icon: 'M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4', badge: '' },
    { id: 'runner', label: 'Runners & Graphics', icon: 'M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z', badge: '' },
    { id: 'emu', label: 'DRM & Goldberg', icon: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z', badge: '' },
    { id: 'paths', label: 'Game Libraries & Paths', icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z', badge: `${$libraryPaths.length}` },
    { id: 'tools', label: 'Tools & Maintenance', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z', badge: gbeStatus?.has_update ? 'Update' : '' },
    { id: 'integrations', label: 'Integrations & APIs', icon: 'M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z', badge: '' },
    { id: 'logs', label: 'Event Logs', icon: 'M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z', badge: `${$notifItems.length}` },
  ];

  function matchesSearch(text) {
    if (!searchQuery.trim()) return true;
    return text.toLowerCase().includes(searchQuery.toLowerCase());
  }
</script>

<div class="flex-1 overflow-y-auto pt-16 sm:pt-20 px-4 sm:px-8 pb-12 max-w-6xl mx-auto w-full space-y-6 text-xs sm:text-sm animate-fade-in custom-scrollbar">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-[#20202c] pb-4">
    <div>
      <h2 class="text-xl sm:text-2xl font-black text-white tracking-tight flex items-center gap-2.5">
        <span>System Settings</span>
      </h2>
      <p class="text-xs text-slate-400 mt-0.5 font-mono">~/.config/gamux/config.json</p>
    </div>

    <!-- Quick Search Bar -->
    <div class="relative w-full sm:w-72">
      <svg class="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Search settings..."
        class="w-full bg-[#12121a] border border-[#262638] focus:border-cyan-500 rounded-xl pl-9 pr-3 py-1.5 text-xs text-slate-100 placeholder-slate-500 outline-none transition-all"
      />
      {#if searchQuery}
        <button
          type="button"
          on:click={() => { searchQuery = ''; }}
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-200 text-xs"
        >
          ✕
        </button>
      {/if}
    </div>
  </div>

  <!-- Master-Detail Split Pane Container -->
  <div class="bg-[#0e0e14] border border-[#222232] rounded-2xl overflow-hidden shadow-2xl flex flex-col md:flex-row min-h-[580px]">
    <!-- Left Navigation Sidebar -->
    <nav class="w-full md:w-64 bg-[#09090d] border-b md:border-b-0 md:border-r border-[#1e1e2c] p-2 sm:p-3 flex flex-row md:flex-col gap-1 overflow-x-auto flex-shrink-0 select-none">
      {#each navCategories as nav}
        <button
          type="button"
          on:click={() => { activeTab = nav.id; }}
          class="flex items-center justify-between px-3 py-2.5 rounded-xl font-bold text-xs transition-all cursor-pointer whitespace-nowrap md:whitespace-normal {activeTab === nav.id ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200 hover:bg-[#14141d]'}"
          aria-label={nav.id === 'system' ? 'System' : nav.id === 'emu' ? 'DRM' : nav.id === 'tools' ? 'Tools' : nav.label}
        >
          <div class="flex items-center space-x-2.5 min-w-0">
            <svg class="w-4 h-4 opacity-80 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={nav.icon} />
            </svg>
            <span class="truncate">{nav.label}</span>
          </div>

          {#if nav.badge}
            <span class="ml-2 px-1.5 py-0.2 rounded-full text-[9px] font-mono font-bold {nav.badge === 'Update' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 animate-pulse' : 'bg-[#181824] text-slate-400 border border-[#2c2c3e]'}">
              {nav.badge}
            </span>
          {/if}
        </button>
      {/each}
    </nav>

    <!-- Right Content Detail Pane -->
    <div class="flex-1 p-5 sm:p-7 space-y-6 overflow-y-auto max-h-[750px] custom-scrollbar">
      <!-- 1. SYSTEM & INTERFACE TAB -->
      {#if activeTab === 'system' || (searchQuery && (matchesSearch('scale') || matchesSearch('sfx') || matchesSearch('audio') || matchesSearch('notification') || matchesSearch('normalize')))}
        <div class="space-y-5">
          <SettingSection
            title="Interface & Display"
            description="Customize display density, tactile sound effects, and launcher notifications."
          >
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <SettingRow label="Display Scale" description="Scale layout for desktop, Deck, or 4K screens." forId="setUiScale" inline={false}>
                <select id="setUiScale" bind:value={currentScale} class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs">
                  <option value={16}>Compact (16px)</option>
                  <option value={18}>Standard (18px)</option>
                  <option value={20}>Large (20px)</option>
                  <option value={22}>Handheld / Steam Deck (22px)</option>
                  <option value={24}>Extra Large / 4K TV (24px)</option>
                </select>
              </SettingRow>

              <SettingRow label="Tactile Audio SFX" description="Synthesized feedback on UI navigation." forId="setSfx" inline={false}>
                <select id="setSfx" bind:value={sfxOn} class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs">
                  <option value={true}>Enabled (Synthesized)</option>
                  <option value={false}>Disabled (Muted)</option>
                </select>
              </SettingRow>

              <SettingRow label="Launch Notifications" description="System notifications on game launch." forId="setLaunchNotify" inline={false}>
                <select id="setLaunchNotify" bind:value={localCfg.enable_launch_notify} class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs">
                  <option value={true}>Enabled</option>
                  <option value={false}>Disabled</option>
                </select>
              </SettingRow>
            </div>
          </SettingSection>

          <SettingSection
            title="Library Normalization"
            description="Automatically sanitize folder names when scanning games."
          >
            <SwitcherRow
              label="Normalize Folder Names"
              description="Standardize scanned directory folder names to match official Steam titles."
              tag="NORMALIZE"
              color="purple"
              bind:value={localCfg.normalize}
            />
          </SettingSection>
        </div>
      {/if}

      <!-- 2. RUNNERS & GRAPHICS TAB -->
      {#if activeTab === 'runner' || (searchQuery && (matchesSearch('runner') || matchesSearch('wine') || matchesSearch('proton') || matchesSearch('mangohud') || matchesSearch('fsr') || matchesSearch('nvapi') || matchesSearch('dlss') || matchesSearch('env')))}
        <div class="space-y-5">
          <SettingSection
            title="Default Compatibility Runner"
            description="Global runtime engine for launching Windows executables on Linux."
          >
            <SettingRow label="Primary Runner Engine" description="UMU Launcher automatically configures Proton fixes and runtimes." forId="setDefaultRunner" inline={false}>
              <select id="setDefaultRunner" bind:value={localCfg.default_runner} class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs">
                <option value="umu">UMU Launcher (umu-run - Recommended)</option>
                <option value="wine">Wine Direct (wine)</option>
                <option value="proton">Proton Direct (proton)</option>
                <option value="linux">Linux Native Executable</option>
              </select>
            </SettingRow>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
              <SettingRow label="Default Wine Prefix" description="Leave blank for runner default." forId="setWinePrefix" inline={false}>
                <input
                  id="setWinePrefix"
                  type="text"
                  bind:value={localCfg.wine_prefix}
                  placeholder="~/.wine (or leave blank)"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
              </SettingRow>

              <SettingRow label="Custom Proton Path" description="Optional custom Proton directory." forId="setProtonPath" inline={false}>
                <input
                  id="setProtonPath"
                  type="text"
                  bind:value={localCfg.proton_path}
                  placeholder="/path/to/Proton (optional)"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
              </SettingRow>
            </div>
          </SettingSection>

          <SettingSection
            title="Global Graphics & Overlays"
            description="Default telemetry and visual scaling flags applied across all games unless overridden."
          >
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              <SwitcherRow
                label="MangoHud Overlay"
                description="FPS and telemetry HUD"
                tag="HUD"
                color="emerald"
                bind:value={localCfg.mangohud}
              />

              <SwitcherRow
                label="Wine Fullscreen FSR"
                description="AMD FidelityFX upscaling"
                tag="FSR"
                color="cyan"
                bind:value={localCfg.wine_fullscreen_fsr}
              />

              <SwitcherRow
                label="Proton NVAPI (DLSS)"
                description="Expose DXVK-NVAPI / DLSS"
                tag="DLSS"
                color="emerald"
                bind:value={localCfg.proton_nvapi}
              />
            </div>
          </SettingSection>

          <!-- Global Environment Variables -->
          <SettingSection
            title="Global Environment Variables"
            description="Applied to all games launched via Standalone runner. Can be overridden per-game."
          >
            <svelte:fragment slot="action">
              <button
                type="button"
                on:click={addGlobalEnv}
                class="bg-[#1c1c28] hover:bg-[#262638] text-white text-xs font-bold px-3 py-1 rounded-lg border border-[#323246] transition-colors cursor-pointer"
              >
                + Add Variable
              </button>
            </svelte:fragment>

            {#if globalEnvList.length > 0}
              <div class="space-y-2 max-h-48 overflow-y-auto pr-1">
                {#each globalEnvList as item, idx}
                  <div class="flex items-center space-x-2">
                    <input
                      type="text"
                      bind:value={item.key}
                      placeholder="KEY (e.g. MANGOHUD)"
                      class="w-1/2 bg-[#0c0c10] border border-[#262636] focus:border-cyan-500 rounded-xl px-3 py-1.5 text-slate-100 font-mono text-xs outline-none"
                    />
                    <input
                      type="text"
                      bind:value={item.value}
                      placeholder="VALUE (e.g. 1)"
                      class="flex-1 bg-[#0c0c10] border border-[#262636] focus:border-cyan-500 rounded-xl px-3 py-1.5 text-slate-100 font-mono text-xs outline-none"
                    />
                    <button
                      type="button"
                      on:click={() => removeGlobalEnv(idx)}
                      class="p-1.5 rounded-lg bg-[#181824] hover:bg-rose-950/40 text-slate-400 hover:text-rose-400 border border-[#28283a] transition-colors cursor-pointer"
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
              <div class="p-3.5 rounded-xl bg-[#0a0a0f] border border-[#1e1e2c] text-center text-slate-500 text-xs font-mono">
                No global environment variables set.
              </div>
            {/if}
          </SettingSection>
        </div>
      {/if}

      <!-- 3. DRM & GOLDBERG TAB -->
      {#if activeTab === 'emu' || (searchQuery && (matchesSearch('drm') || matchesSearch('goldberg') || matchesSearch('gbe') || matchesSearch('persona') || matchesSearch('steamid') || matchesSearch('steamless') || matchesSearch('achievement') || matchesSearch('overlay')))}
        <div class="space-y-5">
          <!-- GBE Fork Release Status Card -->
          <SettingSection
            title="Goldberg Steam Emulator"
            description="Core emulation layer for Steam DRM bypass and local multiplayer."
            badge={gbeStatus?.has_update ? 'Update Available' : isCheckingTools ? 'Checking...' : 'Up to date'}
            badgeColor={gbeStatus?.has_update ? 'emerald' : 'slate'}
          >
            <svelte:fragment slot="action">
              <button
                type="button"
                on:click={loadToolsStatus}
                disabled={isCheckingTools || updatingTools['gbe']}
                class="bg-[#181824] hover:bg-[#222232] text-slate-300 font-bold text-xs px-3 py-1.5 rounded-xl border border-[#2c2c3e] transition-colors disabled:opacity-50 cursor-pointer"
              >
                {isCheckingTools ? 'Checking...' : 'Check'}
              </button>
              <button
                type="button"
                on:click={() => handleUpdateTool('gbe', 'Goldberg Steam Emulator')}
                disabled={updatingTools['gbe']}
                class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-4 py-1.5 rounded-xl transition-colors shadow-sm disabled:opacity-50 cursor-pointer"
              >
                {updatingTools['gbe'] ? 'Updating...' : gbeStatus?.has_update ? 'Update to Latest' : 'Reinstall'}
              </button>
            </svelte:fragment>

            <div class="flex items-center justify-between text-xs text-slate-400 font-mono">
              <div class="flex items-center space-x-2">
                <strong class="text-white font-bold">gbe_fork</strong>
                <span class="text-slate-600">·</span>
                <span>Installed: <strong class="text-slate-200">{gbeStatus?.installed_tag || 'N/A'}</strong></span>
                {#if gbeStatus?.latest_tag}
                  <span> · Latest: <strong class="text-emerald-400">{gbeStatus.latest_tag}</strong></span>
                {/if}
              </div>
              <button
                type="button"
                on:click={() => { isGbeExpanded = !isGbeExpanded; }}
                class="text-xs text-cyan-400 hover:underline font-bold"
              >
                {isGbeExpanded ? 'Hide Release Notes' : 'View Release Notes'}
              </button>
            </div>

            <!-- Expandable Release Notes -->
            {#if isGbeExpanded}
              <div class="pt-3 border-t border-[#1c1c28] text-xs text-slate-300 space-y-2 animate-fade-in">
                <span class="font-bold text-slate-200 text-xs font-mono">{gbeStatus?.release_name || gbeStatus?.latest_tag || 'Release Details'}</span>
                {#if gbeStatus?.release_notes}
                  <div class="bg-[#08080c] p-3 rounded-lg border border-[#1e1e2c] font-mono text-[11px] whitespace-pre-wrap max-h-48 overflow-y-auto leading-relaxed text-slate-300">
                    {gbeStatus.release_notes}
                  </div>
                {/if}
              </div>
            {/if}
          </SettingSection>

          <!-- Operating Mode & Language Defaults -->
          <SettingSection
            title="Operating Mode & Language Defaults"
            description="Standard behavior applied to new game setups."
          >
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <SettingRow label="Default Operating Mode" description="Loader mode preserves original binaries non-destructively." forId="setOperatingMode" inline={false}>
                <select id="setOperatingMode" bind:value={localCfg.operating_mode} class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs">
                  <option value="loader">Loader (Non-destructive, Recommended)</option>
                  <option value="portable">Portable (Direct DLL/SO Replacement)</option>
                  <option value="standalone">Standalone (Direct Play)</option>
                  <option value="none">None (Disabled / DRM-free)</option>
                </select>
              </SettingRow>

              <SettingRow label="Default Game Language" description="Locale passed to Goldberg emulator." forId="setLanguage" inline={false}>
                <select id="setLanguage" bind:value={localCfg.language} class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none capitalize text-xs">
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
              </SettingRow>
            </div>
          </SettingSection>

          <!-- Global Persona Profile -->
          <SettingSection
            title="Global Steam Persona"
            description="Universal identity profile for saves and LAN multiplayer."
          >
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <SettingRow label="Account Name" description="In-game display name." forId="setAccountName" inline={false}>
                <input
                  id="setAccountName"
                  type="text"
                  bind:value={gseUserCfg.account_name}
                  placeholder="GamuxPlayer"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs"
                />
              </SettingRow>

              <SettingRow label="SteamID (64-bit)" description="Unique persona identifier." forId="setSteamId" inline={false}>
                <input
                  id="setSteamId"
                  type="text"
                  bind:value={gseUserCfg.account_steamid}
                  placeholder="76561197960287930"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
              </SettingRow>

              <SettingRow label="Country Code" description="2-letter ISO country." forId="setIpCountry" inline={false}>
                <input
                  id="setIpCountry"
                  type="text"
                  maxlength="2"
                  bind:value={gseUserCfg.ip_country}
                  placeholder="US"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono uppercase text-xs"
                />
              </SettingRow>
            </div>
          </SettingSection>

          <!-- DRM Features Toggles -->
          <SettingSection
            title="DRM Bypass & Network Features"
            description="Automated unpacker and emulation switches."
          >
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <SwitcherRow
                label="Unpack SteamStub DRM"
                description="Run Steamless unpacker on game binaries automatically."
                tag="STEAMLESS"
                color="sky"
                bind:value={localCfg.steamless}
              />

              <SwitcherRow
                label="Fetch Achievements & DLC"
                description="Download achievement icons and DLC schemas."
                tag="ICONS"
                color="amber"
                bind:value={localCfg.achievements}
              />

              <SwitcherRow
                label="Disable Steam Overlay"
                description="Suppress Goldberg emulator in-game overlay."
                tag="OVERLAY"
                color="rose"
                invertLabel={true}
                bind:value={localCfg.disable_overlay}
              />

              <SwitcherRow
                label="Force Offline LAN Mode"
                description="Prevent emulator WAN networking."
                tag="OFFLINE"
                color="cyan"
                bind:value={localCfg.offline_mode}
              />
            </div>
          </SettingSection>
        </div>
      {/if}

      <!-- 4. GAME LIBRARIES & PATHS TAB -->
      {#if activeTab === 'paths' || (searchQuery && (matchesSearch('library') || matchesSearch('folder') || matchesSearch('path') || matchesSearch('storage') || matchesSearch('gbe_dir') || matchesSearch('steamless_dir')))}
        <div class="space-y-5">
          <SettingSection
            title="Game Library Folders"
            description="Folders scanned for installed games across internal drives, SD cards, and mount points."
          >
            <svelte:fragment slot="action">
              <button
                type="button"
                on:click={handleAddLibrary}
                class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-3.5 py-1.5 rounded-xl transition-colors shadow-sm cursor-pointer"
              >
                + Add Library
              </button>
            </svelte:fragment>

            <div class="space-y-2">
              {#if $libraryPaths.length > 0}
                {#each $libraryPaths as libPath}
                  <div class="bg-[#0a0a0f] border border-[#1e1e2c] p-3 rounded-xl flex items-center justify-between gap-3">
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
                        class="text-xs text-slate-300 hover:text-white px-2.5 py-1 bg-[#181824] hover:bg-[#222232] rounded-lg border border-[#2c2c3e] transition-colors cursor-pointer"
                        title="Scan this library folder"
                      >
                        Scan
                      </button>
                      <button
                        type="button"
                        on:click={() => removeLibraryPath(libPath)}
                        class="text-xs text-rose-400 hover:text-rose-300 px-2.5 py-1 bg-[#1c1016] hover:bg-[#2c141e] rounded-lg border border-[#3e1e2a] transition-colors cursor-pointer"
                        title="Remove library folder"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                {/each}
              {:else}
                <div class="p-5 rounded-xl border border-dashed border-[#242436] text-center text-slate-500 font-mono text-xs">
                  No library directories configured. Click "+ Add Library" to add a games folder.
                </div>
              {/if}
            </div>
          </SettingSection>

          <!-- System Binaries & Helper Paths -->
          <SettingSection
            title="Helper Binary Paths"
            description="Locations of external tooling directories."
          >
            <SettingRow label="Goldberg Emulator Path" description="Directory containing GBE binaries." forId="setGbeDir" inline={false}>
              <div class="flex space-x-2">
                <input id="setGbeDir" type="text" bind:value={localCfg.gbe_dir} class="flex-1 bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
                <button type="button" on:click={() => browseFolder('gbe_dir', 'Select Goldberg Directory')} class="bg-[#1c1c28] hover:bg-[#262638] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323246] text-xs cursor-pointer">Browse</button>
              </div>
            </SettingRow>

            <SettingRow label="Steamless Binary Path" description="Directory containing Steamless CLI unpacker." forId="setSteamlessDir" inline={false}>
              <div class="flex space-x-2">
                <input id="setSteamlessDir" type="text" bind:value={localCfg.steamless_dir} class="flex-1 bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs" />
                <button type="button" on:click={() => browseFolder('steamless_dir', 'Select Steamless Directory')} class="bg-[#1c1c28] hover:bg-[#262638] text-slate-200 px-3.5 py-2 rounded-xl font-bold border border-[#323246] text-xs cursor-pointer">Browse</button>
              </div>
            </SettingRow>
          </SettingSection>
        </div>
      {/if}

      <!-- 5. TOOLS & MAINTENANCE TAB -->
      {#if activeTab === 'tools' || (searchQuery && (matchesSearch('tools') || matchesSearch('batch') || matchesSearch('maintenance') || matchesSearch('achievement') || matchesSearch('artwork') || matchesSearch('steamless')))}
        <div class="space-y-5">
          <!-- Toolchain Management -->
          <SettingSection
            title="Helper Tools & Toolchain"
            description="Installed external tools for Steam emulation and DRM unpacking."
          >
            <svelte:fragment slot="action">
              <button
                type="button"
                on:click={loadToolsStatus}
                disabled={isCheckingTools}
                class="bg-[#181824] hover:bg-[#222232] text-slate-300 font-bold text-xs px-3 py-1.5 rounded-xl border border-[#2c2c3e] transition-colors disabled:opacity-50 cursor-pointer"
              >
                {isCheckingTools ? 'Checking...' : 'Check All'}
              </button>
            </svelte:fragment>

            <div class="space-y-2.5">
              {#each toolStatuses as tool}
                <div class="bg-[#0a0a0f] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
                  <div class="space-y-0.5">
                    <div class="flex items-center space-x-2">
                      <span class="font-bold text-slate-100 text-xs">{tool.name || tool.displayName || (tool.key === 'steamless' ? 'Steamless CLI Unpacker' : tool.key === 'gbe' ? 'Goldberg Emulator (GBE)' : tool.key)}</span>
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
                        <span> · Latest: <strong class="text-emerald-400">{tool.latest_tag}</strong></span>
                      {/if}
                    </div>
                  </div>

                  <div class="flex items-center space-x-2">
                    {#if tool.release_url}
                      <a href={tool.release_url} target="_blank" rel="noopener noreferrer" class="text-slate-400 hover:text-cyan-300 text-xs px-2.5 py-1.5 rounded-lg border border-[#262638] bg-[#12121a]">
                        GitHub ↗
                      </a>
                    {/if}
                    <button
                      type="button"
                      on:click={() => handleUpdateTool(tool.key, tool.name)}
                      disabled={updatingTools[tool.key]}
                      class="bg-white hover:bg-neutral-200 text-black font-black text-xs px-3.5 py-1.5 rounded-xl transition-colors shadow-sm disabled:opacity-50 cursor-pointer"
                    >
                      {updatingTools[tool.key] ? 'Updating...' : tool.has_update ? 'Update' : 'Reinstall'}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </SettingSection>

          <!-- Batch Maintenance Operations -->
          <SettingSection
            title="Batch Maintenance Operations"
            description="Automated library operations executed in the background across all detected games."
          >
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <!-- 1. Batch Achievements -->
              <div class="bg-[#0a0a0f] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                    <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                    </svg>
                    <span>Missing Achievements</span>
                  </span>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Download official Steam achievement schemas and icon artwork to steam_settings for all library titles.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={() => handleBatchOp('achievements', 'Batch Achievements Task Queued', 'Downloading missing achievements and icons in background. Track progress in Tasks.')}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#161622] hover:bg-[#202030] disabled:opacity-40 text-sky-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-sky-900/40 hover:border-sky-700/60 transition-all flex items-center justify-center space-x-1.5 cursor-pointer active:scale-98"
                >
                  <span>Download Achievements</span>
                </button>
              </div>

              <!-- 2. Batch Artwork -->
              <div class="bg-[#0a0a0f] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                    <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                    <span>Refresh Artwork</span>
                  </span>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Query SteamGridDB and Steam CDN to resolve and cache vertical posters, hero banners, and logos.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={() => handleBatchOp('artwork', 'Batch Artwork Task Queued', 'Resolving and caching missing posters, heroes, and banners in background.')}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#161622] hover:bg-[#202030] disabled:opacity-40 text-purple-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-purple-900/40 hover:border-purple-700/60 transition-all flex items-center justify-center space-x-1.5 cursor-pointer active:scale-98"
                >
                  <span>Refresh All Artwork</span>
                </button>
              </div>

              <!-- 3. Batch Apply GBE & Steamless -->
              <div class="bg-[#0a0a0f] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
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
                  class="w-full bg-[#161622] hover:bg-[#202030] disabled:opacity-40 text-amber-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-amber-900/40 hover:border-amber-700/60 transition-all flex items-center justify-center space-x-1.5 cursor-pointer active:scale-98"
                >
                  <span>Apply Setup ({$games.filter(g => !g.isPatched).length})</span>
                </button>
              </div>

              <!-- 4. Batch Normalize Directories -->
              <div class="bg-[#0a0a0f] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 transition-colors">
                <div>
                  <span class="font-bold text-slate-100 text-xs flex items-center gap-2">
                    <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                    </svg>
                    <span>Normalize Directory Names</span>
                  </span>
                  <p class="text-[11px] text-slate-400 mt-1 leading-relaxed">
                    Standardize directory names to official Steam titles.
                  </p>
                </div>
                <button
                  type="button"
                  on:click={() => handleBatchOp('normalize', 'Batch Directory Normalization Queued', 'Standardizing directory names across all library games.')}
                  disabled={isSubmittingBatch || $games.length === 0}
                  class="w-full bg-[#161622] hover:bg-[#202030] disabled:opacity-40 text-emerald-300 hover:text-white font-bold text-xs py-2 px-3 rounded-lg border border-emerald-900/40 hover:border-emerald-700/60 transition-all flex items-center justify-center space-x-1.5 cursor-pointer active:scale-98"
                >
                  <span>Normalize Directories</span>
                </button>
              </div>
            </div>
          </SettingSection>
        </div>
      {/if}

      <!-- 6. INTEGRATIONS & APIS TAB -->
      {#if activeTab === 'integrations' || (searchQuery && (matchesSearch('api') || matchesSearch('steam') || matchesSearch('steamgriddb') || matchesSearch('sgdb') || matchesSearch('hubcap') || matchesSearch('artwork') || matchesSearch('depot')))}
        <div class="space-y-5">
          <SettingSection
            title="External Metadata & Artwork Services"
            description="API keys for retrieving high-resolution cover artwork, achievements, and community depots."
          >
            <!-- Steam Web API Key -->
            <SettingRow label="Steam Web API Key" description="Used for metadata schemas and changelogs." forId="setSteamWebKey" inline={false}>
              <div class="space-y-1.5">
                <div class="flex items-center justify-end">
                  <a
                    href="https://steamcommunity.com/dev/apikey"
                    target="_blank"
                    rel="noreferrer"
                    class="text-[10px] text-cyan-400 hover:underline"
                  >
                    Get free Steam API key ↗
                  </a>
                </div>
                <input
                  id="setSteamWebKey"
                  type="password"
                  bind:value={localCfg.steam_web_api_key}
                  placeholder="Enter 32-character Steam Web API key"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
              </div>
            </SettingRow>

            <!-- SteamGridDB API Key -->
            <SettingRow label="SteamGridDB API Key" description="Provides vertical posters, hero banners, and transparent logos." forId="setSGDBKey" inline={false}>
              <div class="space-y-1.5">
                <div class="flex items-center justify-end">
                  <a
                    href="https://www.steamgriddb.com/profile/preferences/api"
                    target="_blank"
                    rel="noreferrer"
                    class="text-[10px] text-cyan-400 hover:underline"
                  >
                    Get free SteamGridDB API key ↗
                  </a>
                </div>
                <input
                  id="setSGDBKey"
                  type="password"
                  bind:value={localCfg.steamgriddb_api_key}
                  placeholder="Enter SteamGridDB API key"
                  class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
                />
              </div>
            </SettingRow>

            <!-- Hubcap Depot API Key -->
            <SettingRow label="Hubcap Depot API Key" description="Optional key for unlocking community depot manifests." forId="setHubcapKey" inline={false}>
              <input
                id="setHubcapKey"
                type="password"
                bind:value={localCfg.hubcap_api_key}
                placeholder="Enter Hubcap API key (optional)"
                class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
              />
            </SettingRow>
          </SettingSection>
        </div>
      {/if}

      <!-- 7. EVENT LOGS TAB -->
      {#if activeTab === 'logs' || (searchQuery && (matchesSearch('log') || matchesSearch('event') || matchesSearch('diagnostic')))}
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
                    class="px-2 py-0.5 rounded-lg capitalize transition-colors cursor-pointer {logFilter === f ? 'bg-[#222232] text-white font-bold' : 'text-slate-500 hover:text-slate-300'}"
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
                  class="text-[11px] font-bold bg-[#181824] hover:bg-rose-950/40 text-slate-400 hover:text-rose-300 px-2.5 py-1 rounded-xl border border-[#2c2c3e] hover:border-rose-900/50 transition-colors cursor-pointer"
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
  </div>

  <!-- Unified Footer Actions -->
  <div class="flex items-center justify-between p-4 bg-[#0e0e14] border border-[#222232] rounded-2xl shadow-xl">
    <button
      type="button"
      on:click={handleReset}
      class="text-slate-400 hover:text-slate-200 text-xs font-bold px-3 py-1.5 rounded-xl hover:bg-[#161622] transition-colors cursor-pointer"
    >
      Reset Defaults
    </button>
    <button
      type="button"
      on:click={handleSave}
      class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black text-xs px-6 py-2 rounded-xl transition-colors shadow-sm cursor-pointer"
    >
      Save Settings
    </button>
  </div>
</div>
