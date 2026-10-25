<script>
  import InheritSelect from '../../../common/InheritSelect.svelte';
  import SeamlessSwitch from '../../../common/SeamlessSwitch.svelte';

  let { networkDraft = $bindable(), generalDraft = $bindable(), cfgDraft = $bindable() } = $props();

  let targetDraft = $derived(networkDraft || generalDraft || cfgDraft || {});
  let customSpeedMB = $state('');

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

  let speedLimit = $derived(targetDraft.transfer_speed_limit ?? 0);

  let selectedSpeedMode = $derived(standardSpeedPresets.includes(speedLimit)
    ? speedLimit
    : -1);

  $effect(() => {
    if (selectedSpeedMode === -1 && !customSpeedMB && speedLimit > 0) {
      customSpeedMB = String(Math.round(speedLimit / 1048576));
    }
  });

  function setSpeedLimit(bytes) {
    if (networkDraft) networkDraft.transfer_speed_limit = bytes;
    if (generalDraft) generalDraft.transfer_speed_limit = bytes;
    if (cfgDraft) cfgDraft.transfer_speed_limit = bytes;
  }

  function handleSpeedSelect(val) {
    const num = Number(val);
    if (num === -1) {
      if (!customSpeedMB) customSpeedMB = '15';
      setSpeedLimit((parseInt(customSpeedMB, 10) || 15) * 1048576);
    } else {
      setSpeedLimit(num);
    }
  }

  function handleCustomSpeedInput(e) {
    const mb = parseInt(e.target.value, 10);
    customSpeedMB = e.target.value;
    if (!isNaN(mb) && mb >= 0) {
      setSpeedLimit(mb * 1048576);
    }
  }

  function handleNotifyChange(val) {
    if (networkDraft) networkDraft.enable_launch_notify = val;
    if (generalDraft) generalDraft.enable_launch_notify = val;
    if (cfgDraft) cfgDraft.enable_launch_notify = val;
  }
</script>

<div class="bg-surface-card border border-surface-border rounded-2xl p-5 sm:p-6 shadow-sm space-y-4">
  <div class="space-y-0.5">
    <h3 class="text-xs font-bold text-white uppercase tracking-wider font-mono">
      Network &amp; Downloads
    </h3>
    <p class="text-xs text-neutral-400 leading-relaxed">
      Global bandwidth limits and launch notification alerts.
    </p>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
    <div class="space-y-2">
      <InheritSelect
        label="Transfer Speed Limit"
        description="Global bandwidth throttle for game downloads and depot streaming"
        value={selectedSpeedMode}
        options={speedLimitOptions}
        onchange={(val) => handleSpeedSelect(val)}
      />

      {#if selectedSpeedMode === -1}
        <div class="p-2.5 bg-surface-input border border-surface-border rounded-xl flex items-center justify-between gap-3 animate-fade-in">
          <span class="text-xs font-semibold text-neutral-300">Custom Limit (MB/s):</span>
          <div class="flex items-center space-x-1.5">
            <input
              type="number"
              min="1"
              max="1000"
              value={customSpeedMB}
              oninput={handleCustomSpeedInput}
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
      value={targetDraft.enable_launch_notify}
      onchange={handleNotifyChange}
    />
  </div>
</div>
