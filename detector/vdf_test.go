package detector

import (
	"strings"
	"testing"
)

func TestParseVDF_NestedACF(t *testing.T) {
	vdfContent := `
"AppState"
{
	"appid"		"1091500"
	"Universe"		"1"
	"name"		"Cyberpunk 2077"
	"installdir"		"Cyberpunk 2077"
	"LastUpdated"		"1708000000"
	"SizeOnDisk"		"75000000000"
	"buildid"		"1337420"
	// User custom configuration
	"InstalledDepots"
	{
		"1091501"
		{
			"manifest"		"8420445566849588826"
			"size"		"60000000000"
		}
		"1091502"
		{
			"manifest"		"3260780708809854852"
			"size"		"15000000000"
		}
	}
	"MountedDepots"
	{
		"1091501"		"8420445566849588826"
		"1091502"		"3260780708809854852"
	}
	"UserConfig"
	{
		"language"		"english"
	}
}
`

	node, err := ParseVDF(strings.NewReader(vdfContent))
	if err != nil {
		t.Fatalf("ParseVDF failed: %v", err)
	}

	if node.Key != "AppState" {
		t.Errorf("expected root key AppState, got %s", node.Key)
	}

	if appid := node.GetString("appid"); appid != "1091500" {
		t.Errorf("expected appid 1091500, got %s", appid)
	}

	if name := node.GetString("name"); name != "Cyberpunk 2077" {
		t.Errorf("expected name 'Cyberpunk 2077', got %s", name)
	}

	if installdir := node.GetString("installdir"); installdir != "Cyberpunk 2077" {
		t.Errorf("expected installdir 'Cyberpunk 2077', got %s", installdir)
	}

	if buildid := node.GetString("buildid"); buildid != "1337420" {
		t.Errorf("expected buildid '1337420', got %s", buildid)
	}

	if size := node.GetInt64("SizeOnDisk"); size != 75000000000 {
		t.Errorf("expected SizeOnDisk 75000000000, got %d", size)
	}

	// Test nested path access
	if lang := node.GetString("UserConfig/language"); lang != "english" {
		t.Errorf("expected language 'english', got %s", lang)
	}

	// Test MountedDepots access
	mounted := node.Get("MountedDepots")
	if mounted == nil {
		t.Fatal("expected MountedDepots section")
	}
	subkeys := mounted.GetSubKeys()
	if len(subkeys) != 2 {
		t.Errorf("expected 2 mounted depots, got %d", len(subkeys))
	}
	if subkeys["1091501"] != "8420445566849588826" {
		t.Errorf("expected manifest 8420445566849588826, got %s", subkeys["1091501"])
	}
}

func FuzzParseVDF(f *testing.F) {
	f.Add([]byte("\"AppState\"\n{\n\t\"appid\"\t\"1091500\"\n\t\"name\"\t\"Cyberpunk\"\n}"))
	f.Add([]byte("\"Key\"\n{\n\t\"Sub\"\n\t{\n\t\t\"Val\"\t\"1\"\n\t}\n}"))
	f.Add([]byte(""))
	f.Add([]byte("   \n\t\r\n"))
	f.Add([]byte("\"Unclosed\" { \"key\" \"val\""))
	f.Add([]byte("// Comment only\n// Another comment"))
	f.Add([]byte("\"Key\" \"ValueNoBraces\""))
	f.Add([]byte("\"Deep\" { \"1\" { \"2\" { \"3\" { \"4\" { \"val\" \"ok\" } } } } }"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Ensure ParseVDF never panics or crashes on arbitrary input
		_, _ = ParseVDF(strings.NewReader(string(data)))
	})
}

