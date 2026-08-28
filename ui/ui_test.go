package ui

import (
	"errors"
	"testing"

	"github.com/staernid/gamux/domain"
	"github.com/urfave/cli/v2"
)

func TestDetectionSummary(t *testing.T) {
	game := domain.NewGame("Test Game", "/tmp/testgame", "123456", "Windows PE (64-bit)")
	game.ExePath = "/tmp/testgame/game.exe"
	game.State = "Original"

	// Verify RenderDetectionSummary & RenderTerseStatus do not panic
	RenderDetectionSummary(game)
	RenderTerseStatus(game)
	RenderDetectionSummary(nil)
	RenderTerseStatus(nil)
}

func TestRenderStepsAndSuccess(t *testing.T) {
	RenderHeader("v0.3.1")
	RenderStep(1, 3, "Detecting Game")
	RenderSubStep("✓", "Detected AppID 12345")
	RenderSubStep("!", "Warning message test")
	RenderSuccess("Setup Complete", "All operations finished cleanly.", []string{"Launch via Gamux", "Check Status"})
	RenderErrorHelp(errors.New("sample error"), []string{"Check file permissions", "Verify game path"})
}

func TestHelpScreenCoverage(t *testing.T) {
	// Verify RenderAppHelp runs cleanly with nil app
	RenderAppHelp(nil, "v1.0.0")

	// Verify RenderAppHelp with cli.App renders commands dynamically
	dummyApp := &cli.App{
		Name:    "gamux",
		Version: "v1.0.0",
		Commands: []*cli.Command{
			{Name: "add", Category: "Setup & Management", Usage: "Add game"},
			{Name: "download", Category: "Acquisition", Usage: "Download game"},
			{Name: "news", Category: "Maintenance", Usage: "Read patch notes"},
		},
	}
	RenderAppHelp(dummyApp, "v1.0.0")
}
