//go:build !linux

package gui

import "context"

func (a *App) monitorSystemThemeChanges(ctx context.Context) {}
