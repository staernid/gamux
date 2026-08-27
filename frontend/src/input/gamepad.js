/**
 * Gamepad Input Normalization and State Tracking
 * Provides unified, controller-agnostic input abstraction with deadzone filtering,
 * just-pressed edge detection, and rotary stick delta tracking.
 */

export const Buttons = {
  SOUTH: 'south', // A (Xbox) / Cross (PlayStation) / B (Nintendo)
  EAST: 'east',   // B (Xbox) / Circle (PlayStation) / A (Nintendo)
  WEST: 'west',   // X (Xbox) / Square (PlayStation) / Y (Nintendo)
  NORTH: 'north', // Y (Xbox) / Triangle (PlayStation) / X (Nintendo)
  LB: 'lb',       // Left Bumper / L1
  RB: 'rb',       // Right Bumper / R1
  LT: 'lt',       // Left Trigger / L2
  RT: 'rt',       // Right Trigger / R2
  SELECT: 'select', // View / Back / Share
  START: 'start',   // Menu / Options
  LS_CLICK: 'ls_click',
  RS_CLICK: 'rs_click',
  DPAD_UP: 'dpad_up',
  DPAD_DOWN: 'dpad_down',
  DPAD_LEFT: 'dpad_left',
  DPAD_RIGHT: 'dpad_right',
  GUIDE: 'guide',
};

const STANDARD_BUTTON_MAP = [
  Buttons.SOUTH,      // 0
  Buttons.EAST,       // 1
  Buttons.WEST,       // 2
  Buttons.NORTH,      // 3
  Buttons.LB,         // 4
  Buttons.RB,         // 5
  Buttons.LT,         // 6
  Buttons.RT,         // 7
  Buttons.SELECT,     // 8
  Buttons.START,      // 9
  Buttons.LS_CLICK,   // 10
  Buttons.RS_CLICK,   // 11
  Buttons.DPAD_UP,    // 12
  Buttons.DPAD_DOWN,  // 13
  Buttons.DPAD_LEFT,  // 14
  Buttons.DPAD_RIGHT, // 15
  Buttons.GUIDE,      // 16
];

export function applyDeadzone(value, threshold = 0.15) {
  if (Math.abs(value) < threshold) return 0;
  return value;
}

/**
 * Normalizes a raw browser Gamepad object into standard semantic inputs.
 */
export function normalizeGamepad(pad, deadzone = 0.15) {
  if (!pad) return null;

  const buttonStates = {};
  for (const b of Object.values(Buttons)) {
    buttonStates[b] = false;
  }

  // W3C Standard mapping
  if (pad.buttons) {
    for (let i = 0; i < pad.buttons.length; i++) {
      const btnName = STANDARD_BUTTON_MAP[i];
      if (btnName) {
        buttonStates[btnName] = !!pad.buttons[i]?.pressed;
      }
    }
  }

  // Axes with deadzone filtering
  const rawAxes = pad.axes || [];
  const leftX = applyDeadzone(rawAxes[0] || 0, deadzone);
  const leftY = applyDeadzone(rawAxes[1] || 0, deadzone);
  const rightX = applyDeadzone(rawAxes[2] || 0, deadzone);
  const rightY = applyDeadzone(rawAxes[3] || 0, deadzone);

  return {
    id: pad.id || 'Standard Gamepad',
    connected: !!pad.connected,
    buttons: buttonStates,
    axes: {
      leftX,
      leftY,
      rightX,
      rightY,
    },
  };
}

/**
 * Tracks previous frame input states for edge detection and rotary stick calculations.
 */
export class GamepadTracker {
  constructor(options = {}) {
    this.deadzone = options.deadzone ?? 0.15;
    this.stickThreshold = options.stickThreshold ?? 0.55;
    this.rotaryThreshold = options.rotaryThreshold ?? 0.30;
    this.rotarySensitivity = options.rotarySensitivity ?? (7.0 / (2 * Math.PI)); // 7 cards per 360 deg

    this.prevButtons = {};
    this.prevStickLeft = false;
    this.prevStickRight = false;
    this.prevStickUp = false;
    this.prevStickDown = false;
    this.lastStickAngle = null;
  }

  /**
   * Process the current frame's raw gamepad and return high-level actions.
   */
  process(rawPad) {
    const norm = normalizeGamepad(rawPad, this.deadzone);
    if (!norm || !norm.connected) {
      this.reset();
      return null;
    }

    const { buttons, axes } = norm;

    // Edge-triggered button presses
    const justPressed = (btn) => buttons[btn] && !this.prevButtons[btn];

    // Analog stick edge-triggered digital steps
    const isStickLeft = axes.leftX < -this.stickThreshold;
    const isStickRight = axes.leftX > this.stickThreshold;
    const isStickUp = axes.leftY < -this.stickThreshold;
    const isStickDown = axes.leftY > this.stickThreshold;

    const stickLeftPressed = isStickLeft && !this.prevStickLeft;
    const stickRightPressed = isStickRight && !this.prevStickRight;
    const stickUpPressed = isStickUp && !this.prevStickUp;
    const stickDownPressed = isStickDown && !this.prevStickDown;

    // Right stick rotary angle calculation
    let rotaryDelta = 0;
    const rMag = Math.sqrt(axes.rightX * axes.rightX + axes.rightY * axes.rightY);
    if (rMag >= this.rotaryThreshold) {
      const angle = Math.atan2(axes.rightY, axes.rightX);
      if (this.lastStickAngle !== null && this.lastStickAngle !== undefined) {
        let d = angle - this.lastStickAngle;
        while (d > Math.PI) d -= 2 * Math.PI;
        while (d < -Math.PI) d += 2 * Math.PI;

        if (Math.abs(d) <= Math.PI * 0.7 && Math.abs(d) > 0.001) {
          rotaryDelta = d * this.rotarySensitivity;
        }
      }
      this.lastStickAngle = angle;
    } else {
      this.lastStickAngle = null;
    }

    // High level semantic actions
    const actions = {
      // Primary Action (A / Enter)
      actionPrimary: justPressed(Buttons.SOUTH),
      // Back / Dismiss Action (B / Escape)
      actionBack: justPressed(Buttons.EAST),
      // Options / Inspector Action (X)
      actionOptions: justPressed(Buttons.WEST),
      // Secondary / Files Action (Y)
      actionFiles: justPressed(Buttons.NORTH),
      // View Cycling (LB / RB)
      prevView: justPressed(Buttons.LB),
      nextView: justPressed(Buttons.RB),
      // Directional Navigation (D-Pad or Left Stick)
      navLeft: justPressed(Buttons.DPAD_LEFT) || stickLeftPressed,
      navRight: justPressed(Buttons.DPAD_RIGHT) || stickRightPressed,
      navUp: justPressed(Buttons.DPAD_UP) || stickUpPressed,
      navDown: justPressed(Buttons.DPAD_DOWN) || stickDownPressed,
      // Continuous rotary wheel scrubbing
      rotaryDelta,
    };

    // Store state for next frame
    this.prevButtons = { ...buttons };
    this.prevStickLeft = isStickLeft;
    this.prevStickRight = isStickRight;
    this.prevStickUp = isStickUp;
    this.prevStickDown = isStickDown;

    return actions;
  }

  reset() {
    this.prevButtons = {};
    this.prevStickLeft = false;
    this.prevStickRight = false;
    this.prevStickUp = false;
    this.prevStickDown = false;
    this.lastStickAngle = null;
  }
}
