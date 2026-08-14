package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// NotifyIssue presents an alert on the console (stderr) and sends a non-intrusive desktop notification via notify-send if available.
func NotifyIssue(ctx context.Context, title, message string) error {
	// 1. Console Output (ANSI color to stderr)
	fmt.Fprintf(os.Stderr, "\n%s\n", yellow("========================================================================"))
	fmt.Fprintf(os.Stderr, "%s\n", yellow(fmt.Sprintf("  ⚠️  gamux Alert: %s", title)))
	fmt.Fprintf(os.Stderr, "%s\n", yellow("------------------------------------------------------------------------"))
	fmt.Fprintf(os.Stderr, "%s\n", message)
	fmt.Fprintf(os.Stderr, "%s\n\n", yellow("========================================================================"))

	// 2. Non-intrusive Desktop Toast Notification via notify-send
	if notifySendPath, err := exec.LookPath("notify-send"); err == nil {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(timeoutCtx, notifySendPath,
			"-u", "normal",
			"-i", "dialog-warning",
			fmt.Sprintf("gamux Alert: %s", title),
			message,
		)
		_ = cmd.Run()
	}

	return nil
}
