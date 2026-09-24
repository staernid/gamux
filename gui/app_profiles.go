package gui

import (
	"fmt"

	"github.com/staernid/gamux/domain"
)

// GetProfiles returns all configured player profiles.
func (a *App) GetProfiles() ([]*domain.Profile, error) {
	if a.Engine == nil || a.Engine.Profiles == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	return a.Engine.Profiles.List(), nil
}

// GetActiveProfile returns the currently active player profile.
func (a *App) GetActiveProfile() (*domain.Profile, error) {
	if a.Engine == nil || a.Engine.Profiles == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	return a.Engine.Profiles.GetActive(), nil
}

// SetActiveProfile switches the currently active profile and emits a change event.
func (a *App) SetActiveProfile(id string) error {
	if a.Engine == nil || a.Engine.Profiles == nil {
		return fmt.Errorf("profile manager not initialized")
	}
	if err := a.Engine.Profiles.SetActive(id); err != nil {
		return fmt.Errorf("set active profile: %w", err)
	}
	active := a.Engine.Profiles.GetActive()
	safeEmit(a.getContext(), "gamux:active-profile-changed", active)
	return nil
}

// CreateProfile creates and persists a new player profile.
func (a *App) CreateProfile(name string, steamID string, language string, country string, avatarURL string) (*domain.Profile, error) {
	if a.Engine == nil || a.Engine.Profiles == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	p, err := a.Engine.Profiles.Create(name, steamID, language, country, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}
	safeEmit(a.getContext(), "gamux:profiles-updated", a.Engine.Profiles.List())
	return p, nil
}

// SaveProfile updates an existing profile's fields and persists changes.
func (a *App) SaveProfile(p domain.Profile) error {
	if a.Engine == nil || a.Engine.Profiles == nil {
		return fmt.Errorf("profile manager not initialized")
	}
	if err := a.Engine.Profiles.Update(&p); err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	active := a.Engine.Profiles.GetActive()
	if active != nil && active.ID == p.ID {
		safeEmit(a.getContext(), "gamux:active-profile-changed", active)
	}
	safeEmit(a.getContext(), "gamux:profiles-updated", a.Engine.Profiles.List())
	return nil
}

// DeleteProfile removes a secondary profile and its isolated directory.
func (a *App) DeleteProfile(id string) error {
	if a.Engine == nil || a.Engine.Profiles == nil {
		return fmt.Errorf("profile manager not initialized")
	}
	if err := a.Engine.Profiles.Delete(id); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	active := a.Engine.Profiles.GetActive()
	safeEmit(a.getContext(), "gamux:active-profile-changed", active)
	safeEmit(a.getContext(), "gamux:profiles-updated", a.Engine.Profiles.List())
	return nil
}
