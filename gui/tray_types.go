package gui

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/staernid/gamux/domain"
)

// TrayGame represents a game entry displayed in the system tray context menu.
type TrayGame struct {
	Name       string `json:"name"`
	GameDir    string `json:"game_dir"`
	LastPlayed int64  `json:"last_played"`
}

// TrayCallbacks defines callbacks for tray events.
type TrayCallbacks struct {
	Icon           []byte
	OnActivate     func()
	OnQuit         func()
	OnLaunchGame   func(gameDir string)
	GetRecentGames func() []TrayGame
}

// SortTrayGames sorts a slice of TrayGame by least time since last play session to most
// (i.e. descending LastPlayed timestamp) and returns at most limit games.
// Games with LastPlayed <= 0 or empty GameDir are omitted.
func SortTrayGames(games []TrayGame, limit int) []TrayGame {
	if limit <= 0 {
		limit = 3
	}
	var played []TrayGame
	for _, g := range games {
		if g.LastPlayed > 0 && strings.TrimSpace(g.GameDir) != "" {
			name := strings.TrimSpace(g.Name)
			if name == "" {
				name = filepath.Base(g.GameDir)
			}
			played = append(played, TrayGame{
				Name:       name,
				GameDir:    g.GameDir,
				LastPlayed: g.LastPlayed,
			})
		}
	}

	sort.Slice(played, func(i, j int) bool {
		if played[i].LastPlayed != played[j].LastPlayed {
			// Higher timestamp = more recent = least time since last play session
			return played[i].LastPlayed > played[j].LastPlayed
		}
		return strings.ToLower(played[i].Name) < strings.ToLower(played[j].Name)
	})

	if len(played) > limit {
		played = played[:limit]
	}
	return played
}

// FilterAndSortRecentGames extracts played games from a domain.Game slice,
// sorts them by least time since last play session to most (descending LastPlayed),
// and returns at most limit items as TrayGame.
func FilterAndSortRecentGames(games []*domain.Game, limit int) []TrayGame {
	if limit <= 0 {
		limit = 3
	}
	var candidates []TrayGame
	for _, g := range games {
		if g == nil || strings.TrimSpace(g.GameDir) == "" {
			continue
		}
		_, lp := g.EffectivePlaytime()
		if lp > 0 {
			candidates = append(candidates, TrayGame{
				Name:       gameTitle(g),
				GameDir:    g.GameDir,
				LastPlayed: lp,
			})
		}
	}

	return SortTrayGames(candidates, limit)
}

func gameTitle(g *domain.Game) string {
	if g == nil {
		return ""
	}
	if g.LaunchProfile != nil && strings.TrimSpace(g.LaunchProfile.Name) != "" {
		return strings.TrimSpace(g.LaunchProfile.Name)
	}
	if strings.TrimSpace(g.Name) != "" {
		return strings.TrimSpace(g.Name)
	}
	if g.GameDir != "" {
		return filepath.Base(g.GameDir)
	}
	return "Unknown Game"
}
