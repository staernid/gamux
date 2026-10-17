package util

import (
	"strings"
	"testing"
)

func TestRunCmd(t *testing.T) {
	// Test a successful command
	output, err := RunCmd("echo", "hello", "world")
	if err != nil {
		t.Fatalf("RunCmd failed: %v", err)
	}
	expectedOutput := "hello world\n"
	if string(output) != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, string(output))
	}

	// Test a command that fails
	_, err = RunCmd("false")
	if err == nil {
		t.Fatalf("RunCmd was expected to fail but succeeded")
	}
	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("Expected 'command failed' error, got %v", err)
	}

	// Test a command with stderr output
	_, err = RunCmd("sh", "-c", "echo 'error' >&2 && exit 1")
	if err == nil {
		t.Fatalf("RunCmd was expected to fail but succeeded")
	}
	if !strings.Contains(err.Error(), "stderr: error") {
		t.Errorf("Expected stderr output in error, got %v", err)
	}
}
