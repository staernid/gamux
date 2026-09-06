import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { multiseatStore, resolveGamepadAssignments, getActiveSession } from './multiseat.js';
import { AppAPI } from '../api/app.js';

describe('multiseatStore', () => {
  beforeEach(() => {
    multiseatStore.toggleCoop(false);
  });

  it('initializes with expected default staging settings', () => {
    const state = get(multiseatStore);
    expect(state.isCoopEnabled).toBe(false);
    expect(state.staging.seat1.mode).toBe('local');
    expect(state.staging.seat2.mode).toBe('local');
    expect(state.staging.layout).toBe('dual_monitor');
  });

  it('toggles co-op mode on and off', () => {
    multiseatStore.toggleCoop(true);
    expect(get(multiseatStore).isCoopEnabled).toBe(true);

    multiseatStore.toggleCoop(false);
    expect(get(multiseatStore).isCoopEnabled).toBe(false);
  });

  it('switches seat output modes independently', () => {
    multiseatStore.setSeatMode('seat1', 'stream');
    expect(get(multiseatStore).staging.seat1.mode).toBe('stream');
    expect(get(multiseatStore).staging.seat2.mode).toBe('local');

    multiseatStore.setSeatMode('seat2', 'stream');
    expect(get(multiseatStore).staging.seat2.mode).toBe('stream');
  });

  it('assigns gamepads, keyboards, and mice to individual seats', async () => {
    multiseatStore.assignGamepad('seat1', '/dev/input/event10');
    expect(get(multiseatStore).staging.seat1.gamepad_node).toBe('/dev/input/event10');

    multiseatStore.assignGamepad('seat2', '/dev/input/event11');
    expect(get(multiseatStore).staging.seat2.gamepad_node).toBe('/dev/input/event11');

    multiseatStore.assignKeyboard('seat2', '/dev/input/event4');
    expect(get(multiseatStore).staging.seat2.keyboard_node).toBe('/dev/input/event4');

    multiseatStore.assignMouse('seat2', '/dev/input/event1');
    expect(get(multiseatStore).staging.seat2.mouse_node).toBe('/dev/input/event1');

    await multiseatStore.refreshKeyboards();
    expect(get(multiseatStore).keyboards.length).toBeGreaterThan(0);

    await multiseatStore.refreshMice();
    expect(get(multiseatStore).mice.length).toBeGreaterThan(0);
  });

  it('updates display layout configuration', () => {
    multiseatStore.setLayout('side_by_side');
    expect(get(multiseatStore).staging.layout).toBe('side_by_side');

    multiseatStore.setLayout('top_bottom');
    expect(get(multiseatStore).staging.layout).toBe('top_bottom');
  });

  it('launches a multi-seat session and updates active session state', async () => {
    const launchSpy = vi.spyOn(AppAPI, 'launchMultiseatSession').mockResolvedValue();

    const mockGame = {
      gameDir: '/mock/games/Halo',
      name: 'Halo',
      appId: '976730',
      exePath: '/mock/games/Halo/halo.exe',
      runner: 'umu',
    };

    await multiseatStore.launchSession(mockGame, { account_name: 'Hero117' });

    expect(launchSpy).toHaveBeenCalled();
    const state = get(multiseatStore);
    expect(state.activeSessions[mockGame.gameDir]?.is_running).toBe(true);

    // Stop session
    const stopSpy = vi.spyOn(AppAPI, 'stopMultiseatSession').mockResolvedValue();
    await multiseatStore.stopSession(mockGame.gameDir);
    expect(stopSpy).toHaveBeenCalledWith(mockGame.gameDir);
    expect(get(multiseatStore).activeSessions[mockGame.gameDir]).toBeUndefined();
  });

  it('preserves Seat 2 compositor as auto when dedicated KBM is configured with auto compositor', async () => {
    const launchSpy = vi.spyOn(AppAPI, 'launchMultiseatSession').mockResolvedValue();

    const mockGame = {
      gameDir: '/mock/games/HaloKBM',
      name: 'Halo KBM',
      appId: '976730',
      exePath: '/mock/games/Halo/halo.exe',
      runner: 'umu',
    };

    multiseatStore.assignKeyboard('seat2', '/dev/input/event10');
    multiseatStore.assignMouse('seat2', '/dev/input/event11');
    multiseatStore.updateStaging({ seat2: { ...get(multiseatStore).staging.seat2, compositor: 'auto' } });

    await multiseatStore.launchSession(mockGame, { account_name: 'Hero117' });

    expect(launchSpy).toHaveBeenCalled();
    const callArgs = launchSpy.mock.calls[launchSpy.mock.calls.length - 1][0];
    expect(callArgs.seat2.compositor).toBe('auto');
    expect(callArgs.seat2.keyboard_node).toBe('/dev/input/event10');
    expect(callArgs.seat2.mouse_node).toBe('/dev/input/event11');
  });

  it('respects explicit gamescope compositor even when dedicated KBM is configured', async () => {
    const launchSpy = vi.spyOn(AppAPI, 'launchMultiseatSession').mockResolvedValue();

    const mockGame = {
      gameDir: '/mock/games/HaloGamescope',
      name: 'Halo Gamescope',
      appId: '976730',
      exePath: '/mock/games/Halo/halo.exe',
      runner: 'umu',
    };

    multiseatStore.assignKeyboard('seat2', '/dev/input/event10');
    multiseatStore.assignMouse('seat2', '/dev/input/event11');
    multiseatStore.updateStaging({ seat2: { ...get(multiseatStore).staging.seat2, compositor: 'gamescope' } });

    await multiseatStore.launchSession(mockGame, { account_name: 'Hero117' });

    expect(launchSpy).toHaveBeenCalled();
    const callArgs = launchSpy.mock.calls[launchSpy.mock.calls.length - 1][0];
    expect(callArgs.seat2.compositor).toBe('gamescope');
  });

  it('does not default to cage when seat2 keyboard and mouse are none or auto', async () => {
    const launchSpy = vi.spyOn(AppAPI, 'launchMultiseatSession').mockResolvedValue();

    const mockGame = {
      gameDir: '/mock/games/HaloNone',
      name: 'Halo None',
      appId: '976730',
      exePath: '/mock/games/Halo/halo.exe',
      runner: 'umu',
    };

    multiseatStore.assignKeyboard('seat2', 'none');
    multiseatStore.assignMouse('seat2', 'none');
    multiseatStore.updateStaging({ seat2: { ...get(multiseatStore).staging.seat2, compositor: 'auto' } });

    await multiseatStore.launchSession(mockGame, { account_name: 'Hero117' });

    expect(launchSpy).toHaveBeenCalled();
    const callArgs = launchSpy.mock.calls[launchSpy.mock.calls.length - 1][0];
    expect(callArgs.seat2.compositor).toBe('auto');
  });

  it('retrieves active session regardless of trailing slash differences', () => {
    const sessions = {
      '/games/Portal 2': { is_running: true, seat1: { is_running: true } },
    };
    expect(getActiveSession(sessions, '/games/Portal 2')).toBeTruthy();
    expect(getActiveSession(sessions, '/games/Portal 2/')).toBeTruthy();
    expect(getActiveSession(sessions, '/games/Portal 2')?.is_running).toBe(true);
    expect(getActiveSession(sessions, '/games/OtherGame')).toBeNull();
  });

  it('stops session and removes entry respecting path normalization', async () => {
    await multiseatStore.launchSession({
      gameDir: '/games/CoopGame/',
      name: 'Co-op Game',
      appId: '123',
      exePath: '/games/CoopGame/game.exe',
      runner: 'umu',
    });

    await multiseatStore.stopSeat('/games/CoopGame', 'p2');
    const sessAfterSeat = getActiveSession(get(multiseatStore).activeSessions, '/games/CoopGame');
    expect(sessAfterSeat?.seat2?.is_running).toBe(false);
    expect(sessAfterSeat?.is_running).toBe(true);

    multiseatStore.setSessionInactive('/games/CoopGame');
    expect(getActiveSession(get(multiseatStore).activeSessions, '/games/CoopGame')).toBeNull();
  });

  describe('resolveGamepadAssignments', () => {
    const mockPads = [
      { name: 'Black Shark Green Ghost', dev_node: '/dev/input/event29', bus: 'usb' },
      { name: 'Xbox Wireless Controller', dev_node: '/dev/input/event30', bus: 'bluetooth' }
    ];

    it('assigns single gamepad to Player 2 and keeps Player 1 on keyboard/mouse under Auto', () => {
      const singlePad = [mockPads[0]];
      const res = multiseatStore.resolveGamepadAssignments
        ? multiseatStore.resolveGamepadAssignments(singlePad, 'auto', 'auto')
        : resolveGamepadAssignments(singlePad, 'auto', 'auto');

      expect(res.p1Dev).toBe('none');
      expect(res.p2Dev).toBe('/dev/input/event29');
    });

    it('assigns first pad to P1 and second pad to P2 when two pads are connected under Auto', () => {
      const res = resolveGamepadAssignments(mockPads, 'auto', 'auto');
      expect(res.p1Dev).toBe('/dev/input/event29');
      expect(res.p2Dev).toBe('/dev/input/event30');
    });

    it('respects explicit none and manual dev node assignments', () => {
      const res = resolveGamepadAssignments(mockPads, 'none', '/dev/input/event29');
      expect(res.p1Dev).toBe('none');
      expect(res.p2Dev).toBe('/dev/input/event29');
    });

    it('handles zero connected gamepads gracefully', () => {
      const res = resolveGamepadAssignments([], 'auto', 'auto');
      expect(res.p1Dev).toBe('none');
      expect(res.p2Dev).toBe('none');
    });
  });
});
