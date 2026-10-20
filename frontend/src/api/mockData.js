// Rich mock datasets for standalone browser dev mode, headless previews, and tests

export const MOCK_CONFIG = {
  gbe_dir: '~/.local/share/gbe_fork',
  steamless_dir: '~/.local/share/steamless',
  steam_userdata: '~/.local/share/Steam/userdata',
  steam_store_api: 'https://store.steampowered.com/api',
  steam_web_api: 'https://api.steampowered.com',
  github_api_url: 'https://api.github.com/repos/Detanup01/gbe_fork/releases/latest',
  steamless_github_api: 'https://api.github.com/repos/staernid/steamless-rs/releases/latest',
  operating_mode: 'loader',
  platform: 'win64',
  runner: 'wine',
  steamless: true,
  achievements: true,
  normalize: true,
  enable_launch_notify: true,
};

export const MOCK_GAMES = [
  {
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    game_dir: '/home/staernid/Games/Cyberpunk 2077',
    store: 'Steam',
    platform: 'win64',
    exe_path: 'bin/x64/Cyberpunk2077.exe',
    exe_args: '-vulkan',
    is_patched: true,
    is_portable: true,
    is_ready: true,
    disk_size_bytes: 73014444032,
    file_count: 1420,
    official_file_count: 1420,
    dlc_count: 3,
    achievement_count: 44,
    operating_mode: 'portable',
    runner: 'wine',
    state: 'Portable-Patched',
    installed_at: 1700000000,
    playtime_seconds: 180000,
    last_played: 1724589000,
    current_language: 'english',
    supported_languages: ['english', 'german', 'french', 'spanish', 'japanese'],
    launch_candidates: [
      { name: 'Cyberpunk 2077 (Main)', executable: 'bin/x64/Cyberpunk2077.exe', arguments: '' },
      { name: 'Cyberpunk 2077 (REDlauncher)', executable: 'REDprelauncher.exe', arguments: '' }
    ],
    news_items: [
      { title: 'Patch 2.13 Available Now', contents: 'Adds support for AMD FidelityFX Super Resolution 3 and Intel XeSS 1.3.', date: 1726142400, feedlabel: 'Community Announcements' },
      { title: 'Night City Wire Special', contents: 'Deep dive into Phantom Liberty enhancements and vehicle combat.', date: 1720000000, feedlabel: 'News' },
      { title: 'Patch 2.12 Update Notes', contents: 'Fixes key binding issues and CPU utilization on hybrid architectures.', date: 1715000000, feedlabel: 'Patch Notes' }
    ]
  },
  {
    name: 'ELDEN RING',
    app_id: '1245620',
    game_dir: '/home/staernid/Games/ELDEN RING',
    store: 'Steam',
    platform: 'win64',
    exe_path: 'Game/eldenring.exe',
    exe_args: '',
    is_patched: true,
    is_portable: true,
    is_ready: true,
    disk_size_bytes: 48900000000,
    file_count: 850,
    official_file_count: 850,
    dlc_count: 1,
    achievement_count: 42,
    operating_mode: 'portable',
    runner: 'umu',
    state: 'Portable-Patched',
    installed_at: 1690000000,
    playtime_seconds: 72000,
    last_played: 1720000000,
    current_language: 'english',
    supported_languages: ['english', 'japanese'],
    launch_candidates: [],
    news_items: []
  },
  {
    name: 'Hades II',
    app_id: '1145350',
    game_dir: '/home/staernid/Games/Hades II',
    store: 'Steam',
    platform: 'win64',
    exe_path: 'Ship/Hades2.exe',
    exe_args: '',
    is_patched: true,
    is_portable: false,
    is_ready: true,
    disk_size_bytes: 8500000000,
    file_count: 320,
    official_file_count: 320,
    dlc_count: 0,
    achievement_count: 0,
    operating_mode: 'loader',
    runner: 'wine',
    state: 'Loader-Configured',
    installed_at: 1715000000,
    playtime_seconds: 36000,
    last_played: 1722000000,
    current_language: 'english',
    supported_languages: ['english'],
    launch_candidates: [],
    news_items: []
  },
  {
    name: 'Hollow Knight: Silksong',
    app_id: '1030300',
    game_dir: '/home/staernid/Games/Silksong',
    store: 'Steam',
    platform: 'linux',
    exe_path: 'Silksong.x86_64',
    exe_args: '',
    is_patched: false,
    is_portable: false,
    is_ready: false,
    disk_size_bytes: 9200000000,
    file_count: 410,
    official_file_count: 410,
    dlc_count: 0,
    achievement_count: 0,
    operating_mode: '',
    runner: 'linux',
    state: 'Original',
    installed_at: 1720000000,
    playtime_seconds: 0,
    last_played: 0,
    current_language: 'english',
    supported_languages: ['english'],
    launch_candidates: [],
    news_items: []
  },
  {
    name: "Baldur's Gate 3",
    app_id: '1086940',
    game_dir: "/home/staernid/Games/Baldurs Gate 3",
    store: 'Steam',
    platform: 'win64',
    exe_path: 'bin/bg3_dx11.exe',
    exe_args: '',
    is_patched: true,
    is_portable: true,
    is_ready: false,
    disk_size_bytes: 131072000000,
    file_count: 2450,
    official_file_count: 2450,
    dlc_count: 2,
    achievement_count: 53,
    operating_mode: 'portable',
    runner: 'wine',
    state: 'Portable-Patched',
    installed_at: 1695000000,
    playtime_seconds: 250000,
    last_played: 1723000000,
    launch_candidates: [
      { name: 'Baldur\'s Gate 3 (DX11)', executable: 'bin/bg3_dx11.exe', arguments: '' },
      { name: 'Baldur\'s Gate 3 (Vulkan)', executable: 'bin/bg3.exe', arguments: '' }
    ]
  },
  {
    name: 'Celeste',
    app_id: '504230',
    game_dir: '/home/staernid/Games/Celeste',
    store: 'Steam',
    platform: 'linux',
    exe_path: 'Celeste.bin.x86_64',
    exe_args: '',
    is_patched: false,
    is_portable: false,
    is_ready: false,
    disk_size_bytes: 1572864000,
    file_count: 42,
    official_file_count: 42,
    dlc_count: 0,
    achievement_count: 32,
    operating_mode: '',
    runner: 'linux',
    state: 'Clean',
    installed_at: 1680000000,
    playtime_seconds: 14400,
    last_played: 1710000000,
    launch_candidates: [
      { name: 'Celeste', executable: 'Celeste.bin.x86_64', arguments: '' }
    ]
  }
];

