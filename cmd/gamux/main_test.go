package main

import (
	"testing"
)

func TestCLIRegistrySanity(t *testing.T) {
	// Verify main CLI commands are well-formed and categorized
	app := buildApp()
	if app == nil {
		t.Fatal("buildApp returned nil")
	}

	if len(app.Commands) == 0 {
		t.Fatal("app.Commands is empty")
	}

	for _, cmd := range app.Commands {
		if cmd.Name == "" {
			t.Errorf("found command with empty Name")
		}
		if cmd.Category == "" && cmd.Name != "help" {
			t.Errorf("command %q has empty Category", cmd.Name)
		}
		if cmd.Usage == "" {
			t.Errorf("command %q has empty Usage text", cmd.Name)
		}
	}
}
