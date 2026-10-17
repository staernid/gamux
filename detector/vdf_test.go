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

func TestParseVDF_URLsAndComments(t *testing.T) {
	vdfContent := `
"Config"
{
	"url"		"https://store.steampowered.com/app/1091500" // Store page URL
	"args"		"//nologo -windowed"
	// Full line comment
	"title"		"Game // Subtitle"
}
`
	node, err := ParseVDF(strings.NewReader(vdfContent))
	if err != nil {
		t.Fatalf("ParseVDF failed: %v", err)
	}

	if url := node.GetString("url"); url != "https://store.steampowered.com/app/1091500" {
		t.Errorf("expected url 'https://store.steampowered.com/app/1091500', got %q", url)
	}
	if args := node.GetString("args"); args != "//nologo -windowed" {
		t.Errorf("expected args '//nologo -windowed', got %q", args)
	}
	if title := node.GetString("title"); title != "Game // Subtitle" {
		t.Errorf("expected title 'Game // Subtitle', got %q", title)
	}
}

func TestParseVDF_EscapedQuotesAndBackslashes(t *testing.T) {
	vdfContent := `
"AppState"
{
	"appid"		"1091500"
	"LaunchOptions"		"\"-vulkan\" -novid \"-dx12\""
	"Description"		"Game with \"quotes\" and \\backslash\\ test"
	"NextKey"		"NextValue"
}
`
	node, err := ParseVDF(strings.NewReader(vdfContent))
	if err != nil {
		t.Fatalf("ParseVDF failed: %v", err)
	}

	if opts := node.GetString("LaunchOptions"); opts != `"-vulkan" -novid "-dx12"` {
		t.Errorf("expected LaunchOptions '\"-vulkan\" -novid \"-dx12\"', got %q", opts)
	}
	if desc := node.GetString("Description"); desc != `Game with "quotes" and \backslash\ test` {
		t.Errorf("expected Description 'Game with \"quotes\" and \\backslash\\ test', got %q", desc)
	}
	if next := node.GetString("NextKey"); next != "NextValue" {
		t.Errorf("expected NextKey 'NextValue', got %q", next)
	}
}

func TestParseVDF_MultilineStrings(t *testing.T) {
	vdfContent := `"AppState"
{
	"appid"		"1091500"
	"Description"		"Line 1
Line 2
Line 3"
	"NextKey"		"ValueAfterMultiline"
}`
	node, err := ParseVDF(strings.NewReader(vdfContent))
	if err != nil {
		t.Fatalf("ParseVDF failed for multiline string: %v", err)
	}

	expectedDesc := "Line 1\nLine 2\nLine 3"
	if desc := node.GetString("Description"); desc != expectedDesc {
		t.Errorf("expected Description %q, got %q", expectedDesc, desc)
	}
	if next := node.GetString("NextKey"); next != "ValueAfterMultiline" {
		t.Errorf("expected NextKey 'ValueAfterMultiline', got %q", next)
	}
}
