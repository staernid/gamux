// Package ui provides interactive CLI menus, progress checklists, visual detection summaries,
// and actionable error guidance for gamux.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

var (
	isTerminal = isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
)

// ANSI Color Codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorDim    = "\033[2m"
	colorBlue   = "\033[34m"
)

func bold(s string) string {
	if !isTerminal {
		return s
	}
	return colorBold + s + colorReset
}

func cyan(s string) string {
	if !isTerminal {
		return s
	}
	return colorCyan + s + colorReset
}

func green(s string) string {
	if !isTerminal {
		return s
	}
	return colorGreen + s + colorReset
}

func yellow(s string) string {
	if !isTerminal {
		return s
	}
	return colorYellow + s + colorReset
}

func red(s string) string {
	if !isTerminal {
		return s
	}
	return colorRed + s + colorReset
}

func dim(s string) string {
	if !isTerminal {
		return s
	}
	return colorDim + s + colorReset
}

// RenderHeader displays the application banner and version.
func RenderHeader(version string) {
	fmt.Println()
	fmt.Println(bold(cyan("========================================================================")))
	fmt.Printf("  🎮 %s - Linux Steam Game Manager (%s)\n", bold("gamux"), cyan(version))
	fmt.Println(dim("  Automated Goldberg Emulation, Lutris Integration & Steam Shortcuts"))
	fmt.Println(bold(cyan("========================================================================")))
	fmt.Println()
}

// RenderAppHelp displays a friendly, hand-holdy quickstart guide and command reference.
func RenderAppHelp(version string) {
	RenderHeader(version)

	fmt.Println(bold("🚀 QUICK START EXAMPLES"))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Printf("  %-30s %s\n", cyan("gamux add ./MyGame"), dim("# Complete setup (Steamless + GBE + Lutris)"))
	fmt.Printf("  %-30s %s\n", cyan("gamux batch ~/Downloads"), dim("# Batch setup all games in a directory"))
	fmt.Printf("  %-30s %s\n", cyan("gamux status ./MyGame"), dim("# Inspect AppID, platform, and patch state"))
	fmt.Printf("  %-30s %s\n", cyan("gamux rollback ./MyGame"), dim("# Restore original binaries & remove emulator settings"))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()

	fmt.Println(bold("📋 AVAILABLE COMMANDS"))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Printf("  %-18s %s\n", cyan("add <path>"), "Full setup (detect AppID, unpack DRM, patch GBE, register Lutris)")
	fmt.Printf("  %-18s %s\n", cyan("apply <path>"), "Apply Goldberg Emulator DLLs/SO and configure DLCs")
	fmt.Printf("  %-18s %s\n", cyan("batch <dir>"), "Scan parent folder containing multiple games and setup each")
	fmt.Printf("  %-18s %s\n", cyan("status <path>"), "Inspect AppID, platform (Linux/Windows PE), and patch state")
	fmt.Printf("  %-18s %s\n", cyan("rollback <path>"), "Restore original files (.ORIGINAL) and delete generated settings")
	fmt.Printf("  %-18s %s\n", cyan("update"), "Update Goldberg Emulator & Steamless release assets")
	fmt.Printf("  %-18s %s\n", cyan("lutris-add <path>"), "Register game in Lutris launcher")
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()

	fmt.Println(bold("⚙️ KEY FLAGS & OPTIONS"))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Printf("  %-22s %s\n", yellow("--no-steamless"), "Skip automatic Steamless SteamStub DRM executable unpacking")
	fmt.Printf("  %-22s %s\n", yellow("--portable"), "Perform direct DLL/SO file replacement instead of loader mode")
	fmt.Printf("  %-22s %s\n", yellow("--lutris"), "Register game in Lutris library configuration")
	fmt.Printf("  %-22s %s\n", yellow("--dry-run"), "Preview actions without moving or modifying files")
	fmt.Printf("  %-22s %s\n", yellow("--yes, -y"), "Non-interactive mode (automatic yes to prompts)")
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()
}

