package ui

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/staernid/gamux/config"
)

func TestRenderConfigSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.HubcapAPIKey = "testkey123"

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RenderConfigSummary(cfg, "/tmp/config.json")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if output == "" {
		t.Errorf("expected rendered summary output")
	}
}
