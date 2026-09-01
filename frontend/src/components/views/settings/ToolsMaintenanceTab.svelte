<script>
  import { createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../../api/app.js';
  import SettingSection from '../../common/SettingSection.svelte';

  export let toolStatuses = [];
  export let isCheckingTools = false;
  export let updatingTools = {};
  export let isSubmittingBatch = false;
  export let gamesCount = 0;
  export let unpatchedGamesCount = 0;

  const dispatch = createEventDispatcher();

  function checkAllTools() {
    dispatch('checkAll');
  }

  function updateTool(key, name) {
    dispatch('updateTool', { key, name });
  }

  function runBatchAchievements() {
    dispatch('batchAchievements');
  }

  function runBatchArtwork() {
    dispatch('batchArtwork');
  }

  function runBatchApply() {
    dispatch('batchApply');
  }

  function runBatchNormalize() {
    dispatch('batchNormalize');
  }

  function runBatchVerify() {
    dispatch('batchVerify');
  }
</script>

<div class="space-y-3">
  <!-- Toolchain Management -->
  <SettingSection
    title="Helper Tools"
    tooltip="External tools for Steam emulation, DRM removal, and Proton execution."
  >
    <svelte:fragment slot="action">
      <button
        type="button"
        on:click={checkAllTools}
        disabled={isCheckingTools}
        class="px-2.5 py-1 bg-[#1a1a26] hover:bg-[#242436] border border-[#303046] text-slate-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer disabled:opacity-50"
      >
        {#if isCheckingTools}
          <svg class="animate-spin w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span>Checking...</span>
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Check All</span>
        {/if}
      </button>
    </svelte:fragment>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5">
      {#each toolStatuses as tool}
        <div class="bg-[#0e0e14] border border-[#1e1e2c] p-3 rounded-xl flex flex-col justify-between gap-2.5">
          <div class="space-y-1 min-w-0">
            <div class="flex items-center justify-between gap-1.5">
              <span class="font-semibold text-xs text-white truncate">{tool.name || tool.display_name || tool.key}</span>
              {#if tool.has_update}
                <span class="px-1.5 py-0.2 rounded text-[9px] font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 flex-shrink-0">
                  Update
                </span>
              {/if}
            </div>

            <div class="text-[11px] font-mono text-slate-400 truncate">
              <span>{tool.installed_tag || tool.installed_version || 'Installed'}</span>
              {#if tool.latest_tag || tool.latest_version}
                <span class="text-slate-600">·</span>
                <span class="text-emerald-400">{tool.latest_tag || tool.latest_version}</span>
              {/if}
            </div>
          </div>

          <div class="flex items-center space-x-1.5 pt-1 border-t border-[#181824]">
            {#if tool.release_url}
              <button
                type="button"
                on:click={() => AppAPI.openURL(tool.release_url)}
                class="text-slate-400 hover:text-cyan-300 text-xs px-2 py-1 rounded-lg border border-[#262638] bg-[#14141e] transition-colors cursor-pointer"
                title="Open GitHub Release"
              >
                ↗
              </button>
            {/if}
            <button
              type="button"
              on:click={() => updateTool(tool.key || tool.tool_name, tool.name || tool.display_name)}
              disabled={updatingTools[tool.key || tool.tool_name]}
              class="flex-1 py-1 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-bold text-xs transition-colors flex items-center justify-center space-x-1 cursor-pointer disabled:opacity-50"
            >
              {#if updatingTools[tool.key || tool.tool_name]}
                <svg class="animate-spin w-3 h-3 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Updating...</span>
              {:else}
                <span>{tool.has_update ? 'Update' : 'Reinstall'}</span>
              {/if}
            </button>
          </div>
        </div>
      {/each}
    </div>
  </SettingSection>

  <!-- Batch Maintenance Operations -->
  <SettingSection
    title="Batch Maintenance"
    tooltip="Automated multi-game library operations executed in the background."
  >
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
      <!-- 1. Batch Achievements -->
      <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
            </svg>
            <span>Achievements</span>
          </span>
          <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
            Download missing achievement schemas and icons across library.
          </p>
        </div>
        <button
          type="button"
          on:click={runBatchAchievements}
          disabled={isSubmittingBatch || gamesCount === 0}
          class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-sky-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-sky-900/40 hover:border-sky-700/60 transition-all cursor-pointer"
        >
          Download Missing
        </button>
      </div>

      <!-- 2. Batch Artwork -->
      <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>Artwork</span>
          </span>
          <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
            Resolve and cache missing posters, hero banners, and logos.
          </p>
        </div>
        <button
          type="button"
          on:click={runBatchArtwork}
          disabled={isSubmittingBatch || gamesCount === 0}
          class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-purple-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-purple-900/40 hover:border-purple-700/60 transition-all cursor-pointer"
        >
          Refresh Artwork
        </button>
      </div>

      <!-- 3. Batch Apply DRM & Emulator -->
      <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <div class="flex items-center justify-between gap-1.5">
            <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5 truncate">
              <svg class="w-3.5 h-3.5 text-amber-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
              <span class="truncate">Apply Setup</span>
            </span>
            <span class="text-[9px] font-mono bg-[#161622] text-amber-300 border border-amber-500/30 px-1.5 py-0.2 rounded">
              {unpatchedGamesCount}
            </span>
          </div>
          <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
            Apply Goldberg emulator and unpacker to unpatched games.
          </p>
        </div>
        <button
          type="button"
          on:click={runBatchApply}
          disabled={isSubmittingBatch || gamesCount === 0}
          class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-amber-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-amber-900/40 hover:border-amber-700/60 transition-all cursor-pointer"
        >
          Apply All
        </button>
      </div>

      <!-- 4. Batch Normalize -->
      <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors">
        <div class="space-y-1">
          <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Normalize Folders</span>
          </span>
          <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
            Standardize directory names to match official Steam titles.
          </p>
        </div>
        <button
          type="button"
          on:click={runBatchNormalize}
          disabled={isSubmittingBatch || gamesCount === 0}
          class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-emerald-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-emerald-900/40 hover:border-emerald-700/60 transition-all cursor-pointer"
        >
          Normalize
        </button>
      </div>

      <!-- 5. Batch Verify Manifests -->
      <div class="bg-[#0e0e14] border border-[#1e1e2c] hover:border-[#2a2a3c] p-3 rounded-xl flex flex-col justify-between gap-2.5 transition-colors sm:col-span-2 lg:col-span-2">
        <div class="space-y-1">
          <span class="font-bold text-slate-100 text-xs flex items-center space-x-1.5">
            <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            <span>Verify Manifests</span>
          </span>
          <p class="text-[11px] text-slate-400 leading-snug line-clamp-2">
            Verify SHA-1 file integrity against official depot manifests.
          </p>
        </div>
        <button
          type="button"
          on:click={runBatchVerify}
          disabled={isSubmittingBatch || gamesCount === 0}
          class="w-full bg-[#181824] hover:bg-[#222232] disabled:opacity-40 text-cyan-300 hover:text-white font-bold text-xs py-1.5 px-2.5 rounded-lg border border-cyan-900/40 hover:border-cyan-700/60 transition-all cursor-pointer"
        >
          Verify All Manifests
        </button>
      </div>
    </div>
  </SettingSection>
</div>
