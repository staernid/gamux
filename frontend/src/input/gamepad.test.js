import { describe, it, expect, beforeEach } from 'vitest';
import { Buttons, normalizeGamepad, applyDeadzone, GamepadTracker } from './gamepad.js';

describe('Gamepad Abstraction', () => {
  it('applies deadzone thresholding correctly', () => {
    expect(applyDeadzone(0.05, 0.15)).toBe(0);
    expect(applyDeadzone(-0.10, 0.15)).toBe(0);
    expect(applyDeadzone(0.25, 0.15)).toBe(0.25);
    expect(applyDeadzone(-0.80, 0.15)).toBe(-0.80);
  });

  it('normalizes standard gamepad buttons and axes', () => {
    const rawPad = {
      id: 'Xbox Wireless Controller',
      connected: true,
      buttons: [
        { pressed: true },  // 0: SOUTH
        { pressed: false }, // 1: EAST
        { pressed: true },  // 2: WEST
        { pressed: false }, // 3: NORTH
        { pressed: false }, // 4: LB
        { pressed: true },  // 5: RB
      ],
      axes: [0.6, 0.05, -0.4, 0.8],
    };

    const norm = normalizeGamepad(rawPad);
    expect(norm.connected).toBe(true);
    expect(norm.buttons[Buttons.SOUTH]).toBe(true);
    expect(norm.buttons[Buttons.EAST]).toBe(false);
    expect(norm.buttons[Buttons.WEST]).toBe(true);
    expect(norm.buttons[Buttons.RB]).toBe(true);

    expect(norm.axes.leftX).toBe(0.6);
    expect(norm.axes.leftY).toBe(0); // below 0.15 deadzone
    expect(norm.axes.rightX).toBe(-0.4);
    expect(norm.axes.rightY).toBe(0.8);
  });

  describe('GamepadTracker', () => {
    let tracker;

    beforeEach(() => {
      tracker = new GamepadTracker();
    });

    it('detects edge-triggered button presses (justPressed)', () => {
      const rawPadPress = {
        connected: true,
        buttons: [{ pressed: true }],
        axes: [0, 0, 0, 0],
      };

      const rawPadHold = {
        connected: true,
        buttons: [{ pressed: true }],
        axes: [0, 0, 0, 0],
      };

      const rawPadRelease = {
        connected: true,
        buttons: [{ pressed: false }],
        axes: [0, 0, 0, 0],
      };

      // Frame 1: Press
      const actions1 = tracker.process(rawPadPress);
      expect(actions1.actionPrimary).toBe(true);

      // Frame 2: Hold (no repeat on edge-trigger)
      const actions2 = tracker.process(rawPadHold);
      expect(actions2.actionPrimary).toBe(false);

      // Frame 3: Release
      const actions3 = tracker.process(rawPadRelease);
      expect(actions3.actionPrimary).toBe(false);
    });

    it('detects directional analog stick pulses', () => {
      const padCenter = {
        connected: true,
        buttons: [],
        axes: [0, 0, 0, 0],
      };

      const padStickRight = {
        connected: true,
        buttons: [],
        axes: [0.8, 0, 0, 0],
      };

      tracker.process(padCenter);
      const actions = tracker.process(padStickRight);
      expect(actions.navRight).toBe(true);
      expect(actions.navLeft).toBe(false);
    });

    it('computes rotary angle delta for right stick click-wheel scrubbing', () => {
      // 1. Right stick at angle 0 (x=1, y=0)
      const padAt0 = {
        connected: true,
        buttons: [],
        axes: [0, 0, 1.0, 0.0],
      };
      tracker.process(padAt0);

      // 2. Right stick rotated 90 degrees (x=0, y=1)
      const padAt90 = {
        connected: true,
        buttons: [],
        axes: [0, 0, 0.0, 1.0],
      };
      const actions = tracker.process(padAt90);

      expect(actions.rotaryDelta).toBeGreaterThan(0);
    });
  });
});
