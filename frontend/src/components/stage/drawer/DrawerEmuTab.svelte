<script>
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import StepperRow from '../../common/StepperRow.svelte';

  export let game = null;
  export let globalCfg = {};
  export let languageOptions = [];

  $: defaultModeLabel = globalCfg.gbe_mode ? (globalCfg.gbe_mode === 'portable' ? 'Portable' : (globalCfg.gbe_mode === 'none' ? 'None' : 'Loader')) : 'Loader';

  $: modeOptions = [
    { value: '', label: `Default (${defaultModeLabel})` },
    { value: 'loader', label: 'Loader (Non-destructive)' },
    { value: 'portable', label: 'Portable (Direct DLL/SO)' },
    { value: 'none', label: 'None (DRM-Free / Disabled)' }
  ];

  $: langOptions = [
    { value: '', label: `Default (${globalCfg.language ? (globalCfg.language === 'auto' ? 'Auto / System' : globalCfg.language) : 'Auto / System'})` },
    ...languageOptions.map(l => ({ value: l, label: l.charAt(0).toUpperCase() + l.slice(1) }))
  ];

  function handleStepperChange() {
    // Modified in-memory draft
  }
</script>

<div class="space-y-3">
  <!-- Emulator Integration Mode -->
  <div class="space-y-2">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      Steam DRM & Emulation
    </span>

    <StepperRow
      label="Emulation Mode"
      options={modeOptions}
      bind:value={$drawerDraft.mode}
      on:change={handleStepperChange}
    />

    <StepperRow
      label="Language"
      options={langOptions}
      bind:value={$drawerDraft.language}
      on:change={handleStepperChange}
    />
  </div>

  <!-- AppID Override -->
  <div class="space-y-1.5 p-3 bg-[#0d0d12] border border-[#222230] rounded-xl focus-within:border-sky-500 transition-all">
    <label for="drawerAppId" class="block font-bold text-slate-200 text-xs">
      AppID Override
    </label>
    <input
      id="drawerAppId"
      type="text"
      bind:value={$drawerDraft.appId}
      placeholder={game?.appId || '123456'}
      class="w-full bg-[#14141e] border border-[#2c2c3e] focus:border-sky-400 rounded-lg px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
    />
  </div>

  <!-- One-Shot Re-Apply Actions -->
  <div class="space-y-2 pt-2 border-t border-[#20202c]">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      One-Shot Setup Actions
    </span>

    <div class="space-y-1.5">
      <!-- Steamless -->
      <button
        type="button"
        on:click={() => { $drawerDraft.optSteamless = !$drawerDraft.optSteamless; handleStepperChange(); }}
        class="w-full flex items-center justify-between p-3 bg-[#0d0d12] hover:bg-[#13131a] border border-[#222230] rounded-xl transition-all select-none text-left"
      >
        <div>
          <span class="block font-bold text-slate-200 text-xs">Unpack SteamStub DRM</span>
          <p class="text-[11px] text-slate-400">Automatically run Steamless-rs unpacker on game binaries.</p>
        </div>
        <div class="w-10 h-6 flex items-center rounded-full p-1 transition-colors {$drawerDraft.optSteamless ? 'bg-sky-500 justify-end' : 'bg-[#222232] justify-start'}">
          <div class="w-4 h-4 rounded-full bg-white shadow-md"></div>
        </div>
      </button>

      <!-- Achievements -->
      <button
        type="button"
        on:click={() => { $drawerDraft.optAchievements = !$drawerDraft.optAchievements; handleStepperChange(); }}
        class="w-full flex items-center justify-between p-3 bg-[#0d0d12] hover:bg-[#13131a] border border-[#222230] rounded-xl transition-all select-none text-left"
      >
        <div>
          <span class="block font-bold text-slate-200 text-xs">Fetch Achievements & DLC</span>
          <p class="text-[11px] text-slate-400">Download Steam achievement icons and DLC list into steam_settings.</p>
        </div>
        <div class="w-10 h-6 flex items-center rounded-full p-1 transition-colors {$drawerDraft.optAchievements ? 'bg-sky-500 justify-end' : 'bg-[#222232] justify-start'}">
          <div class="w-4 h-4 rounded-full bg-white shadow-md"></div>
        </div>
      </button>

      <!-- Normalize Directory -->
      <button
        type="button"
        on:click={() => { $drawerDraft.optNormalize = !$drawerDraft.optNormalize; handleStepperChange(); }}
        class="w-full flex items-center justify-between p-3 bg-[#0d0d12] hover:bg-[#13131a] border border-[#222230] rounded-xl transition-all select-none text-left"
      >
        <div>
          <span class="block font-bold text-slate-200 text-xs">Normalize Folder Name</span>
          <p class="text-[11px] text-slate-400">Standardize folder naming to match official title.</p>
        </div>
        <div class="w-10 h-6 flex items-center rounded-full p-1 transition-colors {$drawerDraft.optNormalize ? 'bg-sky-500 justify-end' : 'bg-[#222232] justify-start'}">
          <div class="w-4 h-4 rounded-full bg-white shadow-md"></div>
        </div>
      </button>
    </div>
  </div>
</div>
