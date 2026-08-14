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
	"github.com/staernid/gamux/util"
	"github.com/urfave/cli/v2"
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

// RenderAppHelp displays a friendly, dynamic quickstart guide and command reference from cli.App.
func RenderAppHelp(app *cli.App, version string) {
	RenderHeader(version)

	fmt.Println(bold("🚀 QUICK START EXAMPLES"))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Printf("  %-32s %s\n", cyan("gamux download 544550"), dim("# Download from Steam CDN & auto-setup"))
	fmt.Printf("  %-32s %s\n", cyan("gamux add ./MyGame"), dim("# Complete setup for existing game folder"))
	fmt.Printf("  %-32s %s\n", cyan("gamux batch ~/Downloads"), dim("# Batch setup all games in a directory"))
	fmt.Printf("  %-32s %s\n", cyan("gamux status ./MyGame --news"), dim("# Inspect AppID, platform, and patch history"))
	fmt.Printf("  %-32s %s\n", cyan("gamux news ./MyGame 1"), dim("# Read full text of patch note #1"))
	fmt.Printf("  %-32s %s\n", cyan("gamux rollback ./MyGame"), dim("# Restore original binaries & remove settings"))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()

	if app != nil && len(app.Commands) > 0 {
		fmt.Println(bold("📋 AVAILABLE COMMANDS"))
		fmt.Println(dim("------------------------------------------------------------------------"))

		categories := []string{"Acquisition", "Setup & Management", "Maintenance"}
		cmdMap := make(map[string][]*cli.Command)
		for _, cmd := range app.Commands {
			cat := cmd.Category
			if cat == "" {
				cat = "Maintenance"
			}
			cmdMap[cat] = append(cmdMap[cat], cmd)
		}

		for _, cat := range categories {
			cmds := cmdMap[cat]
			if len(cmds) == 0 {
				continue
			}
			fmt.Printf(bold("  %s:\n"), cat)
			for _, cmd := range cmds {
				sig := cmd.Name
				if cmd.ArgsUsage != "" {
					sig += " " + cmd.ArgsUsage
				}
				fmt.Printf("    %-24s %s\n", cyan(sig), cmd.Usage)
			}
			fmt.Println()
		}

		fmt.Println(dim("------------------------------------------------------------------------"))
		fmt.Println()

		fmt.Println(bold("⚙️ KEY FLAGS & OPTIONS"))
		fmt.Println(dim("------------------------------------------------------------------------"))
		for _, f := range app.Flags {
			fmt.Printf("  %-24s %s\n", yellow(f.Names()[0]), f.String())
		}
		fmt.Printf("  %-24s %s\n", yellow("--news, --patch-notes"), "Display recent Steam patch notes and update history in status")
		fmt.Printf("  %-24s %s\n", yellow("--platform win64|linux|osx"), "Target platform architecture for downloading depots")
		fmt.Printf("  %-24s %s\n", yellow("--no-setup, --raw"), "Skip automatic post-download DRM unpacking & GBE setup")
		fmt.Printf("  %-24s %s\n", yellow("--portable"), "Perform direct DLL/SO file replacement instead of loader mode")
		fmt.Printf("  %-24s %s\n", yellow("--yes, -y"), "Non-interactive mode (automatic yes to prompts)")
		fmt.Println(dim("------------------------------------------------------------------------"))
		fmt.Println()
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

// NewsItemSummary holds metadata for rendering patch notes.
type NewsItemSummary struct {
	Title     string
	FeedLabel string
	Date      int64
	URL       string
}

// PromptSelectPlatform presents an interactive choice for target platform/depot selection.
func PromptSelectPlatform(gameTitle string, availablePlatforms []string) (string, error) {
	if len(availablePlatforms) == 0 {
		availablePlatforms = []string{"win64", "linux", "osx", "all"}
	}

	fmt.Println()
	fmt.Printf("%s for %s:\n", bold(cyan("🖥️ Select Target Platform / Architecture")), bold(gameTitle))
	fmt.Println(dim("------------------------------------------------------------------------"))
	for i, p := range availablePlatforms {
		desc := ""
		switch p {
		case "win64", "win32":
			desc = "(Recommended for Proton / Wine)"
		case "linux":
			desc = "(Native Linux binary)"
		case "osx", "macos":
			desc = "(Native macOS binary)"
		case "all":
			desc = "(Download all available depot architectures)"
		}
		fmt.Printf("  [%d] %s %s\n", i+1, bold(p), dim(desc))
	}
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Print(bold(fmt.Sprintf("  Choice [1-%d] (default 1): ", len(availablePlatforms))))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return availablePlatforms[0], nil
	}

	input = strings.TrimSpace(input)
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(availablePlatforms) {
		return availablePlatforms[choice-1], nil
	}

	return availablePlatforms[0], nil
}

// DetectionInfoSummary holds metadata for displaying a rich game status dashboard.
type DetectionInfoSummary struct {
	Name             string
	AppID            string
	Platform         string
	GameDir          string
	ExePath          string
	State            string
	OriginalBackups  int
	ManifestID       string
	BuildID          string
	DiskSizeBytes    int64
	FileCount        int
	DLCCount         int
	AchievementCount int
	LutrisRegistered bool
	RecentPatchNote  string
	NewsItems        []NewsItemSummary
}

// FormatBytes formats byte counts into human-readable strings (e.g. 1.29 GiB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RenderDetectionSummary prints a visual preview of detected game metadata.
func RenderDetectionSummary(info DetectionInfoSummary) {
	fmt.Println()
	fmt.Println(bold(cyan("🔍 Game Status Dashboard")))
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Printf("  %-18s %s\n", bold("Game Title:"), green(info.Name))
	if info.AppID != "" && info.AppID != "0" {
		fmt.Printf("  %-18s %s\n", bold("Steam AppID:"), cyan(info.AppID))
	} else {
		fmt.Printf("  %-18s %s\n", bold("Steam AppID:"), yellow("Not Detected (Non-Steam / Custom)"))
	}
	fmt.Printf("  %-18s %s\n", bold("Platform:"), info.Platform)
	if absDir, err := filepath.Abs(info.GameDir); err == nil {
		fmt.Printf("  %-18s %s\n", bold("Directory:"), absDir)
	} else {
		fmt.Printf("  %-18s %s\n", bold("Directory:"), info.GameDir)
	}
	if info.ExePath != "" {
		fmt.Printf("  %-18s %s\n", bold("Main Executable:"), info.ExePath)
	}
	if info.ManifestID != "" {
		fmt.Printf("  %-18s %s\n", bold("Manifest ID:"), cyan(info.ManifestID))
	}
	if info.BuildID != "" {
		fmt.Printf("  %-18s %s\n", bold("Build ID:"), yellow(info.BuildID))
	}
	if info.DiskSizeBytes > 0 {
		fmt.Printf("  %-18s %s (%d files)\n", bold("Size on Disk:"), green(FormatBytes(info.DiskSizeBytes)), info.FileCount)
	}

	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println(bold("  Integrations & Patch Badges:"))

	stateStr := info.State
	if strings.Contains(strings.ToLower(info.State), "original") {
		stateStr = green("[DRM: Clean]")
	} else {
		stateStr = yellow("[" + info.State + "]")
	}
	fmt.Printf("    • State:      %s\n", stateStr)

	if info.LutrisRegistered {
		fmt.Printf("    • Lutris:     %s\n", green("[Registered]"))
	} else {
		fmt.Printf("    • Lutris:     %s\n", dim("[Not Registered]"))
	}

	if info.DLCCount > 0 {
		fmt.Printf("    • DLCs:       %s\n", green(fmt.Sprintf("[%d Configured]", info.DLCCount)))
	}
	if info.AchievementCount > 0 {
		fmt.Printf("    • Achievements: %s\n", green(fmt.Sprintf("[%d Cached]", info.AchievementCount)))
	}

	if len(info.NewsItems) > 0 {
		fmt.Println(dim("------------------------------------------------------------------------"))
		fmt.Println(bold("  📰 Recent Steam Patch Notes / News History:"))
		for i, n := range info.NewsItems {
			fmt.Printf("    [%d] %s %s\n", i+1, bold(cyan(n.Title)), dim("("+n.FeedLabel+")"))
		}
		fmt.Println()
		fmt.Printf("  💡 %s %s\n", yellow("Tip:"), dim("Run 'gamux news 1' to read full patch details."))
	} else if info.RecentPatchNote != "" {
		fmt.Println(dim("------------------------------------------------------------------------"))
		fmt.Println(bold("  📰 Recent Steam Patch Notes / News:"))
		fmt.Printf("    %s\n", cyan(info.RecentPatchNote))
		fmt.Println()
		fmt.Printf("  💡 %s %s\n", yellow("Tip:"), dim("Run 'gamux news 1' to read full patch details."))
	}

	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()
}

// RenderNewsItem prints a full, clean formatted patch note item.
func RenderNewsItem(gameTitle string, index int, title, feedLabel, contents, url string, date int64) {
	fmt.Println()
	fmt.Println(bold(cyan("========================================================================")))
	fmt.Printf("  📰 %s - Patch Note #%d: %s\n", bold(gameTitle), index, green(title))
	if feedLabel != "" {
		fmt.Printf("  Source: %s\n", dim(feedLabel))
	}
	if url != "" {
		fmt.Printf("  URL:    %s\n", dim(url))
	}
	fmt.Println(bold(cyan("========================================================================")))
	fmt.Println()
	if contents != "" {
		fmt.Println(contents)
	} else {
		fmt.Println(dim("No text contents available for this patch item."))
	}
	fmt.Println()
	fmt.Println(bold(cyan("========================================================================")))
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
	if len(suggestedActions) == 0 {
		suggestedActions = util.ExtractSuggestedFixes(err)
	}

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

// CandidateItem holds metadata for presenting an AppID selection prompt.
type CandidateItem struct {
	AppID uint32
	Name  string
}

// PromptSelectCandidate presents an interactive menu to disambiguate game title search matches.
func PromptSelectCandidate(query string, candidates []CandidateItem) (CandidateItem, error) {
	if len(candidates) == 0 {
		return CandidateItem{}, fmt.Errorf("no candidates to select")
	}

	fmt.Println()
	fmt.Printf("%s: %q\n", bold(cyan("🔍 Disambiguate Game Title")), query)
	fmt.Println(dim("------------------------------------------------------------------------"))
	for i, c := range candidates {
		fmt.Printf("  [%d] %s %s\n", i+1, bold(c.Name), dim(fmt.Sprintf("(AppID %d)", c.AppID)))
	}
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Print(bold("  Select candidate [1-") + fmt.Sprintf("%d", len(candidates)) + "]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return candidates[0], nil
	}

	input = strings.TrimSpace(input)
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(candidates) {
		return candidates[choice-1], nil
	}

	return candidates[0], nil
}

// LaunchOptionItem holds metadata for selecting game launch executable options.
type LaunchOptionItem struct {
	Name       string
	Executable string
	Arguments  string
}

// PromptSelectLaunchOption presents an interactive menu to choose between multiple launch options.
func PromptSelectLaunchOption(gameTitle string, options []LaunchOptionItem) (LaunchOptionItem, error) {
	if len(options) == 0 {
		return LaunchOptionItem{}, fmt.Errorf("no launch options to select")
	}
	if len(options) == 1 {
		return options[0], nil
	}

	fmt.Println()
	fmt.Printf("%s for %s:\n", bold(cyan("🎮 Select Target Launch Executable")), bold(gameTitle))
	fmt.Println(dim("------------------------------------------------------------------------"))
	for i, opt := range options {
		label := opt.Name
		if label == "" {
			label = filepath.Base(opt.Executable)
		}
		argsStr := ""
		if opt.Arguments != "" {
			argsStr = fmt.Sprintf(" %s", dim("(args: "+opt.Arguments+")"))
		}
		fmt.Printf("  [%d] %s %s%s\n", i+1, bold(label), dim(fmt.Sprintf("(%s)", opt.Executable)), argsStr)
	}
	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Print(bold("  Choice [1-") + fmt.Sprintf("%d", len(options)) + "] (default 1): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return options[0], nil
	}

	input = strings.TrimSpace(input)
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(options) {
		return options[choice-1], nil
	}

	return options[0], nil
}

