package manifest

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// KeyProvider defines the interface for resolving depot decryption keys and binary manifests.
type KeyProvider interface {
	Name() string
	Resolve(ctx context.Context, appID uint32) (*ParsedLua, error)
}

// LocalLuaProvider resolves keys from a user-specified .lua file.
type LocalLuaProvider struct {
	LuaPath string
}

func (p *LocalLuaProvider) Name() string {
	return "Local Lua File"
}

func (p *LocalLuaProvider) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if p.LuaPath == "" {
		return nil, fmt.Errorf("no local lua path specified")
	}
	content, err := os.ReadFile(p.LuaPath)
	if err != nil {
		return nil, fmt.Errorf("read local lua file %s: %w", p.LuaPath, err)
	}
	parsed, err := ParseLua(string(content))
	if err != nil {
		return nil, err
	}
	parsed.AuditTrace = append(parsed.AuditTrace, fmt.Sprintf("Local Lua File override ('%s'): %d depot keys, %d manifest files", p.LuaPath, len(parsed.Depots), len(parsed.ManifestFiles)))
	return parsed, nil
}

// RevoBDProvider resolves keys and binary manifests from the RevoBD public API.
type RevoBDProvider struct{}

func (p *RevoBDProvider) Name() string {
	return "RevoBD API"
}

func (p *RevoBDProvider) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	return FetchRevoBDKeys(ctx, appID)
}

// SteamDepotDBProvider resolves fallback keys from the KoriaPolis Steam-Depot raw GitHub database.
type SteamDepotDBProvider struct{}

func (p *SteamDepotDBProvider) Name() string {
	return "Steam-Depot DB"
}

func (p *SteamDepotDBProvider) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	return FetchSteamDepotDBKeys(ctx, appID)
}

// ManifesthubProvider resolves fallback keys from the GitLab steamautocracks/manifesthub repository.
type ManifesthubProvider struct{}

func (p *ManifesthubProvider) Name() string {
	return "GitLab Manifesthub"
}

func (p *ManifesthubProvider) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	return FetchManifesthubKeys(ctx, appID)
}

// HubcapProvider resolves depot keys and binary manifests from the Hubcap Manifest API (with local disk cache).
type HubcapProvider struct {
	APIKey string
}

func (p *HubcapProvider) Name() string {
	return "Hubcap API"
}

func (p *HubcapProvider) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	return FetchHubcapKeys(ctx, appID, p.APIKey)
}

// ProviderChain orchestrates a multi-tier fallback chain of KeyProviders.
type ProviderChain struct {
	Providers []KeyProvider
}

// NewDefaultProviderChain creates the standard resolution chain.
func NewDefaultProviderChain(luaPath string, hubcapAPIKey string) *ProviderChain {
	var providers []KeyProvider
	if luaPath != "" {
		providers = append(providers, &LocalLuaProvider{LuaPath: luaPath})
	}
	providers = append(providers,
		&RevoBDProvider{},
		&SteamDepotDBProvider{},
		&ManifesthubProvider{},
	)
	if hubcapAPIKey != "" {
		providers = append(providers, &HubcapProvider{APIKey: hubcapAPIKey})
	}
	return &ProviderChain{Providers: providers}
}

// Resolve executes the provider chain in order, merging manifests and depot keys.
func (c *ProviderChain) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	var trace []string
	var bestResult *ParsedLua
	var errs []string

	for _, provider := range c.Providers {
		// If we already have full keys and binary manifests, skip remaining network providers
		if bestResult != nil && len(bestResult.ManifestFiles) > 0 {
			trace = append(trace, fmt.Sprintf("%s: Skipped (already resolved keys and manifests)", provider.Name()))
			continue
		}

		parsed, err := provider.Resolve(ctx, appID)
		if err == nil && parsed != nil && len(parsed.Depots) > 0 {
			trace = append(trace, fmt.Sprintf("%s: Found %d depot keys, %d .manifest files", provider.Name(), len(parsed.Depots), len(parsed.ManifestFiles)))
			if bestResult == nil {
				bestResult = parsed
			} else {
				// Merge newly obtained manifest files
				if len(parsed.ManifestFiles) > 0 {
					if bestResult.ManifestFiles == nil {
						bestResult.ManifestFiles = make(map[string][]byte)
					}
					for k, v := range parsed.ManifestFiles {
						bestResult.ManifestFiles[k] = v
					}
				}
				// Merge newly obtained depot keys
				existingDepots := make(map[uint32]bool)
				for _, d := range bestResult.Depots {
					existingDepots[d.DepotID] = true
				}
				for _, d := range parsed.Depots {
					if !existingDepots[d.DepotID] && d.DecryptionKey != "" {
						bestResult.Depots = append(bestResult.Depots, d)
					}
				}
			}
		} else if err != nil {
			trace = append(trace, fmt.Sprintf("%s: %v", provider.Name(), err))
			errs = append(errs, fmt.Sprintf("%s: %v", provider.Name(), err))
		}
	}

	if bestResult != nil {
		bestResult.AuditTrace = trace
		return bestResult, nil
	}

	return nil, fmt.Errorf("failed to resolve depot keys for AppID %d via provider chain: %s", appID, strings.Join(errs, " | "))
}
