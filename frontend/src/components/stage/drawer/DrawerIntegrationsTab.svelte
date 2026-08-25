<script>
  export let globalCfg = {};
  export let customOperatingMode = '';
  export let customRunner = '';
  export let customWinePrefix = '';
  export let customMangoHud = null;
  export let customFSR = null;
  export let customNVAPI = null;
  export let customDXVKHud = '';
</script>

<div class="space-y-4">
  <!-- Operating Mode & Runner Controls -->
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 bg-[#09090d] p-3.5 rounded-xl border border-[#202020]">
    <div>
      <label for="drawerOpMode" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
        Primary Operating Mode
      </label>
      <select
        id="drawerOpMode"
        bind:value={customOperatingMode}
        class="w-full bg-[#121218] border border-[#2c2c3e] rounded-xl px-3 py-2 text-slate-100 outline-none font-bold text-xs"
      >
        <option value="">Default ({globalCfg.operating_mode === 'lutris' ? 'Lutris Helper' : 'Standalone Launcher'})</option>
        <option value="standalone">Standalone Launcher Mode (Direct)</option>
        <option value="lutris">Lutris Helper Mode</option>
      </select>
      <p class="text-[10px] text-slate-400 mt-1">
        {customOperatingMode === 'lutris' || (!customOperatingMode && globalCfg.operating_mode === 'lutris') ? 'Delegates launch to Lutris client.' : 'Direct execution via host runner.'}
      </p>
    </div>

    <div>
      <label for="drawerRunner" class="block text-slate-300 font-bold text-[11px] uppercase tracking-wider mb-1">
        Runner Engine
      </label>
      <select
        id="drawerRunner"
        bind:value={customRunner}
        class="w-full bg-[#121218] border border-[#2c2c3e] rounded-xl px-3 py-2 text-slate-100 outline-none font-bold text-xs"
      >
        <option value="">Default ({globalCfg.default_runner ? globalCfg.default_runner.toUpperCase() : 'UMU'})</option>
        <option value="umu">UMU Launcher (umu-run)</option>
        <option value="wine">Direct Wine</option>
        <option value="proton">Direct Proton</option>
        <option value="linux">Linux Native</option>
      </select>
      <p class="text-[10px] text-slate-400 mt-1">Runner used when launching Standalone.</p>
    </div>
  </div>

  <!-- Custom Wine Prefix -->
  <div>
    <label for="drawerPrefix" class="block text-slate-300 font-bold uppercase tracking-wider text-[10px] mb-1">
      Custom Wine Prefix Path
    </label>
    <input
      id="drawerPrefix"
      type="text"
      bind:value={customWinePrefix}
      placeholder="Inherited ({globalCfg.wine_prefix || '~/.local/share/wineprefixes/default'})"
      class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs focus:border-white"
    />
  </div>

  <!-- Performance Overlays & Tri-State Controls -->
  <div class="space-y-3 pt-2">
    <span class="block text-slate-400 font-bold uppercase tracking-wider text-[10px]">
      Graphics & Performance Overlays (Tri-State)
    </span>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <!-- MangoHud -->
      <div class="bg-[#0b0b10] p-3 rounded-xl border border-[#20202e] space-y-1.5">
        <span class="block font-bold text-slate-200 text-xs">MangoHud</span>
        <select
          bind:value={customMangoHud}
          class="w-full bg-[#14141e] border border-[#2c2c3e] rounded-lg px-2 py-1 text-slate-200 outline-none text-xs"
        >
          <option value={null}>Inherited ({globalCfg.mangohud ? 'Enabled' : 'Disabled'})</option>
          <option value={true}>Force Enabled</option>
          <option value={false}>Force Disabled</option>
        </select>
      </div>

      <!-- Wine Fullscreen FSR -->
      <div class="bg-[#0b0b10] p-3 rounded-xl border border-[#20202e] space-y-1.5">
        <span class="block font-bold text-slate-200 text-xs">Wine FSR</span>
        <select
          bind:value={customFSR}
          class="w-full bg-[#14141e] border border-[#2c2c3e] rounded-lg px-2 py-1 text-slate-200 outline-none text-xs"
        >
          <option value={null}>Inherited ({globalCfg.wine_fullscreen_fsr ? 'Enabled' : 'Disabled'})</option>
          <option value={true}>Force Enabled</option>
          <option value={false}>Force Disabled</option>
        </select>
      </div>

      <!-- Proton NVAPI -->
      <div class="bg-[#0b0b10] p-3 rounded-xl border border-[#20202e] space-y-1.5">
        <span class="block font-bold text-slate-200 text-xs">Proton NVAPI</span>
        <select
          bind:value={customNVAPI}
          class="w-full bg-[#14141e] border border-[#2c2c3e] rounded-lg px-2 py-1 text-slate-200 outline-none text-xs"
        >
          <option value={null}>Inherited ({globalCfg.proton_nvapi ? 'Enabled' : 'Disabled'})</option>
          <option value={true}>Force Enabled</option>
          <option value={false}>Force Disabled</option>
        </select>
      </div>
    </div>

    <!-- Custom DXVK HUD -->
    <div>
      <label for="drawerDxvk" class="block text-slate-300 font-bold uppercase tracking-wider text-[10px] mb-1">
        DXVK HUD Flags
      </label>
      <input
        id="drawerDxvk"
        type="text"
        bind:value={customDXVKHud}
        placeholder="Inherited ({globalCfg.dxvk_hud || 'fps,devinfo'})"
        class="w-full bg-[#0c0c10] border border-[#262636] rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs"
      />
    </div>
  </div>
</div>
