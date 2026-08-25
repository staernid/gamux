<script>
  export let isCheckingIntegrity = false;
  export let isNormalizing = false;
  export let integrityError = '';
  export let integrityReport = null;
  export let onRunIntegrityCheck = () => {};
  export let onUpdateGame = () => {};
  export let onNormalizeDirectory = () => {};
</script>

<div class="space-y-3">
  <p class="text-slate-400 text-xs">
    Verify local depot checksums against official SteamPipe SHA-256 manifests:
  </p>

  <div class="flex items-center space-x-2 flex-wrap gap-y-2">
    <button
      type="button"
      on:click={onRunIntegrityCheck}
      disabled={isCheckingIntegrity}
      class="bg-white hover:bg-neutral-200 text-black font-black px-4 py-2 rounded-xl transition-colors text-xs disabled:opacity-50"
    >
      {isCheckingIntegrity ? 'Verifying Files...' : 'Run Integrity Check'}
    </button>

    <button
      type="button"
      on:click={onUpdateGame}
      class="bg-emerald-600 hover:bg-emerald-500 text-white font-bold px-4 py-2 rounded-xl transition-colors text-xs shadow-sm"
      title="Download missing/updated depot files"
    >
      Update / Repair Depots
    </button>

    <button
      type="button"
      on:click={onNormalizeDirectory}
      disabled={isNormalizing}
      class="bg-[#1a1a24] hover:bg-[#252535] text-slate-200 border border-[#2e2e42] font-semibold px-4 py-2 rounded-xl transition-colors text-xs disabled:opacity-50"
      title="Rename game directory to steam naming conventions"
    >
      {isNormalizing ? 'Renaming...' : 'Normalize Directory'}
    </button>
  </div>

  {#if integrityError}
    <div class="p-3.5 rounded-xl bg-[#1e1014] border border-rose-900/50 text-rose-300 text-xs space-y-1">
      <strong class="block font-bold">Verification Notice</strong>
      <p>{integrityError}</p>
    </div>
  {/if}

  {#if integrityReport}
    {@const total = integrityReport.official_count ?? integrityReport.OfficialCount ?? 0}
    {@const valid = integrityReport.intact_count ?? integrityReport.IntactCount ?? 0}
    {@const modified = integrityReport.modified_files ?? integrityReport.ModifiedFiles ?? []}
    {@const missing = integrityReport.missing_files ?? integrityReport.MissingFiles ?? []}
    {@const untracked = integrityReport.untracked_files ?? integrityReport.UntrackedFiles ?? []}

    <div class="bg-[#0b0b10] border border-[#242436] p-4 rounded-xl space-y-2 font-mono text-xs">
      <div class="flex items-center justify-between pb-2 border-b border-[#1c1c28]">
        <span class="text-slate-400">Total Official Files:</span>
        <strong class="text-white">{total} files</strong>
      </div>
      <div class="flex items-center justify-between text-emerald-400">
        <span>✓ Intact & Valid:</span>
        <span>{valid} files</span>
      </div>
      {#if modified.length > 0}
        <div class="flex items-center justify-between text-rose-400">
          <span>✗ Modified / Corrupted:</span>
          <span>{modified.length} files</span>
        </div>
      {/if}
      {#if missing.length > 0}
        <div class="flex items-center justify-between text-amber-400">
          <span>⚠ Missing Files:</span>
          <span>{missing.length} files</span>
        </div>
      {/if}
      {#if untracked.length > 0}
        <div class="flex items-center justify-between text-slate-400">
          <span>+ Extra / Untracked Mods:</span>
          <span>{untracked.length} files</span>
        </div>
      {/if}

      {#if modified.length > 0 || missing.length > 0}
        <div class="pt-2 border-t border-[#1c1c28] flex justify-end">
          <button
            type="button"
            on:click={onUpdateGame}
            class="bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs px-3 py-1.5 rounded-lg transition-colors shadow-sm"
          >
            Repair & Download Missing Files
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
