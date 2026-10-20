package domain

import "strings"

// Canonical asset types
const (
	AssetCover  = "cover"
	AssetBanner = "banner"
	AssetHero   = "hero"
	AssetLogo   = "logo"
	AssetIcon   = "icon"
)

// ArtworkOption represents a selectable artwork asset candidate discovered from a metadata provider.
type ArtworkOption struct {
	ID       string `json:"id"`
	Provider string `json:"provider"` // "steam", "steamgriddb", "gog", "custom"
	Type     string `json:"type"`     // "cover", "banner", "hero", "logo", "icon"
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Style    string `json:"style,omitempty"` // "official", "alternate", "blurred", "material", "no_logo"
	Score    int    `json:"score,omitempty"`
	Author   string `json:"author,omitempty"`
}

// ArtworkQuery encapsulates structured criteria for resolving artwork candidates.
type ArtworkQuery struct {
	GameDir    string `json:"game_dir,omitempty"`
	Title      string `json:"title,omitempty"`
	SteamAppID uint32 `json:"steam_app_id,omitempty"`
	GOGID      string `json:"gog_id,omitempty"`
	SGDBID     string `json:"sgdb_id,omitempty"`
	AssetType  string `json:"asset_type"` // "cover", "banner", "hero", "logo", "icon"
}

// AssetLocation encapsulates paths and URLs for a specific graphical asset.
type AssetLocation struct {
	RemoteURL string `json:"remote_url,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
	LocalURL  string `json:"local_url,omitempty"`
	CustomURL string `json:"custom_url,omitempty"`
}

// GameArtwork stores remote URLs and local file paths for graphical assets.
type GameArtwork struct {
	CoverURL        string `json:"cover_url,omitempty"`
	BannerURL       string `json:"banner_url,omitempty"`
	HeroURL         string `json:"hero_url,omitempty"`
	LogoURL         string `json:"logo_url,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`
	LocalCover      string `json:"local_cover,omitempty"`
	LocalBanner     string `json:"local_banner,omitempty"`
	LocalHero       string `json:"local_hero,omitempty"`
	LocalLogo       string `json:"local_logo,omitempty"`
	LocalIcon       string `json:"local_icon,omitempty"`
	LocalCoverURL   string `json:"local_cover_url,omitempty"`
	LocalBannerURL  string `json:"local_banner_url,omitempty"`
	LocalHeroURL    string `json:"local_hero_url,omitempty"`
	LocalLogoURL    string `json:"local_logo_url,omitempty"`
	CustomCoverURL  string `json:"custom_cover_url,omitempty"`
	CustomBannerURL string `json:"custom_banner_url,omitempty"`
	CustomHeroURL   string `json:"custom_hero_url,omitempty"`
	CustomLogoURL   string `json:"custom_logo_url,omitempty"`
	CustomIconURL   string `json:"custom_icon_url,omitempty"`
	LockedArtwork   bool   `json:"locked_artwork,omitempty"`
}

// GetAsset returns the location information for a specific asset type.
func (g *Game) GetAsset(assetType string) AssetLocation {
	switch strings.ToLower(assetType) {
	case AssetCover:
		return AssetLocation{
			RemoteURL: g.CoverURL,
			LocalPath: g.LocalCover,
			LocalURL:  g.LocalCoverURL,
			CustomURL: g.CustomCoverURL,
		}
	case AssetBanner:
		return AssetLocation{
			RemoteURL: g.BannerURL,
			LocalPath: g.LocalBanner,
			LocalURL:  g.LocalBannerURL,
			CustomURL: g.CustomBannerURL,
		}
	case AssetHero:
		return AssetLocation{
			RemoteURL: g.HeroURL,
			LocalPath: g.LocalHero,
			LocalURL:  g.LocalHeroURL,
			CustomURL: g.CustomHeroURL,
		}
	case AssetLogo:
		return AssetLocation{
			RemoteURL: g.LogoURL,
			LocalPath: g.LocalLogo,
			LocalURL:  g.LocalLogoURL,
			CustomURL: g.CustomLogoURL,
		}
	case AssetIcon:
		return AssetLocation{
			RemoteURL: g.IconURL,
			LocalPath: g.LocalIcon,
			CustomURL: g.CustomIconURL,
		}
	default:
		return AssetLocation{}
	}
}

// SetAsset updates the fields corresponding to the specified asset type.
func (g *Game) SetAsset(assetType string, loc AssetLocation) {
	switch strings.ToLower(assetType) {
	case AssetCover:
		if loc.RemoteURL != "" {
			g.CoverURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalCover = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalCoverURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomCoverURL = loc.CustomURL
		}
	case AssetBanner:
		if loc.RemoteURL != "" {
			g.BannerURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalBanner = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalBannerURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomBannerURL = loc.CustomURL
		}
	case AssetHero:
		if loc.RemoteURL != "" {
			g.HeroURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalHero = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalHeroURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomHeroURL = loc.CustomURL
		}
	case AssetLogo:
		if loc.RemoteURL != "" {
			g.LogoURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalLogo = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalLogoURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomLogoURL = loc.CustomURL
		}
	case AssetIcon:
		if loc.RemoteURL != "" {
			g.IconURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalIcon = loc.LocalPath
		}
		if loc.CustomURL != "" {
			g.CustomIconURL = loc.CustomURL
		}
	}
}
