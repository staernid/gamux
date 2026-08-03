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

// MenuOption represents an item in the interactive menu.
type MenuOption struct {
	Key         string
	Title       string
	Description string
}

// RenderMenu presents an interactive menu and returns the selected option key.
func RenderMenu(options []MenuOption) string {
	fmt.Println(bold("What would you like to do?"))
	fmt.Println()

	for _, opt := range options {
		fmt.Printf("  [%s] %s\n", cyan(opt.Key), bold(opt.Title))
		if opt.Description != "" {
			fmt.Printf("      %s\n", dim(opt.Description))
		}
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(bold("Select an option: "))
		input, err := reader.ReadString('\n')
		if err != nil {
			return ""
		}
		input = strings.TrimSpace(input)

		for _, opt := range options {
			if strings.EqualFold(input, opt.Key) {
				return opt.Key
			}
		}
		fmt.Println(yellow("Invalid selection. Please try again."))
	}
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
