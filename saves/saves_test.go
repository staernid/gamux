package saves

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestIndexingAndLookup(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManifestManager(tmpDir, "", nil)

	mockRaw := map[string]ManifestGame{
		"Valheim": {
			Steam: struct {
				ID uint32 `json:"id,omitempty"`
			}{ID: 892970},
			Files: map[string]ManifestFileRule{
				"<home>/AppData/LocalLow/IronGate/Valheim": {
					Tags: []string{"save"},
					When: []ManifestConstraint{{OS: "windows"}},
				},
				"<root>/userdata/<storeUserId>/892970": {
					Tags: []string{"save", "config"},
					When: []ManifestConstraint{{Store: "steam"}},
				},
			},
		},
		"Cyberpunk 2077": {
			Steam: struct {
				ID uint32 `json:"id,omitempty"`
			}{ID: 1091500},
			Files: map[string]ManifestFileRule{
				"<winSavedGames>/CD Projekt Red/Cyberpunk 2077": {
					Tags: []string{"save"},
					When: []ManifestConstraint{{OS: "windows"}},
				},
			},
		},
		"Valheim Plus": {
			Alias: "Valheim",
		},
	}

	mgr.indexMap(mockRaw)

	// AppID lookup
	g1, found := mgr.FindGame(892970, "", 0)
	if !found || g1 == nil {
		t.Fatalf("expected to find Valheim by AppID 892970")
	}
	if len(g1.Files) != 2 {
		t.Errorf("expected 2 files rules, got %d", len(g1.Files))
	}

	// Title lookup
	g2, found := mgr.FindGame(0, "Cyberpunk 2077", 0)
	if !found || g2 == nil {
		t.Fatalf("expected to find Cyberpunk 2077 by title")
	}
	if g2.Steam.ID != 1091500 {
		t.Errorf("expected AppID 1091500, got %d", g2.Steam.ID)
	}

	// Alias lookup
	g3, found := mgr.FindGame(0, "Valheim Plus", 0)
	if !found || g3 == nil {
		t.Fatalf("expected to find Valheim Plus resolving to Valheim")
	}
	if g3.Steam.ID != 892970 {
		t.Errorf("expected alias to resolve to AppID 892970, got %d", g3.Steam.ID)
	}
}

func TestPlaceholderExpansionAndResolution(t *testing.T) {
	tmpDir := t.TempDir()
	fakeGameDir := filepath.Join(tmpDir, "games", "Valheim")
	fakePrefix := filepath.Join(tmpDir, "prefix")
	fakeUserDir := filepath.Join(fakePrefix, "drive_c", "users", "steamuser")
	fakeSaveDir := filepath.Join(fakeUserDir, "AppData", "LocalLow", "IronGate", "Valheim")

	_ = os.MkdirAll(fakeGameDir, 0755)
	_ = os.MkdirAll(fakeSaveDir, 0755)
	_ = os.WriteFile(filepath.Join(fakeSaveDir, "player.fch"), []byte("save data content"), 0644)

	mgr := NewManifestManager(tmpDir, "", nil)
	mgr.indexMap(map[string]ManifestGame{
		"Valheim": {
			Steam: struct {
				ID uint32 `json:"id,omitempty"`
			}{ID: 892970},
			Files: map[string]ManifestFileRule{
				"<home>/AppData/LocalLow/IronGate/Valheim": {
					Tags: []string{"save"},
					When: []ManifestConstraint{{OS: "windows"}},
				},
			},
		},
	})

	resolver := NewResolver(mgr)
	res, err := resolver.Resolve(context.Background(), ResolveOptions{
		AppID:      "892970",
		GameName:   "Valheim",
		GameDir:    fakeGameDir,
		Platform:   "win64",
		WinePrefix: fakePrefix,
	})

	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if len(res.Locations) == 0 {
		t.Fatalf("expected at least 1 save location resolved")
	}

	var foundSave *SaveLocation
	for i := range res.Locations {
		if res.Locations[i].Path == fakeSaveDir {
			foundSave = &res.Locations[i]
			break
		}
	}

	if foundSave == nil {
		t.Fatalf("expected to find save location %s in result, got: %+v", fakeSaveDir, res.Locations)
	}

	if !foundSave.Exists {
		t.Errorf("expected save location to exist")
	}
	if foundSave.FileCount != 1 {
		t.Errorf("expected file count 1, got %d", foundSave.FileCount)
	}
	if foundSave.SizeBytes <= 0 {
		t.Errorf("expected positive size, got %d", foundSave.SizeBytes)
	}
}

