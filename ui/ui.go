// Package ui provides interactive CLI menus, progress checklists, visual detection summaries,
// and actionable error guidance for gamux.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

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
	colorDim     = "\033[2m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
)

func magenta(s string) string {
	if !isTerminal {
		return s
	}
	return colorMagenta + s + colorReset
}

func blue(s string) string {
	if !isTerminal {
		return s
	}
	return colorBlue + s + colorReset
}

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

// RenderAppHelp displays a clean, concise, single-source workflow guide from cli.App.
func RenderAppHelp(app *cli.App, version string) {
	RenderHeader(version)

	if app != nil && len(app.Commands) > 0 {
		fmt.Println(bold("📋 WORKFLOW & COMMANDS"))
		fmt.Println(dim("------------------------------------------------------------------------"))

		categories := []string{"Step 1: Acquisition", "Step 2: Setup & Integration", "Step 3: Inspection & Maintenance", "Maintenance & Tools"}
		cmdMap := make(map[string][]*cli.Command)
		for _, cmd := range app.Commands {
			cat := cmd.Category
			if cat == "" {
				cat = "Maintenance & Tools"
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
				fmt.Printf("    %-28s %s\n", cyan(sig), cmd.Usage)
			}
			fmt.Println()
		}

		fmt.Println(dim("------------------------------------------------------------------------"))
		fmt.Println()

		if len(app.Flags) > 0 {
			fmt.Println(bold("⚙️ GLOBAL OPTIONS"))
			fmt.Println(dim("------------------------------------------------------------------------"))

			for _, f := range app.Flags {
				names := f.Names()
				if len(names) == 0 {
					continue
				}
				var formattedNames []string
				for _, n := range names {
					if len(n) == 1 {
						formattedNames = append(formattedNames, "-"+n)
					} else {
						formattedNames = append(formattedNames, "--"+n)
					}
				}

				usage := ""
				switch tf := f.(type) {
				case *cli.BoolFlag:
					usage = tf.Usage
				case *cli.StringFlag:
					usage = tf.Usage
				case *cli.UintFlag:
					usage = tf.Usage
				case *cli.IntFlag:
					usage = tf.Usage
				}

				fmt.Printf("  %-28s %s\n", yellow(strings.Join(formattedNames, ", ")), usage)
			}
			fmt.Println(dim("------------------------------------------------------------------------"))
			fmt.Println()
		}
	}
}

// RenderCommandHelp displays clean formatted help for a specific subcommand.
func RenderCommandHelp(cmd *cli.Command, version string) {
	RenderHeader(version)

	if cmd == nil {
		return
	}

	sig := cmd.Name
	if cmd.ArgsUsage != "" {
		sig += " " + cmd.ArgsUsage
	}

	fmt.Printf("%s %s\n", bold("📋 COMMAND:"), cyan(sig))
	fmt.Println(dim("------------------------------------------------------------------------"))
	if cmd.Usage != "" {
		fmt.Printf("  %s\n\n", cmd.Usage)
	}

	if cmd.Description != "" {
		fmt.Printf("  %s\n\n", dim(cmd.Description))
	}

	if len(cmd.Flags) > 0 {
		fmt.Println(bold("⚙️ OPTIONS & FLAGS"))
		fmt.Println(dim("------------------------------------------------------------------------"))
		for _, f := range cmd.Flags {
			names := f.Names()
			if len(names) == 0 {
				continue
			}
			var formattedNames []string
			for _, n := range names {
				if len(n) == 1 {
					formattedNames = append(formattedNames, "-"+n)
				} else {
					formattedNames = append(formattedNames, "--"+n)
				}
			}

			usage := ""
			switch tf := f.(type) {
			case *cli.BoolFlag:
				usage = tf.Usage
			case *cli.StringFlag:
				usage = tf.Usage
			case *cli.UintFlag:
				usage = tf.Usage
			case *cli.IntFlag:
				usage = tf.Usage
			}

			fmt.Printf("  %-28s %s\n", yellow(strings.Join(formattedNames, ", ")), usage)
		}
		fmt.Println(dim("------------------------------------------------------------------------"))
	}
	fmt.Println()
}

var progressMutex sync.Mutex

// RenderProgress renders a clean single-line progress bar with current/total count and item label.
func RenderProgress(current, total int, item string) {
	if total <= 0 {
		return
	}
	progressMutex.Lock()
	defer progressMutex.Unlock()

	percent := float64(current) / float64(total) * 100
	width := 20
	completed := int((percent / 100) * float64(width))
	if completed > width {
		completed = width
	}
	if completed < 0 {
		completed = 0
	}

	bar := strings.Repeat("█", completed) + strings.Repeat("░", width-completed)

	base := filepath.Base(item)
	if len(base) > 28 {
		base = base[:25] + "..."
	}

	fmt.Printf("\r\033[K  %s %s %3.0f%% [%d/%d] %-28s",
		cyan("📥"),
		green("["+bar+"]"),
		percent,
		current,
		total,
		dim(base),
	)
}