// PromptString asks for text input with a default value.
func PromptString(promptText string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", bold(promptText), cyan(defaultValue))
	} else {
		fmt.Printf("%s: ", bold(promptText))
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// PromptYesNoWithExplanation asks a yes/no question with an explanatory hint.
func PromptYesNoWithExplanation(question string, explanation string, defaultYes bool) bool {
	fmt.Println()
	fmt.Println(bold("▶ " + question))
	if explanation != "" {
		fmt.Println(dim("  💡 " + explanation))
	}

	hint := " [Y/n]: "
	if !defaultYes {
		hint = " [y/N]: "
	}
	fmt.Print(bold("  Choice") + hint)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultYes
	}
	if input == "y" || input == "yes" {
		return true
	}
	if input == "n" || input == "no" {
		return false
	}
	return defaultYes
}

// DetectionInfoSummary holds metadata for displaying a detection preview.
type DetectionInfoSummary struct {
	Name            string
	AppID           string
	Platform        string
	GameDir         string
	ExePath         string
	State           string
	OriginalBackups int
}

// RenderDetectionSummary prints a visual preview of detected game metadata.
func RenderDetectionSummary(info DetectionInfoSummary) {
	fmt.Println()
	fmt.Println(bold(cyan("🔍 Game Detection Preview")))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Printf("  %-16s %s\n", bold("Game Title:"), green(info.Name))
	if info.AppID != "" && info.AppID != "0" {
		fmt.Printf("  %-16s %s\n", bold("Steam AppID:"), cyan(info.AppID))
	} else {
		fmt.Printf("  %-16s %s\n", bold("Steam AppID:"), yellow("Not Detected (Non-Steam / Custom)"))
	}
	fmt.Printf("  %-16s %s\n", bold("Platform:"), info.Platform)
	if absDir, err := filepath.Abs(info.GameDir); err == nil {
		fmt.Printf("  %-16s %s\n", bold("Directory:"), absDir)
	} else {
		fmt.Printf("  %-16s %s\n", bold("Directory:"), info.GameDir)
	}
	if info.ExePath != "" {
		fmt.Printf("  %-16s %s\n", bold("Executable:"), info.ExePath)
	}
	if info.State != "" {
		stateStr := info.State
		if strings.Contains(strings.ToLower(info.State), "original") {
			stateStr = green(info.State) + " (Clean / Ready for setup)"
		} else {
			stateStr = yellow(info.State)
		}
		fmt.Printf("  %-16s %s\n", bold("Current State:"), stateStr)
	}
	if info.OriginalBackups > 0 {
		fmt.Printf("  %-16s %d file(s) backed up\n", bold("Backups:"), info.OriginalBackups)
	}
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()
}

// RenderStep prints a progress step indicator.
func RenderStep(stepNum, totalSteps int, title string) {
	fmt.Printf("\n[%d/%d] %s...\n", stepNum, totalSteps, bold(cyan(title)))
}

// RenderSubStep prints a detail item under the current step.
func RenderSubStep(symbol, message string) {
	sym := green(symbol)
	if symbol == "!" || symbol == "⚠" {
		sym = yellow(symbol)
	} else if symbol == "✕" || symbol == "x" {
		sym = red(symbol)
	}
	fmt.Printf("      %s %s\n", sym, message)
}

// RenderSuccess prints a final success summary box.
func RenderSuccess(title, details string, nextSteps []string) {
	fmt.Println()
	fmt.Println(bold(green("========================================================================")))
	fmt.Printf("  ✨ %s\n", bold(green(title)))
	if details != "" {
		fmt.Printf("  %s\n", details)
	}
	if len(nextSteps) > 0 {
		fmt.Println()
		fmt.Println(bold("  💡 Next Steps:"))
		for _, step := range nextSteps {
			fmt.Printf("     • %s\n", step)
		}
	}
	fmt.Println(bold(green("========================================================================")))
	fmt.Println()
}

// RenderErrorHelp displays a formatted error diagnostic box with suggested actions.
func RenderErrorHelp(err error, suggestedActions []string) {
	fmt.Println()
	fmt.Println(bold(red("========================================================================")))
	fmt.Printf("  ❌ Error: %s\n", bold(err.Error()))
	fmt.Println(bold(red("========================================================================")))

	if len(suggestedActions) > 0 {
		fmt.Println()
		fmt.Println(bold("  💡 Suggested Fixes:"))
		for _, action := range suggestedActions {
			fmt.Printf("     • %s\n", action)
		}
		fmt.Println()
	}
}