export const MOCK_TASKS = [
  {
    id: 'tr_mock_1',
    type: 'download',
    app_id: '292030',
    title: 'The Witcher 3: Wild Hunt',
    game_name: 'The Witcher 3: Wild Hunt',
    status: 'downloading',
    stage: 'Downloading Depots',
    details: '292031 (Content)',
    capabilities: 15,
    progress: {
      percent: 68.5,
      current_bytes: 35651584000,
      total_bytes: 52000000000,
      bytes_per_sec: 25165824,
      eta_seconds: 650,
      current_items: 3400,
      total_items: 5000
    },
    bytes_downloaded: 35651584000,
    total_bytes: 52000000000,
    speed_bps: 25165824,
    eta_seconds: 650,
    store: 'Steam',
    depots: [
      { id: '292031', name: 'Base Content', downloaded_bytes: 28000000000, total_bytes: 35000000000, chunks_downloaded: 2800, total_chunks: 3500, percent: 80.0 },
      { id: '292032', name: 'Executables & Binaries', downloaded_bytes: 1200000000, total_bytes: 1200000000, chunks_downloaded: 120, total_chunks: 120, percent: 100.0 },
      { id: '292033', name: 'English Voiceovers', downloaded_bytes: 6451584000, total_bytes: 15800000000, chunks_downloaded: 480, total_chunks: 1380, percent: 40.8 }
    ]
  },
  {
    id: 'tr_mock_2',
    type: 'download',
    app_id: '413150',
    title: 'Stardew Valley',
    game_name: 'Stardew Valley',
    status: 'paused',
    stage: 'Paused',
    details: 'Paused by user',
    capabilities: 15,
    progress: {
      percent: 42.0,
      current_bytes: 220200960,
      total_bytes: 524288000,
      bytes_per_sec: 0,
      eta_seconds: 0,
      current_items: 210,
      total_items: 500
    },
    bytes_downloaded: 220200960,
    total_bytes: 524288000,
    speed_bps: 0,
    eta_seconds: 0,
    store: 'Steam'
  },
  {
    id: 'tr_mock_3',
    type: 'download',
    app_id: '1145360',
    title: 'Hades II',
    game_name: 'Hades II',
    status: 'queued',
    stage: 'Queued',
    details: 'Waiting in queue',
    capabilities: 15,
    progress: {
      percent: 0.0,
      current_bytes: 0,
      total_bytes: 10485760000,
      bytes_per_sec: 0,
      eta_seconds: 0,
      current_items: 0,
      total_items: 1000
    },
    bytes_downloaded: 0,
    total_bytes: 10485760000,
    speed_bps: 0,
    eta_seconds: 0,
    store: 'Steam'
  },
  {
    id: 'op_mock_verify',
    type: 'verify',
    app_id: '1091500',
    title: 'Verify Integrity (Cyberpunk 2077)',
    game_name: 'Cyberpunk 2077',
    status: 'running',
    stage: 'Verifying Integrity',
    details: 'Scanning archive hashes (650/1420 files)',
    capabilities: 1,
    progress: {
      percent: 45.8,
      current_bytes: 33444000000,
      total_bytes: 73014444032,
      current_items: 650,
      total_items: 1420,
      bytes_per_sec: 0,
      eta_seconds: 0
    },
    store: 'Steam'
  },
  {
    id: 'op_mock_completed',
    type: 'apply',
    app_id: '1245620',
    title: 'Apply Setup (ELDEN RING)',
    game_name: 'ELDEN RING',
    status: 'completed',
    stage: 'Completed',
    details: 'Setup applied successfully with Goldberg Emulator hooks.',
    capabilities: 1,
    progress: {
      percent: 100,
      current_items: 1,
      total_items: 1
    },
    finished_at: Date.now() - 3600000,
    store: 'Steam'
  }
];