// ClearProgress clears the current progress bar line in the terminal.
func ClearProgress() {
	fmt.Print("\r\033[K")
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
	Store            string
	Platform         string

	GameDir          string
	ExePath          string
	State            string
	OriginalBackups  int
	ManifestID       string
	BuildID          string
	DiskSizeBytes     int64
	FileCount         int
	OfficialFileCount  int
	ModifiedFiles     []string
	MissingFiles      []string
	UntrackedFiles    []string
	HasUpdate         bool
	RemoteManifestID  string
	DLCCount          int
	AchievementCount  int
	LutrisRegistered  bool
	RecentPatchNote   string
	NewsItems         []NewsItemSummary
}

// CleanBBCode strips Steam community BBCode tags for clean terminal rendering.
func CleanBBCode(input string) string {
	s := input
	s = strings.ReplaceAll(s, "[p]", "\n")
	s = strings.ReplaceAll(s, "[/p]", "\n")
	s = strings.ReplaceAll(s, "[br]", "\n")
	s = strings.ReplaceAll(s, "[h1]", "\n# ")
	s = strings.ReplaceAll(s, "[/h1]", "\n")
	s = strings.ReplaceAll(s, "[h2]", "\n## ")
	s = strings.ReplaceAll(s, "[/h2]", "\n")
	s = strings.ReplaceAll(s, "[b]", "")
	s = strings.ReplaceAll(s, "[/b]", "")
	s = strings.ReplaceAll(s, "[i]", "")
	s = strings.ReplaceAll(s, "[/i]", "")
	s = strings.ReplaceAll(s, "[list]", "\n")
	s = strings.ReplaceAll(s, "[/list]", "")
	s = strings.ReplaceAll(s, "[*]", " • ")

	// Strip [img...]...[/img] tags
	reImg := regexp.MustCompile(`(?i)\[img[^\]]*\].*?\[/img\]`)
	s = reImg.ReplaceAllString(s, "📷 [Image]")

	// Strip url tags [url=LINK]TEXT[/url] -> TEXT (LINK)
	reURL := regexp.MustCompile(`(?i)\[url=([^\]]+)\](.*?)\[/url\]`)
	s = reURL.ReplaceAllString(s, "$2 ($1)")

	// Strip remaining generic BBCode tags
	reGeneric := regexp.MustCompile(`\[/?[a-zA-Z0-9=\.:_-]+\]`)
	s = reGeneric.ReplaceAllString(s, "")

	// Clean excess empty lines
	reMultipleNewlines := regexp.MustCompile(`\n{3,}`)
	s = reMultipleNewlines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
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

// RenderTerseHeader outputs a clean section header for batch/terse inspection.
func RenderTerseHeader(parentDir string, totalGames int) {
	fmt.Println()
	fmt.Printf("%s (%s - %d games found)\n", bold(cyan("📦 Game Library Status")), parentDir, totalGames)
	fmt.Println(dim("----------------------------------------------------------------------------------------------------"))
}

// RenderTerseStatus prints a crisp 2-line card for a game directory.
func RenderTerseStatus(info DetectionInfoSummary) {
	// Game Title
	title := info.Name

	// Store Badge
	storeBadge := ""
	switch strings.ToLower(info.Store) {
	case "steam":
		storeBadge = cyan("[Steam]")
	case "gog":
		storeBadge = blue("[GOG]")
	case "epic":
		storeBadge = magenta("[Epic]")
	case "itch":
		storeBadge = yellow("[Itch]")
	default:
		storeBadge = dim("[?]")
	}

	// DRM / Patch Badge
	drmBadge := ""
	if strings.Contains(strings.ToLower(info.State), "original") {
		drmBadge = dim("[Clean]")
	} else if strings.Contains(strings.ToLower(info.State), "portable") {
		drmBadge = magenta("[Emu Portable]")
	} else {
		drmBadge = yellow("[Emu Loader]")
	}


	// Lutris Badge (only render when registered to reduce noise)
	lutrisBadge := ""
	if info.LutrisRegistered {
		lutrisBadge = green("[Lutris ✓]")
	}

	// Line 1: Title and inline state badges
	badges := []string{storeBadge, drmBadge}
	if lutrisBadge != "" {
		badges = append(badges, lutrisBadge)
	}

	fmt.Printf("  %s  %s\n",
		bold(title),
		strings.Join(badges, " "),
	)

	// Line 2: Details metadata card using clean box drawing └─
	var details []string

	if info.GameDir != "" {
		folderName := filepath.Base(info.GameDir)
		if folderName != "" && !strings.EqualFold(folderName, info.Name) && folderName != "." {
			details = append(details, fmt.Sprintf("dir: %s", folderName))
		}
	}

	if len(info.ModifiedFiles) > 0 || len(info.MissingFiles) > 0 {
		mismatches := len(info.ModifiedFiles) + len(info.MissingFiles)
		details = append(details, yellow(fmt.Sprintf("%d mismatches", mismatches)))
	} else if info.HasUpdate {
		details = append(details, yellow("update available"))
	} else if info.ManifestID != "" {
		details = append(details, green("up-to-date"))
	} else {
		details = append(details, dim("no manifest"))
	}

	if len(info.UntrackedFiles) > 0 {
		details = append(details, yellow(fmt.Sprintf("%d mods", len(info.UntrackedFiles))))
	}

	if info.AppID != "" && info.AppID != "0" {
		details = append(details, dim(fmt.Sprintf("appid: %s", info.AppID)))
	}

	detailStr := strings.Join(details, dim(" | "))
	fmt.Printf("  %s %s\n\n", dim("└─"), dim(detailStr))
}





// RenderTerseSummary prints summary stats after batch inspection.
func RenderTerseSummary(total, loaderCount, cleanCount, updatesAvail, lutrisCount int) {
	fmt.Println(dim("----------------------------------------------------------------------------------------------------"))
	fmt.Printf("  %s Summary: %d total | %s Emu | %s Clean | %s Updates | %s Lutris\n\n",
		yellow("💡"),
		total,
		yellow(fmt.Sprintf("%d", loaderCount)),
		green(fmt.Sprintf("%d", cleanCount)),
		cyan(fmt.Sprintf("%d", updatesAvail)),
		green(fmt.Sprintf("%d", lutrisCount)),
	)
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

	if len(info.ModifiedFiles) > 0 || len(info.MissingFiles) > 0 {
		count := len(info.ModifiedFiles) + len(info.MissingFiles)
		fmt.Printf("    • Steam Depot: %s (%d files modified/missing)\n", yellow(fmt.Sprintf("[%d File Mismatches]", count)), count)
		for i, mod := range info.ModifiedFiles {
			if i >= 3 {
				break
			}
			fmt.Printf("      - %s %s\n", dim(mod), yellow("(Modified/Stale)"))
		}
		for i, miss := range info.MissingFiles {
			if i >= 3 {
				break
			}
			fmt.Printf("      - %s %s\n", dim(miss), red("(Missing)"))
		}
	} else if info.HasUpdate {
		fmt.Printf("    • Steam Depot: %s (Local: %s → Remote: %s)\n", yellow("[🚀 Update Available]"), info.ManifestID, info.RemoteManifestID)
	} else if info.ManifestID != "" {
		fmt.Printf("    • Steam Depot: %s\n", green("[Up to Date]"))
	} else {
		fmt.Printf("    • Steam Depot: %s\n", dim("[No Local Manifest]"))
	}


	if info.LutrisRegistered {
		fmt.Printf("    • Lutris:     %s\n", green("[Registered]"))
	} else {
		fmt.Printf("    • Lutris:     %s\n", dim("[Not Registered]"))
	}

	if len(info.UntrackedFiles) > 0 {
		fmt.Printf("    • Untracked/Mods: %s\n", yellow(fmt.Sprintf("[%d files detected]", len(info.UntrackedFiles))))
		for i, mod := range info.UntrackedFiles {
			if i >= 5 {
				fmt.Printf("      - %s\n", dim(fmt.Sprintf("...and %d more untracked files", len(info.UntrackedFiles)-5)))
				break
			}
			fmt.Printf("      - %s\n", dim(mod))
		}
	} else if info.OfficialFileCount > 0 {
		fmt.Printf("    • Untracked/Mods: %s\n", green("[0 untracked files]"))
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
			fmt.Printf("    [%d] %s %s\n", i+1, bold(cyan(CleanBBCode(n.Title))), dim("("+n.FeedLabel+")"))
		}
		fmt.Println()
		fmt.Printf("  💡 %s %s\n", yellow("Tip:"), dim("Run 'gamux news 1' to read full patch details."))
	} else if info.RecentPatchNote != "" {
		fmt.Println(dim("------------------------------------------------------------------------"))
		fmt.Println(bold("  📰 Recent Steam Patch Notes / News:"))
		fmt.Printf("    %s\n", cyan(CleanBBCode(info.RecentPatchNote)))
		fmt.Println()
		fmt.Printf("  💡 %s %s\n", yellow("Tip:"), dim("Run 'gamux news 1' to read full patch details."))
	}

	if strings.Contains(strings.ToLower(info.State), "original") {
		fmt.Println()
		fmt.Printf("  💡 %s %s\n", yellow("Next Step:"), dim(fmt.Sprintf("Run 'gamux sync \"%s\"' to set up GBE emulator & Lutris integration.", info.GameDir)))
	}


	fmt.Println(dim("------------------------------------------------------------------------"))
	fmt.Println()
}

// RenderNewsItem prints a full, clean formatted patch note item.
func RenderNewsItem(gameTitle string, index int, title, feedLabel, contents, url string, date int64) {
	fmt.Println()
	fmt.Println(bold(cyan("========================================================================")))
	fmt.Printf("  📰 %s - Patch Note #%d: %s\n", bold(gameTitle), index, green(CleanBBCode(title)))
	if feedLabel != "" {
		fmt.Printf("  Source: %s\n", dim(feedLabel))
	}
	if url != "" {
		fmt.Printf("  URL:    %s\n", dim(url))
	}
	fmt.Println(bold(cyan("========================================================================")))
	fmt.Println()
	if contents != "" {
		fmt.Println(CleanBBCode(contents))
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

