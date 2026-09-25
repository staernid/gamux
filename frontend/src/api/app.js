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
  isGameRunning: () => false,
  getRunningGames: () => [],
  getRunningInstances: () => [],
  getConnectedDisplays: () => [
    { name: 'Display-1', width: 1920, height: 1080, refresh_rate: 60, is_primary: true, scale: 1.0, x: 0, y: 0 }
  ],
  getMultiseatCapabilities: () => ({
    has_bwrap: true,
    has_pasta: true,
    has_gamescope: true,
    has_cage: true,
    has_sunshine: true,
    has_pipewire: true
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
  getGameDetails: (appId) => Promise.resolve({
    name: 'Cyberpunk 2077',
    short_description: 'Cyberpunk 2077 is an open-world, action-adventure RPG set in Night City, a megalopolis obsessed with power, glamour, and body modification.',
    detailed_description: 'Cyberpunk 2077 is an open-world, action-adventure RPG set in Night City, a megalopolis obsessed with power, glamour, and body modification.',
    about_the_game: 'Cyberpunk 2077 is an open-world, action-adventure RPG set in the world of Night City.',
    developers: ['CD PROJEKT RED'],
    publishers: ['CD PROJEKT RED'],
    genres: ['RPG', 'Action', 'Open World', 'Sci-Fi'],
    release_date: 'Dec 10, 2020'
  }),
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
  getGameNews: (appId, count) => Promise.resolve(mockData.MOCK_GAMES[0].news_items || []),
  getPatchNote: () => Promise.resolve([null, '']),
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

function createMethodDispatcher(prop) {
  return async (...args) => {
    // Ensure InspectGame receives (gamePath, newsCount) defaults
    if (prop === 'inspectGame' && args.length === 1) {
      args = [args[0], 0];
    }
    // Ensure SelectDirectory receives (title, defaultDir) defaults
    if (prop === 'selectDirectory' && args.length === 1) {
      args = [args[0], ''];
    }

    // 1. Try Go backend runtime binding (Wails exposes PascalCase methods)
    const wailsApp = typeof window !== 'undefined' ? window.go?.gui?.App : null;
    if (wailsApp) {
      const pascal = toPascalCase(prop);
      if (typeof wailsApp[pascal] === 'function') {
        return await wailsApp[pascal](...args);
      }
      if (typeof wailsApp[prop] === 'function') {
        return await wailsApp[prop](...args);
      }
    }

    // 2. Try configured declarative mock handler
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
  { key: 'onOpenTarget', event: 'gamux:open-target' },
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
