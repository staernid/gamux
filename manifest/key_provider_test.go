package manifest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type mockKeyProvider struct {
	name   string
	result *ParsedLua
	err    error
}

func (m *mockKeyProvider) Name() string {
	return m.name
}

func (m *mockKeyProvider) Resolve(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestProviderChain_PriorityAndMerging(t *testing.T) {
	p1 := &mockKeyProvider{
		name: "Provider 1 (Fails)",
		err:  fmt.Errorf("network timeout"),
	}
	p2 := &mockKeyProvider{
		name: "Provider 2 (Keys only)",
		result: &ParsedLua{
			AppID: 100,
			Depots: []DepotKeyPair{
				{DepotID: 101, DecryptionKey: "key101"},
			},
		},
	}
	p3 := &mockKeyProvider{
		name: "Provider 3 (Manifests)",
		result: &ParsedLua{
			AppID: 100,
			Depots: []DepotKeyPair{
				{DepotID: 102, DecryptionKey: "key102"},
			},
			ManifestFiles: map[string][]byte{
				"101_manifest.manifest": []byte("manifest101"),
			},
		},
	}

	chain := &ProviderChain{
		Providers: []KeyProvider{p1, p2, p3},
	}

	res, err := chain.Resolve(context.Background(), 100)
	if err != nil {
		t.Fatalf("chain.Resolve failed: %v", err)
	}

	if len(res.Depots) != 2 {
		t.Errorf("expected 2 merged depots, got %d", len(res.Depots))
	}
	if len(res.ManifestFiles) != 1 {
		t.Errorf("expected 1 merged manifest file, got %d", len(res.ManifestFiles))
	}
	if len(res.AuditTrace) != 3 {
		t.Errorf("expected 3 audit trace items, got %d", len(res.AuditTrace))
	}
}

func TestLocalLuaProvider(t *testing.T) {
	tmpDir := t.TempDir()
	luaFile := filepath.Join(tmpDir, "test.lua")
	_ = os.WriteFile(luaFile, []byte("addappid(12345, 1, \"deadbeef\")"), 0644)

	provider := &LocalLuaProvider{LuaPath: luaFile}
	res, err := provider.Resolve(context.Background(), 12345)
	if err != nil {
		t.Fatalf("LocalLuaProvider.Resolve failed: %v", err)
	}
	if res.AppID != 12345 || len(res.Depots) != 1 || res.Depots[0].DecryptionKey != "deadbeef" {
		t.Errorf("unexpected parsed lua result: %+v", res)
	}

	// Empty path error
	emptyProvider := &LocalLuaProvider{}
	if _, err := emptyProvider.Resolve(context.Background(), 12345); err == nil {
		t.Error("expected error for empty lua path, got nil")
	}
}
