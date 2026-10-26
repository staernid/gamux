<script>
  import ModalShell from '../common/ModalShell.svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import { tasksStore } from '../../stores/tasks.js';
  import { notificationStore } from '../../stores/notifications.js';

  /**
   * @typedef {Object} Props
   * @property {import('../../types/domain').Game} [game]
   * @property {Record<string, unknown>} [versionStatus]
   * @property {() => void} [onclose]
   * @property {() => void | Promise<void>} [onupdate]
   */

  /** @type {Props} */
  let {
    game = null,
    versionStatus = null,
    onclose = null,
    onupdate = null
  } = $props();

  let isUpdating = $state(false);

  let effectiveStatus = $derived(versionStatus || game?.versionStatus || null);
  let currentBuild = $derived(
    effectiveStatus?.current_build_id ||
    game?.buildId ||
    game?.build_id ||
    'Unknown'
  );
  let latestBuild = $derived(effectiveStatus?.latest_build_id || 'Unknown');
  let branch = $derived(game?.branch || 'public');
  let patchTitle = $derived(effectiveStatus?.latest_patch_title || '');
  let patchDate = $derived(
    effectiveStatus?.latest_patch_date
      ? new Date(effectiveStatus.latest_patch_date * 1000).toLocaleDateString(undefined, {
          year: 'numeric',
          month: 'short',
          day: 'numeric'
        })
      : ''
  );

  async function handleUpdate() {
    isUpdating = true;
    try {
      if (onupdate) {
        await onupdate();
      } else if (game?.gameDir && game?.appId && game.appId !== '0') {
        const branchToUpdate = branch;
        await tasksStore.start({
          app_id: parseInt(game.appId, 10),
          game_name: game.name,
          target_dir: game.gameDir,
          platform: game.platform || 'win64',
          language: (game.currentLanguage || 'english').toLowerCase(),
          auto_apply: true,
          store_type: (game.store || 'steam').toLowerCase(),
          branch: branchToUpdate,
        });
        notificationStore.items.add({
          type: 'info',
          title: 'Update Enqueued',
          message: `Update task started for ${game.name}${branchToUpdate !== 'public' ? ` (${branchToUpdate})` : ''}`,
          gameName: game.name
        });
      }
      onclose?.();
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Update Failed',
        message: err?.message || String(err)
      });
    } finally {
      isUpdating = false;
    }
  }
</script>

<ModalShell
  title="Update Available"
  badge="STEAM"
  maxWidth="max-w-lg"
  padding="p-4 sm:p-6"
  {onclose}
>
  <div class="space-y-5">
    <!-- Game Header Snapshot -->
    <div class="flex items-center space-x-4 p-3.5 rounded-2xl bg-surface-header border border-surface-border">
      <div class="w-14 h-20 rounded-xl overflow-hidden shrink-0 bg-black/40 shadow-md">
        <PosterArt
          appId={game?.appId}
          name={game?.name}
          posterUrl={game?.coverUrl}
          localCoverUrl={game?.localCoverUrl}
          className="w-full h-full object-cover"
        />
      </div>
      <div class="min-w-0 flex-1 space-y-1">
        <h3 class="text-sm font-bold text-white truncate">{game?.name || 'Game Update'}</h3>
        <div class="flex items-center space-x-2 text-2xs font-mono text-neutral-400">
          <span class="px-2 py-0.5 rounded bg-surface-card border border-surface-border uppercase font-semibold">
            {branch}
          </span>
          {#if game?.appId}
            <span>AppID: {game.appId}</span>
          {/if}
        </div>
      </div>
    </div>

    <!-- Version Comparison Box -->
    <div class="grid grid-cols-2 gap-3">
      <div class="p-3.5 rounded-xl bg-surface-card border border-surface-border space-y-1">
        <span class="text-3xs uppercase font-mono text-neutral-400 font-bold block">Current Installed</span>
        <div class="text-xs font-mono font-bold text-neutral-300 truncate">
          {#if currentBuild && currentBuild !== '0' && currentBuild !== 'Unknown'}
            Build #{currentBuild}
          {:else if game?.manifestId || game?.manifest_id}
            Manifest #{game.manifestId || game.manifest_id}
          {:else}
            Build #{currentBuild}
          {/if}
        </div>
      </div>

      <div class="p-3.5 rounded-xl bg-amber-500/10 border border-amber-500/30 space-y-1">
        <span class="text-3xs uppercase font-mono text-amber-400 font-bold block">Latest Available</span>
        <div class="text-xs font-mono font-bold text-amber-200 truncate">
          Build #{latestBuild}
        </div>
      </div>
    </div>

    <!-- Patch Notes Preview (if available) -->
    {#if patchTitle || patchDate}
      <div class="p-4 rounded-xl bg-surface-card border border-surface-border space-y-2">
        <div class="flex items-center justify-between text-2xs">
          <span class="font-bold text-neutral-300">Latest Release Notes</span>
          {#if patchDate}
            <span class="font-mono text-neutral-400 text-3xs">{patchDate}</span>
          {/if}
        </div>
        {#if patchTitle}
          <p class="text-xs text-neutral-200 font-medium leading-relaxed">
            {patchTitle}
          </p>
        {/if}
      </div>
    {/if}

    <!-- Modal Actions -->
    <div class="flex items-center justify-end space-x-2.5 pt-3 border-t border-surface-border">
      <button
        type="button"
        onclick={onclose}
        disabled={isUpdating}
        class="px-4 py-2 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-xs font-bold text-neutral-300 hover:text-white transition-colors cursor-pointer"
      >
        Later
      </button>

      <button
        type="button"
        onclick={handleUpdate}
        disabled={isUpdating}
        class="px-5 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-neutral-950 text-xs font-black transition-all shadow-lg flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isUpdating}
          <svg class="w-3.5 h-3.5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Enqueuing...</span>
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>Update Game Now</span>
        {/if}
      </button>
    </div>
  </div>
</ModalShell>
