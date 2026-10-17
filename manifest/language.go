package manifest

import (
	"os"
	"strings"
)

var languageAliases = map[string]string{
	"english":                 "english",
	"en":                      "english",
	"french":                  "french",
	"fr":                      "french",
	"fr-fr":                   "french",
	"german":                  "german",
	"de":                      "german",
	"de-de":                   "german",
	"italian":                 "italian",
	"it":                      "italian",
	"it-it":                   "italian",
	"spanish":                 "spanish",
	"es":                      "spanish",
	"es-es":                   "spanish",
	"spanish - spain":         "spanish",
	"spanish - latin america": "latam",
	"latam":                   "latam",
	"japanese":                "japanese",
	"ja":                      "japanese",
	"jp":                      "japanese",
	"ja-jp":                   "japanese",
	"korean":                  "koreana",
	"koreana":                 "koreana",
	"ko":                      "koreana",
	"kr":                      "koreana",
	"ko-kr":                   "koreana",
	"polish":                  "polish",
	"pl":                      "polish",
	"pl-pl":                   "polish",
	"russian":                 "russian",
	"ru":                      "russian",
	"ru-ru":                   "russian",
	"simplified chinese":      "schinese",
	"schinese":                "schinese",
	"zh-cn":                   "schinese",
	"zh-hans":                 "schinese",
	"traditional chinese":     "tchinese",
	"tchinese":                "tchinese",
	"zh-tw":                   "tchinese",
	"zh-hant":                 "tchinese",
	"chinese":                 "schinese",
	"zh":                      "schinese",
	"portuguese":              "portuguese",
	"pt":                      "portuguese",
	"pt-pt":                   "portuguese",
	"portuguese - brazil":     "brazilian",
	"brazilian":               "brazilian",
	"pt-br":                   "brazilian",
	"arabic":                  "arabic",
	"ar":                      "arabic",
	"czech":                   "czech",
	"cs":                      "czech",
	"danish":                  "danish",
	"da":                      "danish",
	"dutch":                   "dutch",
	"nl":                      "dutch",
	"finnish":                 "finnish",
	"fi":                      "finnish",
	"greek":                   "greek",
	"el":                      "greek",
	"hungarian":               "hungarian",
	"hu":                      "hungarian",
	"norwegian":               "norwegian",
	"no":                      "norwegian",
	"swedish":                 "swedish",
	"sv":                      "swedish",
	"thai":                    "thai",
	"th":                      "thai",
	"turkish":                 "turkish",
	"tr":                      "turkish",
	"ukrainian":               "ukrainian",
	"uk":                      "ukrainian",
	"ua":                      "ukrainian",
	"vietnamese":              "vietnamese",
	"vi":                      "vietnamese",
}

// ResolveSystemLanguage inspects environment variables (LC_ALL, LC_MESSAGES, LANG)
// to determine the system's preferred language, defaulting to "english".
func ResolveSystemLanguage() string {
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		val := strings.TrimSpace(os.Getenv(env))
		if val != "" && val != "C" && val != "POSIX" {
			if idx := strings.Index(val, "."); idx != -1 {
				val = val[:idx]
			}
			val = strings.ReplaceAll(val, "_", "-")
			norm := NormalizeLanguageCode(val)
			if norm != "" && norm != "auto" && norm != "all" {
				return norm
			}
		}
	}
	return "english"
}

// NormalizeLanguageCode normalizes common language names and ISO codes to canonical Steam language names.
func NormalizeLanguageCode(lang string) string {
	clean := strings.ToLower(strings.TrimSpace(lang))
	if clean == "" || clean == "auto" || clean == "all" {
		return clean
	}
	if canonical, ok := languageAliases[clean]; ok {
		return canonical
	}
	for alias, canonical := range languageAliases {
		if strings.EqualFold(alias, clean) || strings.HasPrefix(clean, alias) {
			return canonical
		}
	}
	return clean
}

// LanguagesMatch returns true if two language descriptors refer to the same language.
func LanguagesMatch(lang1, lang2 string) bool {
	l1 := NormalizeLanguageCode(lang1)
	l2 := NormalizeLanguageCode(lang2)
	if l1 == "" || l2 == "" || l1 == "all" || l2 == "all" {
		return true
	}
	if l1 == "auto" {
		l1 = ResolveSystemLanguage()
	}
	if l2 == "auto" {
		l2 = ResolveSystemLanguage()
	}
	if l1 == "" || l2 == "" || l1 == "all" || l2 == "all" {
		return true
	}
	if l1 == l2 {
		return true
	}
	// Broad compatibility for Chinese variants and Portuguese variants
	if (l1 == "schinese" || l1 == "tchinese") && (l2 == "schinese" || l2 == "tchinese") {
		return true
	}
	if (l1 == "portuguese" || l1 == "brazilian") && (l2 == "portuguese" || l2 == "brazilian") {
		return true
	}
	return false
}
