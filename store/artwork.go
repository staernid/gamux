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
	"github.com/staernid/gamux/util"
)

type assetSpec struct {
	AssetType           string
	CacheFilename       string
	LocalCandidates     []string
	SteamCDNURL         func(appID string) string
	SteamCDNFallbackURL func(appID string) string
	ArtworkGetter       func(art *Artwork) string
}

var canonicalAssetSpecs = []assetSpec{
	{
		AssetType:     domain.AssetCover,
		CacheFilename: "cover.jpg",
		LocalCandidates: []string{
			"cover.jpg", "cover.png", "poster.jpg", "poster.png",
			filepath.Join(".gamux", "art", "cover.jpg"),
			filepath.Join(".gamux", "art", "cover.png"),
		},
		SteamCDNURL: func(appID string) string {
			return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", appID)
		},
		SteamCDNFallbackURL: func(appID string) string {
			return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)
		},
		ArtworkGetter: func(art *Artwork) string {
			return art.CoverURL
		},
	},
	{
		AssetType:     domain.AssetBanner,
		CacheFilename: "banner.jpg",
		LocalCandidates: []string{
			"banner.jpg", "banner.png", "header.jpg",
			filepath.Join(".gamux", "art", "banner.jpg"),
		},
		SteamCDNURL: func(appID string) string {
			return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", appID)
		},
		ArtworkGetter: func(art *Artwork) string {
			return art.BannerURL
		},
	},
	{
		AssetType:     domain.AssetHero,
		CacheFilename: "hero.jpg",
		LocalCandidates: []string{
			"hero.jpg", "hero.png", "library_hero.jpg",
			filepath.Join(".gamux", "art", "hero.jpg"),
		},
		SteamCDNURL: func(appID string) string {
			return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg", appID)
		},
		ArtworkGetter: func(art *Artwork) string {
			return art.HeroURL
		},
	},
	{
		AssetType:     domain.AssetLogo,
		CacheFilename: "logo.png",
		LocalCandidates: []string{
			"logo.png", "logo.jpg",
			filepath.Join(".gamux", "art", "logo.png"),
		},
		SteamCDNURL: func(appID string) string {
			return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/logo.png", appID)
		},
		ArtworkGetter: func(art *Artwork) string {
			return art.LogoURL
		},
	},
	{
		AssetType:     domain.AssetIcon,
		CacheFilename: "icon.jpg",
		LocalCandidates: []string{
			"icon.jpg", "icon.png",
			filepath.Join(".gamux", "art", "icon.jpg"),
		},
		SteamCDNURL: func(appID string) string {
			return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_sm_120.jpg", appID)
		},
		ArtworkGetter: func(art *Artwork) string {
			return art.IconURL
		},
	},
}

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

	slug := util.Slugify(game.Name)
	if slug == "" {
		slug = "game-" + game.AppID
	}

	gameArtDir := filepath.Join(s.cacheDir, slug)

	// Step 1: Fast local checks (Custom overrides, game folder candidates, disk cache)
	for _, spec := range canonicalAssetSpecs {
		s.resolveLocalAsset(ctx, game, spec, gameArtDir)
	}

	// Step 2: Steam CDN direct resolution for Steam titles
	steamID, hasSteamID := game.SteamAppID()
	if !hasSteamID && game.AppID != "" && game.AppID != "0" {
		steamID, hasSteamID = util.ParseAppID(game.AppID)
	}

	if hasSteamID && steamID > 0 && !game.LockedArtwork {
		idStr := fmt.Sprintf("%d", steamID)
		for _, spec := range canonicalAssetSpecs {
			s.resolveSteamCDN(ctx, game, spec, idStr, gameArtDir)
		}
	}

	// Step 3: Lazy Provider queries (SteamGridDB & GOG) for remaining missing assets
	if s.registry != nil && !game.LockedArtwork {
		if s.hasMissingAssets(game) {
			s.resolveFromSGDB(ctx, game, steamID, hasSteamID, gameArtDir)
		}
		if s.hasMissingAssets(game) && game.GOGID != "" {
			s.resolveFromGOG(ctx, game, gameArtDir)
		}
	}

	// Step 4: Finalize local URLs & cache synchronization
	for _, spec := range canonicalAssetSpecs {
		s.finalizeAsset(game, spec, slug, gameArtDir)
	}

	return &game.GameArtwork, nil
}

