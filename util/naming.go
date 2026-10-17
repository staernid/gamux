package util

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TitleCase upper-cases the first letter of each word, leaving the rest untouched.
func TitleCase(s string) string {
	return cases.Title(language.Und, cases.NoLower).String(s)
}

// SanitizeFilename replaces illegal filesystem characters with dashes to return a clean folder name.
func SanitizeFilename(name string) string {
	invalidChars := []string{":", "/", "\\", "?", "*", "<", ">", "|", "\""}
	res := name
	for _, c := range invalidChars {
		res = strings.ReplaceAll(res, c, " - ")
	}
	fields := strings.Fields(res)
	clean := strings.Join(fields, " ")
	if clean == "" {
		return "Game"
	}
	return clean
}

// SanitizeInstallDir converts a game title into a clean, 1:1 Steam-style installdir folder name.
// It removes trademark/registered symbols (™, ®, ©), removes illegal OS path characters,
// and normalizes whitespace while preserving standard folder readability.
func SanitizeInstallDir(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Game"
	}
	s := title
	s = strings.ReplaceAll(s, "™", "")
	s = strings.ReplaceAll(s, "®", "")
	s = strings.ReplaceAll(s, "©", "")

	invalidChars := []string{":", "/", "\\", "?", "*", "<", ">", "|", "\""}
	for _, c := range invalidChars {
		s = strings.ReplaceAll(s, c, "")
	}

	fields := strings.Fields(s)
	clean := strings.Join(fields, " ")
	clean = strings.Trim(clean, ". ")
	if clean == "" {
		return "Game"
	}
	return clean
}

// Slugify converts a string into a URL/filesystem friendly kebab-case slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	res := strings.Trim(b.String(), "-")
	return res
}

// CleanGameTitle normalizes title strings (removes trademarks, editions, extra punctuation) for fuzzy comparison.
func CleanGameTitle(title string) string {
	t := strings.ToLower(title)
	replacer := strings.NewReplacer(
		"™", "", "®", "", "©", "",
		":", " ", "-", " ", "_", " ",
		".", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", "'", "",
		"\"", "",
	)
	t = replacer.Replace(t)
	words := strings.Fields(t)
	var filtered []string
	for _, w := range words {
		if w == "the" || w == "edition" || w == "collection" || w == "remastered" || w == "goty" || w == "game" || w == "of" || w == "year" {
			continue
		}
		filtered = append(filtered, w)
	}
	return strings.Join(filtered, " ")
}

// IsTitleMatch checks if two game title strings match reasonably.
func IsTitleMatch(candidateName, targetTitle string) bool {
	c := CleanGameTitle(candidateName)
	t := CleanGameTitle(targetTitle)
	if c == "" || t == "" {
		return false
	}
	if c == t {
		return true
	}
	if strings.Contains(c, t) || strings.Contains(t, c) {
		diff := len(c) - len(t)
		if diff < 0 {
			diff = -diff
		}
		if diff <= 8 || (len(c) > 0 && float64(diff)/float64(len(c)) < 0.35) {
			return true
		}
	}
	return false
}
