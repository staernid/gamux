// Safe Wails API Wrapper & Event Subscription Bridge

export const AppAPI = {
  async getConfig() {
    if (window.go?.gui?.App?.GetConfig) {
      return await window.go.gui.App.GetConfig();
    }
    return {
      gbe_dir: '~/.local/share/gbe_fork',
      steamless_dir: '~/.local/share/steamless',
      lutris_dir: '~/.config/lutris/games',
      steam_userdata: '~/.local/share/Steam/userdata',
      steam_store_api: 'https://store.steampowered.com/api',
      steam_web_api: 'https://api.steampowered.com',
      github_api_url: 'https://api.github.com/repos/Detanup01/gbe_fork/releases/latest',
      steamless_github_api: 'https://api.github.com/repos/staernid/steamless-rs/releases/latest',
      gbe_mode: 'loader',
      platform: 'win64',
      runner: 'wine',
      lutris: true,
      steamless: true,
      achievements: true,
      normalize: true,
      enable_launch_notify: true,
    };
  },

  async saveConfig(cfg) {
    if (window.go?.gui?.App?.SaveConfig) {
      return await window.go.gui.App.SaveConfig(cfg);
    }
  },

  async detectGame(path) {
    if (window.go?.gui?.App?.DetectGame) {
      return await window.go.gui.App.DetectGame(path);
    }
  },

  async inspectGame(path) {
    if (window.go?.gui?.App?.InspectGame) {
      return await window.go.gui.App.InspectGame(path);
    }
  },

  async inspectGameEx(path, count = 5) {
    if (window.go?.gui?.App?.InspectGameEx) {
      return await window.go.gui.App.InspectGameEx(path, count);
    }
  },

  async batchInspect(parentDir) {
    if (window.go?.gui?.App?.BatchInspect) {
      return await window.go.gui.App.BatchInspect(parentDir);
    }
    return [];
  },

  async processGame(opts) {
    if (window.go?.gui?.App?.ProcessGame) {
      return await window.go.gui.App.ProcessGame(opts);
    }
  },

  async applyGame(opts) {
    if (window.go?.gui?.App?.ApplyGame) {
      return await window.go.gui.App.ApplyGame(opts);
    }
  },

  async rollback(path, dryRun = false) {
    if (window.go?.gui?.App?.Rollback) {
      return await window.go.gui.App.Rollback(path, dryRun);
    }
  },

  async downloadGame(opts) {
    if (window.go?.gui?.App?.DownloadGame) {
      return await window.go.gui.App.DownloadGame(opts);
    }
  },

  async updateGame(path) {
    if (window.go?.gui?.App?.UpdateGame) {
      return await window.go.gui.App.UpdateGame(path);
    }
  },

  async getPatchNote(path, index) {
    if (window.go?.gui?.App?.GetPatchNote) {
      return await window.go.gui.App.GetPatchNote(path, index);
    }
  },

  async fetchNews(appID, count = 8) {
    if (window.go?.gui?.App?.FetchNews) {
      return await window.go.gui.App.FetchNews(appID, count);
    }
    return [];
  },

  async searchSteamGames(query) {
    if (window.go?.gui?.App?.SearchSteamGames) {
      return await window.go.gui.App.SearchSteamGames(query);
    }
    return [];
  },

  async selectDirectory(title, defaultDir = '') {
    if (window.go?.gui?.App?.SelectDirectory) {
      return await window.go.gui.App.SelectDirectory(title, defaultDir);
    }
    return prompt(title, defaultDir) || '';
  },

  async selectFile(title, filterName = '', filterPattern = '') {
    if (window.go?.gui?.App?.SelectFile) {
      return await window.go.gui.App.SelectFile(title, filterName, filterPattern);
    }
    return prompt(title) || '';
  },

  async getToolsStatus() {
    if (window.go?.gui?.App?.GetToolsStatus) {
      return await window.go.gui.App.GetToolsStatus();
    }
    return [];
  },

  async updateTool(toolKey) {
    if (window.go?.gui?.App?.UpdateTool) {
      return await window.go.gui.App.UpdateTool(toolKey);
    }
    return 'Tool updated';
  },

  async updateGBE() {
    if (window.go?.gui?.App?.UpdateGBE) {
      return await window.go.gui.App.UpdateGBE();
    }
    return 'GBE updated (mock)';
  },

  async updateSteamless() {
    if (window.go?.gui?.App?.UpdateSteamless) {
      return await window.go.gui.App.UpdateSteamless();
    }
    return 'Steamless updated (mock)';
  },

  async verifyIntegrity(path) {
    if (window.go?.gui?.App?.VerifyIntegrity) {
      return await window.go.gui.App.VerifyIntegrity(path);
    }
  },

  async openFolder(dirPath) {
    if (window.go?.gui?.App?.OpenFolder) {
      return await window.go.gui.App.OpenFolder(dirPath);
    }
  },

  async launchLutrisGame(slug) {
    if (window.go?.gui?.App?.LaunchLutrisGame) {
      return await window.go.gui.App.LaunchLutrisGame(slug);
    }
  }
};

// Setup runtime event listeners
export function initRuntimeEvents({ onNotification, onDownloadProgress, onStepProgress }) {
  if (window.runtime?.EventsOn) {
    if (onNotification) {
      window.runtime.EventsOn('gamux:notification', onNotification);
    }
    if (onDownloadProgress) {
      window.runtime.EventsOn('gamux:download-progress', onDownloadProgress);
    }
    if (onStepProgress) {
      window.runtime.EventsOn('gamux:step-progress', onStepProgress);
    }
  }
}
