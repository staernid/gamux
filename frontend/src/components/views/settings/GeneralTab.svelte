<script>
  import { configStore } from '../../../stores/config.js';
  import {
    isSfxEnabled,
    setSfxEnabled,
    getSfxVolume,
    setSfxVolume,
    isMuteInBackground,
    setMuteInBackground,
    playSelect
  } from '../../../utils/audio.js';
  import SettingSection from '../../common/SettingSection.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import { UI_SCALE_OPTIONS, LAUNCH_BEHAVIOR_OPTIONS } from '../../../constants/options.js';

  export let cfgDraft;

  const currentUiScale = configStore.uiScale;
  let customSpeedMB = '';

  let sfxEnabled = isSfxEnabled();
  let sfxVolume = getSfxVolume();
  let sfxMuteInBackground = isMuteInBackground();

  const uiScaleOptions = UI_SCALE_OPTIONS;
  const launchBehaviorOptions = LAUNCH_BEHAVIOR_OPTIONS;

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

  function handleSfxToggle(e) {
    setSfxEnabled(e.detail);
    sfxEnabled = isSfxEnabled();
  }

  function handleMuteBgToggle(e) {
    setMuteInBackground(e.detail);
    sfxMuteInBackground = isMuteInBackground();
  }

  function handleVolumeInput(e) {
    const vol = parseFloat(e.target.value);
    setSfxVolume(vol);
    sfxVolume = getSfxVolume();
  }

  function handleTestSound() {
    playSelect();
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

  <!-- Audio & Sound Effects -->
  <SettingSection title="Audio & Sound Effects">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
      <SeamlessSwitch
        label="UI Sound Effects"
        description="Synthesized mechanical clicks and chimes for navigation and interactions"
        value={sfxEnabled}
        on:change={handleSfxToggle}
      />

      <SeamlessSwitch
        label="Mute in Background"
        description="Automatically silence UI audio when the Gamux window loses focus"
        value={sfxMuteInBackground}
        on:change={handleMuteBgToggle}
      />
    </div>

    {#if sfxEnabled}
      <div class="p-2.5 bg-[#101013] border border-surface-border rounded-xl flex items-center justify-between gap-3 animate-fade-in mt-2.5">
        <div class="space-y-0.5 min-w-0">
          <span class="text-xs font-semibold text-neutral-200">Sound Effects Volume</span>
          <p class="text-2xs text-neutral-400">Master output volume for synthetic interface sounds</p>
        </div>
        <div class="flex items-center space-x-2.5 flex-shrink-0">
          <span class="text-xs font-mono text-neutral-300 w-10 text-right">{Math.round(sfxVolume * 100)}%</span>
          <input
            type="range"
            min="0"
            max="1"
            step="0.05"
            value={sfxVolume}
            on:input={handleVolumeInput}
            class="w-24 sm:w-32 accent-white cursor-pointer"
            aria-label="Sound Effects Volume"
          />
          <button
            type="button"
            on:click={handleTestSound}
            class="px-2.5 py-1 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors cursor-pointer"
            title="Play test audio chime"
          >
            Test
          </button>
        </div>
      </div>
    {/if}
  </SettingSection>

  <!-- Process Launch & Window Behavior -->
  <SettingSection title="Process Launch & Window Behavior">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
      <InheritSelect
        label="Launch Window Behavior"
        description="Gamux window visibility while an external game process is running"
        bind:value={cfgDraft.launch_window_behavior}
        options={launchBehaviorOptions}
      />

      <SeamlessSwitch
        label="Close to System Tray"
        description="Minimize Gamux to the background instead of quitting when closing the window"
        bind:value={cfgDraft.close_to_tray}
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
          <div class="p-2.5 bg-[#101013] border border-surface-border rounded-xl flex items-center justify-between gap-3 animate-fade-in">
            <span class="text-xs font-semibold text-neutral-300">Custom Limit (MB/s):</span>
            <div class="flex items-center space-x-1.5">
              <input
                type="number"
                min="1"
                max="1000"
                value={customSpeedMB}
                on:input={handleCustomSpeedInput}
                placeholder="15"
                class="w-20 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-2.5 py-1 text-xs text-white font-mono text-right outline-none"
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
