package gbe

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAchievements_EmptyDir(t *testing.T) {
	_, err := LoadAchievements(context.Background(), "", 0, nil, "")
	if err == nil {
		t.Fatal("expected error for empty gameDir")
	}
}

func TestLoadAchievements_LocalSchemaAndUnlocked(t *testing.T) {
	tmpDir := t.TempDir()
	settingsDir := filepath.Join(tmpDir, "steam_settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("mkdir steam_settings: %v", err)
	}

	schemas := []map[string]any{
		{
			"name":        "ACH_FIRST",
			"displayName": "First Achievement",
			"description": "Do the first thing",
			"hidden":      0,
			"icon":        "img/ach_first.png",
			"icongray":    "img/ach_first_gray.png",
		},
		{
			"name":        "ACH_SECOND",
			"displayName": "Second Achievement",
			"description": "Do the second thing",
			"hidden":      1,
			"icon":        "img/ach_second.png",
			"icongray":    "img/ach_second_gray.png",
		},
	}
	schemaData, _ := json.Marshal(schemas)
	if err := os.WriteFile(filepath.Join(settingsDir, "achievements.json"), schemaData, 0644); err != nil {
		t.Fatalf("write achievements.json: %v", err)
	}

	unlocked := map[string]any{
		"ACH_FIRST": map[string]any{
			"earned":      true,
			"earned_time": 1700000000.0,
		},
	}
	unlockedData, _ := json.Marshal(unlocked)
	if err := os.WriteFile(filepath.Join(settingsDir, "achievements_unlocked.json"), unlockedData, 0644); err != nil {
		t.Fatalf("write achievements_unlocked.json: %v", err)
	}

	items, err := LoadAchievements(context.Background(), tmpDir, 12345, []string{settingsDir}, "")
	if err != nil {
		t.Fatalf("LoadAchievements failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 achievement items, got %d", len(items))
	}

	// Unlocked items should be sorted first
	if items[0].ID != "ACH_FIRST" || !items[0].Unlocked || items[0].UnlockTime != 1700000000 {
		t.Errorf("expected ACH_FIRST to be unlocked and first, got %+v", items[0])
	}
	if items[1].ID != "ACH_SECOND" || items[1].Unlocked {
		t.Errorf("expected ACH_SECOND to be locked and second, got %+v", items[1])
	}
	if !items[1].Hidden {
		t.Errorf("expected ACH_SECOND to be hidden")
	}
}

func TestLoadAchievements_MapSchemaFormat(t *testing.T) {
	tmpDir := t.TempDir()
	settingsDir := filepath.Join(tmpDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)

	schemaMap := map[string]any{
		"ACH_WIN": map[string]any{
			"displayName": "Winner",
			"description": "Win the game",
			"hidden":      0,
			"icon":        "img/win.png",
			"icongray":    "img/win_gray.png",
		},
	}
	data, _ := json.Marshal(schemaMap)
	_ = os.WriteFile(filepath.Join(settingsDir, "achievements.json"), data, 0644)

	items, err := LoadAchievements(context.Background(), tmpDir, 999, []string{settingsDir}, "")
	if err != nil {
		t.Fatalf("LoadAchievements failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 achievement item, got %d", len(items))
	}
	if items[0].Name != "Winner" {
		t.Errorf("expected Name 'Winner', got %q", items[0].Name)
	}
}
