package detector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// LaunchCandidate represents a Steam app launch option.
type LaunchCandidate struct {
	Name        string `json:"name"`
	Executable  string `json:"executable"`
	Arguments   string `json:"arguments"`
	Description string `json:"description"`
}

// ACFData represents extracted metadata from a Steam appmanifest_<appid>.acf file.
type ACFData struct {
	AppID         string
	Name          string
	InstallDir    string
	BuildID       string
	Language      string
	LaunchOptions []LaunchCandidate
}

var kvRegex = regexp.MustCompile(`^\s*"([^"]+)"\s+"([^"]*)"`)

// ParseACF parses a Steam ACF/VDF manifest file from a reader.
func ParseACF(r io.Reader) (*ACFData, error) {
	scanner := bufio.NewScanner(r)
	data := &ACFData{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		matches := kvRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := strings.ToLower(matches[1])
			val := matches[2]
			switch key {
			case "appid":
				data.AppID = val
			case "name":
				data.Name = val
			case "installdir":
				data.InstallDir = val
			case "buildid":
				data.BuildID = val
			case "language":
				data.Language = val
			case "executable":
				c := LaunchCandidate{Executable: val}
				data.LaunchOptions = append(data.LaunchOptions, c)
			case "arguments":
				if len(data.LaunchOptions) > 0 {
					data.LaunchOptions[len(data.LaunchOptions)-1].Arguments = val
				}
			case "description":
				if len(data.LaunchOptions) > 0 {
					data.LaunchOptions[len(data.LaunchOptions)-1].Description = val
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ACF file: %w", err)
	}

	if data.AppID == "" && data.Name == "" {
		return nil, fmt.Errorf("invalid ACF file: missing appid and name")
	}

	return data, nil
}

// ParseACFFile opens and parses an ACF manifest file from disk.
func ParseACFFile(path string) (*ACFData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ACF file: %w", err)
	}
	defer f.Close()

	return ParseACF(f)
}

// GenerateACFContent serializes ACFData into standard 1:1 Steam KeyValues VDF string representation.
func GenerateACFContent(data *ACFData) string {
	if data == nil {
		return ""
	}
	appID := data.AppID
	if appID == "" {
		appID = "0"
	}
	name := data.Name
	if name == "" {
		name = "Game"
	}
	installDir := data.InstallDir
	if installDir == "" {
		installDir = name
	}
	buildID := data.BuildID
	if buildID == "" {
		buildID = "0"
	}
	lang := data.Language
	if lang == "" {
		lang = "english"
	}

	var sb strings.Builder
	sb.WriteString("\"AppState\"\n{\n")
	fmt.Fprintf(&sb, "\t\"appid\"\t\t\"%s\"\n", appID)
	fmt.Fprintf(&sb, "\t\"name\"\t\t\"%s\"\n", name)
	fmt.Fprintf(&sb, "\t\"installdir\"\t\t\"%s\"\n", installDir)
	fmt.Fprintf(&sb, "\t\"StateFlags\"\t\t\"4\"\n")
	fmt.Fprintf(&sb, "\t\"buildid\"\t\t\"%s\"\n", buildID)
	fmt.Fprintf(&sb, "\t\"UserConfig\"\n\t{\n\t\t\"language\"\t\t\"%s\"\n\t}\n", lang)
	sb.WriteString("}\n")
	return sb.String()
}