func (s *ArtworkService) resolveLocalAsset(ctx context.Context, game *domain.Game, spec assetSpec, gameArtDir string) {
	loc := game.GetAsset(spec.AssetType)
	cachedPath := filepath.Join(gameArtDir, spec.CacheFilename)

	// 1. Check custom override
	if loc.CustomURL == "none" {
		loc.LocalPath = ""
		loc.LocalURL = ""
		loc.RemoteURL = ""
		game.SetAsset(spec.AssetType, loc)
		return
	}

	if loc.CustomURL != "" {
		if strings.HasPrefix(loc.CustomURL, "http://") || strings.HasPrefix(loc.CustomURL, "https://") {
			if !util.FileExists(cachedPath) || loc.LocalPath == "" {
				_ = s.DownloadFile(ctx, loc.CustomURL, cachedPath)
			}
			loc.LocalPath = cachedPath
			loc.RemoteURL = loc.CustomURL
		} else if util.FileExists(loc.CustomURL) {
			loc.LocalPath = loc.CustomURL
		}
		game.SetAsset(spec.AssetType, loc)
		return
	}

	// 2. Check local game folder candidates
	if game.GameDir != "" && loc.LocalPath == "" {
		for _, candidate := range spec.LocalCandidates {
			p := filepath.Join(game.GameDir, candidate)
			if util.FileExists(p) {
				loc.LocalPath = p
				break
			}
		}
	}

	// 3. Check disk cache
	if loc.LocalPath == "" && util.FileExists(cachedPath) {
		loc.LocalPath = cachedPath
	}

	game.SetAsset(spec.AssetType, loc)
}

func (s *ArtworkService) resolveSteamCDN(ctx context.Context, game *domain.Game, spec assetSpec, idStr string, gameArtDir string) {
	loc := game.GetAsset(spec.AssetType)
	if loc.CustomURL == "none" || spec.SteamCDNURL == nil {
		return
	}

	cachedPath := filepath.Join(gameArtDir, spec.CacheFilename)

	if loc.RemoteURL == "" {
		loc.RemoteURL = spec.SteamCDNURL(idStr)
	}

	if loc.LocalPath == "" && spec.AssetType != domain.AssetIcon {
		if err := s.DownloadFile(ctx, loc.RemoteURL, cachedPath); err == nil {
			loc.LocalPath = cachedPath
		} else if spec.SteamCDNFallbackURL != nil {
			fallbackURL := spec.SteamCDNFallbackURL(idStr)
			if err := s.DownloadFile(ctx, fallbackURL, cachedPath); err == nil {
				loc.LocalPath = cachedPath
			}
		}
	}

	game.SetAsset(spec.AssetType, loc)
}

func (s *ArtworkService) hasMissingAssets(game *domain.Game) bool {
	for _, spec := range canonicalAssetSpecs {
		if spec.AssetType == domain.AssetIcon {
			continue
		}
		loc := game.GetAsset(spec.AssetType)
		if loc.CustomURL != "none" && loc.LocalPath == "" {
			return true
		}
	}
	return false
}

func (s *ArtworkService) resolveFromSGDB(ctx context.Context, game *domain.Game, steamID uint32, hasSteamID bool, gameArtDir string) {
	p, ok := s.registry.Get("steamgriddb")
	if !ok {
		return
	}

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
			for _, res := range results {
				if isTitleMatch(res.Name, game.Name) {
					sgdbID = res.ID
					break
				}
			}
		}
	}

	if sgdbID == "" {
		return
	}

	game.SGDBID = sgdbID
	art, err := p.GetArtwork(ctx, sgdbID)
	if err != nil || art == nil {
		return
	}

	s.applyProviderArtwork(ctx, game, art, gameArtDir)
}

