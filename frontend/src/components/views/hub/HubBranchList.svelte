<script>
  import { formatDate } from '../../../utils/format.js';
  import { filterBranches } from '../../../utils/betas.js';

  /**
   * @typedef {Object} Props
   * @property {Array<{ name: string, description?: string, build_id?: string, pwd_required?: boolean, time_updated?: number }>} [branches]
   * @property {string} [currentBranch]
   * @property {boolean} [isSwitching]
   * @property {string} [switchingBranch]
   * @property {((branch: { name: string, description?: string, build_id?: string, pwd_required?: boolean, time_updated?: number }) => void)} [onselectBranch]
   */

  /** @type {Props} */
  let {
    branches = [],
    currentBranch = 'public',
    isSwitching = false,
    switchingBranch = '',
    onselectBranch = undefined
  } = $props();

  let searchQuery = $state('');
  let filteredBranches = $derived(filterBranches(branches, searchQuery));
</script>

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
                onclick={() => onselectBranch?.(branch)}
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
