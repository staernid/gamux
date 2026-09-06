package ini

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLoad_Basic(t *testing.T) {
	input := `
# Global comment
global_key = global_val

[user::general]
account_name = "Player One"
account_steamid = 76561197960287930
language = english
ip_country = 'US'
empty_key =

[overlay::general]
enable_experimental_overlay=1
`
	f, err := LoadString(input)
	if err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	// Global / default section lookup
	if val, ok := f.Get("", "global_key"); !ok || val != "global_val" {
		t.Errorf("expected global_val, got %q (ok=%v)", val, ok)
	}

	// Case-insensitive section and key lookup
	sec, ok := f.GetSection("USER::GENERAL")
	if !ok {
		t.Fatalf("failed to find section USER::GENERAL")
	}

	// Unquoting double quotes
	if val, ok := sec.Get("Account_Name"); !ok || val != "Player One" {
		t.Errorf("expected 'Player One', got %q", val)
	}
	// Raw value
	if raw, ok := sec.GetRaw("Account_Name"); !ok || raw != `"Player One"` {
		t.Errorf("expected '\"Player One\"', got %q", raw)
	}

	// Unquoting single quotes
	if val, ok := sec.Get("IP_Country"); !ok || val != "US" {
		t.Errorf("expected 'US', got %q", val)
	}

	// Empty value
	if val, ok := sec.Get("empty_key"); !ok || val != "" {
		t.Errorf("expected empty string, got %q (ok=%v)", val, ok)
	}

	// Boolean lookup in overlay::general
	overlaySec, ok := f.GetSection("overlay::general")
	if !ok {
		t.Fatalf("failed to find overlay::general")
	}
	if b, ok := overlaySec.GetBool("enable_experimental_overlay"); !ok || !b {
		t.Errorf("expected overlay enabled true, got %v (ok=%v)", b, ok)
	}
}

func TestLoad_CommentsAndFormattingPreserved(t *testing.T) {
	input := `# Top comment
; Semicolon comment

[app::dlcs]
; DLC definitions
unlock_all=0
# Expansion 1
480=Spacewar DLC #1
481=Spacewar DLC #2
`
	f, err := LoadString(input)
	if err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	sec := f.Section("app::dlcs")
	if val, ok := sec.Get("480"); !ok || val != "Spacewar DLC #1" {
		t.Errorf("expected 'Spacewar DLC #1', got %q", val)
	}

	// Update existing key in place
	sec.Set("unlock_all", "1")

	// Append a new key
	sec.Set("482", "Spacewar DLC #3")

	out := f.String()

	// Verify comments and order are intact
	if !strings.Contains(out, "# Top comment") {
		t.Errorf("top comment lost: %s", out)
	}
	if !strings.Contains(out, "; Semicolon comment") {
		t.Errorf("semicolon comment lost: %s", out)
	}
	if !strings.Contains(out, "; DLC definitions") {
		t.Errorf("DLC definitions comment lost: %s", out)
	}
	if !strings.Contains(out, "# Expansion 1") {
		t.Errorf("Expansion 1 comment lost: %s", out)
	}
	if !strings.Contains(out, "unlock_all=1") {
		t.Errorf("unlock_all not updated to 1: %s", out)
	}
	if !strings.Contains(out, "482=Spacewar DLC #3") {
		t.Errorf("missing newly appended key: %s", out)
	}
}

func TestSection_Types(t *testing.T) {
	f := NewFile()
	sec := f.Section("settings")

	// Boolean truthy variants
	for _, v := range []string{"1", "true", "True", "TRUE", "yes", "YES", "on", "t"} {
		sec.Set("bool_test", v)
		b, ok := sec.GetBool("bool_test")
		if !ok || !b {
			t.Errorf("expected %q to parse as true, got %v (ok=%v)", v, b, ok)
		}
	}

	// Boolean falsy variants
	for _, v := range []string{"0", "false", "False", "no", "NO", "off", "f"} {
		sec.Set("bool_test", v)
		b, ok := sec.GetBool("bool_test")
		if !ok || b {
			t.Errorf("expected %q to parse as false, got %v (ok=%v)", v, b, ok)
		}
	}

	// SetBool defaults to 1/0
	sec.SetBool("new_flag", true)
	if v, _ := sec.Get("new_flag"); v != "1" {
		t.Errorf("expected '1', got %q", v)
	}
	sec.SetBool("new_flag", false)
	if v, _ := sec.Get("new_flag"); v != "0" {
		t.Errorf("expected '0', got %q", v)
	}

	// SetBool preserves word style if existing was true/false
	sec.Set("word_flag", "true")
	sec.SetBool("word_flag", false)
	if v, _ := sec.Get("word_flag"); v != "false" {
		t.Errorf("expected 'false', got %q", v)
	}

	// Int
	sec.SetInt("count", 42)
	if n, ok := sec.GetInt("count"); !ok || n != 42 {
		t.Errorf("expected 42, got %d (ok=%v)", n, ok)
	}
}

func TestDeleteKeyAndSection(t *testing.T) {
	f := NewFile()
	sec := f.Section("main")
	sec.Set("a", "1")
	sec.Set("b", "2")

	if !sec.Delete("a") {
		t.Errorf("expected Delete('a') to succeed")
	}
	if sec.HasKey("a") {
		t.Errorf("key 'a' still exists after delete")
	}
	if !sec.HasKey("b") {
		t.Errorf("key 'b' should still exist")
	}

	if !f.DeleteSection("main") {
		t.Errorf("expected DeleteSection('main') to succeed")
	}
	if f.HasSection("main") {
		t.Errorf("section 'main' still exists after DeleteSection")
	}
}

func TestSaveFile_AtomicAndPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sub", "test.ini")

	f := NewFile()
	f.Section("user::general").Set("account_name", "TestUser")

	if err := f.SaveFile(path); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify file was written and can be re-loaded
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if val, ok := loaded.Get("user::general", "account_name"); !ok || val != "TestUser" {
		t.Errorf("expected 'TestUser', got %q (ok=%v)", val, ok)
	}

	// Modify permissions and save again, verify permissions maintained
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	f.Section("user::general").Set("account_name", "UpdatedUser")
	if err := f.SaveFile(path); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %o", fi.Mode().Perm())
	}
}

func TestConcurrency(t *testing.T) {
	f := NewFile()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			sec := f.Section("concurrent")
			sec.SetInt("worker", workerID)
			_ = sec.GetString("worker", "")
			_ = f.String()
		}(i)
	}

	wg.Wait()
}
