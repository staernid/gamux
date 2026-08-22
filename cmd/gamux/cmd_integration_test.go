package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func executeCLICommand(t *testing.T, args []string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w
	os.Stderr = w

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	app := buildApp()
	runErr := app.Run(args)

	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	output := <-outChan
	return output, runErr
}

func TestCLI_AppHelp(t *testing.T) {
	out, err := executeCLICommand(t, []string{"gamux", "--help"})
	if err != nil {
		t.Fatalf("gamux --help failed: %v", err)
	}

	if !strings.Contains(out, "gamux - Linux Steam Game Manager") {
		t.Errorf("expected app title in help output, got:\n%s", out)
	}
	if !strings.Contains(out, "WORKFLOW & COMMANDS") {
		t.Errorf("expected WORKFLOW & COMMANDS section in help output, got:\n%s", out)
	}
}

func TestCLI_CommandHelp_Apply(t *testing.T) {
	out, err := executeCLICommand(t, []string{"gamux", "apply", "--help"})
	if err != nil {
		t.Fatalf("gamux apply --help failed: %v", err)
	}

	if !strings.Contains(out, "COMMAND: apply [path]") {
		t.Errorf("expected COMMAND: apply [path] header, got:\n%s", out)
	}
	if !strings.Contains(out, "--portable") {
		t.Errorf("expected --portable flag listed in help, got:\n%s", out)
	}
	if strings.Contains(out, "--promote") {
		t.Errorf("expected --promote flag to NOT be present, got:\n%s", out)
	}
}

func TestCLI_NoArgs_TriggersCommandHelp(t *testing.T) {
	out, err := executeCLICommand(t, []string{"gamux", "apply"})
	if err != nil {
		t.Fatalf("gamux apply (0 args) failed: %v", err)
	}

	if !strings.Contains(out, "COMMAND: apply [path]") {
		t.Errorf("expected command help screen when running gamux apply with 0 args, got:\n%s", out)
	}
}

func TestCLI_ConfigShow(t *testing.T) {
	out, err := executeCLICommand(t, []string{"gamux", "config", "show"})
	if err != nil {
		t.Fatalf("gamux config show failed: %v", err)
	}

	if !strings.Contains(out, "gamux Configuration Dashboard") {
		t.Errorf("expected configuration dashboard header, got:\n%s", out)
	}
	if !strings.Contains(out, "gbe_mode") {
		t.Errorf("expected gbe_mode in dashboard, got:\n%s", out)
	}
	if strings.Contains(out, "promote") {
		t.Errorf("expected promote to NOT be in dashboard, got:\n%s", out)
	}
}
