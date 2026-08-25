// Dynamic Zero-Boilerplate Wails API Bridge & Event Subscription Hub
import * as mockData from './mockData.js';

// Declarative default mock implementations for browser testing & offline dev
const DEFAULT_MOCKS = {
  getConfig: () => mockData.MOCK_CONFIG,
  saveConfig: () => Promise.resolve(),
  getGSEUserConfig: () => ({
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US',
  }),
  saveGSEUserConfig: () => Promise.resolve(),
  detectGame: (path) => mockData.MOCK_GAMES[0],
  inspectGame: (path, count = 0) => mockData.MOCK_GAMES.find(g => (g.game_dir || g.GameDir) === path) || mockData.MOCK_GAMES[0],
  inspectLibraries: (dirs) => mockData.MOCK_GAMES,
  getToolsStatus: () => mockData.MOCK_TOOLS_STATUS,
  updateTool: () => 'mock_tool_task_id',
  listTasks: () => mockData.MOCK_TASKS,
  getAvailableRunners: () => ['umu', 'wine', 'proton', 'lutris', 'linux'],
  getTransferSpeedLimit: () => 0,
  setTransferSpeedLimit: () => Promise.resolve(),
  startDownloadTransfer: () => 'mock_transfer_id',
  submitApply: (opts) => 'mock_task_apply_id',
  submitNormalize: (path) => 'mock_task_normalize_id',
  submitRollback: (path, dryRun) => 'mock_task_rollback_id',
  submitMigrateMode: (path, mode, unregister) => 'mock_task_migrate_id',
  submitVerify: (path) => 'mock_task_verify_id',
  submitUpdate: (path) => 'mock_task_update_id',
  pauseTask: () => Promise.resolve(),
  resumeTask: () => Promise.resolve(),
  cancelTask: () => Promise.resolve(),
  removeTask: () => Promise.resolve(),
  reorderTasks: () => Promise.resolve(),
  clearFinishedTasks: () => Promise.resolve(),
  launchGame: () => Promise.resolve(),
  stopGame: () => Promise.resolve(),
  isGameRunning: () => false,
  verifyIntegrity: () => ({
    official_count: 1420,
    intact_count: 1420,
    modified_files: [],
    missing_files: [],
    untracked_files: []
  }),
  saveLaunchProfile: (path, p) => ({ ...mockData.MOCK_GAMES[0], launch_profile: p }),
  resetLaunchProfile: (path) => mockData.MOCK_GAMES[0],
  migrateOperatingMode: () => Promise.resolve(),
  openDirectory: () => Promise.resolve(),
  searchStoreGames: () => [
    { id: '1091500', name: 'Cyberpunk 2077', store: 'steam' },
    { id: '292030', name: 'The Witcher 3: Wild Hunt', store: 'steam' }
  ],
  queryArtworkOptions: (query) => [
    {
      id: 'mock-1',
      provider: 'steam',
      type: query?.asset_type || 'cover',
      url: 'https://cdn.akamai.steamstatic.com/steam/apps/1091500/library_600x900_2x.jpg',
      thumb_url: 'https://cdn.akamai.steamstatic.com/steam/apps/1091500/library_600x900.jpg',
      width: 600,
      height: 900,
      style: 'official'
    }
  ],
  searchArtworkOptions: (query, assetType) => [
    {
      id: 'mock-1',
      provider: 'steam',
      type: assetType || 'cover',
      url: 'https://cdn.akamai.steamstatic.com/steam/apps/1091500/library_600x900_2x.jpg',
      thumb_url: 'https://cdn.akamai.steamstatic.com/steam/apps/1091500/library_600x900.jpg',
      width: 600,
      height: 900,
      style: 'official'
    }
  ],
  applyCustomArtwork: (path, assetType, source) => Promise.resolve({
    cover_url: source,
    local_cover: source,
    local_cover_url: source
  }),
  linkGameMetadata: (path, store, id) => Promise.resolve({
    ...mockData.MOCK_GAMES[0],
    store,
    app_id: id,
    AppID: id
  }),
  resolveArtwork: () => Promise.resolve({ cover_url: '', local_cover: '' }),
  fetchNews: (appId, count) => mockData.MOCK_GAMES[0].news_items || [],
  selectDirectory: (title, defaultPath) => defaultPath || '/mock/path',
  selectFile: (title, defaultPath) => defaultPath || '/mock/file.exe',
};

