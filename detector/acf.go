package detector

import (
	"fmt"
	"io"
	"os"
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

// ParseACF parses a Steam ACF/VDF manifest file from a reader using the recursive VDF parser.
func ParseACF(r io.Reader) (*ACFData, error) {
	node, err := ParseVDF(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidManifest, err.Error())
	}

	data := &ACFData{
		AppID:      node.GetString("appid"),
		Name:       node.GetString("name"),
		InstallDir: node.GetString("installdir"),
		BuildID:    node.GetString("buildid"),
		Language:   node.GetString("UserConfig/language"),
	}

	if data.Language == "" {
		data.Language = node.GetString("language")
	}

	// Parse launch options from config/launch or direct keys
	if launchNode := node.Get("config"); launchNode != nil {
		if sub := launchNode.Get("launch"); sub != nil {
			for _, child := range sub.Children {
				exe := child.GetString("executable")
				args := child.GetString("arguments")
				desc := child.GetString("description")
				if exe != "" {
					data.LaunchOptions = append(data.LaunchOptions, LaunchCandidate{
						Name:        child.Key,
						Executable:  exe,
						Arguments:   args,
						Description: desc,
					})
				}
			}
		}
	}

	if data.AppID == "" && data.Name == "" {
		return nil, ErrInvalidManifest
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

	if len(data.LaunchOptions) > 0 {
		sb.WriteString("\t\"config\"\n\t{\n\t\t\"launch\"\n\t\t{\n")
		for i, opt := range data.LaunchOptions {
			fmt.Fprintf(&sb, "\t\t\t\"%d\"\n\t\t\t{\n", i)
			if opt.Executable != "" {
				fmt.Fprintf(&sb, "\t\t\t\t\"executable\"\t\t\"%s\"\n", opt.Executable)
			}
			if opt.Arguments != "" {
				fmt.Fprintf(&sb, "\t\t\t\t\"arguments\"\t\t\"%s\"\n", opt.Arguments)
			}
			if opt.Description != "" {
				fmt.Fprintf(&sb, "\t\t\t\t\"description\"\t\t\"%s\"\n", opt.Description)
			}
			sb.WriteString("\t\t\t}\n")
		}
		sb.WriteString("\t\t}\n\t}\n")
	}

	sb.WriteString("}\n")
	return sb.String()
}

