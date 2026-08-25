<script>
  export let game = null;
  export let globalCfg = {};
  export let customMode = '';
  export let customLanguage = '';
  export let languageOptions = [];
  export let customAppID = '';
  export let optSteamless = true;
  export let optAchievements = true;
  export let optNormalize = true;
</script>

<div class="space-y-4">
  <!-- Mode & Language Configuration -->
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
    <div>
      <label for="drawerMode" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
        DEFAULT MODE
      </label>
      <select
        id="drawerMode"
        bind:value={customMode}
        class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none text-xs"
      >
        <option value="loader">Loader Mode (ColdClientLoader)</option>
        <option value="portable">Portable Mode (Direct DLL)</option>
      </select>
    </div>

    <div>
      <label for="drawerLanguage" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
        Language
      </label>
      <select
        id="drawerLanguage"
        bind:value={customLanguage}
        class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none capitalize text-xs"
      >
        <option value="">Default ({globalCfg.language ? (globalCfg.language === 'auto' ? 'Auto / System Locale' : globalCfg.language) : 'Auto / System Locale'})</option>
        {#each languageOptions as lang}
          <option value={lang}>{lang}</option>
        {/each}
      </select>
    </div>
  </div>

  <!-- AppID Override -->
  <div>
    <label for="drawerAppId" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
      AppID Override
    </label>
    <input
      id="drawerAppId"
      type="text"
      bind:value={customAppID}
      placeholder={game?.appId || '123456'}
      class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
    />
  </div>

  <!-- One-Shot Setup Action Toggles -->
  <div class="pt-3 border-t border-[#20202c] space-y-2.5">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      One-Shot Re-Apply Actions
    </span>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
      <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
        <input type="checkbox" bind:checked={optSteamless} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
        <span>Unpack SteamStub DRM automatically</span>
      </label>
      <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
        <input type="checkbox" bind:checked={optAchievements} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
        <span>Fetch achievements & DLC metadata</span>
      </label>
      <label class="flex items-center space-x-2.5 text-slate-300 cursor-pointer select-none">
        <input type="checkbox" bind:checked={optNormalize} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
        <span>Normalize game folder name</span>
      </label>
    </div>
  </div>
</div>
