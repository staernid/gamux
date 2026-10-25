import { AppAPI } from '../../api/app.js';

export class DevicesState {
  capabilities = $state({
    hasBwrap: true,
    hasGamescope: true,
    hasCage: false,
    hasSunshine: true,
    hasPipewire: true,
    hasInputPermissions: true,
    inputPermissionsError: '',
  });
  gamepads = $state([]);
  keyboards = $state([]);
  mice = $state([]);
  displays = $state([]);

  async loadCapabilities() {
    try {
      const caps = await AppAPI.getMultiseatCapabilities();
      if (caps) {
        this.capabilities = {
          hasBwrap: Boolean(caps.hasBwrap),
          hasGamescope: Boolean(caps.hasGamescope),
          hasCage: Boolean(caps.hasCage),
          hasSunshine: Boolean(caps.hasSunshine),
          hasPipewire: Boolean(caps.hasPipewire),
          hasInputPermissions: Boolean(caps.hasInputPermissions),
          inputPermissionsError: caps.inputPermissionsError || '',
        };
      }
    } catch (err) {
      console.warn('Failed to probe multiseat capabilities:', err);
    }
  }

  async refreshDisplays() {
    try {
      const displays = await AppAPI.getConnectedDisplays();
      if (Array.isArray(displays)) {
        this.displays = displays;
      }
    } catch (err) {
      console.warn('Failed to probe displays:', err);
    }
  }

  async refreshGamepads() {
    try {
      const pads = await AppAPI.getConnectedGamepads();
      if (Array.isArray(pads)) {
        this.gamepads = pads;
      }
    } catch (err) {
      console.warn('Failed to refresh connected gamepads:', err);
    }
  }

  async refreshKeyboards() {
    try {
      const kbds = await AppAPI.getConnectedKeyboards();
      if (Array.isArray(kbds)) {
        this.keyboards = kbds;
      }
    } catch (err) {
      console.warn('Failed to refresh connected keyboards:', err);
    }
  }

  async refreshMice() {
    try {
      const mice = await AppAPI.getConnectedMice();
      if (Array.isArray(mice)) {
        this.mice = mice;
      }
    } catch (err) {
      console.warn('Failed to refresh connected mice:', err);
    }
  }
}
