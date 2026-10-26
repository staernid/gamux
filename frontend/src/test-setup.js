import '@testing-library/jest-dom/vitest';
import { beforeEach, vi } from 'vitest';

// Mock @wailsio/runtime Call and Events so unit tests don't make real backend calls,
// while preserving Create and other runtime utilities needed by generated models.
vi.mock('@wailsio/runtime', async (importOriginal) => {
  const origWarn = console.warn;
  console.warn = (...args) => {
    if (typeof args[0] === 'string' && args[0].includes('Browser Environment Detected')) return;
    origWarn.apply(console, args);
  };
  try {
    const actual = await importOriginal();
    return {
      ...actual,
      Call: {
        ...actual.Call,
        ByName: vi.fn(),
        ByID: vi.fn(),
      },
      Events: {
        ...actual.Events,
        On: vi.fn(),
        Emit: vi.fn(),
      },
    };
  } finally {
    console.warn = origWarn;
  }
});

// Global mock for Wails window.runtime & window.go if needed
beforeEach(() => {
  window.runtime = {
    EventsOn: vi.fn(),
    EventsEmit: vi.fn(),
  };
  window.go = {
    gui: {
      App: {
        ListTransfers: vi.fn().mockResolvedValue([]),
        StartDownloadTransfer: vi.fn().mockResolvedValue('dl_mock_1'),
        PauseTransfer: vi.fn().mockResolvedValue(true),
        ResumeTransfer: vi.fn().mockResolvedValue(true),
        CancelTransfer: vi.fn().mockResolvedValue(true),
        RemoveTransfer: vi.fn().mockResolvedValue(true),
        ReorderTransfers: vi.fn().mockResolvedValue(true),
        SetTransferSpeedLimit: vi.fn().mockResolvedValue(true),
        GetTransferSpeedLimit: vi.fn().mockResolvedValue(0),
        SearchStoreGames: vi.fn().mockResolvedValue([]),
        UpdateGBE: vi.fn().mockResolvedValue('dl_gbe_123'),
        UpdateSteamless: vi.fn().mockResolvedValue('dl_steamless_123'),
        GetToolsStatus: vi.fn().mockResolvedValue([]),
        GetGBEStatus: vi.fn().mockResolvedValue({ has_update: true, latest_tag: 'v0.2.7' }),
        GetSteamlessStatus: vi.fn().mockResolvedValue({ has_update: false }),
        CheckGameUpdates: vi.fn().mockResolvedValue({ checked: true, is_up_to_date: true }),
        CheckLibraryUpdates: vi.fn().mockResolvedValue({ total_checked: 0, updates_found: 0, updated_games: [] }),
        SelectDirectory: vi.fn().mockResolvedValue('~/Games'),
        GetInitialTarget: vi.fn().mockResolvedValue(''),
        CreateGameStub: vi.fn().mockImplementation((lib, id, name, store, branch = 'public') => Promise.resolve({
          name: name || `Game ${id}`,
          app_id: id,
          branch: branch || 'public',
          game_dir: `${lib}/${name || id}`,
          store: store || 'Steam',
          file_count: 0,
          disk_size_bytes: 0,
          is_installed: false,
          is_stub: true,
          is_ready: false,
        })),
        GetSteamBranches: vi.fn().mockResolvedValue([
          { name: 'public', description: 'Default / Public release', build_id: '10180494', time_updated: 1671497797, pwd_required: false },
          { name: 'beta', description: 'Beta branch for testing new content and risky fixes. Unstable', build_id: '10174821', time_updated: 1671497193, pwd_required: false },
          { name: 'v1.0', description: 'Legacy 2019-23-01', build_id: '3495375', time_updated: 1562010477, pwd_required: false },
        ]),
        SetGameBranch: vi.fn().mockImplementation((dir, branch, password) => Promise.resolve({
          name: 'Game',
          game_dir: dir,
          branch: branch || 'public',
          launch_profile: {
            branch: branch || 'public',
            branch_password: password || '',
          },
        })),
      }
    }
  };
});

