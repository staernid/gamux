<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import {
    selectedGame,
    activeView,
    queueApplyGame,
    setGameRunning,
    runningGames,
    refreshGame
  } from '../../stores/library.js';
  import { tasksStore, isDownloadTask } from '../../stores/tasks.js';
  import { modalStore } from '../../stores/modal.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { drawerDraft } from '../../stores/drawerDraft.js';
  import { configStore } from '../../stores/config.js';
  import { AppAPI } from '../../api/app.js';
  import { formatBytes, formatETA } from '../../utils/format.js';
  import {
    getPlatformColors,
    getPlatformType,
    getMetadataInfo,
    isSteamApp,
    hasMetadataMatch,
    formatPlaytime,
    formatLastPlayed,
    isSamePath
  } from '../../utils/game.js';
  import PosterArt from '../common/PosterArt.svelte';
  import SettingSection from '../common/SettingSection.svelte';
  import SettingRow from '../common/SettingRow.svelte';
  import SeamlessSwitch from '../common/SeamlessSwitch.svelte';
  import InheritSelect from '../common/InheritSelect.svelte';
  import GhostInput from '../common/GhostInput.svelte';
  import EnvDiffEditor from '../common/EnvDiffEditor.svelte';
  import DrawerAboutTab from '../stage/drawer/DrawerAboutTab.svelte';
  import DrawerResourcesTab from '../stage/drawer/DrawerResourcesTab.svelte';
  import DrawerAchievementsTab from '../stage/drawer/DrawerAchievementsTab.svelte';
  import DrawerNewsTab from '../stage/drawer/DrawerNewsTab.svelte';
  import DrawerSavesTab from '../stage/drawer/DrawerSavesTab.svelte';
  import DrawerDLCTab from '../stage/drawer/DrawerDLCTab.svelte';
  import DrawerIntegrityTab from '../stage/drawer/DrawerIntegrityTab.svelte';
  import DrawerLogsTab from '../stage/drawer/DrawerLogsTab.svelte';

  let playButtonRef;

  // 6 Primary Consolidated Tabs: 'overview' | 'community' | 'general' | 'environment' | 'saves' | 'files'
  let activeTab = 'overview';

  // Sub-Navigation Tabs
  let activeCommunitySubTab = 'achievements'; // 'achievements' | 'news'
  let activeFilesSubTab = 'depots'; // 'depots' | 'integrity' | 'logs'

  let logoFailed = false;
  let heroFailed = false;
  let globalCfg = {};
  let isSaving = false;
  let saveSuccess = false;
  let saveSuccessTimer = null;

  // News State
  let newsItems = [];
  let isLoadingNews = false;
  let expandedNewsIndex = 0;
  let loadedNewsAppID = '';
  let newsError = '';

  // Integrity State
  let integrityReport = null;
  let integrityError = '';
  let isCheckingIntegrity = false;
  let isNormalizing = false;

  // 3-Pillar Inline Wizards State
  let showSourceWizard = false;
  let showMetadataWizard = false;

  // Metadata Wizard Search State
  let metaTargetStore = 'steam'; // 'steam' | 'gog' | 'rawg' | 'none'
  let metaSearchQuery = '';
  let metaCustomId = '';
  let metaSearchResults = [];
  let isSearchingMeta = false;
  let isLinkingMeta = false;

  const metadataProviders = [
    { id: 'steam', display_name: 'Steam', badge: 'AppID' },
    { id: 'gog', display_name: 'GOG.com', badge: 'GOG ID' },
    { id: 'rawg', display_name: 'RAWG Database', badge: 'Open DB' },
    { id: 'none', display_name: 'None', badge: 'Unlinked' }
  ];

  async function handleMetaSearch() {
    if (!metaSearchQuery.trim()) return;
    isSearchingMeta = true;
    try {
      const results = await AppAPI.searchStoreGames(metaSearchQuery.trim());
      metaSearchResults = (results || []).filter(c => {
        const storeLower = (c.store || '').toLowerCase();
        return storeLower === metaTargetStore.toLowerCase();
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Search Failed',
        message: err?.message || String(err)
      });
      metaSearchResults = [];
    } finally {
      isSearchingMeta = false;
    }
  }

  async function handleApplyMetaLink(idToUse = null) {
    if (!game?.gameDir) return;
    const targetId = idToUse !== null ? String(idToUse) : (metaTargetStore === 'none' || metaTargetStore === 'standalone' ? '0' : metaCustomId.trim());
    isLinkingMeta = true;
    try {
      const updated = await AppAPI.linkGameMetadata(game.gameDir, metaTargetStore, targetId);
      notificationStore.items.add({
        type: 'success',
        title: 'Metadata Linked',
        message: metaTargetStore === 'none' || metaTargetStore === 'standalone' || !targetId || targetId === '0'
          ? `Unlinked metadata for ${game.name}`
          : `Linked ${game.name} to ${metaTargetStore.toUpperCase()} (#${targetId})`
      });
      if (updated) {
        selectedGame.update(curr => (curr && isSamePath(curr.gameDir, game.gameDir) ? { ...curr, ...updated } : curr));
      }
      showMetadataWizard = false;
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Linking Failed',
        message: err?.message || String(err)
      });
    } finally {
      isLinkingMeta = false;
    }
  }

  async function handleUnlinkMeta() {
    if (!game?.gameDir) return;
    isLinkingMeta = true;
    try {
      const updated = await AppAPI.unlinkGameMetadata(game.gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Metadata Reset',
        message: `Restored auto-detected identity for ${game.name}.`
      });
      if (updated) {
        selectedGame.update(curr => (curr && isSamePath(curr.gameDir, game.gameDir) ? { ...curr, ...updated } : curr));
      }
      showMetadataWizard = false;
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Reset Failed',
        message: err?.message || String(err)
      });
    } finally {
      isLinkingMeta = false;
    }
  }

  const allTasks = tasksStore.items;

  $: game = $selectedGame;
  $: if (game) { logoFailed = false; heroFailed = false; }
  $: isGameRunning = Boolean(game?.gameDir && $runningGames.has(game.gameDir));
  $: pColors = game ? getPlatformColors(game) : { border: '#1a9fff', badgeBg: '#1a9fff', label: 'WIN64', glow: 'rgba(26,159,255,0.4)' };
  $: platformLabel = game ? getPlatformType(game) : 'Windows';
  $: metaInfo = game ? getMetadataInfo(game) : { provider: 'None', id: '' };
  $: isReady = game ? (game.isReady || game.isPatched || !!game.exePath) : false;
  $: isSteam = game ? isSteamApp(game.appId) : false;
  $: hasMeta = game ? hasMetadataMatch(game) : false;
  $: heroBackdropUrl = game ? (game.localHeroUrl || game.heroUrl || game.localBannerUrl || game.bannerUrl || (isSteam ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${game.appId}/library_hero.jpg` : '') || game.localCoverUrl || game.coverUrl || '') : '';

  $: activeGameTask = ($allTasks || []).find(t => {
    if (t.status !== 'running' && t.status !== 'downloading' && t.status !== 'queued' && t.status !== 'pending') return false;
    if (game?.gameDir && isSamePath(t.game_dir || t.gameDir, game.gameDir)) return true;
    if (game?.appId && game.appId !== '0' && (String(t.app_id || t.appID) === String(game.appId))) return true;
    return false;
  });
  $: isTaskActive = Boolean(activeGameTask);

  function getTaskBadgeLabel(task) {
    if (!task) return 'TASK';
    const op = (task.operation || task.type || '').toLowerCase();
    if (isDownloadTask(task)) return 'DOWNLOADING';
    if (op === 'apply') return 'APPLYING SETUP';
    if (op === 'verify') return 'VERIFYING';
    if (op === 'normalize') return 'NORMALIZING';
    if (op === 'update') return 'UPDATING';
    if (op === 'tool_update') return 'TOOL UPDATE';
    if (op === 'rollback') return 'ROLLBACK';
    return 'PROCESSING';
  }

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

  const runnerOptions = [
    { value: 'umu', label: 'UMU (Unified Linux Wine/Proton Runtime)' },
    { value: 'proton', label: 'Proton (Valve Compatibility Tool)' },
    { value: 'wine', label: 'Wine (System Wine)' },
    { value: 'linux', label: 'Linux Native (Direct Binary)' }
  ];

  const operatingModeOptions = [
    { value: 'loader', label: 'Seamless Loader (Preserve Files - Recommended)' },
    { value: 'portable', label: 'Standalone Portable (Direct Files)' },
    { value: 'none', label: 'Direct Launch (DRM-Free / None)' }
  ];

  $: gameLanguageOptions = (languageOptions || []).map(l => ({
    value: l,
    label: l.charAt(0).toUpperCase() + l.slice(1)
  }));

  $: overrideCount = computeOverrideCount($drawerDraft, game, globalCfg);

  function computeOverrideCount(draft, g, cfg) {
    if (!draft || !g) return 0;
    let count = 0;
    if (draft.runner && draft.runner.trim() !== '') count++;
    if (draft.mangoHud !== null && draft.mangoHud !== undefined) count++;
    if (draft.winePrefix && draft.winePrefix.trim() !== '') count++;
    if (draft.protonPath && draft.protonPath.trim() !== '') count++;
    if (draft.exeArgs && draft.exeArgs.trim() !== '') count++;
    if (draft.dllOverrides && draft.dllOverrides.trim() !== '') count++;
    if (draft.language && draft.language.trim() !== '') count++;
    if (draft.operatingMode && draft.operatingMode.trim() !== '') count++;
    if (draft.metadataProvider && draft.metadataProvider.trim() !== '') count++;
    if (draft.metadataId && draft.metadataId.trim() !== '') count++;
    if (draft.customEnv && Object.keys(draft.customEnv).length > 0) count++;
    return count;
  }

  async function handleSaveSettings() {
    if (!game) return;
    isSaving = true;
    try {
      const ok = await drawerDraft.save(game);
      if (ok) {
        saveSuccess = true;
        if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
        saveSuccessTimer = setTimeout(() => { saveSuccess = false; }, 3000);
      }
    } catch (err) {
      console.error('Failed to save settings:', err);
    } finally {
      isSaving = false;
    }
  }

  async function handleResetSettings() {
    if (!game) return;
    await drawerDraft.resetDefaults(game);
    drawerDraft.init(game, globalCfg);
  }

  $: if (game && globalCfg) {
    drawerDraft.init(game, globalCfg);
  }

  onMount(async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const tabParam = urlParams.get('tab');
    const isDrawerOpen = urlParams.get('drawer') === '1' || urlParams.get('properties') === '1';

    if (isDrawerOpen) {
      if (['env', 'environment', 'runner', 'compat', 'emu', 'emulation', 'graphics'].includes(tabParam)) {
        activeTab = 'environment';
      } else {
        activeTab = 'general';
      }
    } else if (tabParam) {
      if (['overview', 'about', 'resources'].includes(tabParam)) {
        activeTab = 'overview';
      } else if (tabParam === 'community' || tabParam === 'achievements') {
        activeTab = 'community';
        activeCommunitySubTab = 'achievements';
      } else if (tabParam === 'news') {
        activeTab = 'community';
        activeCommunitySubTab = 'news';
      } else if (['general', 'config', 'settings', 'target', 'metadata', 'art', 'artwork', 'covers'].includes(tabParam)) {
        activeTab = 'general';
      } else if (['environment', 'env', 'runner', 'compat', 'emu', 'emulation', 'graphics'].includes(tabParam)) {
        activeTab = 'environment';
      } else if (tabParam === 'saves') {
        activeTab = 'saves';
      } else if (['files', 'dlc', 'depots'].includes(tabParam)) {
        activeTab = 'files';
        activeFilesSubTab = 'depots';
      } else if (tabParam === 'integrity') {
        activeTab = 'files';
        activeFilesSubTab = 'integrity';
      } else if (tabParam === 'logs') {
        activeTab = 'files';
        activeFilesSubTab = 'logs';
      }
    } else {
      activeTab = 'overview';
    }

    const cfg = await configStore.load();
    if (cfg) globalCfg = cfg;
    if (game) {
      drawerDraft.init(game, globalCfg);
    }

    await tick();
    if (playButtonRef) {
      playButtonRef.focus();
    }
  });

  onDestroy(() => {
    if (saveSuccessTimer) {
      clearTimeout(saveSuccessTimer);
    }
  });

  async function handleSave() {
    if (!game || isSaving) return;
    isSaving = true;
    saveSuccess = false;
    if (saveSuccessTimer) clearTimeout(saveSuccessTimer);
    try {
      const success = await drawerDraft.save(game);
      if (success) {
        await refreshGame(game.gameDir);
        saveSuccess = true;
        saveSuccessTimer = setTimeout(() => {
          saveSuccess = false;
        }, 2500);
      }
    } finally {
      isSaving = false;
    }
  }

  async function handleResetDefaults() {
    if (!game) return;
    const ok = await drawerDraft.resetDefaults(game);
    if (ok) {
      await refreshGame(game.gameDir);
    }
  }

  function handleRollback() {
    if (!game) return;
    drawerDraft.rollback(game);
    refreshGame(game.gameDir);
  }

  function returnToShelf() {
    $activeView = 'library';
  }

  function handlePrimaryAction() {
    if (!game) return;

    if (isGameRunning) {
      AppAPI.stopGame(game.gameDir);
      setGameRunning(game.gameDir, false);
      notificationStore.info('Game Stopped', `Terminated process for ${game.name}`);
      return;
    }

    const launchPayload = {
      game_dir: game.gameDir,
      name: game.name,
      app_id: game.appId,
      platform: game.platform,
      exe_path: game.exePath,
      exe_args: game.exeArgs,
      operating_mode: game.operatingMode,
      runner: game.runner,
    };

    if (isReady) {
      AppAPI.launchGame(launchPayload);
    } else {
      AppAPI.quickPlay(launchPayload).catch(() => {
        queueApplyGame(game);
      });
    }
  }

  function openDirectory() {
    if (game?.gameDir) {
      AppAPI.openDirectory(game.gameDir);
    }
  }

  function openSettingsTab(targetTab = 'general') {
    if (['env', 'environment', 'runner', 'compat', 'emu', 'emulation', 'graphics'].includes(targetTab)) {
      activeTab = 'environment';
    } else {
      activeTab = 'general';
    }
  }

  async function loadNews(force = false) {
    if (!game?.appId || game.appId === '0') return;
    if (!force && loadedNewsAppID === game.appId && newsItems.length > 0) return;

    isLoadingNews = true;
    newsError = '';
    try {
      const items = await AppAPI.fetchNews(String(game.appId), 6);
      newsItems = items || [];
      loadedNewsAppID = game.appId;
      if (!items || items.length === 0) {
        newsError = 'No patch notes found on Steam for this title.';
      }
    } catch (err) {
      newsError = err?.message || String(err);
    } finally {
      isLoadingNews = false;
    }
  }

  function toggleNewsExpand(idx) {
    expandedNewsIndex = expandedNewsIndex === idx ? -1 : idx;
  }

  async function runIntegrityCheck() {
    if (!game?.gameDir) return;
    isCheckingIntegrity = true;
    integrityError = '';
    integrityReport = null;
    try {
      const rep = await AppAPI.verifyIntegrity(game.gameDir);
      const total = rep?.official_count ?? rep?.OfficialCount ?? 0;
      if (!rep || total === 0) {
        integrityError = 'No official depot manifest entries found for this title.';
      } else {
        integrityReport = rep;
      }
    } catch (err) {
      integrityError = err?.message || String(err);
      notificationStore.error('Integrity Check Notice', integrityError);
    } finally {
      isCheckingIntegrity = false;
    }
  }

  async function handleNormalizeDirectory() {
    if (!game?.gameDir) return;
    const oldDir = game.gameDir;
    try {
      isNormalizing = true;
      await AppAPI.submitNormalize(oldDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Normalizing Directory',
        message: `Queued directory normalization for ${game.name || oldDir}`
      });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Normalization Failed',
        message: err?.message || String(err)
      });
    } finally {
      isNormalizing = false;
    }
  }

  async function handleUpdateGame() {
    if (!game?.gameDir) return;
    const appIdNum = parseInt(game.appId, 10);
    if (appIdNum && !isNaN(appIdNum)) {
      try {
        await tasksStore.start({
          app_id: appIdNum,
          game_name: game.name,
          target_dir: game.gameDir,
          platform: game.platform,
          auto_apply: true,
        });
      } catch (err) {
        // Handled in store
      }
    } else {
      try {
        await AppAPI.submitUpdate(game.gameDir);
        notificationStore.items.add({
          type: 'info',
          title: 'Update Enqueued',
          message: `Update task started for ${game.name}`,
          gameName: game.name
        });
      } catch (err) {
        notificationStore.error('Update Failed', err);
      }
    }
  }

  const primaryTabs = ['overview', 'community', 'general', 'environment', 'saves', 'files'];

  function cycleTab(dir) {
    const idx = primaryTabs.indexOf(activeTab);
    let next = idx + dir;
    if (next < 0) next = primaryTabs.length - 1;
    if (next >= primaryTabs.length) next = 0;
    activeTab = primaryTabs[next];
  }

  $: if (activeTab === 'community' && activeCommunitySubTab === 'news' && loadedNewsAppID !== game?.appId && !isLoadingNews && game?.appId) {
    loadNews();
  }

  function handleKeydown(e) {
    const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);
    if (isInput || $modalStore) return;

    if (e.key === 'Escape' || e.key === 'b' || e.key === 'B') {
      e.preventDefault();
      returnToShelf();
    } else if (e.key === 'x' || e.key === 'X') {
      e.preventDefault();
      openSettingsTab('general');
    } else if (e.key === 'y' || e.key === 'Y') {
      e.preventDefault();
      openDirectory();
    } else if (e.key === 'q' || e.key === 'Q' || e.key === 'PageUp') {
      e.preventDefault();
      cycleTab(-1);
    } else if (e.key === 'e' || e.key === 'E' || e.key === 'PageDown') {
      e.preventDefault();
      cycleTab(1);
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if game}
  <div class="flex-1 overflow-y-auto w-full relative flex flex-col animate-fade-in custom-scrollbar">
    <!-- Hero Banner Background & Vignette (Bleeds behind floating topbar) -->
    <div class="relative w-full h-96 sm:h-[420px] flex-shrink-0 overflow-hidden select-none">
      <!-- Background Image with dynamic blur -->
      <div class="absolute inset-0 bg-[#07070a]">
        <!-- Seamless Cover Texture Anchor (Prevents layout/color pop during view enter) -->
        {#if game.localCoverUrl || game.coverUrl}
          <img
            src={game.localCoverUrl || game.coverUrl}
            alt=""
            class="absolute inset-0 w-full h-full object-cover opacity-25 filter blur-xl scale-125 pointer-events-none"
          />
        {/if}

        {#if heroBackdropUrl && !heroFailed}
          <img
            src={heroBackdropUrl}
            alt=""
            class="w-full h-full object-cover object-top opacity-70 filter blur-[0.5px] scale-105 transition-opacity duration-700"
            on:error={() => { heroFailed = true; }}
          />
        {:else}
          <!-- Art-Poor Fallback: Dynamic Ambient Grid & Subtle Watermark -->
          <div class="absolute inset-0 bg-gradient-to-br from-[#0e0f18] via-[#09090e] to-[#040407]">
            <div class="absolute inset-0 opacity-[0.04]" style="background-image: radial-gradient(#ffffff 1px, transparent 1px); background-size: 24px 24px;"></div>
            <div class="absolute -right-10 -bottom-12 text-[160px] font-black text-white/[0.03] select-none pointer-events-none font-mono tracking-tighter">
              {game.name ? game.name.slice(0, 3).toUpperCase() : 'GMX'}
            </div>
          </div>
        {/if}
        <!-- Ambient Radial Color Glow (Top Left) -->
        <div
          class="absolute -top-20 -left-20 w-[600px] h-[350px] rounded-full opacity-25 filter blur-3xl pointer-events-none"
          style="background-color: {pColors.glow || 'rgba(26,159,255,0.4)'};"
        ></div>
        <!-- Ambient Secondary Glow (Bottom Right) -->
        <div
          class="absolute -bottom-10 right-10 w-[450px] h-[250px] rounded-full opacity-15 filter blur-3xl pointer-events-none"
          style="background-color: {pColors.border || '#1a9fff'};"
        ></div>
      </div>

      <!-- Ambient Vignette & Gradient Overlays -->
      <div class="absolute inset-0 bg-gradient-to-t from-[#09090c] via-[#09090c]/50 to-transparent"></div>
      <div class="absolute inset-0 bg-gradient-to-r from-[#09090c]/80 via-transparent to-[#09090c]/60"></div>
      <div class="absolute inset-x-0 top-0 h-24 bg-gradient-to-b from-black/40 to-transparent pointer-events-none"></div>

      <!-- Top Nav Breadcrumb -->
      <div class="absolute top-4 left-6 z-20 flex items-center space-x-3">
        <button
          type="button"
          on:click={returnToShelf}
          class="flex items-center space-x-2 bg-[#12121a]/80 hover:bg-[#1e1e2c] border border-[#27273a] text-slate-300 hover:text-white px-3.5 py-1.5 rounded-xl text-xs font-bold transition shadow-lg backdrop-blur-md group cursor-pointer"
          title="Back to Shelf (Esc / B)"
        >
          <span class="text-slate-400 group-hover:text-cyan-400 transition-colors">←</span>
          <span>Shelf</span>
          <span class="text-[10px] text-slate-500 font-mono px-1 py-0.2 rounded bg-black/40 border border-[#2c2c3e]">B / Esc</span>
        </button>
      </div>

      <!-- Hero Content Container -->
      <div class="absolute bottom-6 left-6 right-6 z-20 flex flex-col sm:flex-row sm:items-end justify-between gap-6 max-w-6xl mx-auto">
        <!-- Left: Game Poster + Title/Logo + Badges -->
        <div class="flex items-end space-x-5 min-w-0">
          <!-- Poster Art (Cover 600x900) -->
          <div
            class="w-24 sm:w-32 aspect-[2/3] rounded-2xl overflow-hidden shadow-2xl border flex-shrink-0 hidden xs:block bg-black/40 group/poster relative cursor-pointer"
            style="border-color: {pColors.border}; box-shadow: 0 12px 30px -8px {pColors.glow};"
            on:click={() => modalStore.open('artwork-picker', { game, initialTab: 'cover' })}
            title="Change Cover Artwork"
            role="button"
            tabindex="0"
            on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); modalStore.open('artwork-picker', { game, initialTab: 'cover' }); } }}
          >
            <PosterArt
              appId={game.appId}
              name={game.name}
              posterUrl={game.coverUrl}
              localCoverUrl={game.localCoverUrl}
              className="w-full h-full object-cover group-hover/poster:scale-105 transition-transform duration-300"
            />
            <div class="absolute inset-0 bg-black/60 opacity-0 group-hover/poster:opacity-100 transition-opacity flex flex-col items-center justify-center gap-1 text-white p-1 text-center backdrop-blur-[2px]">
              <svg class="w-4 h-4 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span class="text-[10px] font-bold">Change Art</span>
            </div>
          </div>

          <!-- Title/Logo & Meta Block -->
          <div class="space-y-2 min-w-0">
            <!-- Store / Running Badge -->
            <div class="flex items-center space-x-2 flex-wrap gap-y-1">
              <span
                class="px-2.5 py-0.5 rounded-lg text-[10px] font-black uppercase tracking-wider text-white shadow-md"
                style="background-color: {pColors.badgeBg};"
              >
                {platformLabel} Source
              </span>
              {#if metaInfo.provider && metaInfo.provider !== 'None'}
                <span class="px-2.5 py-0.5 rounded-lg text-[10px] font-mono font-bold bg-[#141420] border border-[#2c2c42] text-sky-300 shadow-md">
                  Meta: {metaInfo.provider}{metaInfo.id ? ` #${metaInfo.id}` : ''}
                </span>
              {/if}
              {#if isGameRunning}
                <span class="px-2 py-0.5 rounded-lg text-[10px] font-bold bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 font-mono flex items-center space-x-1.5 animate-pulse">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                  <span>RUNNING</span>
                </span>
              {/if}
            </div>

            {#if (game.localLogoUrl || game.logoUrl) && !logoFailed}
              <div class="py-1">
                <img
                  src={game.localLogoUrl || game.logoUrl}
                  alt={game.name}
                  class="max-h-20 sm:max-h-24 md:max-h-28 max-w-[280px] sm:max-w-[360px] md:max-w-[420px] object-contain object-left drop-shadow-2xl"
                  on:error={() => { logoFailed = true; }}
                />
              </div>
            {:else}
              <h1 class="text-2xl sm:text-4xl font-black text-white tracking-tight drop-shadow-md truncate max-w-2xl">
                {game.name}
              </h1>
            {/if}

            <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-slate-400 font-mono">
              <span>{formatBytes(game.diskSizeBytes)}</span>
              <span class="text-slate-600">·</span>
              <span class="truncate max-w-sm">{game.runner || globalCfg?.default_runner || 'UMU'}</span>
              {#if !game.isReady}
                <span class="text-slate-600">·</span>
                <span class="text-amber-400">Setup Required</span>
              {/if}
              {#if game.playtimeSeconds > 0}
                <span class="text-slate-600">·</span>
                <span class="text-sky-300 font-bold flex items-center gap-1.5">
                  <svg class="w-3.5 h-3.5 opacity-80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>{formatPlaytime(game.playtimeSeconds)}</span>
                </span>
              {/if}
              {#if game.lastPlayed > 0}
                <span class="text-slate-600">·</span>
                <span class="text-slate-400">Played {formatLastPlayed(game.lastPlayed)}</span>
              {/if}
              {#if game.appId && game.appId !== '0'}
                <span class="text-slate-600">·</span>
                <span class="text-slate-500">#{game.appId}</span>
              {/if}
            </div>
          </div>
        </div>

        <!-- Right: Primary CTA & Actions -->
        <div class="flex items-center space-x-3 flex-shrink-0">
          <!-- Primary Action Button (Play / Stop / Setup / Task Running) -->
          <button
            bind:this={playButtonRef}
            type="button"
            on:click={handlePrimaryAction}
            disabled={isTaskActive}
            class="px-6 py-3 rounded-2xl font-black text-sm transition-all flex items-center space-x-2.5 shadow-2xl active:scale-95 focus:outline-none focus:ring-2 focus:ring-white/40 cursor-pointer disabled:cursor-not-allowed {isGameRunning ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-600/30' : isTaskActive ? 'bg-[#1e1e2c] border border-cyan-500/50 text-cyan-300 shadow-cyan-950/40' : isReady ? 'bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 shadow-emerald-500/30' : 'bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-400 hover:to-orange-400 text-slate-950 shadow-amber-500/30'}"
          >
            {#if isGameRunning}
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
              </svg>
              <span>Stop Game</span>
            {:else if isTaskActive}
              <svg class="animate-spin w-4 h-4 text-cyan-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
              </svg>
              <span>{activeGameTask.status === 'queued' ? 'Queued...' : 'Task in Progress...'}</span>
            {:else if isReady}
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
              </svg>
              <span>Play Now</span>
            {:else}
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd" />
              </svg>
              <span>Setup & Play</span>
            {/if}
          </button>

          <!-- Files Button (Y) -->
          <button
            type="button"
            on:click={openDirectory}
            class="p-3 rounded-2xl bg-[#14141e]/90 hover:bg-[#1f1f2e] border border-[#2b2b3e] text-slate-300 hover:text-white transition shadow-xl flex items-center justify-center group cursor-pointer"
            title="Open Directory in File Manager (Y)"
          >
            <svg class="w-5 h-5 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Main Content & 5-Tab Section -->
    <div class="max-w-6xl w-full mx-auto px-6 py-4 flex-1 flex flex-col space-y-4">
      <!-- Live Task Progress HUD (Displayed when an operation or download is active for this game) -->
      {#if activeGameTask}
        <div class="bg-gradient-to-r from-[#12121e] via-[#101018] to-[#12121e] border border-cyan-500/40 rounded-2xl p-4 shadow-xl space-y-2.5 animate-fade-in">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center space-x-2.5 min-w-0">
              <span class="px-2 py-0.5 rounded-lg text-[10px] font-black uppercase tracking-wider bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 flex items-center space-x-1.5 flex-shrink-0">
                <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse"></span>
                <span>{getTaskBadgeLabel(activeGameTask)}</span>
              </span>
              <h4 class="font-bold text-white text-xs truncate">
                {activeGameTask.stage || activeGameTask.title || 'Task executing'}
              </h4>
              {#if activeGameTask.details}
                <span class="text-slate-500 text-xs hidden sm:inline">·</span>
                <span class="text-slate-400 text-xs font-mono truncate hidden sm:inline max-w-sm">
                  {activeGameTask.details}
                </span>
              {/if}
            </div>

            <!-- Task Controls -->
            <div class="flex items-center space-x-2 flex-shrink-0">
              <button
                type="button"
                on:click={() => { $activeView = 'tasks'; }}
                class="px-2.5 py-1 rounded-xl bg-[#1c1c28] hover:bg-[#252538] text-slate-300 hover:text-white text-[11px] font-bold border border-[#2e2e42] transition flex items-center space-x-1 cursor-pointer"
                title="Open full Tasks view (2)"
              >
                <span>View Tasks</span>
                <span class="text-slate-500 font-mono text-[9px] px-1 py-0.2 rounded bg-black/40">2</span>
              </button>
              <button
                type="button"
                on:click={() => tasksStore.cancel(activeGameTask.id)}
                class="px-2.5 py-1 rounded-xl bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 hover:text-white text-[11px] font-bold border border-rose-900/40 hover:border-rose-700/60 transition flex items-center space-x-1 cursor-pointer active:scale-95"
                title="Cancel active task"
              >
                Cancel
              </button>
            </div>
          </div>

          <!-- Progress Bar -->
          <div class="space-y-1">
            <div class="w-full bg-[#0a0a0f] border border-[#1e1e2c] h-2 rounded-full overflow-hidden">
              <div
                class="bg-gradient-to-r from-cyan-500 to-blue-500 h-full rounded-full transition-all duration-300 relative"
                style="width: {activeGameTask.percent || 0}%"
              >
                <div class="absolute inset-0 bg-white/20 animate-pulse"></div>
              </div>
            </div>
            <div class="flex items-center justify-between text-[10px] font-mono text-slate-400">
              <span>
                {#if activeGameTask.total_bytes && activeGameTask.total_bytes > 0}
                  {formatBytes(activeGameTask.bytes_downloaded || activeGameTask.current_bytes || 0)} / {formatBytes(activeGameTask.total_bytes)} ({activeGameTask.percent?.toFixed(0) || 0}%)
                {:else if activeGameTask.total_items && activeGameTask.total_items > 0}
                  Step {activeGameTask.current_items || 0} of {activeGameTask.total_items} ({activeGameTask.percent?.toFixed(0) || 0}%)
                {:else}
                  {activeGameTask.percent?.toFixed(0) || 0}%
                {/if}
              </span>
              <div class="flex items-center space-x-3">
                {#if activeGameTask.speed_bps > 0}
                  <span class="text-cyan-400 font-bold">{formatBytes(activeGameTask.speed_bps)}/s</span>
                {/if}
                {#if activeGameTask.eta_seconds > 0}
                  <span>ETA: {formatETA(activeGameTask.eta_seconds)}</span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {/if}

      <!-- 6 Primary Navigation Tabs -->
      <div class="flex items-center space-x-2 border-b border-[#20202c] pb-2 text-xs font-bold select-none overflow-x-auto custom-scrollbar">
        <!-- 1. Overview Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'overview'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'overview' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Overview</span>
        </button>

        <!-- 2. Community Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'community'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'community' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
          </svg>
          <span>Community</span>
        </button>

        <!-- 3. General Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'general'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'general' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          <span>General</span>
        </button>

        <!-- 4. Environment Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'environment'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'environment' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          <span>Environment</span>
        </button>

        <!-- 5. Saves Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'saves'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'saves' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
          </svg>
          <span>Saves</span>
        </button>

        <!-- 6. Files & Diagnostics Tab -->
        <button
          type="button"
          on:click={() => { activeTab = 'files'; }}
          class="px-4 py-2 rounded-xl transition-all flex items-center space-x-2 flex-shrink-0 cursor-pointer {activeTab === 'files' ? 'bg-[#1e1e2c] text-white shadow-md border border-[#303046]' : 'text-slate-400 hover:text-slate-200'}"
        >
          <svg class="w-4 h-4 opacity-80 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
          <span>Files & Diagnostics</span>
          {#if isGameRunning}
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" title="Execution log active"></span>
          {/if}
        </button>
      </div>

      <!-- Tab Content Area -->
      <div class="flex-1 pb-12">
        <!-- ================= 1. OVERVIEW TAB ================= -->
        {#if activeTab === 'overview'}
          <div class="space-y-6">
            <!-- Game Summary Synopsis & Facts -->
            <DrawerAboutTab {game} />

            <!-- Categorized Technical & Community Resources Grid -->
            <DrawerResourcesTab {game} />
          </div>

        <!-- ================= 2. COMMUNITY TAB ================= -->
        {:else if activeTab === 'community'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl space-y-6">
            <!-- Community Sub-Pills -->
            <div class="flex items-center space-x-2 bg-[#0c0c10] p-1.5 rounded-xl border border-[#20202e] w-fit">
              <button
                type="button"
                on:click={() => { activeCommunitySubTab = 'achievements'; }}
                class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeCommunitySubTab === 'achievements' ? 'bg-[#202030] text-amber-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
                <span>Achievements</span>
              </button>

              {#if isSteam}
                <button
                  type="button"
                  on:click={() => { activeCommunitySubTab = 'news'; }}
                  class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-2 cursor-pointer {activeCommunitySubTab === 'news' ? 'bg-[#202030] text-sky-300 shadow-sm border border-[#34344a]' : 'text-slate-400 hover:text-slate-200'}"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
                  </svg>
                  <span>News & Updates</span>
                  {#if newsItems.length > 0}
                    <span class="px-1.5 py-0.2 rounded-full bg-cyan-500/20 text-cyan-300 text-[10px] font-mono">
                      {newsItems.length}
                    </span>
                  {/if}
                </button>
              {/if}
            </div>

            {#if activeCommunitySubTab === 'achievements'}
              <DrawerAchievementsTab gameDir={game.gameDir} />
            {:else if activeCommunitySubTab === 'news'}
              <DrawerNewsTab
                {isLoadingNews}
                {newsError}
                {newsItems}
                {expandedNewsIndex}
                onRefreshNews={() => loadNews(true)}
                onToggleExpand={toggleNewsExpand}
              />
            {/if}
          </div>

        <!-- ================= 3. GENERAL TAB ================= -->
        {:else if activeTab === 'general'}
          <div class="space-y-4">
            <!-- Action Toolbar -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 bg-[#121218] border border-[#222232] rounded-xl p-2.5 shadow-xl">
              <div class="flex items-center space-x-2 px-2">
                <span class="text-xs font-bold text-slate-200">General Configuration</span>
              </div>

              <!-- Status Badge & Save Actions -->
              <div class="flex items-center space-x-2 flex-shrink-0">
                {#if overrideCount > 0}
                  <span class="px-2 py-0.5 rounded-lg text-xs font-mono font-bold bg-sky-500/15 text-sky-300 border border-sky-500/30 flex items-center space-x-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-sky-400"></span>
                    <span>{overrideCount} Override{overrideCount === 1 ? '' : 's'}</span>
                  </span>
                {:else}
                  <span class="px-2 py-0.5 rounded-lg text-xs font-mono text-slate-400 bg-[#0c0c10] border border-[#1e1e2c]">
                    Inheriting Defaults
                  </span>
                {/if}

                {#if saveSuccess}
                  <span class="text-xs font-mono font-bold text-emerald-400 flex items-center space-x-1 animate-fade-in">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Saved</span>
                  </span>
                {/if}

                <button
                  type="button"
                  on:click={handleResetSettings}
                  class="p-1.5 bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-400 hover:text-white rounded-lg text-xs transition-colors cursor-pointer"
                  title="Reset all settings to system defaults"
                  aria-label="Reset all settings to system defaults"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </button>

                <button
                  type="button"
                  on:click={handleSaveSettings}
                  disabled={isSaving}
                  class="px-3.5 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs shadow-md transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
                >
                  {#if isSaving}
                    <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                    </svg>
                    <span>Saving...</span>
                  {:else}
                    <span>Save</span>
                  {/if}
                </button>
              </div>
            </div>

            <!-- Tab Content -->
            <div class="bg-[#121218] border border-[#222232] rounded-xl p-4 sm:p-5 shadow-xl space-y-4">
              <div class="space-y-3">
                <SettingSection title="Game Identification">
                  <SettingRow label="Title Override" forId="draftGameName" inline={false}>
                    <input
                      id="draftGameName"
                      type="text"
                      bind:value={$drawerDraft.name}
                      placeholder={game?.name || "Game Title"}
                      class="w-full bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-lg px-3 py-2 text-slate-100 outline-none text-xs transition-colors"
                    />
                  </SettingRow>
                </SettingSection>

                <SettingSection title="Identity, Source & Artwork">
                  {@const metaInfo = getMetadataInfo(game)}
                  {@const hasMatch = hasMetadataMatch(game)}

                  <!-- 3 Pillar Action Cards -->
                  <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                    <!-- 1. Physical Source Pillar -->
                    <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 {showSourceWizard ? 'border-cyan-500/50 bg-[#12121c]' : ''}">
                      <div class="space-y-1">
                        <div class="flex items-center justify-between">
                          <span class="text-[10px] uppercase font-mono font-bold tracking-wider text-slate-400 flex items-center gap-1.5">
                            <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7" />
                            </svg>
                            Physical Source
                          </span>
                          <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-slate-800 text-slate-200 border border-slate-700">
                            {$drawerDraft.sourceOverride || game.source || game.store || 'Standalone'}
                          </span>
                        </div>
                        <p class="text-[11px] text-slate-400 leading-snug">
                          File layout, DRM emulation, and runner execution pipeline.
                        </p>
                      </div>
                      <button
                        type="button"
                        on:click={() => { showSourceWizard = !showSourceWizard; if (showSourceWizard) showMetadataWizard = false; }}
                        class="w-full py-1.5 px-3 rounded-lg bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
                      >
                        <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                        <span>{showSourceWizard ? 'Close Source Setup' : 'Configure Source...'}</span>
                      </button>
                    </div>

                    <!-- 2. Store Metadata Pillar -->
                    <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col justify-between space-y-3 {showMetadataWizard ? 'border-purple-500/50 bg-[#12121c]' : ''}">
                      <div class="space-y-1">
                        <div class="flex items-center justify-between">
                          <span class="text-[10px] uppercase font-mono font-bold tracking-wider text-slate-400 flex items-center gap-1.5">
                            <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                            </svg>
                            Store Metadata
                          </span>
                          {#if hasMatch}
                            <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-sky-500/15 text-sky-300 border border-sky-500/30">
                              {metaInfo.provider} {metaInfo.id ? '#' + metaInfo.id : ''}
                            </span>
                          {:else}
                            <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-amber-500/15 text-amber-300 border border-amber-500/30">
                              Unlinked
                            </span>
                          {/if}
                        </div>
                        <p class="text-[11px] text-slate-400 leading-snug">
                          Game synopsis, release date, developer credits, and patch notes.
                        </p>
                      </div>
                      <button
                        type="button"
                        on:click={() => { showMetadataWizard = !showMetadataWizard; if (showMetadataWizard) showSourceWizard = false; }}
                        class="w-full py-1.5 px-3 rounded-lg bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-200 hover:text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer"
                      >
                        <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                        </svg>
                        <span>{showMetadataWizard ? 'Close Matcher' : (hasMatch ? 'Change Match...' : 'Match Metadata...')}</span>
                      </button>
                    </div>

                    <!-- 3. Artwork Hub Pillar -->
                    <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3.5 rounded-xl flex flex-col justify-between space-y-3">
                      <div class="space-y-1">
                        <div class="flex items-center justify-between">
                          <span class="text-[10px] uppercase font-mono font-bold tracking-wider text-slate-400 flex items-center gap-1.5">
                            <svg class="w-3.5 h-3.5 text-pink-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                            Artwork Hub
                          </span>
                          <span class="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-pink-500/15 text-pink-300 border border-pink-500/30">
                            {game.lockedArtwork ? 'Custom' : 'Active'}
                          </span>
                        </div>
                        <p class="text-[11px] text-slate-400 leading-snug">
                          Covers (600×900), Hero banners, Grid banners, and Logos.
                        </p>
                      </div>
                      <button
                        type="button"
                        on:click={() => modalStore.open('artwork-picker', { game })}
                        class="w-full py-1.5 px-3 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-bold transition shadow flex items-center justify-center gap-1.5 cursor-pointer"
                      >
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        </svg>
                        <span>Open Artwork Hub...</span>
                      </button>
                    </div>
                  </div>

                  <!-- ============================================================ -->
                  <!-- INLINE POP-OUT: PHYSICAL SOURCE WIZARD -->
                  <!-- ============================================================ -->
                  {#if showSourceWizard}
                    <div class="bg-[#12121a] border border-cyan-500/40 rounded-xl p-4 space-y-3.5 animate-in fade-in duration-150 mt-2">
                      <div class="flex items-center justify-between border-b border-[#222232] pb-2">
                        <div class="flex items-center space-x-2">
                          <span class="text-xs font-bold text-white uppercase tracking-wider font-mono">Configure Physical Installation Source</span>
                          <span class="text-[10px] text-slate-400 font-mono">(Overrides detected source)</span>
                        </div>
                        <button type="button" on:click={() => showSourceWizard = false} class="text-slate-400 hover:text-white text-xs font-mono cursor-pointer">Close</button>
                      </div>

                      <div class="grid grid-cols-2 sm:grid-cols-5 gap-2">
                        {#each [
                          { id: '', label: 'Auto-Detect', desc: 'Detect from files' },
                          { id: 'Steam', label: 'Steam', desc: 'Steamworks + Goldberg' },
                          { id: 'GOG', label: 'GOG', desc: 'GOG game layout' },
                          { id: 'Epic', label: 'Epic', desc: 'Epic Games manifest' },
                          { id: 'Standalone', label: 'Standalone', desc: 'Direct DRM-free' }
                        ] as opt}
                          {@const isSel = ($drawerDraft.sourceOverride || '') === opt.id}
                          <button
                            type="button"
                            on:click={() => drawerDraft.setField('sourceOverride', opt.id)}
                            class="p-2.5 rounded-lg border text-left transition flex flex-col justify-between gap-1 cursor-pointer {isSel ? 'bg-cyan-950/40 border-cyan-400 text-white shadow-sm' : 'bg-[#0a0a0f] border-[#222232] text-slate-400 hover:text-slate-200 hover:border-slate-600'}"
                          >
                            <span class="font-bold text-xs">{opt.label}</span>
                            <span class="text-[10px] text-slate-500 font-mono">{opt.desc}</span>
                          </button>
                        {/each}
                      </div>

                      <div class="p-3 bg-[#0a0a0f] border border-[#1e1e2c] rounded-lg text-xs text-slate-400 space-y-1">
                        <div class="font-bold text-slate-300">
                          Current Source: <span class="text-cyan-400 font-mono font-bold">{$drawerDraft.sourceOverride || game.source || game.store || 'Standalone'}</span>
                          {#if !$drawerDraft.sourceOverride}
                            <span class="text-[10px] text-slate-500 font-normal ml-1">(Auto-detected)</span>
                          {/if}
                        </div>
                        <p class="text-[11px] text-slate-400">
                          {$drawerDraft.sourceOverride === 'Steam' 
                            ? 'Steam mode enables Goldberg emulator injection and Steamworks API mocking.' 
                            : $drawerDraft.sourceOverride === 'GOG' 
                            ? 'GOG mode enables GOG registry & offline info detection.'
                            : $drawerDraft.sourceOverride === 'Standalone'
                            ? 'Standalone mode executes binary directly via Proton/Wine without injecting Steamworks emulators.'
                            : 'Auto-detect inspects binaries, steam_api DLLs, and manifests to configure runtime hooks.'}
                        </p>
                      </div>
                    </div>
                  {/if}

                  <!-- ============================================================ -->
                  <!-- INLINE POP-OUT: METADATA MATCH WIZARD -->
                  <!-- ============================================================ -->
                  {#if showMetadataWizard}
                    <div class="bg-[#12121a] border border-purple-500/40 rounded-xl p-4 space-y-3.5 animate-in fade-in duration-150 mt-2">
                      <div class="flex items-center justify-between border-b border-[#222232] pb-2">
                        <div class="flex items-center space-x-2">
                          <span class="text-xs font-bold text-white uppercase tracking-wider font-mono">Match Online Store Metadata</span>
                          <span class="text-[10px] text-slate-400 font-mono">(Does not modify physical game files)</span>
                        </div>
                        <div class="flex items-center space-x-2">
                          {#if hasMatch}
                            <button
                              type="button"
                              on:click={handleUnlinkMeta}
                              disabled={isLinkingMeta}
                              class="text-rose-400 hover:text-rose-300 text-xs font-mono px-2 py-0.5 rounded bg-rose-500/10 border border-rose-500/20 cursor-pointer"
                            >
                              Reset to Auto
                            </button>
                          {/if}
                          <button type="button" on:click={() => showMetadataWizard = false} class="text-slate-400 hover:text-white text-xs font-mono cursor-pointer">Close</button>
                        </div>
                      </div>

                      <!-- Provider Selection Pills -->
                      <div class="flex items-center gap-2 overflow-x-auto pb-1">
                        {#each metadataProviders as prov}
                          {@const isAct = metaTargetStore === prov.id}
                          <button
                            type="button"
                            on:click={() => { metaTargetStore = prov.id; if (metaTargetStore !== 'none' && metaTargetStore !== 'standalone' && metaSearchQuery.trim()) handleMetaSearch(); }}
                            class="px-3 py-1 rounded-lg text-xs font-bold transition flex items-center space-x-1.5 cursor-pointer {isAct ? 'bg-purple-600 text-white shadow border border-purple-400/40' : 'bg-[#0a0a0f] border border-[#222232] text-slate-400 hover:text-white'}"
                          >
                            <span>{prov.display_name}</span>
                            <span class="text-[9px] bg-black/30 px-1 rounded font-mono">{prov.badge}</span>
                          </button>
                        {/each}
                      </div>

                      {#if metaTargetStore === 'none' || metaTargetStore === 'standalone'}
                        <div class="p-3 bg-[#0a0a0f] border border-[#1e1e2c] rounded-lg text-xs text-slate-300 flex items-center justify-between">
                          <p class="text-[11px] text-slate-400">Unlinks external store metadata. Game will display local library title and fallback artwork.</p>
                          <button
                            type="button"
                            on:click={() => handleApplyMetaLink('0')}
                            disabled={isLinkingMeta}
                            class="px-4 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs rounded-lg transition cursor-pointer"
                          >
                            {isLinkingMeta ? 'Unlinking...' : 'Set None (Unlinked)'}
                          </button>
                        </div>
                      {:else}
                        <!-- Live Search Bar -->
                        <div class="flex flex-col sm:flex-row items-center gap-2">
                          <div class="relative flex-1 w-full">
                            <input
                              type="text"
                              bind:value={metaSearchQuery}
                              placeholder={`Search ${metaTargetStore.toUpperCase()} catalogue (e.g. ${game.name})...`}
                              on:keydown={(e) => { if (e.key === 'Enter') handleMetaSearch(); }}
                              class="w-full bg-[#0a0a0f] border border-[#222232] focus:border-purple-500 rounded-lg px-3.5 py-1.5 text-xs text-white outline-none font-mono placeholder:text-slate-500"
                            />
                          </div>
                          <button
                            type="button"
                            on:click={handleMetaSearch}
                            disabled={isSearchingMeta || !metaSearchQuery.trim()}
                            class="w-full sm:w-auto px-4 py-1.5 bg-[#1c1c28] hover:bg-[#28283a] text-white text-xs font-bold rounded-lg border border-[#2f2f42] transition flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50"
                          >
                            {#if isSearchingMeta}
                              <div class="w-3 h-3 border-2 border-purple-400 border-t-transparent rounded-full animate-spin"></div>
                              <span>Searching...</span>
                            {:else}
                              <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                              </svg>
                              <span>Search</span>
                            {/if}
                          </button>
                        </div>

                        <!-- Candidate Results List -->
                        {#if metaSearchResults.length > 0}
                          <div class="space-y-1.5 max-h-60 overflow-y-auto pr-1">
                            <span class="text-[10px] font-bold uppercase tracking-wider text-slate-400 block">
                              Found Candidates ({metaSearchResults.length})
                            </span>
                            <div class="grid grid-cols-1 gap-1.5">
                              {#each metaSearchResults as cand}
                                <div class="flex items-center justify-between p-2.5 rounded-lg bg-[#0a0a0f] border border-[#1e1e2c] hover:border-purple-500/50 transition">
                                  <div class="flex items-center space-x-2.5 min-w-0">
                                    {#if cand.poster_url}
                                      <img src={cand.poster_url} alt="" class="w-8 h-11 object-cover rounded bg-black/40 flex-shrink-0" />
                                    {/if}
                                    <div class="min-w-0">
                                      <div class="text-xs font-bold text-white truncate">{cand.name}</div>
                                      <div class="flex items-center space-x-2 text-[10px] font-mono text-slate-400">
                                        <span class="text-purple-300">ID: {cand.id}</span>
                                        {#if cand.release_date}
                                          <span>• {cand.release_date}</span>
                                        {/if}
                                      </div>
                                    </div>
                                  </div>
                                  <button
                                    type="button"
                                    on:click={() => handleApplyMetaLink(cand.id)}
                                    disabled={isLinkingMeta}
                                    class="px-3 py-1 bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold rounded-lg transition flex-shrink-0 cursor-pointer"
                                  >
                                    Link Metadata
                                  </button>
                                </div>
                              {/each}
                            </div>
                          </div>
                        {/if}

                        <!-- Manual ID Fallback Row -->
                        <div class="flex items-center gap-2 pt-2 border-t border-[#1e1e2c]">
                          <span class="text-[10px] font-bold text-slate-400 font-mono flex-shrink-0">
                            {metaTargetStore === 'steam' ? 'Steam AppID:' : metaTargetStore === 'gog' ? 'GOG ID / Slug:' : 'RAWG ID / Slug:'}
                          </span>
                          <input
                            type="text"
                            bind:value={metaCustomId}
                            placeholder={metaTargetStore === 'steam' ? 'e.g. 1091500' : metaTargetStore === 'gog' ? 'e.g. the_witcher_3' : 'e.g. portal-2'}
                            class="flex-1 bg-[#0a0a0f] border border-[#222232] focus:border-purple-500 rounded-lg px-2.5 py-1 text-xs text-white outline-none font-mono placeholder:text-slate-600"
                          />
                          <button
                            type="button"
                            on:click={() => handleApplyMetaLink()}
                            disabled={!metaCustomId.trim() || isLinkingMeta}
                            class="px-3.5 py-1 bg-purple-600 hover:bg-purple-500 disabled:opacity-40 text-white text-xs font-bold rounded-lg transition flex-shrink-0 cursor-pointer"
                          >
                            Link ID
                          </button>
                        </div>
                      {/if}
                    </div>
                  {/if}
                </SettingSection>

                <SettingSection title="Target Executable">
                  <!-- Detected Candidate Executables -->
                  {#if candidates.length > 0}
                    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 pb-2">
                      {#each candidates as cand, idx}
                        {@const isSelected = $drawerDraft.exePath === cand.executable}
                        <button
                          type="button"
                          on:click={() => drawerDraft.selectCandidate(cand)}
                          class="p-2.5 rounded-lg border text-left transition-all flex items-start justify-between gap-2 cursor-pointer {isSelected ? 'bg-sky-950/40 border-sky-500/70 shadow-sm shadow-sky-950/30' : 'bg-[#0d0d14] border-[#1e1e2c] hover:border-[#2a2a3e] text-slate-300'}"
                        >
                          <div class="space-y-0.5 min-w-0 flex-1">
                            <div class="flex items-center space-x-1.5 flex-wrap">
                              <span class="font-bold text-xs {isSelected ? 'text-white' : 'text-slate-200'} truncate">
                                {cand.name || cand.executable.split('/').pop()}
                              </span>
                              {#if idx === 0}
                                <span class="px-1.5 py-0.2 rounded text-[9px] font-bold bg-sky-500/20 text-sky-300 border border-sky-500/40">
                                  Primary
                                </span>
                              {/if}
                            </div>
                            <p class="font-mono text-[10px] text-slate-400 truncate">
                              {cand.executable}
                            </p>
                          </div>

                          <div class="w-3.5 h-3.5 rounded-full border flex items-center justify-center flex-shrink-0 mt-0.5 {isSelected ? 'border-sky-400 bg-sky-500' : 'border-[#303044]'}" aria-hidden="true">
                            {#if isSelected}
                              <div class="w-1.5 h-1.5 rounded-full bg-black"></div>
                            {/if}
                          </div>
                        </button>
                      {/each}
                    </div>
                  {/if}

                  <div class="flex items-center space-x-2 pt-1 pb-3">
                    <input
                      type="text"
                      bind:value={$drawerDraft.exePath}
                      placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
                      class="flex-1 bg-[#14141e] border border-[#262638] focus:border-sky-500 rounded-lg px-3 py-2 text-slate-100 font-mono text-xs outline-none transition-colors"
                    />
                    <button
                      type="button"
                      on:click={async () => { await drawerDraft.browseExecutable(game?.gameDir); }}
                      class="bg-[#1c1c28] hover:bg-[#262638] text-slate-200 px-3.5 py-2 rounded-lg font-bold border border-[#323246] text-xs flex-shrink-0 cursor-pointer"
                    >
                      Browse
                    </button>
                  </div>

                  <div class="pt-3 border-t border-[#1a1a26]">
                    <GhostInput
                      label="Launch Arguments"
                      description="Command-line flags passed directly to game executable"
                      bind:value={$drawerDraft.exeArgs}
                      globalPlaceholder="e.g. -dx11 -fullscreen -novid"
                      mono={true}
                    />
                  </div>
                </SettingSection>
              </div>
            </div>
          </div>

        <!-- ================= 4. ENVIRONMENT TAB ================= -->
        {:else if activeTab === 'environment'}
          <div class="space-y-4">
            <!-- Action Toolbar -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 bg-[#121218] border border-[#222232] rounded-xl p-2.5 shadow-xl">
              <div class="flex items-center space-x-2 px-2">
                <span class="text-xs font-bold text-slate-200">Runtime & Environment</span>
              </div>

              <!-- Status Badge & Save Actions -->
              <div class="flex items-center space-x-2 flex-shrink-0">
                {#if overrideCount > 0}
                  <span class="px-2 py-0.5 rounded-lg text-xs font-mono font-bold bg-sky-500/15 text-sky-300 border border-sky-500/30 flex items-center space-x-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-sky-400"></span>
                    <span>{overrideCount} Override{overrideCount === 1 ? '' : 's'}</span>
                  </span>
                {:else}
                  <span class="px-2 py-0.5 rounded-lg text-xs font-mono text-slate-400 bg-[#0c0c10] border border-[#1e1e2c]">
                    Inheriting Defaults
                  </span>
                {/if}

                {#if saveSuccess}
                  <span class="text-xs font-mono font-bold text-emerald-400 flex items-center space-x-1 animate-fade-in">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Saved</span>
                  </span>
                {/if}

                <button
                  type="button"
                  on:click={handleResetSettings}
                  class="p-1.5 bg-[#181824] hover:bg-[#222232] border border-[#2c2c3e] text-slate-400 hover:text-white rounded-lg text-xs transition-colors cursor-pointer"
                  title="Reset all settings to system defaults"
                  aria-label="Reset all settings to system defaults"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </button>

                <button
                  type="button"
                  on:click={handleSaveSettings}
                  disabled={isSaving}
                  class="px-3.5 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs shadow-md transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
                >
                  {#if isSaving}
                    <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                    </svg>
                    <span>Saving...</span>
                  {:else}
                    <span>Save</span>
                  {/if}
                </button>
              </div>
            </div>

            <!-- Sub-Tab Content -->
            <div class="bg-[#121218] border border-[#222232] rounded-xl p-4 sm:p-5 shadow-xl space-y-4">
              <div class="space-y-3">
                <SettingSection title="Runner & Prefix">
                  <div class="space-y-2.5">
                    <InheritSelect
                      label="Runner Layer"
                      bind:value={$drawerDraft.runner}
                      globalValue={globalCfg?.default_runner || 'umu'}
                      options={runnerOptions}
                    />

                    <GhostInput
                      label="Wine Prefix"
                      description="Custom isolated Wine bottle location"
                      bind:value={$drawerDraft.winePrefix}
                      globalPlaceholder={globalCfg?.wine_prefix || "Isolated: steam_settings/pfx"}
                      mono={true}
                    />

                    <GhostInput
                      label="Proton Path"
                      description="Direct directory path to custom Proton distribution"
                      bind:value={$drawerDraft.protonPath}
                      globalPlaceholder={globalCfg?.proton_path || "System Proton default"}
                      mono={true}
                    />
                  </div>
                </SettingSection>

                <SettingSection title="Graphics & Overlays">
                  <div class="space-y-2.5">
                    <SeamlessSwitch
                      label="MangoHud Overlay"
                      description="Display real-time FPS, frame timing, and GPU/CPU telemetry HUD"
                      bind:value={$drawerDraft.mangoHud}
                      inheritedValue={Boolean(globalCfg?.mangohud)}
                    />
                  </div>
                </SettingSection>

                <SettingSection title="Steam Emulation">
                  <div class="space-y-2.5">
                    <InheritSelect
                      label="Operating Mode"
                      bind:value={$drawerDraft.operatingMode}
                      globalValue={globalCfg?.operating_mode || 'loader'}
                      options={operatingModeOptions}
                    />

                    <InheritSelect
                      label="Game Language"
                      bind:value={$drawerDraft.language}
                      globalValue={globalCfg?.language || 'auto'}
                      options={gameLanguageOptions}
                    />

                    <GhostInput
                      label="DLL Overrides"
                      description="Custom native/builtin DLL loading rules (WINEDLLOVERRIDES)"
                      bind:value={$drawerDraft.dllOverrides}
                      globalPlaceholder="e.g. version=n,b;dinput8=n,b;dxgi=n,b"
                      mono={true}
                    />
                  </div>
                </SettingSection>

                <SettingSection title="Environment Variables">
                  <EnvDiffEditor
                    globalEnv={globalCfg?.global_env || {}}
                    bind:customEnv={$drawerDraft.customEnv}
                  />
                </SettingSection>
              </div>
            </div>
          </div>

        <!-- ================= 5. SAVES TAB ================= -->
        {:else if activeTab === 'saves'}
          <div class="bg-[#121218] border border-[#222232] rounded-2xl p-6 shadow-xl">
            <DrawerSavesTab {game} />
          </div>

        <!-- ================= 6. FILES & DIAGNOSTICS TAB ================= -->
        {:else if activeTab === 'files'}
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
                onRunIntegrityCheck={runIntegrityCheck}
                onUpdateGame={handleUpdateGame}
                onNormalizeDirectory={handleNormalizeDirectory}
              />
            {:else if activeFilesSubTab === 'logs'}
              <DrawerLogsTab {game} />
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
