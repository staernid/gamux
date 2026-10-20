/**
 * Canonical Domain Entity Type Definitions for Gamux Frontend
 *
 * All frontend entities are standardized on camelCase idioms with
 * backwards-compatible snake_case aliases across the IPC boundary.
 */

export interface LaunchCandidate {
  name: string;
  executable: string;
  arguments?: string;
  description?: string;
}

export interface LaunchProfile {
  gameDir?: string;
  appId?: string;
  name?: string;
  platform?: string;
  exePath?: string;
  exeArgs?: string;
  operatingMode?: string;
  runner?: string;
  winePrefix?: string;
  protonPath?: string;
  language?: string;
  mangohud?: boolean;
  gamescope?: boolean;
  gamescopeWidth?: number;
  gamescopeHeight?: number;
  gamescopeFps?: number;
  disableOverlay?: boolean;
  offlineMode?: boolean;
  dllOverrides?: string;
  inhibitSleep?: boolean;
  preferSdl?: boolean;
  enableWayland?: boolean;
  customEnv?: Record<string, string>;
  workingDir?: string;
  profileId?: string;
  forceEmulation?: boolean;
  store?: string;
  storeAppId?: string;
  branch?: string;
  branchPassword?: string;
}

export interface Game {
  name: string;
  appId: string;
  gameDir: string;
  store: string;
  source: string;
  gogId?: string;
  metadataProvider?: string;
  metadataId?: string;
  platform: string;
  exePath: string;
  exeArgs?: string;
  launchCandidates: LaunchCandidate[];
  launchProfile?: LaunchProfile | null;
  branch?: string;
  branchPassword?: string;
  isInstalled: boolean;
  isStub: boolean;
  isVirtual: boolean;
  isPatched: boolean;
  isPortable?: boolean;
  isReady?: boolean;
  hasUpdatesOrIssues?: boolean;
  diskSizeBytes: number;
  fileCount: number;
  officialFileCount?: number;
  dlcCount?: number;
  achievementCount?: number;
  playtimeSeconds?: number;
  lastPlayed?: number;
  installedAt?: number;
  state?: string;
  targetLibPath?: string;
  coverUrl?: string;
  bannerUrl?: string;
  heroUrl?: string;
  logoUrl?: string;
  iconUrl?: string;
  rawDepotLayout?: boolean;
  depots?: any[];
  availablePlatforms?: string[];
  supportedLanguages?: string[];
  versionStatus?: any;
  hasUpdate?: boolean;
  remoteManifestId?: string;
  manifestId?: string;
  buildId?: string;
  newsItems?: any[];
  summary?: string;
  releaseDate?: string;
  developers?: string[];
  publishers?: string[];
  genres?: string[];
}

export interface RunningInstance {
  instanceId: string;
  gameDir: string;
  gameName: string;
  appId: string;
  profileId: string;
  seatId: string;
  pid: number;
  isMultiseat: boolean;
  startTime: number;
  mode?: string;
}

export interface TaskProgress {
  percent: number;
  currentBytes: number;
  totalBytes: number;
  speedBps: number;
  etaSeconds: number;
  currentItems: number;
  totalItems: number;
}

export interface Task {
  id: string;
  type: string;
  title: string;
  gameName: string;
  gameDir: string;
  appId: string;
  stage: string;
  details: string;
  status: 'queued' | 'pending' | 'running' | 'downloading' | 'paused' | 'completed' | 'failed' | 'cancelled';
  percent: number;
  currentBytes: number;
  totalBytes: number;
  speedBps: number;
  bytesPerSec: number;
  etaSeconds: number;
  currentItems: number;
  totalItems: number;
  store: string;
  posterUrl: string;
  depots: any[];
  capabilities: number;
  createdAt: number;
  finishedAt: number;
  error?: string;
  progress: TaskProgress;
}

export interface Profile {
  id: string;
  index: number;
  name: string;
  steamId: string;
  language: string;
  ipCountry: string;
  avatarUrl: string;
  isMain: boolean;
}

export interface SystemCapabilities {
  hasBwrap: boolean;
  hasGamescope: boolean;
  hasCage: boolean;
  hasSunshine: boolean;
  hasPipewire: boolean;
  hasInputPermissions: boolean;
  inputPermissionsError?: string;
}
