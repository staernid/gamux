<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../api/app.js';
  import { isSteamApp } from '../../../utils/game.js';
  import { formatDate } from '../../../utils/format.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import { updateGameInStore, selectedGame } from '../../../stores/library.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   * @property {boolean} [isTaskActive]
   * @property {any} [activeGameTask]
   * @property {any} [onbranchChange]
   * @property {any} [onupdateGame]
   */

  /** @type {Props} */
  let {
    game = $bindable(null),
    isTaskActive = false,
    activeGameTask = null,
    onbranchChange = undefined,
    onupdateGame = undefined
  } = $props();

  let branches = $state([]);
  let isLoading = $state(false);
  let error = $state('');
  let isSwitching = $state(false);
  let switchingBranch = $state('');
  let searchQuery = $state('');

  // Private beta state
  let privateBetaCode = $state('');
  let isCheckingCode = $state(false);
  let codeFeedback = $state(null); // { type: 'success' | 'error', message: string }









  let loadedAppId = $state('');


  async function loadBranches() {
    if (!effectiveAppId) return;
    isLoading = true;
    error = '';
    codeFeedback = null;
    try {
      const appIDNum = parseInt(effectiveAppId, 10);
      const res = await AppAPI.getSteamBranches(appIDNum);
      branches = res || [];
    } catch (err) {
      error = err?.message || 'Failed to fetch Steam branches.';
      branches = [];
    } finally {
      isLoading = false;
    }
  }

  function dispatchBranchChange(detail) {
    onbranchChange?.(detail);
  }

  async function handleSelectBranch(branch) {
    if (branch.name === currentBranch) return;

    // Check if password required and not entered
    let password = '';
    if (branch.pwd_required) {
      if (!privateBetaCode.trim()) {
        codeFeedback = {
          type: 'error',
          message: `Branch "${branch.name}" is password-protected. Please enter the beta access code below.`
        };
        return;
      }
      password = privateBetaCode.trim();
    }

    isSwitching = true;
    switchingBranch = branch.name;
    codeFeedback = null;

    try {
      if (isVirtual) {
        // Virtual title: save in local store and dispatch
        selectedGame.update(curr => curr ? { ...curr, branch: branch.name, branchPassword: password } : curr);
        if (game) {
          game.branch = branch.name;
          game.branchPassword = password;
        }
        dispatchBranchChange({ branch: branch.name, password, targetBuildId: branch.build_id });
        notificationStore.items.add({
          type: 'info',
          title: 'Target Branch Selected',
          message: `Selected branch "${branch.name}" for ${game.name}. This will be applied upon adding to library or downloading.`
        });
      } else {
        // Stub or Installed game: update launch profile and ACF via Go engine
        const updated = await AppAPI.setGameBranch(game.gameDir, branch.name, password);
        if (updated) {
          updateGameInStore(updated);
          selectedGame.update(curr => curr && curr.gameDir === updated.game_dir ? { ...curr, ...updated, branch: branch.name } : curr);
        }
        dispatchBranchChange({ branch: branch.name, password, targetBuildId: branch.build_id });

        if (isStub) {
          notificationStore.items.add({
            type: 'success',
            title: 'Branch Saved to Stub',
            message: `Profile updated for ${game.name}. Download will target branch "${branch.name}" (Build #${branch.build_id || 'latest'}).`
          });
        } else {
          notificationStore.items.add({
            type: 'success',
            title: 'Branch Switched',
            message: `Active branch switched to "${branch.name}".`
          });
        }
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Branch Switch Failed',
        message: err?.message || String(err)
      });
    } finally {
      isSwitching = false;
      switchingBranch = '';
    }
  }

  async function handleCheckPrivateCode() {
    if (!privateBetaCode.trim()) {
      codeFeedback = {
        type: 'error',
        message: 'Please enter a beta access code.'
      };
      return;
    }

    isCheckingCode = true;
    codeFeedback = null;

    // Look for branches that require password
    const pwdBranches = branches.filter(b => b.pwd_required);
    if (pwdBranches.length === 0 && branches.length > 0) {
      codeFeedback = {
        type: 'error',
        message: 'No password-protected private beta branches were reported by Steam for this title.'
      };
      isCheckingCode = false;
      return;
    }

    // Attempt to unlock / switch to the first password-protected branch or active branch
    const targetBranch = pwdBranches.find(b => b.name === currentBranch) || pwdBranches[0];

    try {
      if (game?.gameDir && !isVirtual) {
        const updated = await AppAPI.setGameBranch(game.gameDir, targetBranch.name, privateBetaCode.trim());
        if (updated) {
          updateGameInStore(updated);
        }
      } else if (game) {
        game.branch = targetBranch.name;
        game.branchPassword = privateBetaCode.trim();
        selectedGame.update(curr => curr ? { ...curr, branch: targetBranch.name, branchPassword: privateBetaCode.trim() } : curr);
      }

      codeFeedback = {
        type: 'success',
        message: `Beta code accepted. Selected branch "${targetBranch.name}".`
      };
      dispatchBranchChange({ branch: targetBranch.name, password: privateBetaCode.trim(), targetBuildId: targetBranch.build_id });
    } catch (err) {
      codeFeedback = {
        type: 'error',
        message: err?.message || 'Invalid beta code or failed to unlock branch.'
      };
    } finally {
      isCheckingCode = false;
    }
  }

  function handleTriggerUpdate() {
    const detail = { branch: currentBranch };
    onupdateGame?.(detail);
  }
  let effectiveAppId = $derived((game?.appId && isSteamApp(game.appId))
    ? String(game.appId).trim()
    : ((game?.metadataProvider || '').toLowerCase() === 'steam' && isSteamApp(game?.metadataId) ? String(game.metadataId).trim() : ''));
  let isSteam = $derived(Boolean(effectiveAppId));
  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);
  let isStub = $derived(game ? (game.isStub ?? (Boolean(game.gameDir) && !isInstalled)) : false);
  let isVirtual = $derived(game ? (game.isVirtual ?? !game.gameDir) : false);
  let currentBranch = $derived(game?.branch || 'public');
  let activeBranchObj = $derived(branches.find(b => b.name === currentBranch) || null);
  let targetBuildId = $derived(activeBranchObj?.build_id || '');
  let installedBuildId = $derived(game?.buildId || '');
  let cleanInstalledBuild = $derived((installedBuildId && installedBuildId !== '0' && installedBuildId !== 'unknown') ? String(installedBuildId).trim() : '');
  let isValidInstalledBuild = $derived(Boolean(cleanInstalledBuild));
  let isBuildMismatch = $derived(isInstalled && isValidInstalledBuild && Boolean(targetBuildId && targetBuildId !== '0' && targetBuildId !== cleanInstalledBuild));
  let sortedBranches = $derived([...branches].sort((a, b) => {
    if (a.name === currentBranch && b.name !== currentBranch) return -1;
    if (b.name === currentBranch && a.name !== currentBranch) return 1;
    if (a.name === 'public') return -1;
    if (b.name === 'public') return 1;
    return a.name.localeCompare(b.name);
  }));
  let filteredBranches = $derived(sortedBranches.filter(b => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return b.name.toLowerCase().includes(q) || (b.description && b.description.toLowerCase().includes(q)) || (b.build_id && b.build_id.includes(q));
  }));
  $effect(() => {
    if (effectiveAppId && effectiveAppId !== loadedAppId) {
      loadedAppId = effectiveAppId;
      loadBranches();
    }
  });
