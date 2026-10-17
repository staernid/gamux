package manifest

import "testing"

func files(paths ...string) []ManifestFileEntry {
	out := make([]ManifestFileEntry, 0, len(paths))
	for _, p := range paths {
		out = append(out, ManifestFileEntry{Path: p})
	}
	return out
}

func TestClassifyDepotContent_Rules(t *testing.T) {
	dirEntry := ManifestFileEntry{Path: "bin/", Flags: 0x40}

	tests := []struct {
		name      string
		depotName string
		files     []ManifestFileEntry
		meta      *DepotMeta
		wantCat   string
		wantLang  string
	}{
		{"empty input is core", "", nil, nil, ContentCategoryCore, ""},
		{"executable beats soundtrack metadata", "", files("game.exe"), &DepotMeta{Name: "Game Soundtrack"}, ContentCategoryCore, ""},
		{"executable beats language name", "French Pack", files("bin/game"), nil, ContentCategoryCore, ""},
		{"directory entries are ignored for exe check", "", append([]ManifestFileEntry{dirEntry}, files("readme.txt")...), nil, ContentCategoryCore, ""},
		{"meta soundtrack", "", nil, &DepotMeta{Name: "Game Soundtrack"}, ContentCategorySoundtrack, ""},
		{"meta ost suffix", "", nil, &DepotMeta{Name: "Game OST"}, ContentCategorySoundtrack, ""},
		{"meta hd name", "", nil, &DepotMeta{Name: "4K Texture Pack"}, ContentCategoryOptionalHD, ""},
		{"meta optional language", "", files("data/a.dat"), &DepotMeta{Name: "Depot", Language: "french", Optional: true}, ContentCategoryLocalePrefix + "french", "french"},
		{"meta language voice name", "", files("data/a.dat"), &DepotMeta{Name: "German Voice", Language: "german"}, ContentCategoryLocalePrefix + "german", "german"},
		{"meta language without files", "", nil, &DepotMeta{Name: "Depot", Language: "german"}, ContentCategoryLocalePrefix + "german", "german"},
		{"meta base language stays core", "", files("data/a.dat"), &DepotMeta{Name: "Base", Language: "english"}, ContentCategoryCore, ""},
		{"meta dlc flag", "", files("data/a.dat"), &DepotMeta{Name: "Extra", IsDLC: true}, ContentCategoryDLC, ""},
		{"meta dlc app id", "", files("data/a.dat"), &DepotMeta{Name: "Extra", DLCAppID: 42}, ContentCategoryDLC, ""},
		{"name soundtrack", "Official Soundtrack", nil, nil, ContentCategorySoundtrack, ""},
		{"name hd", "High Resolution Pack", nil, nil, ContentCategoryOptionalHD, ""},
		{"name dlc", "DLC - Extra Levels", nil, nil, ContentCategoryDLC, ""},
		{"name language keyword", "Spanish Voices", nil, nil, ContentCategoryLocalePrefix + "spanish", "spanish"},
		{"meta name overrides depot name", "French Audio", nil, &DepotMeta{Name: "Soundtrack"}, ContentCategorySoundtrack, ""},
		{"files media only", "Depot 1", files("a.mp3", "b.flac"), nil, ContentCategorySoundtrack, ""},
		{"files mostly hd", "Depot 1", files("tex/4k_a.dat", "tex/4k_b.dat", "tex/4k_c.dat", "readme.txt"), nil, ContentCategoryOptionalHD, ""},
		{"files hd below threshold", "Depot 1", files("tex/4k_a.dat", "a.dat", "b.dat", "c.dat"), nil, ContentCategoryCore, ""},
		{"files dominant language", "Depot 1", files("data/german/a.dat", "data/german/b.dat", "data/german/c.dat", "readme.txt"), nil, ContentCategoryLocalePrefix + "german", "german"},
		{"files single language hit is core", "Depot 1", files("data/german/a.dat", "b.dat", "c.dat", "d.dat"), nil, ContentCategoryCore, ""},
		{"files short code language", "Depot 1", files("loc/fr-fr/a.dat", "loc/fr-fr/b.dat"), nil, ContentCategoryLocalePrefix + "french", "french"},
		{"plain files are core", "Depot 1", files("data/a.dat", "data/b.dat"), nil, ContentCategoryCore, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, lang := ClassifyDepotContent(1, tt.depotName, tt.files, tt.meta)
			if cat != tt.wantCat || lang != tt.wantLang {
				t.Errorf("got (%q, %q), want (%q, %q)", cat, lang, tt.wantCat, tt.wantLang)
			}
		})
	}
}
