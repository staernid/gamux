<script>
  import { configStore } from '../../../stores/config.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';

  export let cfgDraft;

  const currentUiScale = configStore.uiScale;
  let customSpeedMB = '';

  const uiScaleOptions = [
    { value: 14, label: '14px — Compact' },
    { value: 16, label: '16px — Standard' },
    { value: 18, label: '18px — Balanced (Default)' },
    { value: 20, label: '20px — Handheld (Deck)' },
    { value: 24, label: '24px — Large (TV / Couch)' }
  ];

  const standardSpeedPresets = [0, 5242880, 10485760, 26214400, 52428800, 104857600];

  const speedLimitOptions = [
    { value: 0, label: 'Unlimited (Maximum Speed)' },
    { value: 5242880, label: '5 MB/s' },
    { value: 10485760, label: '10 MB/s' },
    { value: 26214400, label: '25 MB/s' },
    { value: 52428800, label: '50 MB/s' },
    { value: 104857600, label: '100 MB/s' },
    { value: -1, label: 'Custom Limit (MB/s)...' }
  ];

  $: selectedSpeedMode = standardSpeedPresets.includes(cfgDraft.transfer_speed_limit)
    ? cfgDraft.transfer_speed_limit
    : -1;

  $: if (selectedSpeedMode === -1 && !customSpeedMB && cfgDraft.transfer_speed_limit > 0) {
    customSpeedMB = String(Math.round(cfgDraft.transfer_speed_limit / 1048576));
  }

  function handleSpeedSelect(val) {
    const num = Number(val);
    if (num === -1) {
      if (!customSpeedMB) customSpeedMB = '15';
      cfgDraft.transfer_speed_limit = (parseInt(customSpeedMB, 10) || 15) * 1048576;
    } else {
      cfgDraft.transfer_speed_limit = num;
    }
    cfgDraft = { ...cfgDraft };
  }

  function handleCustomSpeedInput(e) {
    const mb = parseInt(e.target.value, 10);
    customSpeedMB = e.target.value;
    if (!isNaN(mb) && mb >= 0) {
      cfgDraft.transfer_speed_limit = mb * 1048576;
      cfgDraft = { ...cfgDraft };
    }
  }

  function handleScaleChange(size) {
    configStore.setScale(size);
  }
</script>

<div class="space-y-3">
  <!-- Display & Interface -->
  <SettingSection title="Display & Interface">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
      <InheritSelect
        label="Interface Scale"
        description="Base UI scale and font sizing for handhelds, desktops, or TVs (applies immediately)"
        value={$currentUiScale}
        options={uiScaleOptions}
        on:change={(e) => handleScaleChange(e.detail)}
      />
    </div>
  </SettingSection>

  <!-- Network & Downloads -->
  <SettingSection title="Network & Downloads">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
      <div class="space-y-2">
        <InheritSelect
          label="Transfer Speed Limit"
          description="Global bandwidth throttle for game downloads and depot streaming"
          value={selectedSpeedMode}
          options={speedLimitOptions}
          on:change={(e) => handleSpeedSelect(e.detail)}
        />

        {#if selectedSpeedMode === -1}
          <div class="p-2.5 bg-[#101013] border border-[#27272a] rounded-xl flex items-center justify-between gap-3 animate-fade-in">
            <span class="text-xs font-semibold text-neutral-300">Custom Limit (MB/s):</span>
            <div class="flex items-center space-x-1.5">
              <input
                type="number"
                min="1"
                max="1000"
                value={customSpeedMB}
                on:input={handleCustomSpeedInput}
                placeholder="15"
                class="w-20 bg-[#141417] border border-[#27272a] focus:border-neutral-400 rounded-lg px-2.5 py-1 text-xs text-white font-mono text-right outline-none"
              />
              <span class="text-xs font-mono text-neutral-400">MB/s</span>
            </div>
          </div>
        {/if}
      </div>

      <SeamlessSwitch
        label="Launch Notifications"
        description="Display system toast alert when a game runner process initializes"
        bind:value={cfgDraft.enable_launch_notify}
      />
    </div>
  </SettingSection>
</div>
