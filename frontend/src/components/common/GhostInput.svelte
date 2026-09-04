<script>
  import { createEventDispatcher } from 'svelte';

  export let label = '';
  export let description = '';
  export let tooltip = '';
  export let value = ''; // current local value ('' means inherit if hasInheritance)
  export let placeholder = ''; // standard placeholder
  export let globalPlaceholder = ''; // global default watermark (if in inherit mode)
  export let disabled = false;
  export let mono = false;
  export let type = 'text'; // 'text' | 'password'
  export let status = ''; // optional status badge (e.g. 'SET')
  export let stacked = false; // stacked vertical card layout
  export let id = `input-${Math.random().toString(36).slice(2, 9)}`;

  const dispatch = createEventDispatcher();

  let showPassword = false;

  $: hasInheritance = Boolean(globalPlaceholder);
  $: isInherited = hasInheritance && (!value || value.trim() === '');
  $: isOverridden = hasInheritance && !isInherited;
  $: effectiveDescription = description || tooltip;
  $: effectiveType = type === 'password' ? (showPassword ? 'text' : 'password') : type;

  function handleInput(e) {
    if (disabled) return;
    value = e.target.value;
    dispatch('input', value);
  }

  function revertToInherit(e) {
    e.stopPropagation();
    if (disabled || !hasInheritance) return;
    value = '';
    dispatch('input', '');
    dispatch('change', '');
  }

  function togglePasswordVisibility(e) {
    e.stopPropagation();
    showPassword = !showPassword;
  }
</script>

{#if stacked}
  <div
    class="flex flex-col justify-between gap-2.5 p-3 rounded-xl transition-all h-full
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-surface-card'}
      {isOverridden ? 'bg-surface-card border border-neutral-600' : 'bg-[#101013] border border-surface-border'}"
  >
    <!-- Top: Header with Label, Status, and Override Indicator -->
    <div class="space-y-1 min-w-0">
      <div class="flex items-center justify-between gap-1.5 min-w-0">
        <div class="flex items-center space-x-2 min-w-0 flex-1">
          <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {isOverridden ? 'bg-white' : 'bg-transparent'}" aria-hidden="true"></div>
          <label for={id} class="text-xs font-semibold cursor-pointer truncate {isOverridden ? 'text-white font-bold' : 'text-neutral-200'}">
            {label}
          </label>
        </div>

        <div class="flex items-center space-x-1.5 flex-shrink-0">
          {#if isOverridden}
            <span class="text-3xs font-mono text-neutral-300 font-medium">
              (Override)
            </span>
          {/if}
          {#if status}
            <span class="px-2 py-0.5 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 rounded-lg text-3xs font-mono font-bold">
              {status}
            </span>
          {/if}
        </div>
      </div>

      {#if effectiveDescription}
        <p class="text-xs text-neutral-400 leading-relaxed">
          {effectiveDescription}
        </p>
      {/if}
    </div>

    <!-- Bottom: Full-Width Input + Optional Slot + Reset -->
    <div class="flex items-center space-x-1.5 w-full pt-1">
      <div class="relative flex-1 min-w-0">
        <input
          {id}
          type={effectiveType}
          {value}
          {disabled}
          placeholder={globalPlaceholder || placeholder || ''}
          on:input={handleInput}
          class="w-full bg-surface-input border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors {mono ? 'font-mono' : ''} {type === 'password' ? 'pr-8' : ''}
            {isOverridden
              ? 'border-neutral-500 text-white font-medium focus:border-white'
              : 'border-surface-border text-neutral-200 placeholder-neutral-500 focus:border-neutral-500'}"
        />
        {#if type === 'password'}
          <button
            type="button"
            on:click={togglePasswordVisibility}
            class="absolute inset-y-0 right-0 px-2.5 flex items-center text-neutral-400 hover:text-white transition-colors cursor-pointer"
            title={showPassword ? 'Hide value' : 'Show value'}
            aria-label={showPassword ? 'Hide value' : 'Show value'}
          >
            {#if showPassword}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
              </svg>
            {:else}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
            {/if}
          </button>
        {/if}
      </div>

      <slot name="action" />

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#202024] hover:bg-surface-border border border-surface-borderHighlight text-neutral-300 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title="Reset to global default"
          aria-label="Reset to global default"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      {/if}
    </div>
  </div>
{:else}
  <div
    class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-2.5 sm:p-3 rounded-xl transition-all
      {disabled ? 'opacity-40 pointer-events-none' : 'hover:bg-surface-card'}
      {isOverridden ? 'bg-surface-card border border-neutral-600' : 'bg-[#101013] border border-surface-border'}"
  >
    <!-- Left Side: Label & Description with Override Dot -->
    <div class="flex items-start sm:items-center space-x-2.5 min-w-0 flex-1">
      <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 mt-1 sm:mt-0 {isOverridden ? 'bg-white' : 'bg-transparent'}" aria-hidden="true"></div>

      <div class="space-y-0.5 min-w-0 flex-1">
        <div class="flex items-center space-x-2 flex-wrap">
          <label for={id} class="text-xs font-semibold cursor-pointer {isOverridden ? 'text-white font-bold' : 'text-neutral-200'}">
            {label}
          </label>
          {#if isOverridden}
            <span class="text-3xs font-mono text-neutral-300 font-medium">
              (Override)
            </span>
          {/if}
        </div>

        {#if effectiveDescription}
          <p class="text-xs text-neutral-400 leading-relaxed">
            {effectiveDescription}
          </p>
        {/if}
      </div>
    </div>

    <!-- Right Side: Input + Optional Status/Slot + Reset Button -->
    <div class="flex items-center space-x-2 flex-shrink-0 w-full sm:w-auto">
      <div class="relative flex-1 sm:w-60">
        <input
          {id}
          type={effectiveType}
          {value}
          {disabled}
          placeholder={globalPlaceholder || placeholder || ''}
          on:input={handleInput}
          class="w-full bg-surface-input border rounded-lg px-3 py-1.5 text-xs outline-none transition-colors {mono ? 'font-mono' : ''} {type === 'password' ? 'pr-8' : ''}
            {isOverridden
              ? 'border-neutral-500 text-white font-medium focus:border-white'
              : 'border-surface-border text-neutral-200 placeholder-neutral-500 focus:border-neutral-500'}"
        />
        {#if type === 'password'}
          <button
            type="button"
            on:click={togglePasswordVisibility}
            class="absolute inset-y-0 right-0 px-2.5 flex items-center text-neutral-400 hover:text-white transition-colors cursor-pointer"
            title={showPassword ? 'Hide value' : 'Show value'}
            aria-label={showPassword ? 'Hide value' : 'Show value'}
          >
            {#if showPassword}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
              </svg>
            {:else}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
            {/if}
          </button>
        {/if}
      </div>

      {#if status}
        <span class="px-2 py-1 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 rounded-lg text-3xs font-mono font-bold flex-shrink-0">
          {status}
        </span>
      {/if}

      <slot name="action" />

      {#if isOverridden}
        <button
          type="button"
          on:click={revertToInherit}
          class="p-1.5 rounded-lg bg-[#202024] hover:bg-surface-border border border-surface-borderHighlight text-neutral-300 hover:text-white text-xs transition-colors flex items-center justify-center cursor-pointer flex-shrink-0"
          title="Reset to global default"
          aria-label="Reset to global default"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      {/if}
    </div>
  </div>
{/if}