export const MOCK_TRANSFERS = MOCK_TASKS;

export const MOCK_TOOLS_STATUS = [
  {
    name: 'Goldberg Emulator (GBE)',
    key: 'gbe',
    installed_version: 'v0.2.6',
    installed_tag: 'v0.2.6',
    latest_version: 'v0.2.7',
    latest_tag: 'v0.2.7',
    has_update: true,
    is_installed: true,
    description: 'Steam DRM removal and offline multiplayer emulator'
  },
  {
    tool_name: 'steamless',
    name: 'Steamless DRM Unpacker',
    display_name: 'Steamless DRM Unpacker',
    key: 'steamless',
    installed_version: 'v0.1.2',
    installed_tag: 'v0.1.2',
    latest_version: 'v0.1.2',
    latest_tag: 'v0.1.2',
    has_update: false,
    is_installed: true,
    release_date: '2026-06-15',
    release_notes: '• Native ELF header parsing optimization.\n• Support for newer SteamStub packaging wrappers.',
  },
  {
    name: 'UMU Proton Launcher',
    key: 'umu',
    installed_version: '',
    installed_tag: '',
    latest_version: 'v1.1.4',
    latest_tag: 'v1.1.4',
    has_update: true,
    is_installed: false,
    description: 'Unified Linux Wine and Proton runtime launcher'
  }
];

export const MOCK_GBE_STATUS = {
  has_update: true,
  installed_tag: 'v0.2.6',
  latest_tag: 'v0.2.7',
  release_name: 'GBE Fork v0.2.7 - Enhanced Lobby & DLC Support',
  published_at: '2026-08-15T12:00:00Z',
  release_notes: '• Fixed steamclient64.dll overlay hooks for DirectX 12\n• Added support for automated DLC unlock mapping\n• Improved LAN lobby broadcast reliability on Linux',
  release_url: 'https://github.com/Detanup01/gbe_fork/releases/tag/v0.2.7'
};

export const MOCK_SAVE_LOCATIONS = {
  game_dir: '/home/staernid/Games/Cyberpunk 2077',
  game_name: 'Cyberpunk 2077',
  app_id: '1091500',
  platform: 'win64',
  total_save_bytes: 45219840,
  total_config_files: 3,
  has_cloud_saves: true,
  primary_save_path: '/home/staernid/Games/umu/umu-default/drive_c/users/steamuser/Saved Games/CD Projekt Red/Cyberpunk 2077',
  primary_config_path: '/home/staernid/Games/umu/umu-default/drive_c/users/steamuser/AppData/Local/CD Projekt Red/Cyberpunk 2077',
  locations: [
    {
      path: '/home/staernid/Games/umu/umu-default/drive_c/users/steamuser/Saved Games/CD Projekt Red/Cyberpunk 2077',
      display_path: '~/Games/umu/umu-default/drive_c/users/steamuser/Saved Games/CD Projekt Red/Cyberpunk 2077',
      category: 'save',
      tags: ['save'],
      exists: true,
      is_dir: true,
      file_count: 12,
      size_bytes: 45219840,
      source: 'ludusavi'
    },
    {
      path: '/home/staernid/Games/umu/umu-default/drive_c/users/steamuser/AppData/Local/CD Projekt Red/Cyberpunk 2077',
      display_path: '~/Games/umu/umu-default/drive_c/users/steamuser/AppData/Local/CD Projekt Red/Cyberpunk 2077',
      category: 'config',
      tags: ['config'],
      exists: true,
      is_dir: true,
      file_count: 3,
      size_bytes: 16384,
      source: 'ludusavi'
    },
    {
      path: '/home/staernid/.local/share/GSE Saves/1091500/remote',
      display_path: '~/.local/share/GSE Saves/1091500/remote',
      category: 'cloud',
      tags: ['save', 'cloud'],
      exists: true,
      is_dir: true,
      file_count: 4,
      size_bytes: 819200,
      source: 'gbe'
    }
  ]
};

