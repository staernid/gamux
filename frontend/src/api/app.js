// Wails v3 API bridge (validated against generated bindings.gen.js) & event subscription hub
import { Call, Events } from '@wailsio/runtime';
import { BINDINGS, BINDING_PREFIX } from './bindings.gen.js';
import { DEFAULT_MOCKS } from './mocks.js';

export { DEFAULT_MOCKS };

function toPascalCase(str) {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export function isWailsDesktop() {
  if (typeof window === 'undefined') return false;
  if (window.location?.protocol === 'wails:') return true;
  if (window.location?.hostname === 'wails.localhost') return true;
  if (Boolean(window.webkit?.messageHandlers)) return true;
  if (Boolean(window.chrome?.webview?.postMessage)) return true;
  if (Boolean(window.wails?.invoke)) return true;
  return false;
}

/**
 * Builds the dispatcher for a bound Go method. The method name and argument
 * count are validated against the generated BINDINGS manifest in every
 * environment so wrong calls fail in Vitest/browser preview exactly as they
 * would against the real Wails v3 backend (which enforces strict arity).
 * @param {string} prop camelCase method name present in BINDINGS
 */
function createMethodDispatcher(prop) {
  const arity = BINDINGS[prop];
  return async (...args) => {
    if (args.length !== arity) {
      throw new TypeError(`AppAPI.${prop} expects ${arity} argument(s), got ${args.length}`);
    }

    // 1. Native Wails v3 desktop: call the Go backend directly.
    if (isWailsDesktop()) {
      try {
        return await Call.ByName(BINDING_PREFIX + toPascalCase(prop), ...args);
      } catch (err) {
        console.error(`Wails v3 binding call error for ${prop}:`, err);
        throw err;
      }
    }

    // 2. Browser preview, Vitest, or offline evaluation: declarative mocks.
    const mock = DEFAULT_MOCKS[prop];
    if (typeof mock !== 'function') {
      throw new Error(`AppAPI.${prop} has no mock implementation in DEFAULT_MOCKS (browser/test mode)`);
    }
    return await mock(...args);
  };
}

/** @type {Map<string, Function>} */
const dispatcherCache = new Map();

/**
 * Typed-by-manifest bridge to the Go `gui.App` service. Only methods listed in
 * the generated `bindings.gen.js` exist; any other property is `undefined`, so
 * `AppAPI.typo()` throws immediately. Tests may still assign overrides
 * (`AppAPI.getConfig = vi.fn()`), which take precedence.
 */
export const AppAPI = new Proxy({}, {
  get(target, prop) {
    if (typeof prop !== 'string') return target[prop];
    if (prop in target) return target[prop];
    if (!Object.hasOwn(BINDINGS, prop)) return undefined;
    let fn = dispatcherCache.get(prop);
    if (!fn) {
      fn = createMethodDispatcher(prop);
      dispatcherCache.set(prop, fn);
    }
    return fn;
  },
  set(target, prop, value) {
    target[prop] = value;
    return true;
  },
  deleteProperty(target, prop) {
    delete target[prop];
    return true;
  },
  has(target, prop) {
    return prop in target || (typeof prop === 'string' && Object.hasOwn(BINDINGS, prop));
  },
  getOwnPropertyDescriptor(target, prop) {
    if (prop in target) return Object.getOwnPropertyDescriptor(target, prop);
    if (typeof prop === 'string' && Object.hasOwn(BINDINGS, prop)) {
      return { value: this.get(target, prop), writable: true, enumerable: true, configurable: true };
    }
    return undefined;
  }
});

const RUNTIME_EVENT_MAP = [
  { key: 'onTaskEvent', event: 'gamux:task-event' },
  { key: 'onTransferEvent', event: 'gamux:transfer-event' },
  { key: 'onNotification', event: 'gamux:notification' },
  { key: 'onDownloadProgress', event: 'gamux:download-progress' },
  { key: 'onStepProgress', event: 'gamux:step-progress' },
  { key: 'onGameStatus', event: 'gamux:game-status' },
  { key: 'onOpenTarget', event: 'gamux:open-target' },
  { key: 'onSystemThemeChanged', event: 'gamux:system-theme-changed' },
];

/**
 * Subscribes to a backend runtime event (Wails v3) and mirrors it to the dev
 * bridge so browser previews and GUI audits can replay it.
 * @param {string} event Event name, e.g. `gamux:task-event`
 * @param {(payload: any) => void} cb Receives the event payload (`ev.data`)
 */
export function onRuntimeEvent(event, cb) {
  if (typeof window !== 'undefined' && window.__GAMUX_DEV__) {
    window.__GAMUX_DEV__.registerEventListener(event, cb);
  }

  if (isWailsDesktop()) {
    try {
      Events.On(event, (ev) => {
        const payload = ev?.data !== undefined ? ev.data : ev;
        window.__GAMUX_DEV__?.recordIncoming('wails', event, payload);
        cb(payload);
      });
    } catch (e) {
      console.warn(`Failed to bind Wails v3 event listener for ${event}:`, e);
    }
  }
}

// Initialize and bind all runtime event listeners to Wails and dev bridge
export function initRuntimeEvents(listeners = {}) {
  RUNTIME_EVENT_MAP.forEach(({ key, event }) => {
    const cb = listeners[key];
    if (cb) onRuntimeEvent(event, cb);
  });
}
