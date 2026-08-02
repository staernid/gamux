package detector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// ACFData represents extracted metadata from a Steam appmanifest_<appid>.acf file.
type ACFData struct {
	AppID      string
	Name       string
	InstallDir string
	BuildID    string
	Language   string
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
