<script>
  import { drawerDraft } from '../../../stores/drawerDraft.js';
  import { getOperatingModeDescriptor } from '../../../utils/game.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SettingRow from '../../common/SettingRow.svelte';
  import SwitcherRow from '../../common/SwitcherRow.svelte';

  export let game = null;
  export let globalCfg = {};
  export let languageOptions = [];

  $: defaultDescriptor = getOperatingModeDescriptor(globalCfg.operating_mode || globalCfg.gbe_mode || 'loader');

  $: modeOptions = [
    { value: '', label: `Default (${defaultDescriptor.label})` },
    { value: 'loader', label: 'Seamless Loader (Recommended)' },
    { value: 'portable', label: 'Standalone Portable (Direct Files)' },
    { value: 'none', label: 'Direct Launch (DRM-Free / None)' }
  ];

  $: langOptions = [
    { value: '', label: `Default (${globalCfg.language ? (globalCfg.language === 'auto' ? 'Auto / System' : globalCfg.language) : 'Auto / System'})` },
    ...languageOptions.map(l => ({ value: l, label: l.charAt(0).toUpperCase() + l.slice(1) }))
  ];
</script>

<div class="space-y-4">
  <!-- Steam DRM & Emulation Mode -->
  <SettingSection
    title="Steam DRM & Emulation"
    description="Goldberg Steam emulator mode and language overrides for this title."
  >
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <SettingRow label="Emulation Mode" description="Loader mode preserves original game files." forId="drawerMode" inline={false}>
        <select
          id="drawerMode"
          bind:value={$drawerDraft.mode}
          class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none text-xs"
        >
          {#each modeOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      </SettingRow>

      <SettingRow label="Game Language" description="Language locale passed to Goldberg emulator." forId="drawerLang" inline={false}>
        <select
          id="drawerLang"
          bind:value={$drawerDraft.language}
          class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none capitalize text-xs"
        >
          {#each langOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      </SettingRow>
    </div>

    <!-- AppID Override -->
    <div class="pt-2 border-t border-[#1c1c28]">
      <SettingRow label="AppID Override" description="Steam Application ID used for save directory mapping and achievements." forId="drawerAppId" inline={false}>
        <input
          id="drawerAppId"
          type="text"
          bind:value={$drawerDraft.appId}
          placeholder={game?.appId || '123456'}
          class="w-full bg-[#14141e] border border-[#262638] focus:border-cyan-500 rounded-xl px-3 py-2 text-slate-100 outline-none font-mono text-xs transition-colors"
        />
      </SettingRow>
    </div>
  </SettingSection>

  <!-- One-Shot Setup Actions -->
  <SettingSection
    title="One-Shot Setup Actions"
    description="Automated actions executed when applying emulator configuration."
  >
    <div class="space-y-2">
      <SwitcherRow
        label="Unpack SteamStub DRM"
        description="Automatically run Steamless unpacker on game binaries."
        tag="STEAMLESS"
        color="sky"
        bind:value={$drawerDraft.optSteamless}
      />

      <SwitcherRow
        label="Fetch Achievements & DLC"
        description="Download Steam achievement icons and DLC list into steam_settings."
        tag="ICONS"
        color="amber"
        bind:value={$drawerDraft.optAchievements}
      />

      <SwitcherRow
        label="Normalize Folder Name"
        description="Standardize folder naming to match official Steam title."
        tag="NORMALIZE"
        color="purple"
        bind:value={$drawerDraft.optNormalize}
      />
    </div>
  </SettingSection>
</div>
