package gui

import (
	"fmt"
	"log/slog"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/gbe"
)

// -----------------------------------------------------------------------------
// Configuration & Persona Settings
// -----------------------------------------------------------------------------

// GetConfig returns the current active configuration.
func (a *App) GetConfig() (*config.Config, error) {
	if a.Config == nil {
		a.Config = config.DefaultConfig()
	}
	return a.Config, nil
}

// SaveConfig validates and saves new configuration settings.
func (a *App) SaveConfig(newCfg *config.Config) error {
	if newCfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if err := config.SaveConfig(newCfg, ""); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	a.Config = newCfg
	a.Engine = engine.New(newCfg)
	slog.Info("Configuration updated and saved via GUI")
	return nil
}

// GetGSEUserConfig returns the global Goldberg user persona settings from configs.user.ini.
func (a *App) GetGSEUserConfig() (*gbe.GSEUserConfig, error) {
	if a.Engine != nil {
		return a.Engine.GetGSEUserConfig()
	}
	return gbe.LoadGSEUserConfig()
}

// SaveGSEUserConfig persists the global Goldberg user persona settings to configs.user.ini.
func (a *App) SaveGSEUserConfig(cfg *gbe.GSEUserConfig) error {
	if cfg == nil {
		return fmt.Errorf("gse user config cannot be nil")
	}
	if a.Engine != nil {
		return a.Engine.SaveGSEUserConfig(cfg)
	}
	return gbe.SaveGSEUserConfig(cfg)
}