export const MOCK_BACKUPS = [
  {
    id: '2026-08-25_14-30-00',
    zip_path: '/home/staernid/.local/share/gamux/backups/Cyberpunk_2077/2026-08-25_14-30-00.zip',
    app_id: '1091500',
    game_title: 'Cyberpunk 2077',
    created_at: 1724589000,
    created_by: 'manual',
    note: 'Before Phantom Liberty ending',
    file_count: 15,
    size_bytes: 38400000
  }
];

export const MOCK_CONFIG_SCHEMA = [
  { key: 'library_paths', label: 'Library Folders', description: 'Directories where Gamux scans and discovers installed game directories.', type: 'path_list', category: 'Paths', scope: 'global', default: [] },
  { key: 'gbe_dir', label: 'Goldberg Emulator Directory', description: 'Path to Goldberg Steam Emulator binaries and overlay libraries.', type: 'path', category: 'Paths', scope: 'global', default: '' },
  { key: 'artwork_dir', label: 'Artwork Cache Directory', description: 'Directory where downloaded posters, banners, and grid artwork are stored.', type: 'path', category: 'Paths', scope: 'global', default: '' },
  { key: 'steam_web_api_key', label: 'Steam Web API Key', description: 'API key used to query the official Steam Web API.', type: 'string', category: 'API Keys', scope: 'global', default: '' },
  { key: 'hubcap_api_key', label: 'Hubcap API Key', description: 'API key used to retrieve community manifests and metadata via Hubcap.', type: 'string', category: 'API Keys', scope: 'global', default: '' },
  { key: 'steamgriddb_api_key', label: 'SteamGridDB API Key', description: 'API key for SteamGridDB to fetch custom high-res box art.', type: 'string', category: 'API Keys', scope: 'global', default: '' },
  { key: 'rawg_api_key', label: 'RAWG API Key', description: 'API key for RAWG Video Games Database.', type: 'string', category: 'API Keys', scope: 'global', default: '' },
  { key: 'enable_launch_notify', label: 'Launch Notifications', description: 'Display a desktop notification when a game process launches or exits.', type: 'boolean', category: 'Lifecycle', scope: 'global', default: true },
  { key: 'launch_window_behavior', label: 'Launch Window Behavior', description: 'Action taken with the Gamux main window when a game is launched.', type: 'select', category: 'Lifecycle', scope: 'global', default: 'keep', options: [{ value: 'keep', label: 'Keep Window Open' }, { value: 'minimize', label: 'Minimize to Taskbar' }, { value: 'hide', label: 'Hide Window' }] },
  { key: 'close_to_tray', label: 'Close to System Tray', description: 'Minimize application to system tray when the main window is closed.', type: 'boolean', category: 'Lifecycle', scope: 'global', default: false },
  { key: 'operating_mode', label: 'Operating Mode', description: 'Default integration method for applying Goldberg emulator hooks.', type: 'select', category: 'Operational', scope: 'both', default: 'loader', options: [{ value: 'loader', label: 'Loader Mode' }, { value: 'portable', label: 'Portable Mode' }, { value: 'none', label: 'None' }] },
  { key: 'language', label: 'Game Language', description: 'Game language reported to the Goldberg emulator.', type: 'select', category: 'Operational', scope: 'both', default: 'auto', options: [{ value: 'auto', label: 'Automatic' }, { value: 'english', label: 'English' }] },
  { key: 'runner', label: 'Runner Layer', description: 'Execution environment used to run this title.', type: 'select', category: 'Runtime', scope: 'game', default: 'umu', inherits_from: 'default_runner', options: [{ value: 'umu', label: 'UMU / Proton' }, { value: 'wine', label: 'Wine Direct' }, { value: 'proton', label: 'Proton' }, { value: 'linux', label: 'Linux Native' }] },
  { key: 'dll_overrides', label: 'DLL Overrides', description: 'Custom native/builtin DLL loading rules.', type: 'string', category: 'Runtime', scope: 'game', default: '' },
  { key: 'disable_overlay', label: 'Disable In-Game Overlay', description: 'Disable Goldberg overlay hooks and telemetry.', type: 'boolean', category: 'Operational', scope: 'both', default: false },
  { key: 'offline_mode', label: 'Offline Mode', description: 'Force offline mode across all emulated Steam interfaces.', type: 'boolean', category: 'Operational', scope: 'both', default: false },
  { key: 'achievements', label: 'Achievement Tracking', description: 'Capture and display in-game achievement unlocks.', type: 'boolean', category: 'Operational', scope: 'both', default: true },
  { key: 'steamless', label: 'Automatic DRM Unpacking', description: 'Run Steamless on main executables during pipeline scan.', type: 'boolean', category: 'Operational', scope: 'both', default: true },
  { key: 'normalize', label: 'Normalize Game Folders', description: 'Clean and standardize directory layout.', type: 'boolean', category: 'Operational', scope: 'both', default: true },
  { key: 'transfer_speed_limit', label: 'Transfer Speed Limit', description: 'Maximum transfer speed limit in bytes per second.', type: 'number', category: 'Operational', scope: 'global', default: 0 },
  { key: 'default_runner', label: 'Default Runner', description: 'Default compatibility runner used to execute Windows PE titles.', type: 'select', category: 'Runtime', scope: 'both', default: 'umu', options: [{ value: 'umu', label: 'UMU / Proton' }, { value: 'wine', label: 'Wine Direct' }] },
  { key: 'wine_prefix', label: 'Wine Prefix', description: 'Path to custom WINEPREFIX container directory.', type: 'path', category: 'Runtime', scope: 'both', default: '' },
  { key: 'global_env', label: 'Global Environment Variables', description: 'Key-value environment variables passed to all launched games.', type: 'env', category: 'Operational', scope: 'global', default: {} },
  { key: 'mangohud', label: 'MangoHud Overlay', description: 'Render Vulkan / OpenGL HUD with FPS and hardware metrics.', type: 'boolean', category: 'Graphics', scope: 'both', default: false },
  { key: 'auto_backup_saves', label: 'Auto-Backup Saves', description: 'Automatically create a zip snapshot backup before launching.', type: 'boolean', category: 'Saves & Backups', scope: 'both', default: false },
  { key: 'backup_retention_count', label: 'Backup Retention Limit', description: 'Maximum number of rolling backup snapshots to retain.', type: 'number', category: 'Saves & Backups', scope: 'both', default: 5, min: 1, max: 50, step: 1 }
];

