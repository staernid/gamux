import { DevicesState } from './devices.svelte.js';
import { StagingState } from './staging.svelte.js';
import { SessionState } from './session.svelte.js';
import { configStore } from '../config.js';
import {
  getActiveSession,
  resolveMultiGamepadAssignments,
  resolveGamepadAssignments,
  hydrateStagingFromConfig,
} from './assignments.js';

export {
  getActiveSession,
  resolveMultiGamepadAssignments,
  resolveGamepadAssignments,
  hydrateStagingFromConfig,
};

export class MultiseatState {
  devices = new DevicesState();
  stagingState = new StagingState();
  sessionState = new SessionState();

  #subscribers = new Set();

  get capabilities() {
    return this.devices.capabilities;
  }

  get gamepads() {
    return this.devices.gamepads;
  }

  get keyboards() {
    return this.devices.keyboards;
  }

  get mice() {
    return this.devices.mice;
  }

  get displays() {
    return this.devices.displays;
  }

  get isCoopEnabled() {
    return this.stagingState.isCoopEnabled;
  }

  get staging() {
    return this.stagingState.staging;
  }

  get activeSessions() {
    return this.sessionState.activeSessions;
  }

  #notify() {
    const snap = this.snapshot();
    for (const fn of this.#subscribers) {
      try { fn(snap); } catch {}
    }
  }

  snapshot() {
    return {
      capabilities: this.devices.capabilities,
      gamepads: this.devices.gamepads,
      keyboards: this.devices.keyboards,
      mice: this.devices.mice,
      displays: this.devices.displays,
      isCoopEnabled: this.stagingState.isCoopEnabled,
      staging: this.stagingState.staging,
      activeSessions: this.sessionState.activeSessions,
    };
  }

  // Svelte store contract interop
  subscribe = (fn) => {
    this.#subscribers.add(fn);
    fn(this.snapshot());
    return () => this.#subscribers.delete(fn);
  };

  syncFromConfig = (cfg) => {
    this.stagingState.syncFromConfig(cfg);
    this.#notify();
  };

  loadCapabilities = async () => {
    await this.devices.loadCapabilities();
    this.#notify();
  };

  refreshDisplays = async () => {
    await this.devices.refreshDisplays();
    this.#notify();
  };

  refreshGamepads = async () => {
    await this.devices.refreshGamepads();
    this.#notify();
  };

  refreshKeyboards = async () => {
    await this.devices.refreshKeyboards();
    this.#notify();
  };

  refreshMice = async () => {
    await this.devices.refreshMice();
    this.#notify();
  };

  toggleCoop = (forceVal) => {
    this.stagingState.toggleCoop(forceVal);
    this.#notify();
  };

  addSeat = () => {
    this.stagingState.addSeat();
    this.#notify();
  };

  toggleSeatActive = (seatId) => {
    this.stagingState.toggleSeatActive(seatId);
    this.#notify();
  };

  setSeatDisplay = (seatId, displayIndex) => {
    this.stagingState.setSeatDisplay(seatId, displayIndex);
    this.#notify();
  };

  setSeatSlot = (seatId, targetSlot) => {
    this.stagingState.setSeatSlot(seatId, targetSlot);
    this.#notify();
  };

  swapSeatsSlots = (seatId1, seatId2) => {
    this.stagingState.swapSeatsSlots(seatId1, seatId2);
    this.#notify();
  };

  removeSeat = (index) => {
    this.stagingState.removeSeat(index);
    this.#notify();
  };

  updateSeat = (indexOrId, partial) => {
    this.stagingState.updateSeat(indexOrId, partial);
    this.#notify();
  };

  setSeatMode = (seatId, mode) => {
    this.stagingState.setSeatMode(seatId, mode);
    this.#notify();
  };

  setLayout = (layout) => {
    this.stagingState.setLayout(layout);
    this.#notify();
  };

  assignGamepad = (seatId, devNode) => {
    this.stagingState.assignGamepad(seatId, devNode);
    this.#notify();
  };

  assignKeyboard = (seatId, devNode) => {
    this.stagingState.assignKeyboard(seatId, devNode);
    this.#notify();
  };

  assignMouse = (seatId, devNode) => {
    this.stagingState.assignMouse(seatId, devNode);
    this.#notify();
  };

  updateStaging = (partial) => {
    this.stagingState.updateStaging(partial);
    this.#notify();
  };

  launchSession = async (game, userCfg = {}) => {
    await this.sessionState.launchSession(game, userCfg, this.devices, this.stagingState);
    this.#notify();
  };

  stopSession = async (gameDir) => {
    await this.sessionState.stopSession(gameDir);
    this.#notify();
  };

  stopSeat = async (gameDir, seatId) => {
    await this.sessionState.stopSeat(gameDir, seatId);
    this.#notify();
  };

  setSessionInactive = (gameDir) => {
    this.sessionState.setSessionInactive(gameDir);
    this.#notify();
  };

  refreshStatus = async (gameDir) => {
    await this.sessionState.refreshStatus(gameDir);
    this.#notify();
  };
}

export const multiseatStore = new MultiseatState();

if (typeof configStore?.subscribe === 'function') {
  configStore.subscribe(cfg => {
    if (cfg?.multiseat) {
      multiseatStore.syncFromConfig(cfg);
    }
  });
}
