<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../../../api/app.js';
  import { notificationStore } from '../../../../stores/notifications.js';
  import { formatBytes, formatTimeAgo } from '../../../../utils/format.js';

  /**
   * @typedef {Object} Props
   * @property {any} [game]
   */

  /** @type {Props} */
  let { game = null } = $props();

  let isLoading = $state(true);
  let errorMsg = $state('');
  let saveResult = $state(null);
  let backups = $state([]);
  let isCreatingBackup = $state(false);
  let isRestoring = $state(false);
  let showBackupInput = $state(false);
  let backupNote = $state('');
  let copiedPath = $state('');
  let copyTimeout = null;

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    if (!game?.gameDir) return;
    isLoading = true;
    errorMsg = '';
    try {
      const [locs, bks] = await Promise.all([
        AppAPI.getGameSaveLocations(game.gameDir),
        AppAPI.getGameBackups(game.gameDir)
      ]);
      saveResult = locs;
      backups = bks || [];
    } catch (err) {
      errorMsg = err?.message || String(err);
    } finally {
      isLoading = false;
    }
  }

  async function handleOpenPath(path) {
    if (!path) return;
    try {
      await AppAPI.openDirectory(path);
    } catch (err) {
      notificationStore.error('Open Folder Failed', err?.message || String(err));
    }
  }

  function handleCopyPath(path) {
    if (!path) return;
    navigator.clipboard.writeText(path);
    copiedPath = path;
    if (copyTimeout) clearTimeout(copyTimeout);
    copyTimeout = setTimeout(() => {
      copiedPath = '';
    }, 2000);
  }

  async function handleCreateBackup() {
    if (!game?.gameDir || isCreatingBackup) return;
    isCreatingBackup = true;
    try {
      const info = await AppAPI.createSaveBackup(game.gameDir, backupNote.trim());
      notificationStore.items.add({
        type: 'info',
        title: 'Save Backup Created',
        message: `Created snapshot for ${game.name} (${formatBytes(info?.size_bytes || 0)})`
      });
      backupNote = '';
      showBackupInput = false;
      await loadData();
    } catch (err) {
      notificationStore.error('Backup Failed', err?.message || String(err));
    } finally {
      isCreatingBackup = false;
    }
  }

  async function handleRestoreBackup(backupId) {
    if (!game?.gameDir || !backupId || isRestoring) return;
    if (!confirm(`Restore save backup "${backupId}"? This will overwrite active save files.`)) {
      return;
    }
    isRestoring = true;
    try {
      await AppAPI.restoreSaveBackup(game.gameDir, backupId);
      notificationStore.items.add({
        type: 'info',
        title: 'Save Backup Restored',
        message: `Restored snapshot ${backupId} for ${game.name}`
      });
      await loadData();
    } catch (err) {
      notificationStore.error('Restore Failed', err?.message || String(err));
    } finally {
      isRestoring = false;
    }
  }

  async function handleDeleteBackup(backupId) {
    if (!game?.gameDir || !backupId) return;
    if (!confirm(`Delete snapshot "${backupId}" permanently?`)) return;
    try {
      await AppAPI.deleteSaveBackup(game.gameDir, backupId);
      backups = backups.filter(b => b.id !== backupId);
      notificationStore.items.add({
        type: 'info',
        title: 'Snapshot Deleted',
        message: `Removed backup ${backupId}`
      });
    } catch (err) {
      notificationStore.error('Delete Failed', err?.message || String(err));
    }
  }

  async function handleUpdateManifest() {
    try {
      await AppAPI.updateSaveManifest();
      notificationStore.items.add({
        type: 'info',
        title: 'Manifest Updated',
        message: 'Updated Ludusavi save database from GitHub.'
      });
      await loadData();
    } catch (err) {
      notificationStore.error('Manifest Update Failed', err?.message || String(err));
    }
  }

  function getCategoryBadge(category) {
    switch (category) {
      case 'save':
        return { label: 'Save Data', class: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40' };
      case 'config':
        return { label: 'Config', class: 'bg-neutral-800 text-neutral-200 border-neutral-700' };
      case 'cloud':
        return { label: 'Goldberg Cloud', class: 'bg-neutral-800 text-neutral-200 border-neutral-700' };
      case 'portable':
        return { label: 'Portable Save', class: 'bg-neutral-800 text-neutral-200 border-neutral-700' };
      default:
        return { label: 'Data', class: 'bg-surface-header text-neutral-400 border-surface-border' };
    }
  }
</script>

<div class="space-y-4">
  {#if isLoading}
    <div class="flex flex-col items-center justify-center p-12 text-neutral-400 space-y-3">
      <div class="w-7 h-7 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
      <span class="text-xs font-mono">Discovering game save and config paths...</span>
    </div>
  {:else if errorMsg}
    <div class="p-4 rounded-xl bg-rose-950/40 border border-rose-800/40 text-rose-300 text-xs">
      {errorMsg}
    </div>
  {:else}
    <!-- Top Summary Metrics Banner -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
      <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-0.5">
        <span class="text-3xs text-neutral-400 uppercase font-bold tracking-wider">Total Save Size</span>
        <p class="font-mono text-sm font-bold text-neutral-100">
          {formatBytes(saveResult?.total_save_bytes || 0)}
        </p>
      </div>

      <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-0.5">
        <span class="text-3xs text-neutral-400 uppercase font-bold tracking-wider">Discovered Paths</span>
        <p class="font-mono text-sm font-bold text-neutral-100">
          {(saveResult?.locations || []).filter(l => l.exists).length} / {(saveResult?.locations || []).length}
        </p>
      </div>

      <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-0.5">
        <span class="text-3xs text-neutral-400 uppercase font-bold tracking-wider">Steam Cloud Emulation</span>
        <p class="font-mono text-sm font-bold {saveResult?.has_cloud_saves ? 'text-neutral-100' : 'text-neutral-500'}">
          {saveResult?.has_cloud_saves ? 'Active (GSE)' : 'None'}
        </p>
      </div>

      <div class="bg-surface-header border border-surface-border p-3 rounded-xl space-y-0.5">
        <span class="text-3xs text-neutral-400 uppercase font-bold tracking-wider">Backup Snapshots</span>
        <p class="font-mono text-sm font-bold text-neutral-100">
          {backups.length}
        </p>
      </div>
    </div>

    <!-- Discovered Save & Config Locations Section -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <span class="block text-neutral-400 font-bold uppercase tracking-wider text-3xs">
          Discovered Save & Config Directories
        </span>
        <button
          type="button"
          onclick={handleUpdateManifest}
          class="text-3xs text-neutral-300 hover:text-white font-bold transition-colors cursor-pointer"
          title="Force update Ludusavi database from GitHub"
        >
          Refresh Manifest
        </button>
      </div>

      {#if (saveResult?.locations || []).length > 0}
        <div class="space-y-2">
          {#each saveResult.locations as loc}
            {@const badge = getCategoryBadge(loc.category)}
            <div class="p-3 bg-[#0d0d12] border border-[#20202c] rounded-xl flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 transition-all">
              <div class="space-y-1 min-w-0 flex-1">
                <div class="flex items-center space-x-2 flex-wrap gap-y-1">
                  <span class="px-2 py-0.5 rounded-md text-4xs font-bold border font-mono {badge.class}">
                    {badge.label}
                  </span>
                  {#if loc.exists}
                    <span class="text-3xs text-slate-400 font-mono">
                      {formatBytes(loc.size_bytes)} ({loc.file_count} {loc.file_count === 1 ? 'file' : 'files'})
                    </span>
                  {:else}
                    <span class="text-3xs text-slate-500 italic">
                      Not created on disk yet
                    </span>
                  {/if}
                  {#if loc.source}
                    <span class="text-4xs text-slate-500 font-mono">
                      · {loc.source}
                    </span>
                  {/if}
                </div>

                <p class="font-mono text-xs text-slate-300 truncate select-all" title={loc.path}>
                  {loc.display_path || loc.path}
                </p>
              </div>

              <div class="flex items-center space-x-2 flex-shrink-0 self-end sm:self-center">
                <button
                  type="button"
                  onclick={() => handleCopyPath(loc.path)}
                  class="px-2.5 py-1.5 rounded-lg bg-[#1a1a24] hover:bg-[#252535] text-slate-300 hover:text-white border border-[#2b2b3c] text-2xs font-bold transition-colors flex items-center space-x-1"
                  title="Copy directory path to clipboard"
                >
                  {#if copiedPath === loc.path}
                    <span class="text-emerald-400 font-bold flex items-center space-x-1">
                      <svg class="w-3 h-3 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
                      </svg>
                      <span>Copied</span>
                    </span>
                  {:else}
                    <span>Copy</span>
                  {/if}
                </button>

                <button
                  type="button"
                  onclick={() => handleOpenPath(loc.path)}
                  disabled={!loc.exists}
                  class="px-3 py-1.5 rounded-lg bg-surface-card hover:bg-surface-cardHover disabled:opacity-40 disabled:cursor-not-allowed text-neutral-200 hover:text-white border border-surface-border text-2xs font-bold transition-colors flex items-center space-x-1 cursor-pointer"
                  title="Open folder in File Manager"
                >
                  <span>Open</span>
                </button>
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="p-4 rounded-xl bg-surface-header border border-surface-border text-center text-neutral-400 text-xs">
          No save locations identified yet. Launch the game once to create initial saves.
        </div>
      {/if}
    </div>

    <!-- Snapshot History & Backup Manager Section -->
    <div class="space-y-2 pt-2 border-t border-surface-border">
      <div class="flex items-center justify-between">
        <div>
          <span class="block text-neutral-300 font-bold uppercase tracking-wider text-3xs">
            Save Snapshots & Backups
          </span>
          <span class="text-3xs text-neutral-400">
            Portable cross-device .zip snapshots (PC ↔ Steam Deck).
          </span>
        </div>

        <button
          type="button"
          onclick={() => { showBackupInput = !showBackupInput; }}
          class="bg-emerald-950/50 hover:bg-emerald-900/70 text-emerald-300 text-2xs font-bold px-3 py-1.5 rounded-xl border border-emerald-800/50 transition-colors flex items-center space-x-1 shadow-sm cursor-pointer"
        >
          <span>+ Create Backup</span>
        </button>
      </div>

      <!-- New Backup Note Input Form -->
      {#if showBackupInput}
        <div class="p-3 bg-surface-header border border-emerald-700/50 rounded-xl space-y-2 animate-fade-in">
          <label for="backupNoteInput" class="block text-neutral-200 text-xs font-bold">
            Backup Description / Note (Optional)
          </label>
          <div class="flex items-center space-x-2">
            <input
              id="backupNoteInput"
              type="text"
              bind:value={backupNote}
              placeholder="e.g. Before Final Boss / Chapter 4"
              class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-3 py-1.5 text-white font-mono text-xs outline-none transition-colors"
            />
            <button
              type="button"
              onclick={handleCreateBackup}
              disabled={isCreatingBackup}
              class="bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 disabled:opacity-50 text-neutral-950 font-black text-xs px-4 py-1.5 rounded-lg transition shadow-md flex-shrink-0 cursor-pointer"
            >
              {isCreatingBackup ? 'Saving...' : 'Take Snapshot'}
            </button>
            <button
              type="button"
              onclick={() => { showBackupInput = false; }}
              class="text-neutral-400 hover:text-white text-xs px-2 py-1.5 rounded-lg cursor-pointer"
            >
              Cancel
            </button>
          </div>
        </div>
      {/if}

      <!-- Backups List -->
      {#if backups.length > 0}
        <div class="space-y-1.5 max-h-48 overflow-y-auto pr-1">
          {#each backups as bk}
            <div class="p-2.5 bg-surface-header border border-surface-border rounded-xl flex items-center justify-between gap-2.5 text-xs">
              <div class="space-y-0.5 min-w-0">
                <div class="flex items-center space-x-2">
                  <span class="font-mono font-bold text-neutral-200">{bk.id}</span>
                  <span class="text-3xs text-neutral-400">({formatTimeAgo(bk.created_at)})</span>
                  <span class="text-4xs px-1.5 py-0.2 rounded border font-mono {bk.created_by === 'auto_exit' ? 'bg-amber-500/10 text-amber-300 border-amber-500/30' : 'bg-neutral-800 text-neutral-300 border-neutral-700'}">
                    {bk.created_by === 'auto_exit' ? 'Auto Exit' : 'Manual'}
                  </span>
                </div>
                {#if bk.note}
                  <p class="text-2xs text-emerald-300/90 italic truncate">{bk.note}</p>
                {/if}
                <div class="flex items-center space-x-2 text-3xs text-neutral-400 font-mono">
                  <span>{formatBytes(bk.size_bytes)}</span>
                  <span>·</span>
                  <span>{bk.file_count} {bk.file_count === 1 ? 'file' : 'files'}</span>
                </div>
              </div>

              <div class="flex items-center space-x-1.5 flex-shrink-0">
                <button
                  type="button"
                  onclick={() => handleRestoreBackup(bk.id)}
                  disabled={isRestoring}
                  class="px-2.5 py-1 rounded-lg bg-emerald-950/40 hover:bg-emerald-900/60 text-emerald-300 border border-emerald-800/40 text-2xs font-bold transition-colors"
                  title="Restore this save snapshot"
                >
                  {isRestoring ? 'Restoring...' : 'Restore'}
                </button>

                <button
                  type="button"
                  onclick={() => handleDeleteBackup(bk.id)}
                  class="p-1 rounded-lg bg-[#181822] hover:bg-rose-950/40 text-slate-400 hover:text-rose-400 border border-[#2b2b3c] transition-colors"
                  title="Delete Snapshot"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="p-3 bg-[#09090d] border border-[#20202e] rounded-xl text-center text-slate-500 text-xs">
          No backup snapshots created yet. Click "+ Create Backup" to take your first save snapshot.
        </div>
      {/if}
    </div>
  {/if}
</div>