</script>

<div class="space-y-5">
  {#if !isSteam}
    <!-- Non-Steam Warning -->
    <div class="bg-surface-card border border-surface-border rounded-2xl p-8 text-center space-y-4 max-w-md mx-auto">
      <div class="w-12 h-12 rounded-2xl bg-surface-header border border-surface-border text-neutral-400 flex items-center justify-center mx-auto">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <div class="space-y-1">
        <h3 class="font-bold text-white text-base">Steam AppID Required</h3>
        <p class="text-xs text-neutral-400 leading-relaxed">
          Betas and depot branches are a Steam feature. Link this title to a valid Steam AppID in the General tab to manage branches.
        </p>
      </div>
    </div>
  {:else}
    <!-- Context-Aware Status Banner -->
    <div class="bg-surface-card border border-surface-border rounded-2xl p-5 shadow-sm space-y-3">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-center space-x-3">
          <div class="w-9 h-9 rounded-xl bg-surface-header border border-surface-border text-neutral-300 flex items-center justify-center flex-shrink-0">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v10M8 7a3 3 0 100-6 3 3 0 000 6zm0 10a3 3 0 100 6 3 3 0 000-6zm8-6a3 3 0 100-6 3 3 0 000 6zm-8 4h3a5 5 0 005-5V7" />
            </svg>
          </div>
          <div>
            <div class="flex items-center space-x-2">
              <h2 class="text-sm font-bold text-white">Steam Betas & Branches</h2>
              <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold bg-surface-header border border-surface-border text-neutral-400">
                #{effectiveAppId}
              </span>
            </div>
            <p class="text-xs text-neutral-400 leading-relaxed mt-0.5">
              Select testing branches, Vulkan/DirectX variants, or legacy versions provided by the game developers.
            </p>
          </div>
        </div>

        <button
          type="button"
          onclick={loadBranches}
          disabled={isLoading}
          class="px-3 py-1.5 rounded-xl bg-surface-header hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white text-xs font-semibold transition-colors flex items-center space-x-1.5 self-start sm:self-auto cursor-pointer disabled:opacity-50"
          title="Refresh branch list from Steam"
        >
          <svg class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Refresh</span>
        </button>
      </div>

      <!-- Context Notice Badge -->
      <div class="pt-2 border-t border-surface-border flex flex-wrap items-center justify-between gap-2 text-xs">
        <div class="flex items-center space-x-2">
          {#if isVirtual}
            <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-neutral-800 text-neutral-300 border border-neutral-700">
              Virtual Store Title
            </span>
            <span class="text-neutral-400 text-xs">
              Selected branch will be saved when adding to library or downloading.
            </span>
          {:else if isStub}
            <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-neutral-800 text-neutral-300 border border-neutral-700">
              Library Stub
            </span>
            <span class="text-neutral-400 text-xs">
              Configured in launch profile. Downloads will target branch <strong class="text-neutral-200">{currentBranch}</strong>{#if targetBuildId} (Build #{targetBuildId}){/if}.
            </span>
          {:else}
            <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-surface-header text-neutral-300 border border-surface-border">
              Installed Game
            </span>
            <span class="text-neutral-400 text-xs">
              Configured in <code class="font-mono text-neutral-300">appmanifest_{effectiveAppId}.acf</code> & launch profile.
            </span>
          {/if}
        </div>

        <!-- Current Branch Badge -->
        <div class="flex items-center space-x-1.5 font-mono text-2xs">
          <span class="text-neutral-500">Active Branch:</span>
          <span class="px-2 py-0.5 rounded bg-surface-header border border-surface-border text-white font-bold">
            {currentBranch}
          </span>
          {#if targetBuildId}
            <span class="text-neutral-500 font-mono">#{targetBuildId}</span>
          {/if}
        </div>
      </div>

      <!-- Build Mismatch / Update Pending Banner for Installed Game -->
      {#if isBuildMismatch}
        <div class="p-3.5 rounded-xl bg-surface-header border border-surface-borderHighlight flex flex-col sm:flex-row sm:items-center justify-between gap-3 animate-fade-in">
          <div class="space-y-0.5">
            <div class="flex items-center space-x-2">
              <span class="w-2 h-2 rounded-full bg-amber-400"></span>
              <span class="text-xs font-bold text-neutral-100">Files Pending Branch Sync</span>
            </div>
            <p class="text-2xs text-neutral-400 font-mono">
              Installed files: #{cleanInstalledBuild} &rarr; Target branch files: #{targetBuildId}
            </p>
            {#if activeGameTask && (activeGameTask.status === 'downloading' || activeGameTask.status === 'running' || activeGameTask.status === 'paused')}
              <p class="text-3xs text-neutral-400 font-mono">
                Active task: {activeGameTask.status === 'paused' ? 'Paused - ' : ''}{activeGameTask.stage || activeGameTask.details || 'Syncing files...'}
              </p>
            {/if}
          </div>

          <button
            type="button"
            onclick={handleTriggerUpdate}
            disabled={isTaskActive}
            class="px-4 py-2 rounded-xl bg-white hover:bg-neutral-200 text-neutral-950 font-black text-xs transition-colors flex items-center space-x-1.5 self-start sm:self-auto cursor-pointer disabled:opacity-50"
          >
            {#if isTaskActive}
              {#if activeGameTask?.status === 'paused'}
                <span>Download Paused</span>
              {:else}
                <svg class="animate-spin w-3.5 h-3.5 text-neutral-950" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Download in Progress...</span>
              {/if}
            {:else}
              <svg class="w-3.5 h-3.5 text-neutral-950" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              <span>Switch Files to Branch</span>
            {/if}
          </button>
        </div>
      {/if}
    </div>

    <!-- Loading State -->
    {#if isLoading}
      <div class="bg-surface-card border border-surface-border rounded-2xl p-12 text-center space-y-3">
        <svg class="animate-spin w-6 h-6 text-neutral-400 mx-auto" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
        </svg>
        <p class="text-xs text-neutral-400 font-mono">Fetching depot branch metadata from Steam...</p>
      </div>
    {:else if error}
      <!-- Error State -->
      <div class="bg-surface-card border border-surface-border rounded-2xl p-6 space-y-3">
        <div class="flex items-center space-x-2 text-rose-400">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span class="text-xs font-bold">Failed to load branches</span>
        </div>
        <p class="text-xs text-neutral-400">{error}</p>
        <button
          type="button"
          onclick={loadBranches}
          class="px-3 py-1.5 rounded-lg bg-surface-header hover:bg-surface-cardHover border border-surface-border text-xs text-neutral-200 font-semibold cursor-pointer"
        >
          Try Again
        </button>
      </div>
    {:else}
      <!-- Branch Selection Section -->
      <div class="bg-surface-card border border-surface-border rounded-2xl p-5 shadow-sm space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div>
            <h3 class="text-xs font-bold text-neutral-200">Available Branches</h3>
            <p class="text-2xs text-neutral-400 mt-0.5">
              Select a branch below to opt into or out of beta builds.
            </p>
          </div>

          <!-- Search Filter if many branches -->
          {#if branches.length > 4}
            <div class="relative w-full sm:w-60">
              <input
                type="text"
                bind:value={searchQuery}
                placeholder="Filter branches..."
                class="w-full bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-xl px-3 py-1.5 pl-8 text-neutral-100 text-xs outline-none transition-colors"
              />
              <svg class="w-3.5 h-3.5 text-neutral-500 absolute left-2.5 top-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
          {/if}
        </div>

        <!-- Branches List -->
        {#if filteredBranches.length === 0}
          <div class="p-8 text-center text-xs text-neutral-400 font-mono">
            {#if searchQuery}
              No branches match "{searchQuery}".
            {:else}
              No alternative branches reported for this game. Only the standard public release is available.
            {/if}
          </div>
        {:else}
          <div class="space-y-2">
            {#each filteredBranches as branch (branch.name)}
              {@const isActive = branch.name === currentBranch}
              {@const isThisSwitching = isSwitching && switchingBranch === branch.name}

              <div
                class="border rounded-xl p-3.5 transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 {isActive ? 'bg-surface-header border-surface-borderHighlight shadow-md' : 'bg-surface-base hover:bg-surface-cardHover border-surface-border'}"
              >
                <!-- Branch Info -->
                <div class="space-y-1 min-w-0 flex-1">
                  <div class="flex items-center space-x-2 flex-wrap gap-y-1">
                    <span class="text-xs font-bold text-white font-mono flex items-center gap-1.5">
                      {#if branch.name === 'public'}
                        <span>None / Default (public)</span>
                      {:else}
                        <span>{branch.name}</span>
                      {/if}
                    </span>

                    {#if isActive}
                      <span class="px-2 py-0.5 rounded text-3xs font-bold uppercase tracking-wider bg-white text-neutral-950 font-mono">
                        ACTIVE
                      </span>
                    {/if}

                    {#if branch.pwd_required}
                      <span class="px-1.5 py-0.5 rounded text-3xs font-mono bg-neutral-800 text-neutral-300 border border-neutral-700 flex items-center space-x-1" title="Password required to access this branch">
                        <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                        </svg>
                        <span>Private</span>
                      </span>
                    {/if}

                    {#if branch.build_id}
                      <span class="px-1.5 py-0.5 rounded text-3xs font-mono text-neutral-400 bg-surface-input border border-surface-border">
                        Build #{branch.build_id}
                      </span>
                    {/if}
                  </div>

                  {#if branch.description}
                    <p class="text-xs text-neutral-400 leading-relaxed">
                      {branch.description}
                    </p>
                  {/if}

                  {#if branch.time_updated > 0}
                    <div class="text-3xs text-neutral-500 font-mono">
                      Last updated: {formatDate(branch.time_updated)}
                    </div>
                  {/if}
                </div>

                <!-- Select Action -->
                <div class="flex-shrink-0 self-start sm:self-auto">
                  {#if isActive}
                    <div class="px-3 py-1.5 rounded-xl bg-surface-header text-neutral-400 border border-surface-border text-xs font-mono flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5 text-neutral-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                      </svg>
                      <span>Selected</span>
                    </div>
                  {:else}
                    <button
                      type="button"
                      onclick={() => handleSelectBranch(branch)}
                      disabled={isSwitching}
                      class="px-3.5 py-1.5 rounded-xl bg-surface-header hover:bg-surface-elevated border border-surface-border hover:border-surface-borderHighlight text-neutral-200 hover:text-white font-semibold text-xs transition-colors cursor-pointer disabled:opacity-50 flex items-center space-x-1.5"
                    >
                      {#if isThisSwitching}
                        <svg class="animate-spin w-3 h-3 text-neutral-300" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                        </svg>
                        <span>Switching...</span>
                      {:else}
                        <span>Switch to Branch</span>
                      {/if}
                    </button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Private Betas Section -->
      <div class="bg-surface-card border border-surface-border rounded-2xl p-5 shadow-sm space-y-3">
        <div class="flex items-center space-x-2">
          <svg class="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          <h3 class="text-xs font-bold text-neutral-200">Private Betas</h3>
        </div>
        <p class="text-xs text-neutral-400 leading-relaxed">
          Enter a beta access code to unlock private test builds or enter credentials for password-protected branches.
        </p>

        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-1">
          <input
            type="password"
            bind:value={privateBetaCode}
            placeholder="Enter beta access code..."
            class="flex-1 bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-xl px-3 py-2 text-neutral-100 text-xs outline-none transition-colors font-mono"
            onkeydown={(e) => { if (e.key === 'Enter') handleCheckPrivateCode(); }}
          />
          <button
            type="button"
            onclick={handleCheckPrivateCode}
            disabled={isCheckingCode || !privateBetaCode.trim()}
            class="px-4 py-2 rounded-xl bg-surface-header hover:bg-surface-elevated border border-surface-border text-neutral-200 hover:text-white font-semibold text-xs transition-colors cursor-pointer disabled:opacity-50 flex items-center justify-center space-x-1.5"
          >
            {#if isCheckingCode}
              <svg class="animate-spin w-3 h-3 text-neutral-300" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
              </svg>
              <span>Checking...</span>
            {:else}
              <span>Check Code</span>
            {/if}
          </button>
        </div>

        {#if codeFeedback}
          <div class="p-3 rounded-xl text-xs space-y-1 {codeFeedback.type === 'success' ? 'bg-neutral-900 border border-neutral-700 text-neutral-200' : 'bg-[#181214] border border-rose-900/50 text-rose-300'}">
            <p>{codeFeedback.message}</p>
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>
