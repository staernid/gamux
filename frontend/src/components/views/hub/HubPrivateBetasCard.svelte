<script>
  /**
   * @typedef {Object} Props
   * @property {string} [code]
   * @property {boolean} [isCheckingCode]
   * @property {{ type: 'success' | 'error', message: string } | null} [codeFeedback]
   * @property {() => void} [oncheckCode]
   */

  /** @type {Props} */
  let {
    code = $bindable(''),
    isCheckingCode = false,
    codeFeedback = null,
    oncheckCode = undefined
  } = $props();
</script>

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
      bind:value={code}
      placeholder="Enter beta access code..."
      class="flex-1 bg-surface-input border border-surface-border focus:border-surface-borderHighlight rounded-xl px-3 py-2 text-neutral-100 text-xs outline-none transition-colors font-mono"
      onkeydown={(e) => { if (e.key === 'Enter') oncheckCode?.(); }}
    />
    <button
      type="button"
      onclick={oncheckCode}
      disabled={isCheckingCode || !code.trim()}
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
    <div class="p-3 rounded-xl text-xs space-y-1 {codeFeedback.type === 'success' ? 'bg-neutral-900 border border-neutral-700 text-neutral-200' : 'bg-rose-950/40 border border-rose-900/50 text-rose-300'}">
      <p>{codeFeedback.message}</p>
    </div>
  {/if}
</div>
