<script>
  import { onMount } from 'svelte';
  import SettingSection from '../../common/SettingSection.svelte';
  import GhostInput from '../../common/GhostInput.svelte';
  import InheritSelect from '../../common/InheritSelect.svelte';
  import SeamlessSwitch from '../../common/SeamlessSwitch.svelte';
  import { multiseatStore, resolveGamepadAssignments } from '../../../stores/multiseat.js';
  import { AppAPI } from '../../../api/app.js';
  import { generateRandomSteamID as createRandomSteamID } from '../../../utils/steam.js';
  import SteamPersonaCard from './SteamPersonaCard.svelte';

  export let cfgDraft = {
    multiseat: {
      player2_account_name: 'PlayerTwo',
      player2_account_steamid: '76561198998877665',
      player2_language: 'english',
      player2_ip_country: 'US',
      player2_wine_prefix: '',
      default_seat1_mode: 'local',
      default_seat2_mode: 'local',
      default_layout: 'dual_monitor',
      broadcast_relay_enabled: true,
      seat1_gamepad: 'auto',
      seat2_gamepad: 'auto',
      seat1_keyboard: 'auto',
      seat2_keyboard: 'auto',
      seat1_mouse: 'auto',
      seat2_mouse: 'auto',
      seat2_force_grab_cursor: false,
      seat2_grab_keyboard: false,
      seat2_show_cursor: true,
      seat2_compositor: 'auto',
      seat1_display_index: 0,
      seat2_display_index: 1,
      sunshine_port_p1: 47989,
      sunshine_port_p2: 48010,
    }
  };

  export let userCfgDraft = {
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US'
  };

  export let languagesList = [];
  export let countriesList = [];

  // Ensure nested multiseat defaults exist
  $: if (!cfgDraft.multiseat) {
    cfgDraft.multiseat = {};
  }
  $: if (cfgDraft.multiseat.player2_account_name === undefined) cfgDraft.multiseat.player2_account_name = 'PlayerTwo';
  $: if (cfgDraft.multiseat.player2_account_steamid === undefined) cfgDraft.multiseat.player2_account_steamid = '76561198998877665';
  $: if (cfgDraft.multiseat.player2_language === undefined) cfgDraft.multiseat.player2_language = 'english';
  $: if (cfgDraft.multiseat.player2_ip_country === undefined) cfgDraft.multiseat.player2_ip_country = 'US';
  $: if (cfgDraft.multiseat.player2_wine_prefix === undefined) cfgDraft.multiseat.player2_wine_prefix = '';
  $: if (cfgDraft.multiseat.default_seat1_mode === undefined) cfgDraft.multiseat.default_seat1_mode = 'local';
  $: if (cfgDraft.multiseat.default_seat2_mode === undefined) cfgDraft.multiseat.default_seat2_mode = 'local';
  $: if (cfgDraft.multiseat.default_layout === undefined) cfgDraft.multiseat.default_layout = 'dual_monitor';
  $: if (cfgDraft.multiseat.broadcast_relay_enabled === undefined) cfgDraft.multiseat.broadcast_relay_enabled = true;
  $: if (cfgDraft.multiseat.seat1_gamepad === undefined) cfgDraft.multiseat.seat1_gamepad = 'auto';
  $: if (cfgDraft.multiseat.seat2_gamepad === undefined) cfgDraft.multiseat.seat2_gamepad = 'auto';
  $: if (cfgDraft.multiseat.seat1_keyboard === undefined) cfgDraft.multiseat.seat1_keyboard = 'auto';
  $: if (cfgDraft.multiseat.seat2_keyboard === undefined) cfgDraft.multiseat.seat2_keyboard = 'auto';
  $: if (cfgDraft.multiseat.seat1_mouse === undefined) cfgDraft.multiseat.seat1_mouse = 'auto';
  $: if (cfgDraft.multiseat.seat2_mouse === undefined) cfgDraft.multiseat.seat2_mouse = 'auto';
  $: if (cfgDraft.multiseat.seat2_force_grab_cursor === undefined) cfgDraft.multiseat.seat2_force_grab_cursor = false;
  $: if (cfgDraft.multiseat.seat2_grab_keyboard === undefined) cfgDraft.multiseat.seat2_grab_keyboard = false;
  $: if (cfgDraft.multiseat.seat2_show_cursor === undefined) cfgDraft.multiseat.seat2_show_cursor = true;
  $: if (cfgDraft.multiseat.seat2_compositor === undefined || cfgDraft.multiseat.seat2_compositor === '') cfgDraft.multiseat.seat2_compositor = 'auto';
  $: if (cfgDraft.multiseat.seat1_display_index === undefined) cfgDraft.multiseat.seat1_display_index = 0;
  $: if (cfgDraft.multiseat.seat2_display_index === undefined) cfgDraft.multiseat.seat2_display_index = 1;
  $: if (cfgDraft.multiseat.sunshine_port_p1 === undefined) cfgDraft.multiseat.sunshine_port_p1 = 47989;
  $: if (cfgDraft.multiseat.sunshine_port_p2 === undefined) cfgDraft.multiseat.sunshine_port_p2 = 48010;

  onMount(() => {
    multiseatStore.loadCapabilities();
    multiseatStore.refreshGamepads();
    multiseatStore.refreshKeyboards();
    multiseatStore.refreshMice();
  });

  $: caps = $multiseatStore.capabilities;
  $: hasBwrap = Boolean(caps?.has_bwrap ?? caps?.hasBwrap ?? caps?.HasBwrap);
  $: hasPasta = Boolean(caps?.has_pasta ?? caps?.hasPasta ?? caps?.HasPasta);
  $: hasGamescope = Boolean(caps?.has_gamescope ?? caps?.hasGamescope ?? caps?.HasGamescope);
  $: hasCage = Boolean(caps?.has_cage ?? caps?.hasCage ?? caps?.HasCage);
  $: hasSunshine = Boolean(caps?.has_sunshine ?? caps?.hasSunshine ?? caps?.HasSunshine);
  $: hasPipewire = Boolean(caps?.has_pipewire ?? caps?.hasPipewire ?? caps?.HasPipewire);

  $: gamepads = $multiseatStore.gamepads;

  $: gamepadAssignments = resolveGamepadAssignments(
    gamepads,
    cfgDraft.multiseat?.seat1_gamepad || 'auto',
    cfgDraft.multiseat?.seat2_gamepad || 'auto'
  );

  $: gamepadOptions = [
    { value: 'auto', label: 'Auto (Smart Allocation)' },
    { value: 'none', label: 'Keyboard & Mouse Only (No Gamepad)' },
    ...(gamepads || []).map(pad => ({
      value: pad.dev_node,
      label: `${pad.name} (${pad.dev_node})`
    }))
  ];

  $: keyboards = $multiseatStore.keyboards;
  $: mice = $multiseatStore.mice;

  $: keyboardOptions = [
    { value: 'auto', label: 'Auto (Shared / Host Default)' },
    { value: 'none', label: 'None (Mask All Keyboards)' },
    ...(keyboards || []).map(k => ({
      value: k.dev_node,
      label: `${k.name} (${k.dev_node})`
    }))
  ];

  $: mouseOptions = [
    { value: 'auto', label: 'Auto (Shared / Host Default)' },
    { value: 'none', label: 'None (Mask All Mice)' },
    ...(mice || []).map(m => ({
      value: m.dev_node,
      label: `${m.name} (${m.dev_node})`
    }))
  ];

  const layoutOptions = [
    { value: 'dual_monitor', label: 'Dual Monitor (Auto Display 0 & 1)' },
    { value: 'side_by_side', label: 'Side-by-Side (Left / Right - Vertical Split)' },
    { value: 'top_bottom', label: 'Top / Bottom (Horizontal Split)' },
    { value: 'windowed', label: 'Windowed (Floating Windows)' }
  ];

  function quickSetupPlayer2() {
    cfgDraft.multiseat = {
      ...cfgDraft.multiseat,
      player2_account_name: (userCfgDraft?.account_name || 'Player') + '-Coop',
      player2_account_steamid: createRandomSteamID(),
      player2_language: userCfgDraft?.language || 'english',
      player2_ip_country: userCfgDraft?.ip_country || 'US'
    };
    cfgDraft = { ...cfgDraft };
  }

  async function handleBrowsePlayer2WinePrefix() {
    try {
      const selected = await AppAPI.selectDirectory('Select Player 2 Wine Prefix Directory', cfgDraft.multiseat?.player2_wine_prefix || '');
      if (selected) {
        cfgDraft.multiseat.player2_wine_prefix = selected;
        cfgDraft = { ...cfgDraft };
      }
    } catch (err) {
      console.warn('Failed to select directory:', err);
    }
  }
