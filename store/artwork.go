package store

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/util"
)

// ArtworkService provides high-level multi-tier artwork resolution, caching, and runner export.
type ArtworkService struct {
	cacheDir   string
	registry   *Registry
	httpClient *http.Client
}

// NewArtworkService creates an artwork resolution engine.
func NewArtworkService(cacheDir string, registry *Registry) *ArtworkService {
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".local", "share", "gamux", "art")
	}
	return &ArtworkService{
		cacheDir:   util.ExpandPath(cacheDir),
		registry:   registry,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// DownloadFile downloads a URL to a local destination file path atomically.
func (s *ArtworkService) DownloadFile(ctx context.Context, downloadURL, targetPath string) error {
	if downloadURL == "" || targetPath == "" {
		return fmt.Errorf("invalid download URL or target path")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Game Manager)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", downloadURL, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(targetPath), err)
	}

	tmpPath := targetPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file %s: %w", tmpPath, err)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write file: %w", err)
	}
	out.Close()

	return os.Rename(tmpPath, targetPath)
}

// Resolve executes the multi-tier artwork waterfall for a game and populates its GameArtwork.
func (s *ArtworkService) Resolve(ctx context.Context, game *domain.Game) (*domain.GameArtwork, error) {
	if game == nil {
		return nil, fmt.Errorf("game cannot be nil")
	}

	slug := lutris.Slugify(game.Name)
	if slug == "" {
		slug = "game-" + game.AppID
	}

	gameArtDir := filepath.Join(s.cacheDir, slug)
	cachedCover := filepath.Join(gameArtDir, "cover.jpg")
	cachedBanner := filepath.Join(gameArtDir, "banner.jpg")

	// 0. Check custom/locked artwork preference first
	if game.CustomCoverURL == "none" {
		game.LocalCover = ""
		game.LocalCoverURL = ""
		game.CoverURL = ""
	} else if game.CustomCoverURL != "" {
		if strings.HasPrefix(game.CustomCoverURL, "http://") || strings.HasPrefix(game.CustomCoverURL, "https://") {
			if !util.FileExists(cachedCover) || game.LocalCover == "" {
				_ = s.DownloadFile(ctx, game.CustomCoverURL, cachedCover)
			}
			game.LocalCover = cachedCover
			game.CoverURL = game.CustomCoverURL
		} else if util.FileExists(game.CustomCoverURL) {
			game.LocalCover = game.CustomCoverURL
		}
	}

	if game.CustomBannerURL == "none" {
		game.LocalBanner = ""
		game.LocalBannerURL = ""
		game.BannerURL = ""
	} else if game.CustomBannerURL != "" {
		if strings.HasPrefix(game.CustomBannerURL, "http://") || strings.HasPrefix(game.CustomBannerURL, "https://") {
			if !util.FileExists(cachedBanner) || game.LocalBanner == "" {
				_ = s.DownloadFile(ctx, game.CustomBannerURL, cachedBanner)
			}
			game.LocalBanner = cachedBanner
			game.BannerURL = game.CustomBannerURL
		} else if util.FileExists(game.CustomBannerURL) {
			game.LocalBanner = game.CustomBannerURL
		}
	}

	// 1. Check local game folder first for custom/bundled artwork
	if game.GameDir != "" {
		if game.CustomCoverURL != "none" {
			candidates := []string{
				"cover.jpg", "cover.png", "poster.jpg", "poster.png",
				filepath.Join(".gamux", "art", "cover.jpg"),
				filepath.Join(".gamux", "art", "cover.png"),
			}
			for _, c := range candidates {
				p := filepath.Join(game.GameDir, c)
				if util.FileExists(p) {
					if game.LocalCover == "" {
						game.LocalCover = p
					}
					break
				}
			}
		}
		if game.CustomBannerURL != "none" {
			bannerCandidates := []string{
				"banner.jpg", "banner.png", "hero.jpg", "hero.png", "header.jpg",
				filepath.Join(".gamux", "art", "banner.jpg"),
				filepath.Join(".gamux", "art", "hero.jpg"),
			}
			for _, c := range bannerCandidates {
				p := filepath.Join(game.GameDir, c)
				if util.FileExists(p) {
					if game.LocalBanner == "" {
						game.LocalBanner = p
					}
					break
				}
			}
		}
	}

	// 2. If cached artwork exists on disk, reuse it
	if game.CustomCoverURL != "none" && game.LocalCover == "" && util.FileExists(cachedCover) {
		game.LocalCover = cachedCover
	}
	if game.CustomBannerURL != "none" && game.LocalBanner == "" && util.FileExists(cachedBanner) {
		game.LocalBanner = cachedBanner
	}

	// 3. Strategy: Steam CDN (if Steam AppID is present and artwork not locked)
	steamID, hasSteamID := game.SteamAppID()
	if !hasSteamID && game.AppID != "" && game.AppID != "0" {
		steamID, hasSteamID = util.ParseAppID(game.AppID)
	}

	if hasSteamID && steamID > 0 && !game.LockedArtwork {
		idStr := fmt.Sprintf("%d", steamID)
		if game.CustomCoverURL != "none" {
			if game.CoverURL == "" {
				game.CoverURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", idStr)
			}
			if game.LocalCover == "" {
				if err := s.DownloadFile(ctx, game.CoverURL, cachedCover); err == nil {
					game.LocalCover = cachedCover
				} else {
					// Fallback to 1x poster or header
					fallbackURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", idStr)
					if err := s.DownloadFile(ctx, fallbackURL, cachedCover); err == nil {
						game.LocalCover = cachedCover
					}
				}
			}
		}
		if game.CustomBannerURL != "none" {
			if game.BannerURL == "" {
				game.BannerURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", idStr)
			}
			if game.LocalBanner == "" {
				if err := s.DownloadFile(ctx, game.BannerURL, cachedBanner); err == nil {
					game.LocalBanner = cachedBanner
				}
			}
		}
		if game.HeroURL == "" {
			game.HeroURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg", idStr)
		}
		if game.LogoURL == "" {
			game.LogoURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/logo.png", idStr)
		}
		if game.IconURL == "" {
			game.IconURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_sm_120.jpg", idStr)
		}
	}

	// 4. Strategy: SteamGridDB (if configured and artwork is missing)
	if s.registry != nil && !game.LockedArtwork && ((game.CustomCoverURL != "none" && game.LocalCover == "") || (game.CustomBannerURL != "none" && game.LocalBanner == "")) {
		if p, ok := s.registry.Get("steamgriddb"); ok {
			var sgdbID string
			if hasSteamID && steamID > 0 {
				if cp, ok := p.(*SGDBProvider); ok {
					if id, _, err := cp.GetGameBySteamAppID(ctx, steamID); err == nil {
						sgdbID = id
					}
				}
			}
			if sgdbID == "" && game.Name != "" {
				if results, err := p.Search(ctx, game.Name); err == nil && len(results) > 0 {
					sgdbID = results[0].ID
				}
			}

			if sgdbID != "" {
				game.SGDBID = sgdbID
				if art, err := p.GetArtwork(ctx, sgdbID); err == nil && art != nil {
					if game.CoverURL == "" && art.CoverURL != "" {
						game.CoverURL = art.CoverURL
					}
					if game.BannerURL == "" && art.BannerURL != "" {
						game.BannerURL = art.BannerURL
					}
					if game.HeroURL == "" && art.HeroURL != "" {
						game.HeroURL = art.HeroURL
					}
					if game.LogoURL == "" && art.LogoURL != "" {
						game.LogoURL = art.LogoURL
					}
					if game.IconURL == "" && art.IconURL != "" {
						game.IconURL = art.IconURL
					}

					if game.LocalCover == "" && art.CoverURL != "" {
						if err := s.DownloadFile(ctx, art.CoverURL, cachedCover); err == nil {
							game.LocalCover = cachedCover
						}
					}
					if game.LocalBanner == "" && art.BannerURL != "" {
						if err := s.DownloadFile(ctx, art.BannerURL, cachedBanner); err == nil {
							game.LocalBanner = cachedBanner
						}
					}
				}
			}
		}
	}

	// 5. Strategy: GOG CDN (if GOG ID is present)
	if s.registry != nil && !game.LockedArtwork && (game.LocalCover == "" || game.LocalBanner == "") && game.GOGID != "" {
		if p, ok := s.registry.Get("gog"); ok {
			if art, err := p.GetArtwork(ctx, game.GOGID); err == nil && art != nil {
				if game.CoverURL == "" && art.CoverURL != "" {
					game.CoverURL = art.CoverURL
				}
				if game.BannerURL == "" && art.BannerURL != "" {
					game.BannerURL = art.BannerURL
				}
				if game.LocalCover == "" && art.CoverURL != "" {
					if err := s.DownloadFile(ctx, art.CoverURL, cachedCover); err == nil {
						game.LocalCover = cachedCover
					}
				}
				if game.LocalBanner == "" && art.BannerURL != "" {
					if err := s.DownloadFile(ctx, art.BannerURL, cachedBanner); err == nil {
						game.LocalBanner = cachedBanner
					}
				}
			}
		}
	}

	// 6. Set local web-accessible URLs
	if game.LocalCover != "" && util.FileExists(game.LocalCover) {
		if game.LocalCover != cachedCover {
			_ = os.MkdirAll(gameArtDir, 0755)
			_ = util.CopyFile(game.LocalCover, cachedCover)
		}
		var mtime int64 = 1
		if fi, err := os.Stat(cachedCover); err == nil {
			mtime = fi.ModTime().Unix()
		}
		game.LocalCoverURL = fmt.Sprintf("/art/%s/cover.jpg?t=%d", slug, mtime)
	} else if util.FileExists(cachedCover) {
		var mtime int64 = 1
		if fi, err := os.Stat(cachedCover); err == nil {
			mtime = fi.ModTime().Unix()
		}
		game.LocalCover = cachedCover
		game.LocalCoverURL = fmt.Sprintf("/art/%s/cover.jpg?t=%d", slug, mtime)
	}

	if game.LocalBanner != "" && util.FileExists(game.LocalBanner) {
		if game.LocalBanner != cachedBanner {
			_ = os.MkdirAll(gameArtDir, 0755)
			_ = util.CopyFile(game.LocalBanner, cachedBanner)
		}
		var mtime int64 = 1
		if fi, err := os.Stat(cachedBanner); err == nil {
			mtime = fi.ModTime().Unix()
		}
		game.LocalBannerURL = fmt.Sprintf("/art/%s/banner.jpg?t=%d", slug, mtime)
	} else if util.FileExists(cachedBanner) {
		var mtime int64 = 1
		if fi, err := os.Stat(cachedBanner); err == nil {
			mtime = fi.ModTime().Unix()
		}
		game.LocalBanner = cachedBanner
		game.LocalBannerURL = fmt.Sprintf("/art/%s/banner.jpg?t=%d", slug, mtime)
	}

	// 7. Sync / Export to Lutris directories if local artwork is available
	s.exportToLutris(slug, game.LocalCover, game.LocalBanner)

	return &game.GameArtwork, nil
}