const METHOD_ALIASES = {
  pauseTransfer: 'pauseTask',
  resumeTransfer: 'resumeTask',
  cancelTransfer: 'cancelTask',
  removeTransfer: 'removeTask',
  reorderTransfers: 'reorderTasks',
  openFolder: 'openDirectory',
  batchInspect: 'inspectLibraries',
  batchInspectLibraries: 'inspectLibraries',
  inspectGameEx: 'inspectGame',
  searchSteamGames: 'searchStoreGames',
  searchArtwork: 'searchStoreGames',
  updateGame: 'submitUpdate',
  applyGame: 'submitApply',
  rollback: 'submitRollback',
  normalizeDirectory: 'submitNormalize',
};

function toPascalCase(str) {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}

function createMethodDispatcher(prop) {
  let targetProp = METHOD_ALIASES[prop] || prop;
  return async (...args) => {
    // Ensure InspectGame receives (gamePath, newsCount)
    if (targetProp === 'inspectGame' && args.length === 1) {
      args = [args[0], 0];
    }

    // 1. Try Go backend runtime binding
    const wailsApp = typeof window !== 'undefined' ? window.go?.gui?.App : null;
    if (wailsApp) {
      const pascal = toPascalCase(targetProp);
      if (typeof wailsApp[pascal] === 'function') {
        return await wailsApp[pascal](...args);
      }
      if (typeof wailsApp[targetProp] === 'function') {
        return await wailsApp[targetProp](...args);
      }
      const originalPascal = toPascalCase(prop);
      if (typeof wailsApp[originalPascal] === 'function') {
        return await wailsApp[originalPascal](...args);
      }
      if (typeof wailsApp[prop] === 'function') {
        return await wailsApp[prop](...args);
      }
    }

    // 2. Try configured declarative mock handler
    if (typeof DEFAULT_MOCKS[targetProp] === 'function') {
      return await DEFAULT_MOCKS[targetProp](...args);
    }
    if (typeof DEFAULT_MOCKS[prop] === 'function') {
      return await DEFAULT_MOCKS[prop](...args);
    }

    // 3. Transparent fallback
    return null;
  };
}

// Dynamic Proxy forwarding any AppAPI.methodName(...) to window.go.gui.App.MethodName(...)
export const AppAPI = new Proxy({}, {
  get(target, prop) {
    if (typeof prop !== 'string') return target[prop];
    if (prop in target) return target[prop];
    return createMethodDispatcher(prop);
  },
  set(target, prop, value) {
    target[prop] = value;
    return true;
  },
  has(target, prop) {
    return true;
  },
  getOwnPropertyDescriptor(target, prop) {
    return {
      value: target[prop] || createMethodDispatcher(prop),
      writable: true,
      enumerable: true,
      configurable: true
    };
  }
});

const RUNTIME_EVENT_MAP = [
  { key: 'onTaskEvent', event: 'gamux:task-event' },
  { key: 'onTransferEvent', event: 'gamux:transfer-event' },
  { key: 'onNotification', event: 'gamux:notification' },
  { key: 'onDownloadProgress', event: 'gamux:download-progress' },
  { key: 'onStepProgress', event: 'gamux:step-progress' },
  { key: 'onGameStatus', event: 'gamux:game-status' },
];

// Initialize and bind all runtime event listeners to Wails and dev bridge
export function initRuntimeEvents(listeners = {}) {
  RUNTIME_EVENT_MAP.forEach(({ key, event }) => {
    const cb = listeners[key];
    if (!cb) return;

    if (typeof window !== 'undefined' && window.__GAMUX_DEV__) {
      window.__GAMUX_DEV__.registerEventListener(event, cb);
    }

    if (typeof window !== 'undefined' && window.runtime?.EventsOn) {
      window.runtime.EventsOn(event, (ev) => {
        window.__GAMUX_DEV__?.recordIncoming('wails', event, ev);
        cb(ev);
      });
    }
  });
}