</script>

<div class="space-y-6">
  <!-- 1. System Capabilities & Diagnostics -->
  <SettingSection title="System Diagnostics & Isolation Tools">
    <div class="space-y-3">
      <p class="text-xs text-neutral-400">
        Host system tools used for sandboxing, screen composition, audio partitioning, and networking.
      </p>

      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 pt-1">
        <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
          <div class="space-y-0.5">
            <div class="text-xs font-bold text-neutral-200">Bubblewrap</div>
            <div class="text-3xs text-neutral-400 font-mono">bwrap</div>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasBwrap ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
            {hasBwrap ? 'Available' : 'Missing'}
          </span>
        </div>

        <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
          <div class="space-y-0.5">
            <div class="text-xs font-bold text-neutral-200">Pasta (NetNS)</div>
            <div class="text-3xs text-neutral-400 font-mono">pasta</div>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasPasta ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
            {hasPasta ? 'Available' : 'Missing'}
          </span>
        </div>

        <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
          <div class="space-y-0.5">
            <div class="text-xs font-bold text-neutral-200">Gamescope</div>
            <div class="text-3xs text-neutral-400 font-mono">gamescope</div>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasGamescope ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
            {hasGamescope ? 'Available' : 'Missing'}
          </span>
        </div>

        <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
          <div class="space-y-0.5">
            <div class="text-xs font-bold text-neutral-200">Cage (Wayland)</div>
            <div class="text-3xs text-neutral-400 font-mono">cage</div>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasCage ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
            {hasCage ? 'Available' : 'Optional'}
          </span>
        </div>

        <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
          <div class="space-y-0.5">
            <div class="text-xs font-bold text-neutral-200">Sunshine</div>
            <div class="text-3xs text-neutral-400 font-mono">sunshine</div>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasSunshine ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
            {hasSunshine ? 'Available' : 'Optional'}
          </span>
        </div>

        <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
          <div class="space-y-0.5">
            <div class="text-xs font-bold text-neutral-200">PipeWire</div>
            <div class="text-3xs text-neutral-400 font-mono">pactl</div>
          </div>
          <span class="px-2 py-0.5 rounded text-3xs font-mono font-bold {hasPipewire ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-neutral-800 text-neutral-400 border border-neutral-700'}">
            {hasPipewire ? 'Available' : 'Missing'}
          </span>
        </div>
      </div>

      {#if !hasBwrap || !hasPasta}
        <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl text-xs text-amber-300 flex items-start space-x-2.5">
          <svg class="w-4 h-4 flex-shrink-0 mt-0.5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="space-y-1">
            <div class="font-bold text-amber-200">Recommended Packages Missing</div>
            <div class="text-2xs text-amber-300/90 leading-relaxed">
              Install <code class="font-mono bg-amber-500/20 px-1 py-0.5 rounded">{[!hasBwrap && 'bubblewrap', !hasPasta && 'passt'].filter(Boolean).join(' and ')}</code> via your package manager for complete filesystem sandboxing and rootless network isolation between player seats.
            </div>
          </div>
        </div>
      {/if}
    </div>
  </SettingSection>

  <!-- 2. Player 2 (Guest Persona) Profile -->
  <SettingSection title="Player 2 Profile (Co-op Guest)">
    <div class="space-y-3">
      <div class="flex items-center justify-between pb-2 border-b border-surface-border/50">
        <p class="text-xs text-neutral-400">
          Distinct profile injected into Player 2's isolated Bubblewrap sandbox with separate save files, achievements, and Steamworks identity.
        </p>
        <button
          type="button"
          on:click={quickSetupPlayer2}
          class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
          title="Clone Player 1 locale with a distinct SteamID and Guest nickname"
        >
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          <span>Quick Setup</span>
        </button>
      </div>

      <SteamPersonaCard
        bind:accountName={cfgDraft.multiseat.player2_account_name}
        bind:accountSteamId={cfgDraft.multiseat.player2_account_steamid}
        bind:language={cfgDraft.multiseat.player2_language}
        bind:ipCountry={cfgDraft.multiseat.player2_ip_country}
        collisionSteamId={userCfgDraft?.account_steamid || ''}
        nameLabel="Guest Display Name"
        nameDescription="Persona nickname for the second player in split-screen or streaming sessions"
        namePlaceholder="PlayerTwo"
        langLabel="Guest Language"
        langDescription="Language injected into Player 2's Goldberg Steamworks emulation"
        steamidLabel="Guest Steam ID (64-Bit)"
        steamidDescription="Must differ from Player 1 to avoid Steamworks lobby collisions"
        steamidPlaceholder="76561198998877665"
        countryLabel="Guest Country Code"
        countryDescription="Two-letter ISO country code reported by Player 2's emulator"
        syncTitle="Auto-Sync Player 2 Profile"
        syncSubtitle="Enter Player 2's Steam vanity URL or SteamID64 to sync co-op persona and avatar"
        {languagesList}
        {countriesList}
        on:change={() => { cfgDraft = { ...cfgDraft }; }}
      >
        <div class="space-y-1">
          <GhostInput
            label="Guest Wine Prefix"
            description="Default isolated Wine/Proton prefix directory for Player 2 (defaults to ~/Games/umu-p2)"
            bind:value={cfgDraft.multiseat.player2_wine_prefix}
            placeholder="~/Games/umu-p2"
          >
            <button
              slot="action"
              type="button"
              on:click={handleBrowsePlayer2WinePrefix}
              class="px-3 py-1.5 bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-200 hover:text-white rounded-lg text-xs font-semibold transition-colors flex items-center space-x-1.5 cursor-pointer flex-shrink-0"
              title="Browse for custom Player 2 Wine prefix directory"
            >
              <span>Browse</span>
            </button>
          </GhostInput>
        </div>
      </SteamPersonaCard>
    </div>
  </SettingSection>

  <!-- 3. Screen & Split Layout Profile -->
  <SettingSection title="Display & Screen Layout">
    <div class="space-y-3">
      <p class="text-xs text-neutral-400">
        Controls how Gamescope or Cage maps output video to physical displays or virtual split-screen geometry. Local split layouts automatically partition your Gamescope screen resolution 1:1 between seats.
      </p>

      <div class="space-y-2.5">
        <InheritSelect
          label="Default Split-Screen Layout"
          description="Layout applied when launching games in 2-player local co-op mode"
          bind:value={cfgDraft.multiseat.default_layout}
          options={layoutOptions}
        />

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
          <div class="space-y-1.5">
            <span class="block text-xs font-bold text-neutral-300">Player 1 Default Output</span>
            <div class="grid grid-cols-2 gap-1.5 bg-surface-input p-1 rounded-xl border border-surface-border">
              <button
                type="button"
                on:click={() => { cfgDraft.multiseat.default_seat1_mode = 'local'; cfgDraft = { ...cfgDraft }; }}
                class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {cfgDraft.multiseat.default_seat1_mode === 'local' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
              >
                Local Screen
              </button>
              <button
                type="button"
                on:click={() => { cfgDraft.multiseat.default_seat1_mode = 'stream'; cfgDraft = { ...cfgDraft }; }}
                class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {cfgDraft.multiseat.default_seat1_mode === 'stream' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
              >
                Sunshine Stream
              </button>
            </div>
          </div>

          <div class="space-y-1.5">
            <span class="block text-xs font-bold text-neutral-300">Player 2 Default Output</span>
            <div class="grid grid-cols-2 gap-1.5 bg-surface-input p-1 rounded-xl border border-surface-border">
              <button
                type="button"
                on:click={() => { cfgDraft.multiseat.default_seat2_mode = 'local'; cfgDraft = { ...cfgDraft }; }}
                class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {cfgDraft.multiseat.default_seat2_mode === 'local' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
              >
                Local Screen
              </button>
              <button
                type="button"
                on:click={() => { cfgDraft.multiseat.default_seat2_mode = 'stream'; cfgDraft = { ...cfgDraft }; }}
                class="py-1.5 text-xs font-bold rounded-lg transition text-center cursor-pointer {cfgDraft.multiseat.default_seat2_mode === 'stream' ? 'bg-white text-black shadow-sm' : 'text-neutral-400 hover:text-white'}"
              >
                Sunshine Stream
              </button>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 pt-1">
          <div>
            <label for="seat1-disp-idx" class="block text-2xs font-bold text-neutral-300 mb-1">Player 1 Monitor Index</label>
            <input
              id="seat1-disp-idx"
              type="number"
              min="0"
              max="10"
              bind:value={cfgDraft.multiseat.seat1_display_index}
              class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
            />
          </div>
          <div>
            <label for="seat2-disp-idx" class="block text-2xs font-bold text-neutral-300 mb-1">Player 2 Monitor Index</label>
            <input
              id="seat2-disp-idx"
              type="number"
              min="0"
              max="10"
              bind:value={cfgDraft.multiseat.seat2_display_index}
              class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
            />
          </div>
        </div>
      </div>
    </div>
  </SettingSection>

  <!-- 4. Controller & Gamepad Diagnostics -->
  <SettingSection title="Host Controllers & Gamepads">
    <div class="space-y-4">
      <div class="flex items-center justify-between pb-2 border-b border-surface-border/50">
        <p class="text-xs text-neutral-400">
          Detected gamepads on host system. In Co-Op mode, controllers are dynamically masked and allocated to ensure zero input bleed between instances.
        </p>
        <button
          type="button"
          on:click={() => multiseatStore.refreshGamepads()}
          class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
          title="Rescan host input devices"
        >
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Rescan</span>
        </button>
      </div>

      <!-- Controller Allocation Selectors -->
      <div class="grid grid-cols-2 gap-3 p-3 bg-surface-card border border-surface-border rounded-xl">
        <div class="space-y-1.5">
          <label for="seat1-gamepad-select" class="block text-2xs font-bold text-neutral-300">Player 1 Controller</label>
          <select
            id="seat1-gamepad-select"
            bind:value={cfgDraft.multiseat.seat1_gamepad}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            {#each gamepadOptions as opt}
              <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
            {/each}
          </select>
          <p class="text-3xs text-neutral-400 leading-relaxed">
            {#if cfgDraft.multiseat.seat1_gamepad === 'none'}
              Forces Keyboard &amp; Mouse. All host gamepads are masked from Player 1.
            {:else if cfgDraft.multiseat.seat1_gamepad === 'auto'}
              Auto-allocates controller if available; falls back to Keyboard &amp; Mouse.
            {:else}
              Assigned to selected controller.
            {/if}
          </p>
        </div>

        <div class="space-y-1.5">
          <label for="seat2-gamepad-select" class="block text-2xs font-bold text-neutral-300">Player 2 Controller</label>
          <select
            id="seat2-gamepad-select"
            bind:value={cfgDraft.multiseat.seat2_gamepad}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            {#each gamepadOptions as opt}
              <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
            {/each}
          </select>
          <p class="text-3xs text-neutral-400 leading-relaxed">
            {#if cfgDraft.multiseat.seat2_gamepad === 'none'}
              Forces Keyboard &amp; Mouse. All host gamepads are masked from Player 2.
            {:else if cfgDraft.multiseat.seat2_gamepad === 'auto'}
              Auto-allocates dedicated controller to Player 2.
            {:else}
              Assigned to selected controller.
            {/if}
          </p>
        </div>
      </div>

      <!-- Detected Gamepads List -->
      {#if gamepads && gamepads.length > 0}
        <div class="space-y-2">
          {#each gamepads as pad}
            {@const isP1 = gamepadAssignments.p1Dev === pad.dev_node}
            {@const isP2 = gamepadAssignments.p2Dev === pad.dev_node}
            <div class="p-2.5 bg-surface-card border border-surface-border rounded-xl flex items-center justify-between">
              <div class="flex items-center space-x-2.5">
                <svg class="w-4 h-4 text-neutral-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
                </svg>
                <div class="space-y-0.5">
                  <div class="text-xs font-bold text-neutral-200">{pad.name}</div>
                  <div class="text-3xs text-neutral-400 font-mono">{pad.dev_node} ({pad.bus})</div>
                </div>
              </div>
              <div>
                {#if isP1}
                  <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-white text-black">
                    Player 1{cfgDraft.multiseat.seat1_gamepad === 'auto' ? ' (Auto)' : ''}
                  </span>
                {:else if isP2}
                  <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-white text-black">
                    Player 2{cfgDraft.multiseat.seat2_gamepad === 'auto' ? ' (Auto)' : ''}
                  </span>
                {:else}
                  <span class="text-3xs font-mono font-bold px-2 py-0.5 rounded bg-surface-input border border-surface-border text-neutral-400">
                    Unassigned
                  </span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="p-3 bg-surface-card/60 border border-surface-border rounded-xl text-2xs text-neutral-400">
          No physical gamepads detected. Connect USB or Bluetooth controllers before launching co-op sessions to isolate player inputs.
        </div>
      {/if}
    </div>
  </SettingSection>

  <!-- 5. Dual Keyboards & Mice (Games Without Gamepad Support) -->
  <SettingSection title="Dual Keyboards & Mice (Experimental)">
    <div class="space-y-4">
      <div class="flex items-center justify-between pb-2 border-b border-surface-border/50">
        <p class="text-xs text-neutral-400 leading-relaxed">
          Enables 2-player local co-op for games lacking controller support. Player 2's devices are grabbed exclusively via evdev (<code class="font-mono text-neutral-300">EVIOCGRAB</code>) and injected directly into Gamescope via libei (<code class="font-mono text-neutral-300">gamescope-*-ei</code>). Gamescope with EIS redirection is auto-selected when dedicated KBM is configured.
        </p>
        <button
          type="button"
          on:click={() => { multiseatStore.refreshKeyboards(); multiseatStore.refreshMice(); }}
          class="px-3 py-1 bg-surface-header hover:bg-surface-card border border-surface-border text-neutral-300 hover:text-white rounded-lg text-2xs font-bold transition flex items-center space-x-1.5 flex-shrink-0 cursor-pointer"
          title="Rescan connected keyboards and mice"
        >
          <svg class="w-3 h-3 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Rescan KBM</span>
        </button>
      </div>

      <!-- Keyboard & Mouse Allocation Selectors -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-surface-card border border-surface-border rounded-xl">
        <div class="space-y-1.5">
          <label for="seat1-kbd-select" class="block text-2xs font-bold text-neutral-300">Player 1 Keyboard</label>
          <select
            id="seat1-kbd-select"
            bind:value={cfgDraft.multiseat.seat1_keyboard}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            {#each keyboardOptions as opt}
              <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
            {/each}
          </select>
        </div>

        <div class="space-y-1.5">
          <label for="seat2-kbd-select" class="block text-2xs font-bold text-neutral-300">Player 2 Keyboard</label>
          <select
            id="seat2-kbd-select"
            bind:value={cfgDraft.multiseat.seat2_keyboard}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            {#each keyboardOptions as opt}
              <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
            {/each}
          </select>
        </div>

        <div class="space-y-1.5 pt-1">
          <label for="seat1-mouse-select" class="block text-2xs font-bold text-neutral-300">Player 1 Mouse</label>
          <select
            id="seat1-mouse-select"
            bind:value={cfgDraft.multiseat.seat1_mouse}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            {#each mouseOptions as opt}
              <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
            {/each}
          </select>
        </div>

        <div class="space-y-1.5 pt-1">
          <label for="seat2-mouse-select" class="block text-2xs font-bold text-neutral-300">Player 2 Mouse</label>
          <select
            id="seat2-mouse-select"
            bind:value={cfgDraft.multiseat.seat2_mouse}
            on:change={() => { cfgDraft = { ...cfgDraft }; }}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white outline-none focus:border-neutral-400 cursor-pointer"
          >
            {#each mouseOptions as opt}
              <option value={opt.value} class="bg-surface-base text-neutral-200">{opt.label}</option>
            {/each}
          </select>
        </div>
      </div>

      <!-- Player 2 Cursor & Gamescope Experimentation Toggles -->
      <div class="p-3 bg-surface-card border border-surface-border rounded-xl space-y-3">
        <div class="text-xs font-bold text-neutral-200">Player 2 Pointer &amp; Compositor Options</div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
            <div class="space-y-0.5 max-w-[200px]">
              <div class="text-2xs font-bold text-neutral-200">Force Grab Cursor</div>
              <div class="text-3xs text-neutral-400">Relative mouse mode for 3D/FPS camera control (--force-grab-cursor)</div>
            </div>
            <SeamlessSwitch
              bind:value={cfgDraft.multiseat.seat2_force_grab_cursor}
              tooltip="Force Grab Cursor"
              on:change={(e) => { cfgDraft.multiseat.seat2_force_grab_cursor = e.detail; cfgDraft = { ...cfgDraft }; }}
            />
          </div>

          <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
            <div class="space-y-0.5 max-w-[200px]">
              <div class="text-2xs font-bold text-neutral-200">Grab Keyboard (-g)</div>
              <div class="text-3xs text-neutral-400">Captures all keypresses inside the window to prevent desktop shortcuts</div>
            </div>
            <SeamlessSwitch
              bind:value={cfgDraft.multiseat.seat2_grab_keyboard}
              tooltip="Grab Keyboard"
              on:change={(e) => { cfgDraft.multiseat.seat2_grab_keyboard = e.detail; cfgDraft = { ...cfgDraft }; }}
            />
          </div>

          <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
            <div class="space-y-0.5 max-w-[200px]">
              <div class="text-2xs font-bold text-neutral-200">Software Cursor</div>
              <div class="text-3xs text-neutral-400">Enforces visible pointer via WLR_NO_HARDWARE_CURSORS=1</div>
            </div>
            <SeamlessSwitch
              bind:value={cfgDraft.multiseat.seat2_show_cursor}
              tooltip="Software Cursor"
              on:change={(e) => { cfgDraft.multiseat.seat2_show_cursor = e.detail; cfgDraft = { ...cfgDraft }; }}
            />
          </div>

          <div class="flex items-center justify-between p-2.5 bg-surface-input rounded-lg border border-surface-border">
            <div class="space-y-0.5 max-w-[170px]">
              <div class="text-2xs font-bold text-neutral-200">Compositor Backend</div>
              <div class="text-3xs text-neutral-400 leading-relaxed">Gamescope EIS auto-selected for dual KBM virtual input</div>
            </div>
            <select
              bind:value={cfgDraft.multiseat.seat2_compositor}
              on:change={() => { cfgDraft = { ...cfgDraft }; }}
              class="bg-surface-base border border-surface-border rounded-lg px-2 py-1 text-2xs text-white outline-none focus:border-neutral-400 cursor-pointer"
            >
              <option value="auto">Auto (Gamescope EIS, Cage fallback)</option>
              <option value="gamescope">Gamescope (Direct EIS)</option>
              <option value="cage">Cage (Standalone Kiosk)</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  </SettingSection>

  <!-- 5. Networking & Remote Streaming -->
  <SettingSection title="Networking & Sunshine Streaming">
    <div class="space-y-3">
      <div class="flex items-center justify-between p-3 bg-surface-card border border-surface-border rounded-xl">
        <div class="space-y-0.5 max-w-md">
          <div class="text-xs font-bold text-neutral-200">UDP Broadcast Relay (Port 47584)</div>
          <div class="text-2xs text-neutral-400 leading-relaxed">
            Forwards LAN discovery packets between Player 1 and Player 2 for Goldberg Steamworks lobby auto-detection.
          </div>
        </div>
        <SeamlessSwitch
          bind:value={cfgDraft.multiseat.broadcast_relay_enabled}
          tooltip="UDP Broadcast Relay"
          on:change={(e) => { cfgDraft.multiseat.broadcast_relay_enabled = e.detail; cfgDraft = { ...cfgDraft }; }}
        />
      </div>

      <div class="grid grid-cols-2 gap-3 pt-1">
        <div>
          <label for="sunshine-port-p1-cfg" class="block text-2xs font-bold text-neutral-300 mb-1">Player 1 Sunshine Port</label>
          <input
            id="sunshine-port-p1-cfg"
            type="number"
            bind:value={cfgDraft.multiseat.sunshine_port_p1}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
          />
        </div>
        <div>
          <label for="sunshine-port-p2-cfg" class="block text-2xs font-bold text-neutral-300 mb-1">Player 2 Sunshine Port</label>
          <input
            id="sunshine-port-p2-cfg"
            type="number"
            bind:value={cfgDraft.multiseat.sunshine_port_p2}
            class="w-full bg-surface-input border border-surface-border rounded-xl px-3 py-1.5 text-xs text-white font-mono outline-none focus:border-neutral-400"
          />
        </div>
      </div>
    </div>
  </SettingSection>

  <!-- 6. Audio Architecture -->
  <SettingSection title="Audio Routing & PipeWire">
    <div class="space-y-2">
      <p class="text-xs text-neutral-400 leading-relaxed">
        Co-Op mode automatically allocates isolated PipeWire null sinks (<code class="font-mono text-neutral-300 bg-surface-input px-1.5 py-0.5 rounded border border-surface-border">gamux-p1</code> and <code class="font-mono text-neutral-300 bg-surface-input px-1.5 py-0.5 rounded border border-surface-border">gamux-p2</code>). Games receive dedicated pulse sinks via sandbox environment variables to keep audio streams separated between player headsets or remote Moonlight streams.
      </p>
    </div>
  </SettingSection>
</div>
