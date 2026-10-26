<script>
  import { AppAPI } from '../../../api/app.js';
  import { notificationStore } from '../../../stores/notifications.js';
  import { updateGameInStore, selectedGame } from '../../../stores/library.js';
  import { getEffectiveAppId, sortBranches, isBranchBuildMismatch } from '../../../utils/betas.js';
  import HubBetasStatusBanner from './HubBetasStatusBanner.svelte';
  import HubBranchList from './HubBranchList.svelte';
  import HubPrivateBetasCard from './HubPrivateBetasCard.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../../types/domain').Game} [game]
   * @property {boolean} [isTaskActive]
   * @property {import('../../../types/domain').Task | null} [activeGameTask]
   * @property {((branch: { branch: string, password?: string, targetBuildId?: string }) => void | Promise<void>) | null} [onbranchChange]
   * @property {((detail: { branch: string }) => void | Promise<void>) | null} [onupdateGame]
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

  // Private beta state
  let privateBetaCode = $state('');
  let isCheckingCode = $state(false);
  let codeFeedback = $state(null); // { type: 'success' | 'error', message: string }

  let loadedAppId = $state('');

  let effectiveAppId = $derived(getEffectiveAppId(game));
  let isSteam = $derived(Boolean(effectiveAppId));
  let isInstalled = $derived(game ? (game.isInstalled ?? (Boolean(game.exePath) || (game.fileCount > 0))) : false);
  let isStub = $derived(game ? (game.isStub ?? (Boolean(game.gameDir) && !isInstalled)) : false);
  let isVirtual = $derived(game ? (game.isVirtual ?? !game.gameDir) : false);
  let currentBranch = $derived(game?.branch || 'public');
  let activeBranchObj = $derived(branches.find(b => b.name === currentBranch) || null);
  let targetBuildId = $derived(activeBranchObj?.build_id || '');
  let cleanInstalledBuild = $derived((game?.buildId && game.buildId !== '0' && game.buildId !== 'unknown') ? String(game.buildId).trim() : '');
  let isBuildMismatch = $derived(isBranchBuildMismatch(isInstalled, cleanInstalledBuild, targetBuildId));
  let sortedBranches = $derived(sortBranches(branches, currentBranch));

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

    const pwdBranches = branches.filter(b => b.pwd_required);
    if (pwdBranches.length === 0 && branches.length > 0) {
      codeFeedback = {
        type: 'error',
        message: 'No password-protected private beta branches were reported by Steam for this title.'
      };
      isCheckingCode = false;
      return;
    }

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
    onupdateGame?.({ branch: currentBranch });
  }

  // Fetch Steam depot branches on AppID change
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
    <HubBetasStatusBanner
      {effectiveAppId}
      {isVirtual}
      {isStub}
      {currentBranch}
      {targetBuildId}
      {isBuildMismatch}
      {cleanInstalledBuild}
      {isLoading}
      {isTaskActive}
      {activeGameTask}
      onrefresh={loadBranches}
      ontriggerUpdate={handleTriggerUpdate}
    />

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
      <HubBranchList
        branches={sortedBranches}
        {currentBranch}
        {isSwitching}
        {switchingBranch}
        onselectBranch={handleSelectBranch}
      />

      <HubPrivateBetasCard
        bind:code={privateBetaCode}
        {isCheckingCode}
        {codeFeedback}
        oncheckCode={handleCheckPrivateCode}
      />
    {/if}
  {/if}
</div>
