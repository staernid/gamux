import { describe, it, expect } from 'vitest';
import {
  getActiveSession,
  resolveGamepadAssignments,
  resolveMultiGamepadAssignments,
  syncStaging,
  hydrateStagingFromConfig,
  defaultSeats
} from './assignments.js';

describe('multiseat assignments pure helpers', () => {
  describe('getActiveSession', () => {
    it('retrieves active session regardless of trailing slash differences', () => {
      const sessions = {
        '/games/Portal 2': { is_running: true, seat1: { is_running: true } },
      };
      expect(getActiveSession(sessions, '/games/Portal 2')).toBeTruthy();
      expect(getActiveSession(sessions, '/games/Portal 2/')).toBeTruthy();
      expect(getActiveSession(sessions, '/games/Portal 2')?.is_running).toBe(true);
      expect(getActiveSession(sessions, '/games/OtherGame')).toBeNull();
    });

    it('returns null on missing sessions or gameDir', () => {
      expect(getActiveSession(null, '/games/a')).toBeNull();
      expect(getActiveSession({}, null)).toBeNull();
    });
  });

  describe('resolveGamepadAssignments', () => {
    const mockPads = [
      { name: 'Black Shark Green Ghost', dev_node: '/dev/input/event29', bus: 'usb' },
      { name: 'Xbox Wireless Controller', dev_node: '/dev/input/event30', bus: 'bluetooth' }
    ];

    it('assigns single gamepad to Player 2 and keeps Player 1 on keyboard/mouse under Auto', () => {
      const singlePad = [mockPads[0]];
      const res = resolveGamepadAssignments(singlePad, 'auto', 'auto');

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

  describe('syncStaging & hydrateStagingFromConfig', () => {
    it('hydrates default configuration structure correctly', () => {
      const mCfg = {
        default_layout: 'side_by_side',
        sunshine_port_p1: 47989,
        sunshine_port_p2: 48010,
        seat1_display_index: 0,
        seat2_display_index: 1,
      };
      const hydrated = hydrateStagingFromConfig(mCfg);
      expect(hydrated.layout).toBe('side_by_side');
      expect(hydrated.seats).toHaveLength(2);
      expect(hydrated.seat1.display_index).toBe(0);
      expect(hydrated.seat2.display_index).toBe(0); // side_by_side split layout uses splitDisp
    });
  });
});
