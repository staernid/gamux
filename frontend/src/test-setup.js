import '@testing-library/jest-dom/vitest';
import { beforeEach, vi } from 'vitest';

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
        SelectDirectory: vi.fn().mockResolvedValue('/home/staernid/Games'),
      }
    }
  };
});