export const MOCK_MOD_STATUS = {
  topology: {
    engine: 'unity',
    mod_dir: 'BepInEx/plugins',
    absolute_mod_dir: '/home/staernid/Games/Risk of Rain 2/BepInEx/plugins',
    detected: true,
    supported_formats: ['.dll', '.zip']
  },
  mods: [
    {
      id: 'ItemStatsMod.dll',
      name: 'ItemStatsMod',
      workshop_id: 315783921,
      enabled: true,
      path: '/home/staernid/Games/Risk of Rain 2/BepInEx/plugins/ItemStatsMod.dll',
      relative_path: 'BepInEx/plugins/ItemStatsMod.dll',
      time_updated: 1720000000,
      latest_time_updated: 1729000000,
      has_update: true,
      description: 'Displays detailed item stats and descriptions in the HUD.',
      preview_url: '',
      size: 1048576,
      files: ['ItemStatsMod.dll']
    },
    {
      id: 'ProperSave.dll',
      name: 'ProperSave',
      workshop_id: 315783922,
      enabled: false,
      path: '/home/staernid/Games/Risk of Rain 2/BepInEx/plugins/ProperSave.dll.disabled',
      relative_path: 'BepInEx/plugins/ProperSave.dll.disabled',
      time_updated: 1715000000,
      latest_time_updated: 1715000000,
      has_update: false,
      description: 'Saves your run progress at the beginning of each stage.',
      preview_url: '',
      size: 524288,
      files: ['ProperSave.dll']
    }
  ],
  outdated_count: 1
};

