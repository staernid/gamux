// Wails v3 API bridge (validated against generated bindings.gen.js) & event subscription hub
import { Call, Events } from '@wailsio/runtime';
import * as mockData from './mockData.js';
import { BINDINGS, BINDING_PREFIX } from './bindings.gen.js';

// Declarative default mock implementations for browser testing & offline dev
export const DEFAULT_MOCKS = {
  getConfig: () => mockData.MOCK_CONFIG,
  saveConfig: () => Promise.resolve(),
  getGSEUserConfig: () => ({
    account_name: 'GamuxPlayer',
    account_steamid: '76561197960287930',
    language: 'english',
    ip_country: 'US',
  }),
  saveGSEUserConfig: () => Promise.resolve(),
  getProfiles: () => [
    { id: 'main', index: 1, name: 'GamuxPlayer', steam_id: '76561197960287930', language: 'english', ip_country: 'US', avatar_url: '', is_main: true },
    { id: 'p2', index: 2, name: 'PlayerTwo', steam_id: '76561198998877665', language: 'english', ip_country: 'US', avatar_url: '', is_main: false },
  ],
  getActiveProfile: () => ({ id: 'main', index: 1, name: 'GamuxPlayer', steam_id: '76561197960287930', language: 'english', ip_country: 'US', avatar_url: '', is_main: true }),
  setActiveProfile: () => Promise.resolve(),
  createProfile: (name, steamId, lang, country, avatar) => Promise.resolve({
    id: 'p3', index: 3, name: name || 'PlayerThree', steam_id: steamId || '76561198000000003', language: lang || 'english', ip_country: country || 'US', avatar_url: avatar || '', is_main: false
  }),
  saveProfile: () => Promise.resolve(),
  deleteProfile: () => Promise.resolve(),
  detectGame: (path) => mockData.MOCK_GAMES[0],
  inspectGame: (path, count = 0) => mockData.MOCK_GAMES.find(g => (g.game_dir || g.GameDir) === path) || mockData.MOCK_GAMES[0],
  inspectLibraries: (dirs) => mockData.MOCK_GAMES,
  getToolsStatus: () => mockData.MOCK_TOOLS_STATUS,
  updateTool: () => 'mock_tool_task_id',
  listTasks: () => mockData.MOCK_TASKS,
  getAvailableRunners: () => ['umu', 'wine', 'proton', 'linux'],
  getTransferSpeedLimit: () => 0,
  setTransferSpeedLimit: () => Promise.resolve(),
  startDownloadTransfer: () => 'mock_transfer_id',
  submitApply: (opts) => 'mock_task_apply_id',
  submitNormalize: (path) => 'mock_task_normalize_id',
  submitRollback: (path, dryRun) => 'mock_task_rollback_id',
  submitVerify: (path) => 'mock_task_verify_id',
  submitUpdate: (path) => 'mock_task_update_id',
  pauseTask: () => Promise.resolve(),
  resumeTask: () => Promise.resolve(),
  cancelTask: () => Promise.resolve(),
  removeTask: () => Promise.resolve(),
  reorderTasks: () => Promise.resolve(),
  clearFinishedTasks: () => Promise.resolve(),
  launchGame: () => Promise.resolve(),
  quickPlay: () => Promise.resolve(),
  stopGame: () => Promise.resolve(),
  stopGameInstance: () => Promise.resolve(),
  trashGame: (path) => Promise.resolve(),
  isGameRunning: () => false,
  getRunningGames: () => [],
  getRunningInstances: () => [],
  getConnectedDisplays: () => [
    { name: 'Display-1', width: 1920, height: 1080, refresh_rate: 60, is_primary: true, scale: 1.0, x: 0, y: 0 }
  ],
  getMultiseatCapabilities: () => ({
    hasBwrap: true,
    hasGamescope: true,
    hasCage: true,
    hasSunshine: true,
    hasPipewire: true,
    hasInputPermissions: true,
    inputPermissionsError: ''
  }),
  getConnectedGamepads: () => ([
    { dev_node: '/dev/input/event2', name: 'Xbox Wireless Controller', id: 'event2', is_gamepad: true },
    { dev_node: '/dev/input/event3', name: 'DualSense Wireless Controller', id: 'event3', is_gamepad: true }
  ]),
  getConnectedKeyboards: () => ([
    { dev_node: '/dev/input/event4', name: 'Keychron K2 Mechanical Keyboard', id: 'event4', is_keyboard: true }
  ]),
  getConnectedMice: () => ([
    { dev_node: '/dev/input/event1', name: 'Logitech USB Optical Mouse', id: 'event1', is_mouse: true }
  ]),
  launchMultiseatSession: () => Promise.resolve(),
  stopMultiseatSession: () => Promise.resolve(),
  stopMultiseatSeat: () => Promise.resolve(),
  getMultiseatSessionStatus: () => ({ is_running: false }),
  isMultiseatSessionRunning: () => false,
  submitBatchLibraryOp: (kind) => Promise.resolve(`task_batch_${kind}_1`),
  getGameLog: (path) => Promise.resolve(`=== Gamux Launch Log ===\nTime: Tue, 25 Aug 2026 20:30:00 CEST\nGame: Mock Game Title\nPlatform: win64\nMode: standalone\nRunner: umu\nExecutable: /mock/game.exe\nWorkingDir: /mock\nCommand: umu-run /mock/game.exe\n========================\n\n[umu-run] Initializing Proton prefix...\n[Proton] Game process started successfully (PID: 12345)\n[DXVK] Found Vulkan 1.3 driver\n[Audio] PulseAudio sink connected\n`),
  getGameAchievements: (path) => Promise.resolve([
    { id: 'ACH_1', name: 'First Steps', description: 'Complete the tutorial', icon_url: '', icon_gray_url: '', hidden: false, unlocked: true, unlock_time: 1690000000, global_percent: 74.2, rarity: 'common' },
    { id: 'ACH_2', name: 'Master Collector', description: 'Collect all hidden items', icon_url: '', icon_gray_url: '', hidden: false, unlocked: false, unlock_time: 0, global_percent: 3.8, rarity: 'ultra_rare' }
  ]),
  getGameCurrentPlayers: (appId) => Promise.resolve(14250),
  getGameMods: (path) => Promise.resolve(mockData.MOCK_MOD_STATUS),
  toggleGameMod: (path, modId, enable) => Promise.resolve({
    id: modId,
    name: modId,
    enabled: enable,
    path: enable ? `${path}/${modId}` : `${path}/${modId}.disabled`,
    relative_path: modId,
  }),
  deleteGameMod: () => Promise.resolve(),
  checkGameModUpdates: (path) => Promise.resolve(mockData.MOCK_MOD_STATUS),
  installWorkshopMod: () => Promise.resolve('mock_workshop_task_id'),
  resolveSteamProfile: (identifier) => Promise.resolve({
    steam_id: '76561197960287930',
    persona_name: 'GamuxPlayer',
    avatar_url: '',
    country_code: 'US',
    profile_url: 'https://steamcommunity.com/id/gamuxplayer'
  }),
  verifyIntegrity: () => ({
    official_count: 1420,
    intact_count: 1420,
    modified_files: [],
    missing_files: [],
    untracked_files: []
  }),
  getLaunchProfile: (path) => null,
  saveLaunchProfile: (path, p) => ({ ...mockData.MOCK_GAMES[0], launch_profile: p }),
  previewGameLaunch: (path) => Promise.resolve({
    command: 'WINEPREFIX="/home/user/Games/umu" MANGOHUD=1 umu-run /mock/game.exe -dx11',
    target_exe: '/mock/game.exe',
    args: ['umu-run', '/mock/game.exe', '-dx11'],
    env: {
      WINEPREFIX: '/home/user/Games/umu',
      MANGOHUD: '1',
      SteamAppId: '1091500',
    },
    working_dir: '/mock',
    runner: 'umu',
    operating_mode: 'loader',
    platform: 'win64',
    overrides: [],
  }),
  getGameDepots: (path) => [
    { depot_id: 41501, name: 'Base Content (41501)', category: 'common', content_category: 'core', installed: true, size: 500000000, file_count: 120, manifest_id: 1001 },
    { depot_id: 41502, name: 'Windows Binaries (41502)', category: 'windows', content_category: 'core', installed: true, size: 25000000, file_count: 15, manifest_id: 1002 },
    { depot_id: 41503, name: 'French Audio Pack (41503)', category: 'common', content_category: 'locale:french', language: 'french', installed: false, size: 450000000, file_count: 80, manifest_id: 1003 },
    { depot_id: 41504, name: 'Linux Native Binaries (41504)', category: 'linux', content_category: 'core', installed: false, size: 28000000, file_count: 18, manifest_id: 1004 },
    { depot_id: 41505, name: 'Official Soundtrack (41505)', category: 'soundtrack', content_category: 'soundtrack', installed: false, size: 150000000, file_count: 24, manifest_id: 1005 },
  ],
  getAppDepots: (appId) => [
    { depot_id: 41501, name: 'Base Content (41501)', category: 'common', content_category: 'core', installed: false, size: 500000000, file_count: 120, manifest_id: 1001 },
    { depot_id: 41502, name: 'Windows Binaries (41502)', category: 'windows', content_category: 'core', installed: false, size: 25000000, file_count: 15, manifest_id: 1002 },
    { depot_id: 41503, name: 'French Audio Pack (41503)', category: 'common', content_category: 'locale:french', language: 'french', installed: false, size: 450000000, file_count: 80, manifest_id: 1003 },
    { depot_id: 41504, name: 'German Audio Pack (41504)', category: 'common', content_category: 'locale:german', language: 'german', installed: false, size: 430000000, file_count: 75, manifest_id: 1004 },
    { depot_id: 41505, name: 'High-Resolution 4K Textures (41505)', category: 'common', content_category: 'optional_hd', installed: false, size: 1200000000, file_count: 200, manifest_id: 1005 },
    { depot_id: 41506, name: 'Official Soundtrack (41506)', category: 'soundtrack', content_category: 'soundtrack', installed: false, size: 150000000, file_count: 24, manifest_id: 1006 },
  ],
  checkGameUpdates: (path) => Promise.resolve({
    is_up_to_date: true,
    checked: true,
    current_build_id: '123456',
    latest_build_id: '123456',
    latest_patch_date: 1672531199,
    latest_patch_title: 'Latest Game Update',
    message: 'Game is up to date'
  }),
  checkLibraryUpdates: () => Promise.resolve({
    total_checked: mockData.MOCK_GAMES.length,
    updates_found: 0,
    updated_games: [],
  }),
  getGameDLCs: (path) => Promise.resolve([
    { app_id: '2004110', name: 'Expansion Pass' },
    { app_id: '2147250', name: 'Soundtrack & Artwork' }
  ]),
  fetchDLCs: (path) => Promise.resolve(),
  switchGamePlatform: (path, plat) => Promise.resolve({ ...mockData.MOCK_GAMES[0], platform: plat }),
  uninstallDepot: (path, depotId) => Promise.resolve({ ...mockData.MOCK_GAMES[0] }),
  openDirectory: () => Promise.resolve(),
  createGameStub: (libraryPath, appID, name, storeName, branch = 'public') => Promise.resolve({
    ...mockData.MOCK_GAMES[0],
    name: name || `Game ${appID}`,
    app_id: appID,
    branch: branch || 'public',
    game_dir: `${libraryPath || '/home/staernid/Games'}/${name || appID}`,
    store: storeName || 'Steam',
    file_count: 0,
    disk_size_bytes: 0,
    is_installed: false,
    is_stub: true,
    is_ready: false
  }),
  getSteamBranches: (appId) => Promise.resolve([
    { name: 'public', description: 'Default / Public release', build_id: '10180494', time_updated: 1671497797, pwd_required: false },
    { name: 'beta', description: 'Beta branch for testing new content and risky fixes. Unstable', build_id: '10174821', time_updated: 1671497193, pwd_required: false },
    { name: 'v1.0', description: 'Legacy 2019-23-01', build_id: '3495375', time_updated: 1562010477, pwd_required: false },
  ]),
  setGameBranch: (gameDir, branch, password) => Promise.resolve({
    ...mockData.MOCK_GAMES[0],
    game_dir: gameDir,
    branch: branch || 'public',
    launch_profile: {
      ...mockData.MOCK_GAMES[0].launch_profile,
      branch: branch || 'public',
      branch_password: password || '',
    }
  }),


  openURL: () => Promise.resolve(),
  getGameMetadata: (provider, id) => {
    const p = (provider || 'steam').toLowerCase();
    if (p === 'rawg') {
      return Promise.resolve({
        name: 'RAWG Game Title',
        short_description: 'Detailed RAWG catalog synopsis and open database lore.',
        detailed_description: 'Detailed RAWG catalog synopsis and open database lore.',
        about_the_game: 'Detailed RAWG catalog synopsis and open database lore.',
        developers: ['RAWG Verified Dev'],
        publishers: ['RAWG Publisher'],
        genres: ['Action', 'Adventure'],
        release_date: '2021-05-18'
      });
    }
    if (p === 'gog') {
      return Promise.resolve({
        name: 'GOG DRM-Free Title',
        short_description: 'DRM-free release on GOG.com with complete offline installers and extras.',
        detailed_description: 'DRM-free release on GOG.com with complete offline installers and extras.',
        about_the_game: 'DRM-free release on GOG.com with complete offline installers and extras.',
        developers: ['CD PROJEKT RED'],
        publishers: ['GOG Ltd'],
        genres: ['RPG'],
        release_date: 'May 19, 2015'
      });
    }
    return Promise.resolve({
      name: 'Cyberpunk 2077',
      short_description: 'Cyberpunk 2077 is an open-world, action-adventure RPG set in Night City, a megalopolis obsessed with power, glamour, and body modification.',
      detailed_description: 'Cyberpunk 2077 is an open-world, action-adventure RPG set in Night City, a megalopolis obsessed with power, glamour, and body modification.',
      about_the_game: 'Cyberpunk 2077 is an open-world, action-adventure RPG set in the world of Night City.',
      developers: ['CD PROJEKT RED'],
      publishers: ['CD PROJEKT RED'],
      genres: ['RPG', 'Action', 'Open World', 'Sci-Fi'],
      release_date: 'Dec 10, 2020'
    });
  },
  getDiskSpace: (path) => ({
    path: path || '/home/staernid/Games',
    total_bytes: 1099511627776, // 1 TiB
    free_bytes: 492970344448,   // 459 GiB
    used_bytes: 606541283328,   // 564 GiB
  }),
  searchStoreGames: () => [
    { id: '1091500', name: 'Cyberpunk 2077', store: 'steam', release_date: '2020' },
    { id: '292030', name: 'The Witcher 3: Wild Hunt', store: 'steam', release_date: '2015' },
    { id: '1207658924', name: 'The Witcher 3: Wild Hunt', store: 'gog', release_date: '2015' },
    { id: '3498', name: 'Grand Theft Auto V', store: 'rawg', release_date: '2013-09-17' },
    { id: '4200', name: 'Portal 2', store: 'rawg', release_date: '2011-04-18' },
    { id: '41494', name: 'Cyberpunk 2077', store: 'rawg', release_date: '2020-12-10' }
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
  applyCustomArtwork: (path, assetType, source) => Promise.resolve({
    cover_url: source,
    local_cover: source,
    local_cover_url: source
  }),
  linkGameMetadata: (path, store, id) => Promise.resolve({
    ...mockData.MOCK_GAMES[0],
    store,
    metadata_provider: store,
    metadata_id: id,
    app_id: id,
    AppID: id
  }),
  unlinkGameMetadata: (path) => Promise.resolve({
    ...mockData.MOCK_GAMES[0],
    metadata_provider: '',
    metadata_id: ''
  }),
  listMetadataProviders: () => Promise.resolve([
    { id: 'steam', display_name: 'Steam', capabilities: { can_search: true, can_fetch_metadata: true, can_provide_artwork: true, is_game_store: true } },
    { id: 'gog', display_name: 'GOG', capabilities: { can_search: true, can_fetch_metadata: true, can_provide_artwork: true, is_game_store: true } },
    { id: 'rawg', display_name: 'RAWG Video Games Database', capabilities: { can_search: true, can_fetch_metadata: true, can_provide_artwork: true, is_game_store: false } }
  ]),
  resolveArtwork: () => Promise.resolve({ cover_url: '', local_cover: '' }),
  fetchNews: (appId, count) => Promise.resolve(mockData.MOCK_GAMES[0].news_items || []),
  getPatchNote: () => Promise.resolve([null, '']),
  openURL: (url) => { window.open(url, '_blank', 'noopener,noreferrer'); },
  selectDirectory: (title, defaultPath) => defaultPath || '/mock/path',
  selectFile: (title, filterName, filterPattern) => '/mock/file.exe',
  getGameSaveLocations: (path) => Promise.resolve(mockData.MOCK_SAVE_LOCATIONS),
  getGameBackups: (path) => Promise.resolve(mockData.MOCK_BACKUPS),
  createSaveBackup: (path, note) => Promise.resolve(mockData.MOCK_BACKUPS[0]),
  restoreSaveBackup: (path, id) => Promise.resolve(),
  deleteSaveBackup: (path, id) => Promise.resolve(),
  updateSaveManifest: () => Promise.resolve(),
  getConfigSchema: () => Promise.resolve(mockData.MOCK_CONFIG_SCHEMA || []),
  getGameConfigSchema: () => Promise.resolve((mockData.MOCK_CONFIG_SCHEMA || []).filter(p => p.scope === 'game' || p.scope === 'both')),
};

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