func (s *ArtworkService) exportToLutris(slug, coverPath, bannerPath string) {
	if slug == "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	lutrisCoverDir := filepath.Join(home, ".cache", "lutris", "coverart")
	lutrisBannerDir := filepath.Join(home, ".cache", "lutris", "banners")

	if coverPath != "" && util.FileExists(coverPath) {
		_ = os.MkdirAll(lutrisCoverDir, 0755)
		target := filepath.Join(lutrisCoverDir, slug+".jpg")
		if !util.FileExists(target) {
			_ = util.CopyFile(coverPath, target)
		}
	}

	if bannerPath != "" && util.FileExists(bannerPath) {
		_ = os.MkdirAll(lutrisBannerDir, 0755)
		target := filepath.Join(lutrisBannerDir, slug+".jpg")
		if !util.FileExists(target) {
			_ = util.CopyFile(bannerPath, target)
		}
	}
}

// ApplyCustomArtwork saves a chosen image URL or local file to the game's cache and updates GameArtwork.
func (s *ArtworkService) ApplyCustomArtwork(ctx context.Context, game *domain.Game, assetType string, imageSource string) (*domain.GameArtwork, error) {
	if game == nil {
		return nil, fmt.Errorf("game cannot be nil")
	}
	if imageSource == "" {
		return nil, fmt.Errorf("image source cannot be empty")
	}

	slug := lutris.Slugify(game.Name)
	if slug == "" {
		slug = "game-" + game.AppID
	}

	gameArtDir := filepath.Join(s.cacheDir, slug)
	if err := os.MkdirAll(gameArtDir, 0755); err != nil {
		return nil, fmt.Errorf("create artwork dir: %w", err)
	}

	fileName := "cover.jpg"
	switch assetType {
	case "cover":
		fileName = "cover.jpg"
		game.CustomCoverURL = imageSource
	case "banner":
		fileName = "banner.jpg"
		game.CustomBannerURL = imageSource
	case "hero":
		fileName = "hero.jpg"
		game.CustomHeroURL = imageSource
	case "logo":
		fileName = "logo.png"
		game.CustomLogoURL = imageSource
	case "icon":
		fileName = "icon.jpg"
		game.CustomIconURL = imageSource
	default:
		fileName = "cover.jpg"
		game.CustomCoverURL = imageSource
	}

	targetPath := filepath.Join(gameArtDir, fileName)

	// Handle clearing artwork / forcing fallback
	if imageSource == "none" || imageSource == "clear" {
		_ = os.Remove(targetPath)
		game.LockedArtwork = true
		if assetType == "cover" {
			game.CustomCoverURL = "none"
			game.LocalCover = ""
			game.LocalCoverURL = ""
			game.CoverURL = ""
		} else if assetType == "banner" || assetType == "hero" {
			game.CustomBannerURL = "none"
			game.LocalBanner = ""
			game.LocalBannerURL = ""
			game.BannerURL = ""
		} else if assetType == "logo" {
			game.CustomLogoURL = "none"
			game.LogoURL = ""
		} else if assetType == "icon" {
			game.CustomIconURL = "none"
			game.LocalIcon = ""
			game.IconURL = ""
		}
		s.exportToLutris(slug, game.LocalCover, game.LocalBanner)
		return &game.GameArtwork, nil
	}

	// If local file exists, copy it
	if util.FileExists(imageSource) {
		if err := util.CopyFile(imageSource, targetPath); err != nil {
			return nil, fmt.Errorf("copy local image: %w", err)
		}
	} else if strings.HasPrefix(imageSource, "http://") || strings.HasPrefix(imageSource, "https://") {
		// Remote URL -> Download
		if err := s.DownloadFile(ctx, imageSource, targetPath); err != nil {
			return nil, fmt.Errorf("download image: %w", err)
		}
	} else {
		return nil, fmt.Errorf("invalid image source (must be a valid local file path or http/https URL)")
	}

	var mtime int64 = time.Now().Unix()
	if fi, err := os.Stat(targetPath); err == nil {
		mtime = fi.ModTime().Unix()
	}

	game.LockedArtwork = true
	if assetType == "cover" {
		game.LocalCover = targetPath
		game.LocalCoverURL = fmt.Sprintf("/art/%s/%s?t=%d", slug, fileName, mtime)
		game.CoverURL = imageSource
	} else if assetType == "banner" || assetType == "hero" {
		game.LocalBanner = targetPath
		game.LocalBannerURL = fmt.Sprintf("/art/%s/%s?t=%d", slug, fileName, mtime)
		game.BannerURL = imageSource
	}

	s.exportToLutris(slug, game.LocalCover, game.LocalBanner)
	return &game.GameArtwork, nil
}