func TestBackupCreationAndRestoration(t *testing.T) {
	tmpDir := t.TempDir()
	fakeGameDir := filepath.Join(tmpDir, "games", "Valheim")
	fakePrefix := filepath.Join(tmpDir, "prefix")
	fakeUserDir := filepath.Join(fakePrefix, "drive_c", "users", "steamuser")
	fakeSaveDir := filepath.Join(fakeUserDir, "AppData", "LocalLow", "IronGate", "Valheim")
	backupBaseDir := filepath.Join(tmpDir, "backups")

	_ = os.MkdirAll(fakeGameDir, 0755)
	_ = os.MkdirAll(fakeSaveDir, 0755)
	originalContent := "sample save data 12345"
	saveFilePath := filepath.Join(fakeSaveDir, "player.fch")
	_ = os.WriteFile(saveFilePath, []byte(originalContent), 0644)

	mgr := NewManifestManager(tmpDir, "", nil)
	mgr.indexMap(map[string]ManifestGame{
		"Valheim": {
			Steam: struct {
				ID uint32 `json:"id,omitempty"`
			}{ID: 999999},
			Files: map[string]ManifestFileRule{
				"<home>/AppData/LocalLow/IronGate/Valheim": {
					Tags: []string{"save"},
					When: []ManifestConstraint{{OS: "windows"}},
				},
			},
		},
	})

	resolver := NewResolver(mgr)
	bm := NewBackupManager(backupBaseDir, 3, resolver)

	resolveOpts := ResolveOptions{
		AppID:      "999999",
		GameName:   "Valheim",
		GameDir:    fakeGameDir,
		Platform:   "win64",
		WinePrefix: fakePrefix,
	}

	// 1. Create Backup
	backupInfo, err := bm.CreateBackup(context.Background(), resolveOpts, "Test backup note", "manual")
	if err != nil {
		t.Fatalf("create backup failed: %v", err)
	}

	if backupInfo == nil || backupInfo.ZipPath == "" {
		t.Fatalf("invalid backup info returned")
	}

	if backupInfo.FileCount != 1 {
		t.Errorf("expected 1 file in backup, got %d", backupInfo.FileCount)
	}

	// 2. List Backups
	list, err := bm.ListBackups("Valheim", "999999")
	if err != nil {
		t.Fatalf("list backups failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 backup in list, got %d", len(list))
	}

	// 3. Corrupt or delete original save file
	_ = os.Remove(saveFilePath)

	// 4. Restore Backup
	if err := bm.RestoreBackup(context.Background(), backupInfo.ZipPath, resolveOpts); err != nil {
		t.Fatalf("restore backup failed: %v", err)
	}

	restoredData, err := os.ReadFile(saveFilePath)
	if err != nil {
		t.Fatalf("failed reading restored file: %v", err)
	}

	if string(restoredData) != originalContent {
		t.Errorf("expected restored content %q, got %q", originalContent, string(restoredData))
	}

	// 5. Delete Backup
	if err := bm.DeleteBackup(backupInfo.ZipPath); err != nil {
		t.Fatalf("delete backup failed: %v", err)
	}

	listAfterDelete, _ := bm.ListBackups("Valheim", "999999")
	if len(listAfterDelete) != 0 {
		t.Errorf("expected 0 backups after delete, got %d", len(listAfterDelete))
	}
}

func TestResolveEffectiveWinePrefix_DefaultSharedUMU(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManifestManager(tmpDir, "", nil)
	resolver := NewResolver(mgr)

	// 1. Explicit custom prefix
	customPrefix := filepath.Join(tmpDir, "custom-prefix")
	p1 := resolver.resolveEffectiveWinePrefix(customPrefix, "1091500", "Cyberpunk 2077")
	if p1 != customPrefix {
		t.Errorf("expected custom prefix %s, got %s", customPrefix, p1)
	}

	// 2. Unset prefix -> defaults to isolated per-game prefix or shared umu-default under Games/umu
	p2 := resolver.resolveEffectiveWinePrefix("", "1091500", "Cyberpunk 2077")
	if !strings.HasSuffix(p2, filepath.Join("Games", "umu", "1091500")) && !strings.HasSuffix(p2, filepath.Join("Games", "umu", "umu-default")) {
		t.Errorf("expected UMU prefix under Games/umu, got %s", p2)
	}
}

