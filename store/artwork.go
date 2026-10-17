package store

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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
		CacheFilename: "icon.png",
		LocalCandidates: []string{
			"icon.png", "icon.ico", "icon.jpg",
			filepath.Join(".gamux", "art", "icon.png"),
			filepath.Join(".gamux", "art", "icon.ico"),
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

// InvalidateCache purges cached artwork files for a game when identity changes.
func (s *ArtworkService) InvalidateCache(game *domain.Game) {
	if s == nil || game == nil {
		return
	}
	slug := util.Slugify(game.Name)
	if slug == "" {
		slug = "game-" + game.AppID
	}
	gameArtDir := filepath.Join(s.cacheDir, slug)
	_ = os.RemoveAll(gameArtDir)
}

// PruneOrphans removes unreferenced image assets in the game's .gamux/art/ and system cache directories.
func (s *ArtworkService) PruneOrphans(game *domain.Game) (int, error) {
	if s == nil || game == nil {
		return 0, nil
	}

	referencedPaths := make(map[string]bool)
	for _, spec := range canonicalAssetSpecs {
		loc := game.GetAsset(spec.AssetType)
		if loc.LocalPath != "" {
			referencedPaths[util.CanonicalPath(loc.LocalPath)] = true
		}
	}

	slug := util.Slugify(game.Name)
	if slug == "" {
		slug = "game-" + game.AppID
	}

	dirsToInspect := []string{
		filepath.Join(s.cacheDir, slug),
	}
	if game.GameDir != "" {
		dirsToInspect = append(dirsToInspect, filepath.Join(game.GameDir, ".gamux", "art"))
	}

	removedCount := 0
	validCacheNames := make(map[string]bool)
	for _, spec := range canonicalAssetSpecs {
		validCacheNames[strings.ToLower(spec.CacheFilename)] = true
	}

	for _, dir := range dirsToInspect {
		if !util.DirExists(dir) {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			filePath := filepath.Join(dir, e.Name())
			canonPath := util.CanonicalPath(filePath)

			// If the file is not currently referenced
			if !referencedPaths[canonPath] {
				nameLower := strings.ToLower(e.Name())
				ext := strings.ToLower(filepath.Ext(nameLower))
				isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".ico" || ext == ".webp"
				if isImage {
					if err := os.Remove(filePath); err == nil {
						removedCount++
						slog.Debug("Pruned orphaned artwork file", "path", filePath)
					}
				}
			}
		}
	}

	return removedCount, nil
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

	// Step 2: Resolve according to metadata provider / known IDs
	provider := game.EffectiveMetadataProvider()
	rawgID := ""
	if provider == "rawg" {
		rawgID = game.EffectiveMetadataID()
	}
	if rawgID == "" {
		rawgID = game.RAWGID
	}
	gogID := game.GOGID
	if gogID == "" && provider == "gog" {
		gogID = game.EffectiveMetadataID()
	}
	steamID, hasSteamID := game.SteamAppID()
	if !hasSteamID && game.AppID != "" && game.AppID != "0" {
		steamID, hasSteamID = util.ParseAppID(game.AppID)
	}
	if !hasSteamID && provider == "steam" {
		steamID, hasSteamID = util.ParseAppID(game.EffectiveMetadataID())
	}

	if provider == "rawg" && rawgID != "" && !game.LockedArtwork && s.registry != nil {
		s.resolveFromRAWG(ctx, game, rawgID, gameArtDir)
	} else if provider == "gog" && gogID != "" && !game.LockedArtwork && s.registry != nil {
		s.resolveFromGOG(ctx, game, gogID, gameArtDir)
	} else if hasSteamID && steamID > 0 && !game.LockedArtwork {
		idStr := fmt.Sprintf("%d", steamID)
		for _, spec := range canonicalAssetSpecs {
			s.resolveSteamCDN(ctx, game, spec, idStr, gameArtDir)
		}
	}

	// Step 3: Multi-tier Title-Based Cascade for remaining missing assets
	if s.registry != nil && !game.LockedArtwork && s.hasMissingAssets(game) {
		// 3a. SteamGridDB query
		s.resolveFromSGDB(ctx, game, steamID, hasSteamID, gameArtDir)

		// 3b. Known GOG/RAWG IDs if not primary provider
		if s.hasMissingAssets(game) && gogID != "" {
			s.resolveFromGOG(ctx, game, gogID, gameArtDir)
		}
		if s.hasMissingAssets(game) && rawgID != "" {
			s.resolveFromRAWG(ctx, game, rawgID, gameArtDir)
		}

		// 3c. Steam Title Search (discovers Steam AppID for DRM-free/standalone games)
		if s.hasMissingAssets(game) && game.Name != "" && (!hasSteamID || steamID == 0) {
			s.resolveFromSteamSearch(ctx, game, gameArtDir)
		}

		// 3d. GOG Title Search (queries GOG catalog for vertical/horizontal artwork)
		if s.hasMissingAssets(game) && game.Name != "" && gogID == "" {
			s.resolveFromGOGSearch(ctx, game, gameArtDir)
		}

		// 3e. RAWG Title Search (queries open database for high-res wallpapers/screenshots)
		if s.hasMissingAssets(game) && game.Name != "" && rawgID == "" {
			s.resolveFromRAWGSearch(ctx, game, gameArtDir)
		}

		// 3f. SteamGrid Proxy (Bottles) Fallback for Cover
		if s.hasMissingAssets(game) && game.Name != "" {
			s.resolveFromSteamGridProxy(ctx, game, gameArtDir)
		}
	}

	// Step 4: Cross-Asset Fallback Substitution (ensure no game remains artless)
	s.applyCrossAssetFallbacks(ctx, game, gameArtDir)

	// Step 5: Finalize local URLs & cache synchronization
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

	// 4. For icons, extract embedded high-res icon from PE executable if available
	if spec.AssetType == domain.AssetIcon && loc.LocalPath == "" {
		targetExe := game.ExePath
		if targetExe == "" && len(game.LaunchCandidates) > 0 {
			targetExe = game.LaunchCandidates[0].Executable
		}
		if targetExe != "" {
			if !filepath.IsAbs(targetExe) && game.GameDir != "" {
				targetExe = filepath.Join(game.GameDir, targetExe)
			}
			if strings.HasSuffix(strings.ToLower(targetExe), ".exe") && util.FileExists(targetExe) {
				pngPath := filepath.Join(gameArtDir, "icon.png")
				if err := util.ExtractPEIcon(targetExe, pngPath); err == nil && util.FileExists(pngPath) {
					loc.LocalPath = pngPath
					loc.RemoteURL = ""
				}
			}
		}
	}

	game.SetAsset(spec.AssetType, loc)
}

func (s *ArtworkService) resolveSteamCDN(ctx context.Context, game *domain.Game, spec assetSpec, idStr string, gameArtDir string) {
	loc := game.GetAsset(spec.AssetType)
	if loc.CustomURL == "none" || spec.SteamCDNURL == nil {
		return
	}

	cachedPath := filepath.Join(gameArtDir, spec.CacheFilename)
	cdnURL := spec.SteamCDNURL(idStr)

	if loc.RemoteURL == "" {
		loc.RemoteURL = cdnURL
	}

	if loc.LocalPath == "" && spec.AssetType != domain.AssetIcon {
		if err := s.DownloadFile(ctx, cdnURL, cachedPath); err == nil {
			loc.LocalPath = cachedPath
			loc.RemoteURL = cdnURL
		} else if spec.SteamCDNFallbackURL != nil {
			fallbackURL := spec.SteamCDNFallbackURL(idStr)
			if err := s.DownloadFile(ctx, fallbackURL, cachedPath); err == nil {
				loc.LocalPath = cachedPath
				loc.RemoteURL = fallbackURL
			} else {
				loc.RemoteURL = ""
			}
		} else if spec.AssetType == domain.AssetCover {
			// Additional capsule/header fallback for older Steam titles
			capsuleURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_616x353.jpg", idStr)
			if err := s.DownloadFile(ctx, capsuleURL, cachedPath); err == nil {
				loc.LocalPath = cachedPath
				loc.RemoteURL = capsuleURL
			} else {
				headerURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", idStr)
				if err := s.DownloadFile(ctx, headerURL, cachedPath); err == nil {
					loc.LocalPath = cachedPath
					loc.RemoteURL = headerURL
				} else {
					loc.RemoteURL = ""
				}
			}
		} else {
			// Download failed (e.g. 404 for logo.png), reset RemoteURL so downstream providers or fallbacks can resolve
			loc.RemoteURL = ""
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

func (s *ArtworkService) resolveFromSteamSearch(ctx context.Context, game *domain.Game, gameArtDir string) {
	p, ok := s.registry.Get("steam")
	if !ok {
		return
	}

	results, err := p.Search(ctx, game.Name)
	if err != nil || len(results) == 0 {
		return
	}

	var matchedID string
	for _, res := range results {
		if util.IsTitleMatch(res.Name, game.Name) {
			matchedID = res.ID
			break
		}
	}

	if matchedID == "" {
		return
	}

	for _, spec := range canonicalAssetSpecs {
		s.resolveSteamCDN(ctx, game, spec, matchedID, gameArtDir)
	}
}

func (s *ArtworkService) resolveFromGOG(ctx context.Context, game *domain.Game, gogID string, gameArtDir string) {
	if gogID == "" {
		return
	}

	p, ok := s.registry.Get("gog")
	if !ok {
		return
	}

	art, err := p.GetArtwork(ctx, gogID)
	if err != nil || art == nil {
		return
	}

	s.applyProviderArtwork(ctx, game, art, gameArtDir)
}

func (s *ArtworkService) resolveFromGOGSearch(ctx context.Context, game *domain.Game, gameArtDir string) {
	p, ok := s.registry.Get("gog")
	if !ok {
		return
	}

	results, err := p.Search(ctx, game.Name)
	if err != nil || len(results) == 0 {
		return
	}

	var targetID string
	for _, res := range results {
		if util.IsTitleMatch(res.Name, game.Name) {
			targetID = res.ID
			break
		}
	}
	if targetID == "" && len(results) > 0 {
		targetID = results[0].ID
	}

	if targetID == "" {
		return
	}

	s.resolveFromGOG(ctx, game, targetID, gameArtDir)
}

func (s *ArtworkService) resolveFromRAWG(ctx context.Context, game *domain.Game, rawgID string, gameArtDir string) {
	if rawgID == "" {
		return
	}

	p, ok := s.registry.Get("rawg")
	if !ok {
		return
	}

	art, err := p.GetArtwork(ctx, rawgID)
	if err != nil || art == nil {
		return
	}

	s.applyProviderArtwork(ctx, game, art, gameArtDir)

	// If game still has missing assets (e.g. logo, high-res vertical cover), follow RAWG linked store IDs
	if s.hasMissingAssets(game) {
		if rawgProv, ok := p.(*RAWGProvider); ok {
			if rg, err := rawgProv.FetchRawGame(ctx, rawgID); err == nil && rg != nil {
				steamAppID, gogSlug := ExtractLinkedStoreIDs(rg)
				if steamAppID > 0 {
					idStr := fmt.Sprintf("%d", steamAppID)
					for _, spec := range canonicalAssetSpecs {
						if loc := game.GetAsset(spec.AssetType); loc.LocalPath == "" {
							s.resolveSteamCDN(ctx, game, spec, idStr, gameArtDir)
						}
					}
				}
				if gogSlug != "" && s.hasMissingAssets(game) {
					s.resolveFromGOG(ctx, game, gogSlug, gameArtDir)
				}
			}
		}
	}
}

func (s *ArtworkService) resolveFromRAWGSearch(ctx context.Context, game *domain.Game, gameArtDir string) {
	p, ok := s.registry.Get("rawg")
	if !ok {
		return
	}

	results, err := p.Search(ctx, game.Name)
	if err != nil || len(results) == 0 {
		return
	}

	var targetID string
	for _, res := range results {
		if util.IsTitleMatch(res.Name, game.Name) {
			targetID = res.ID
			break
		}
	}
	if targetID == "" && len(results) > 0 {
		targetID = results[0].ID
	}

	if targetID == "" {
		return
	}

	s.resolveFromRAWG(ctx, game, targetID, gameArtDir)
}

func (s *ArtworkService) resolveFromSteamGridProxy(ctx context.Context, game *domain.Game, gameArtDir string) {
	if game == nil || game.Name == "" {
		return
	}
	p, ok := s.registry.Get("steamgrid-proxy")
	if !ok {
		return
	}

	proxyProvider, ok := p.(*SteamGridProxyProvider)
	if !ok {
		return
	}

	coverURL, err := proxyProvider.FetchCoverURL(ctx, game.Name)
	if err != nil || coverURL == "" {
		return
	}

	coverSpec := canonicalAssetSpecs[0] // domain.AssetCover
	cachedPath := filepath.Join(gameArtDir, coverSpec.CacheFilename)
	loc := game.GetAsset(domain.AssetCover)
	if loc.LocalPath == "" {
		if err := s.DownloadFile(ctx, coverURL, cachedPath); err == nil {
			loc.LocalPath = cachedPath
			loc.RemoteURL = coverURL
			game.SetAsset(domain.AssetCover, loc)
		}
	}
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

		if loc.RemoteURL == "" || loc.LocalPath == "" {
			loc.RemoteURL = providerURL
		}
		if loc.LocalPath == "" && spec.AssetType != domain.AssetIcon {
			cachedPath := filepath.Join(gameArtDir, spec.CacheFilename)
			if err := s.DownloadFile(ctx, providerURL, cachedPath); err == nil {
				loc.LocalPath = cachedPath
				loc.RemoteURL = providerURL
			}
		}
		game.SetAsset(spec.AssetType, loc)
	}
}

func (s *ArtworkService) applyCrossAssetFallbacks(ctx context.Context, game *domain.Game, gameArtDir string) {
	coverLoc := game.GetAsset(domain.AssetCover)
	heroLoc := game.GetAsset(domain.AssetHero)
	bannerLoc := game.GetAsset(domain.AssetBanner)

	coverPath := filepath.Join(gameArtDir, "cover.jpg")
	heroPath := filepath.Join(gameArtDir, "hero.jpg")
	bannerPath := filepath.Join(gameArtDir, "banner.jpg")

	// 1. If Cover is missing, fallback to hero or banner
	if coverLoc.LocalPath == "" || !util.FileExists(coverLoc.LocalPath) {
		if heroLoc.LocalPath != "" && util.FileExists(heroLoc.LocalPath) {
			_ = util.CopyFile(heroLoc.LocalPath, coverPath)
			coverLoc.LocalPath = coverPath
			if coverLoc.RemoteURL == "" {
				coverLoc.RemoteURL = heroLoc.RemoteURL
			}
			game.SetAsset(domain.AssetCover, coverLoc)
		} else if bannerLoc.LocalPath != "" && util.FileExists(bannerLoc.LocalPath) {
			_ = util.CopyFile(bannerLoc.LocalPath, coverPath)
			coverLoc.LocalPath = coverPath
			if coverLoc.RemoteURL == "" {
				coverLoc.RemoteURL = bannerLoc.RemoteURL
			}
			game.SetAsset(domain.AssetCover, coverLoc)
		}
	}

	// 2. If Hero is missing, fallback to banner or cover
	if heroLoc.LocalPath == "" || !util.FileExists(heroLoc.LocalPath) {
		if bannerLoc.LocalPath != "" && util.FileExists(bannerLoc.LocalPath) {
			_ = util.CopyFile(bannerLoc.LocalPath, heroPath)
			heroLoc.LocalPath = heroPath
			if heroLoc.RemoteURL == "" {
				heroLoc.RemoteURL = bannerLoc.RemoteURL
			}
			game.SetAsset(domain.AssetHero, heroLoc)
		} else if coverLoc.LocalPath != "" && util.FileExists(coverLoc.LocalPath) {
			_ = util.CopyFile(coverLoc.LocalPath, heroPath)
			heroLoc.LocalPath = heroPath
			if heroLoc.RemoteURL == "" {
				heroLoc.RemoteURL = coverLoc.RemoteURL
			}
			game.SetAsset(domain.AssetHero, heroLoc)
		}
	}

	// 3. If Banner is missing, fallback to hero or cover
	if bannerLoc.LocalPath == "" || !util.FileExists(bannerLoc.LocalPath) {
		if heroLoc.LocalPath != "" && util.FileExists(heroLoc.LocalPath) {
			_ = util.CopyFile(heroLoc.LocalPath, bannerPath)
			bannerLoc.LocalPath = bannerPath
			if bannerLoc.RemoteURL == "" {
				bannerLoc.RemoteURL = heroLoc.RemoteURL
			}
			game.SetAsset(domain.AssetBanner, bannerLoc)
		} else if coverLoc.LocalPath != "" && util.FileExists(coverLoc.LocalPath) {
			_ = util.CopyFile(coverLoc.LocalPath, bannerPath)
			bannerLoc.LocalPath = bannerPath
			if bannerLoc.RemoteURL == "" {
				bannerLoc.RemoteURL = coverLoc.RemoteURL
			}
			game.SetAsset(domain.AssetBanner, bannerLoc)
		}
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

	mtime := time.Now().Unix()
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
	_, _ = s.PruneOrphans(game)

	return &game.GameArtwork, nil
}