func (s *ArtworkService) resolveFromGOG(ctx context.Context, game *domain.Game, gameArtDir string) {
	p, ok := s.registry.Get("gog")
	if !ok {
		return
	}

	art, err := p.GetArtwork(ctx, game.GOGID)
	if err != nil || art == nil {
		return
	}

	s.applyProviderArtwork(ctx, game, art, gameArtDir)
}

func (s *ArtworkService) applyProviderArtwork(ctx context.Context, game *domain.Game, art *Artwork, gameArtDir string) {
	for _, spec := range canonicalAssetSpecs {
		loc := game.GetAsset(spec.AssetType)
		if loc.CustomURL == "none" {
			continue
		}
		providerURL := spec.ArtworkGetter(art)
		if providerURL == "" {
			continue
		}

		if loc.RemoteURL == "" {
			loc.RemoteURL = providerURL
		}
		if loc.LocalPath == "" && spec.AssetType != domain.AssetIcon {
			cachedPath := filepath.Join(gameArtDir, spec.CacheFilename)
			if err := s.DownloadFile(ctx, providerURL, cachedPath); err == nil {
				loc.LocalPath = cachedPath
			}
		}
		game.SetAsset(spec.AssetType, loc)
	}
}

func (s *ArtworkService) finalizeAsset(game *domain.Game, spec assetSpec, slug, gameArtDir string) {
	loc := game.GetAsset(spec.AssetType)
	cachedPath := filepath.Join(gameArtDir, spec.CacheFilename)

	if loc.LocalPath != "" && util.FileExists(loc.LocalPath) {
		if loc.LocalPath != cachedPath {
			_ = os.MkdirAll(gameArtDir, 0755)
			_ = util.CopyFile(loc.LocalPath, cachedPath)
		}
		var mtime int64 = 1
		if fi, err := os.Stat(cachedPath); err == nil {
			mtime = fi.ModTime().Unix()
		}
		loc.LocalURL = fmt.Sprintf("/art/%s/%s?t=%d", slug, spec.CacheFilename, mtime)
	} else if util.FileExists(cachedPath) {
		var mtime int64 = 1
		if fi, err := os.Stat(cachedPath); err == nil {
			mtime = fi.ModTime().Unix()
		}
		loc.LocalPath = cachedPath
		loc.LocalURL = fmt.Sprintf("/art/%s/%s?t=%d", slug, spec.CacheFilename, mtime)
	}

	game.SetAsset(spec.AssetType, loc)
}

// ApplyCustomArtwork saves a chosen image URL or local file to the game's cache and updates GameArtwork.
func (s *ArtworkService) ApplyCustomArtwork(ctx context.Context, game *domain.Game, assetType string, imageSource string) (*domain.GameArtwork, error) {
	if game == nil {
		return nil, fmt.Errorf("game cannot be nil")
	}
	if imageSource == "" {
		return nil, fmt.Errorf("image source cannot be empty")
	}

	slug := util.Slugify(game.Name)
	if slug == "" {
		slug = "game-" + game.AppID
	}

	gameArtDir := filepath.Join(s.cacheDir, slug)
	if err := os.MkdirAll(gameArtDir, 0755); err != nil {
		return nil, fmt.Errorf("create artwork dir: %w", err)
	}

	fileName := "cover.jpg"
	for _, spec := range canonicalAssetSpecs {
		if strings.EqualFold(spec.AssetType, assetType) {
			fileName = spec.CacheFilename
			break
		}
	}

	targetPath := filepath.Join(gameArtDir, fileName)

	// Handle clearing artwork / forcing fallback
	if imageSource == "none" || imageSource == "clear" {
		game.LockedArtwork = true
		loc := domain.AssetLocation{
			CustomURL: "none",
			LocalPath: "",
			LocalURL:  "",
			RemoteURL: "",
		}
		game.SetAsset(assetType, loc)
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
	loc := domain.AssetLocation{
		CustomURL: imageSource,
		LocalPath: targetPath,
		LocalURL:  fmt.Sprintf("/art/%s/%s?t=%d", slug, fileName, mtime),
		RemoteURL: imageSource,
	}
	game.SetAsset(assetType, loc)

	return &game.GameArtwork, nil
}