func TestService_EndToEndLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	fakeGameDir := filepath.Join(tmpDir, "games", "TestRPG")
	fakePrefix := filepath.Join(tmpDir, "prefix")
	fakeUserDir := filepath.Join(fakePrefix, "drive_c", "users", "steamuser")
	fakeSaveDir := filepath.Join(fakeUserDir, "AppData", "LocalLow", "Studio", "TestRPG")
	backupBaseDir := filepath.Join(tmpDir, "backups")

	_ = os.MkdirAll(fakeGameDir, 0755)
	_ = os.MkdirAll(fakeSaveDir, 0755)
	saveContent := "rpg_save_state_123"
	saveFilePath := filepath.Join(fakeSaveDir, "save.dat")
	_ = os.WriteFile(saveFilePath, []byte(saveContent), 0644)

	// Write launch_profile.json in fakeGameDir to test automatic parameter resolution
	gamuxDir := filepath.Join(fakeGameDir, ".gamux")
	_ = os.MkdirAll(gamuxDir, 0755)
	profileJSON := `{
		"game_dir": "` + fakeGameDir + `",
		"app_id": "777777",
		"platform": "win64",
		"wine_prefix": "` + fakePrefix + `"
	}`
	_ = os.WriteFile(filepath.Join(gamuxDir, "launch_profile.json"), []byte(profileJSON), 0644)

	svc := NewService(ServiceConfig{
		CacheDir:         tmpDir,
		BackupDir:        backupBaseDir,
		RetentionCount:   3,
		GlobalPrefix:     fakePrefix,
		DisableAutoFetch: true,
	})

	// Inject manifest rule for 777777
	svc.manifestMgr.indexMap(map[string]ManifestGame{
		"TestRPG": {
			Steam: struct {
				ID uint32 `json:"id,omitempty"`
			}{ID: 777777},
			Files: map[string]ManifestFileRule{
				"<home>/AppData/LocalLow/Studio/TestRPG": {
					Tags: []string{"save"},
					When: []ManifestConstraint{{OS: "windows"}},
				},
			},
		},
	})

	// 1. Inspect Locations
	locs, err := svc.InspectLocations(context.Background(), fakeGameDir)
	if err != nil {
		t.Fatalf("InspectLocations failed: %v", err)
	}
	if len(locs.Locations) == 0 {
		t.Fatalf("expected at least 1 resolved save location")
	}

	// 2. Create Backup
	backup, err := svc.CreateBackup(context.Background(), fakeGameDir, "EndToEnd Test Snapshot", "manual")
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if backup == nil || backup.ID == "" {
		t.Fatalf("invalid backup returned")
	}

	// 3. List Backups
	list, err := svc.ListBackups(context.Background(), fakeGameDir)
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 backup in list, got %d", len(list))
	}

	// 4. Corrupt and Restore
	_ = os.Remove(saveFilePath)
	if err := svc.RestoreBackup(context.Background(), fakeGameDir, backup.ID); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	restored, err := os.ReadFile(saveFilePath)
	if err != nil {
		t.Fatalf("reading restored save failed: %v", err)
	}
	if string(restored) != saveContent {
		t.Errorf("expected restored content %q, got %q", saveContent, string(restored))
	}

	// 5. Delete Backup
	if err := svc.DeleteBackup(context.Background(), fakeGameDir, backup.ID); err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}
	listAfter, _ := svc.ListBackups(context.Background(), fakeGameDir)
	if len(listAfter) != 0 {
		t.Errorf("expected 0 backups after deletion, got %d", len(listAfter))
	}
}

func TestDeriveSemanticPrefix(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "/games/My Documents Game/prefix/drive_c/users/steamuser/AppData/Roaming",
			expected: "<winAppData>",
		},
		{
			path:     "/games/prefix/drive_c/users/steamuser/AppData/Roaming/Company/Game/save.dat",
			expected: filepath.Join("<winAppData>", "Company/Game/save.dat"),
		},
		{
			path:     "/games/prefix/drive_c/users/steamuser/AppData/LocalLow/Game/save.dat",
			expected: filepath.Join("<winLocalAppDataLow>", "Game/save.dat"),
		},
		{
			path:     "/games/prefix/drive_c/users/steamuser/Saved Games/Game/save.dat",
			expected: filepath.Join("<winSavedGames>", "Game/save.dat"),
		},
		{
			path:     "/games/My Documents Game/prefix/drive_c/users/steamuser/Documents/My Games/save.dat",
			expected: filepath.Join("<winDocuments>", "My Games/save.dat"),
		},
	}

	for _, tt := range tests {
		got := deriveSemanticPrefix(tt.path, "", "123", "win64")
		if got != tt.expected {
			t.Errorf("deriveSemanticPrefix(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}
