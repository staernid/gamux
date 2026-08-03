package ui

import (
	"errors"
	"testing"
)

func TestDetectionSummary(t *testing.T) {
	info := DetectionInfoSummary{
		Name:     "Test Game",
		AppID:    "123456",
		Platform: "Windows PE (64-bit)",
		GameDir:  "/tmp/testgame",
		ExePath:  "/tmp/testgame/game.exe",
		State:    "Original",
	}

	// Verify RenderDetectionSummary does not panic
	RenderDetectionSummary(info)
}

func TestRenderStepsAndSuccess(t *testing.T) {
	RenderHeader("v0.3.1")
	RenderStep(1, 3, "Detecting Game")
	RenderSubStep("✓", "Detected AppID 12345")
	RenderSubStep("!", "Warning message test")
	RenderSuccess("Setup Complete", "All operations finished cleanly.", []string{"Launch via Steam", "Check Lutris"})
	RenderErrorHelp(errors.New("sample error"), []string{"Check file permissions", "Verify game path"})
}
