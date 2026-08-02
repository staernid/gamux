// Package steamshortcut adds non-Steam game shortcuts to Steam's shortcuts
// configuration. It writes both binary VDF entries (primary) and .desktop
// files (fallback) so the shortcut appears in Steam regardless of version.
package steamshortcut

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gamux/config"
	"gamux/steam"
)

// ShortcutConfig holds parameters for a Steam non-Steam game shortcut.
type ShortcutConfig struct {
	Name      string // Display name in Steam
	ExePath   string // Absolute path to the executable
	AppID     string // AppID (used for grid images, etc.)
	LaunchOpt string // Optional: launch options / arguments
	StartDir  string // Optional: working directory
	IconPath  string // Optional: path to a PNG icon
}

// ── Binary VDF helpers ───────────────────────────────────────────────

// VDF types as used by Steam's binary shortcuts.vdf.
const (
	vdfTypeNull    = 0
	vdfTypeString  = 1
	vdfTypeInt32   = 2
	vdfTypeInt64   = 3
	vdfTypeColor   = 4
	vdfTypePtr     = 6
	vdfNodeTypeObj = 1 // used as value type to indicate nested object
	vdfNodeTypeArr = 2 // used as value type to indicate nested array
)

// vdfValue represents a single VDF value.
type vdfValue struct {
	Type  int
	Value string // used for strings and hex-encoded bytes
	Int64 int64  // used for int32/int64
}

// vdfNode represents a KV entry in the VDF tree.
type vdfNode struct {
	Key      string
	NodeType int // 1=object, 2=array, 3=string, 4=int32, 5=int64, 6=color, 7=ptr
	Children []vdfChild
}

// vdfChild is a key-value pair inside an object/array node.
type vdfChild struct {
	Key   string
	Value vdfValue
	Child *vdfNode
}

// magicHeader is the 4-byte magic at the start of Steam's binary VDF.
var magicHeader = []byte{0x00, 0x02, 0x04, 0x00}

// readVDF reads a binary VDF file and returns the root node.
func readVDF(path string) (*vdfNode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read VDF file: %w", err)
	}

	if len(data) < 4 || string(data[:4]) != string(magicHeader) {
		return nil, fmt.Errorf("invalid VDF magic header")
	}

	reader := &vdfReader{r: data[4:]}
	root, err := reader.readNode()
	if err != nil {
		return nil, fmt.Errorf("parse VDF: %w", err)
	}
	return root, nil
}

type vdfReader struct {
	r []byte
	i int
}

func (r *vdfReader) readBytes(n int) ([]byte, error) {
	if r.i+n > len(r.r) {
		return nil, fmt.Errorf("unexpected EOF reading %d bytes", n)
	}
	data := r.r[r.i : r.i+n]
	r.i += n
	return data, nil
}

func (r *vdfReader) readString() (string, error) {
	lenBytes, err := r.readBytes(4)
	if err != nil {
		return "", err
	}
	strLen := int(binary.LittleEndian.Uint32(lenBytes))
	if strLen < 0 {
		return "", fmt.Errorf("negative string length: %d", strLen)
	}
	if r.i+strLen > len(r.r) {
		return "", fmt.Errorf("unexpected EOF reading string of length %d", strLen)
	}
	s := string(r.r[r.i : r.i+strLen])
	r.i += strLen
	// Skip trailing null byte if present
	if r.i < len(r.r) && r.r[r.i] == 0 {
		r.i++
	}
	return s, nil
}

func (r *vdfReader) readValue() (vdfValue, error) {
	typeBytes, err := r.readBytes(4)
	if err != nil {
		return vdfValue{}, err
	}
	vt := int(binary.LittleEndian.Uint32(typeBytes))

	switch vt {
	case vdfTypeNull:
		return vdfValue{Type: vdfTypeNull}, nil
	case vdfTypeString:
		s, err := r.readString()
		if err != nil {
			return vdfValue{}, err
		}
		return vdfValue{Type: vdfTypeString, Value: s}, nil
	case vdfTypeInt32:
		b, err := r.readBytes(4)
		if err != nil {
			return vdfValue{}, err
		}
		return vdfValue{Type: vdfTypeInt32, Int64: int64(binary.LittleEndian.Uint32(b))}, nil
	case vdfTypeInt64:
		b, err := r.readBytes(8)
		if err != nil {
			return vdfValue{}, err
		}
		return vdfValue{Type: vdfTypeInt64, Int64: int64(binary.LittleEndian.Uint64(b))}, nil
	case vdfTypeColor:
		b, err := r.readBytes(4)
		if err != nil {
			return vdfValue{}, err
		}
		return vdfValue{Type: vdfTypeColor, Value: fmt.Sprintf("%08x", binary.LittleEndian.Uint32(b))}, nil
	case vdfTypePtr:
		b, err := r.readBytes(8)
		if err != nil {
			return vdfValue{}, err
		}
		return vdfValue{Type: vdfTypePtr, Int64: int64(binary.LittleEndian.Uint64(b))}, nil
	default:
		return vdfValue{}, fmt.Errorf("unknown VDF type: %d", vt)
	}
}

func (r *vdfReader) readNode() (*vdfNode, error) {
	key, err := r.readString()
	if err != nil {
		return nil, err
	}

	typeBytes, err := r.readBytes(4)
	if err != nil {
		return nil, err
	}
	nt := int(binary.LittleEndian.Uint32(typeBytes))

	node := &vdfNode{Key: key, NodeType: nt}

	if nt == 1 || nt == 2 { // object or array
		node.Children = []vdfChild{}
		for {
			// Read key
			lenBytes, err := r.readBytes(4)
			if err != nil {
				if err.Error() == "unexpected EOF reading 4 bytes" {
					break
				}
				return nil, err
			}
			keyLen := int(binary.LittleEndian.Uint32(lenBytes))
			if keyLen == 0xFFFFFFFF {
				// End of array/object marker
				break
			}
			keyBytes, err := r.readBytes(keyLen)
			if err != nil {
				return nil, err
			}
			childKey := string(keyBytes)

			// Read value
			val, err := r.readValue()
			if err != nil {
				return nil, err
			}

			child := vdfChild{Key: childKey, Value: val}

			// If value type indicates a nested node, read the child node
			if val.Type == vdfNodeTypeObj || val.Type == vdfNodeTypeArr {
				childNode, err := r.readNode()
				if err != nil {
					return nil, err
				}
				child.Child = childNode
			}

			node.Children = append(node.Children, child)
		}
	}

	return node, nil
}

// ── Public API ──────────────────────────────────────────────────────

// SteamUserdataDir returns the first non-empty Steam userdata directory
// (~/.local/share/Steam/userdata/<numeric_id>/), or an error.
func SteamUserdataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("steamshortcut: get home dir: %w", err)
	}

	base := filepath.Join(home, config.SteamUserdata)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		// Also try the flatpak path
		flatpakBase := filepath.Join(home, "var", "lib", "steam", "userdata")
		if _, err := os.Stat(flatpakBase); os.IsNotExist(err) {
			return "", fmt.Errorf("steamshortcut: Steam userdata not found at %s or %s", base, flatpakBase)
		}
		base = flatpakBase
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("steamshortcut: read Steam userdata dir: %w", err)
	}

	var numericDirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err == nil {
			numericDirs = append(numericDirs, filepath.Join(base, e.Name()))
		}
	}

	if len(numericDirs) == 0 {
		return "", fmt.Errorf("steamshortcut: no numeric Steam user directories found")
	}

	// Return the most recently modified directory
	best := numericDirs[0]
	bestMod, _ := os.Stat(best)
	for _, d := range numericDirs[1:] {
		info, err := os.Stat(d)
		if err != nil {
			continue
		}
		if info.ModTime().After(bestMod.ModTime()) {
			best = d
			bestMod = info
		}
	}

	return best, nil
}

// AddShortcut adds a non-Steam game shortcut to Steam's shortcuts.vdf.
// It finds the Steam userdata directory, locates shortcuts.vdf, and
// appends an entry. Returns an error if Steam userdata can't be found.
// Accepts a dryRun parameter.
func AddShortcut(cfg ShortcutConfig, dryRun bool) error {
	if cfg.Name == "" {
		return fmt.Errorf("steamshortcut: Name is required")
	}
	if cfg.ExePath == "" {
		return fmt.Errorf("steamshortcut: ExePath is required")
	}

	if dryRun {
		slog.Info("[DRY RUN] Would add Steam shortcut", "name", cfg.Name, "exe", cfg.ExePath)
		slog.Info("[DRY RUN] Would also create .desktop entry as fallback")
		return nil
	}

	// Resolve to absolute path
	exePath, err := filepath.Abs(cfg.ExePath)
	if err != nil {
		return fmt.Errorf("steamshortcut: resolve exe path: %w", err)
	}
	cfg.ExePath = exePath

	startDir := cfg.StartDir
	if startDir == "" {
		startDir = filepath.Dir(exePath)
	}

	// Try binary VDF first
	vdfErr := tryWriteVDF(cfg, exePath, startDir)
	if vdfErr == nil {
		slog.Info("Added Steam shortcut (VDF)", "name", cfg.Name, "exe", exePath)
	} else {
		slog.Warn("VDF write failed, falling back to .desktop entry", "error", vdfErr)
	}

	// Always create .desktop entry as a reliable fallback
	desktopPath, err := DesktopEntry(cfg, false)
	if err != nil {
		return fmt.Errorf("steamshortcut: write .desktop entry: %w", err)
	}
	slog.Info("Wrote .desktop entry", "path", desktopPath)

	return nil
}

// tryWriteVDF attempts to write the shortcut to shortcuts.vdf in binary format.
func tryWriteVDF(cfg ShortcutConfig, exePath, startDir string) error {
	steamDir, err := SteamUserdataDir()
	if err != nil {
		return fmt.Errorf("steamshortcut: find userdata: %w", err)
	}

	vdfPath := filepath.Join(steamDir, "config", "shortcuts.vdf")
	if _, err := os.Stat(vdfPath); os.IsNotExist(err) {
		// Try alternate location
		vdfPath = filepath.Join(steamDir, "shortcuts.vdf")
		if _, err := os.Stat(vdfPath); os.IsNotExist(err) {
			return fmt.Errorf("shortcuts.vdf not found in Steam userdata")
		}
	}

	// Read existing shortcuts
	root, err := readVDF(vdfPath)
	if err != nil {
		return fmt.Errorf("read shortcuts.vdf: %w", err)
	}

	// Find or create the "shortcuts" root node
	var shortcutsNode *vdfNode
	for _, child := range root.Children {
		if child.Child != nil && child.Child.Key == "shortcuts" {
			shortcutsNode = child.Child
			break
		}
	}
	if shortcutsNode == nil {
		return fmt.Errorf("shortcuts.vdf has unexpected structure")
	}

	// Find the next available AppID index
	nextID := 0
	for _, child := range shortcutsNode.Children {
		if id, err := strconv.Atoi(child.Key); err == nil && id >= nextID {
			nextID = id + 1
		}
	}

	// Create new shortcut node
	sc := vdfChild{
		Key: strconv.Itoa(nextID),
		Child: &vdfNode{
			NodeType: 1, // object
			Children: []vdfChild{
				{Key: "AppID", Value: vdfValue{Type: vdfTypeInt32, Int64: 0}},
				{Key: "Exe", Value: vdfValue{Type: vdfTypeString, Value: exePath}},
				{Key: "StartDir", Value: vdfValue{Type: vdfTypeString, Value: startDir}},
				{Key: "LaunchOptions", Value: vdfValue{Type: vdfTypeString, Value: cfg.LaunchOpt}},
				{Key: "IsHidden", Value: vdfValue{Type: vdfTypeInt32, Int64: 0}},
				{Key: "IsFavorite", Value: vdfValue{Type: vdfTypeInt32, Int64: 0}},
				{Key: "Name", Value: vdfValue{Type: vdfTypeString, Value: cfg.Name}},
			},
		},
	}

	shortcutsNode.Children = append(shortcutsNode.Children, sc)

	// Write back
	if err := writeVDF(vdfPath, root); err != nil {
		return fmt.Errorf("write shortcuts.vdf: %w", err)
	}

	slog.Info("Wrote shortcut to shortcuts.vdf", "name", cfg.Name, "index", nextID)
	return nil
}

// writeVDF serializes a VDF tree back to binary format.
func writeVDF(path string, root *vdfNode) error {
	var buf []byte
	buf = append(buf, magicHeader...)
	buf = writeNode(buf, root)

	return os.WriteFile(path, buf, 0644)
}

func writeNode(buf []byte, n *vdfNode) []byte {
	// Write key
	buf = writeString(buf, n.Key)

	// Write node type
	buf = writeInt32(buf, uint32(n.NodeType))

	if n.NodeType == 1 || n.NodeType == 2 { // object or array
		for _, child := range n.Children {
			// Write child key
			buf = writeString(buf, child.Key)

			if child.Child != nil {
				// Value is a nested node — write the node type as value type
				buf = writeInt32(buf, uint32(child.Child.NodeType))
				buf = writeNode(buf, child.Child)
			} else {
				// Write the value
				buf = writeValue(buf, child.Value)
			}
		}
		// Write end-of-array/object marker (0xFFFFFFFF key length)
		buf = writeInt32(buf, 0xFFFFFFFF)
	}

	return buf
}

func writeString(buf []byte, s string) []byte {
	buf = writeInt32(buf, uint32(len(s)))
	buf = append(buf, []byte(s)...)
	buf = append(buf, 0) // null terminator
	return buf
}

func writeInt32(buf []byte, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(buf, b...)
}

func writeInt64(buf []byte, v int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	return append(buf, b...)
}

func writeValue(buf []byte, v vdfValue) []byte {
	buf = writeInt32(buf, uint32(v.Type))
	switch v.Type {
	case vdfTypeString:
		buf = writeString(buf, v.Value)
	case vdfTypeInt32, vdfTypeInt64:
		buf = writeInt64(buf, v.Int64)
	case vdfTypeColor:
		var n uint32
		fmt.Sscanf(v.Value, "%x", &n)
		buf = writeInt32(buf, n)
	case vdfTypePtr:
		buf = writeInt64(buf, v.Int64)
	}
	return buf
}

// DesktopEntry writes a .desktop file to ~/.local/share/applications/
// that launches the game via Steam's protocol handler.
// This is a fallback for when direct VDF manipulation isn't available.
// Returns the file path.
func DesktopEntry(cfg ShortcutConfig, dryRun bool) (string, error) {
	appID := "0"
	if cfg.AppID != "" {
		appID = cfg.AppID
	}

	// Generate a safe filename from the name
	slug := strings.ToLower(cfg.Name)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "game"
	}

	desktopName := "steam-" + slug + ".desktop"
	desktopPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications", desktopName)

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Name=%s
Type=Application
Exec=steam://rungameid/%s
Icon=steam
Terminal=false
Categories=Game;
`, cfg.Name, appID)

	if dryRun {
		slog.Info("[DRY RUN] Would write .desktop entry", "path", desktopPath)
		return desktopPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(desktopPath), 0755); err != nil {
		return "", fmt.Errorf("steamshortcut: create applications dir: %w", err)
	}

	if err := os.WriteFile(desktopPath, []byte(desktopContent), 0644); err != nil {
		return "", fmt.Errorf("steamshortcut: write .desktop: %w", err)
	}

	return desktopPath, nil
}

// FetchSteamGridArtwork downloads grid images for an AppID into Steam's grid folder.
func FetchSteamGridArtwork(ctx context.Context, appID string, dryRun bool) error {
	if appID == "" || appID == "0" {
		return nil
	}

	steamDir, err := SteamUserdataDir()
	if err != nil {
		return fmt.Errorf("steamshortcut: find userdata for grid art: %w", err)
	}

	gridDir := filepath.Join(steamDir, "config", "grid")
	coverPath := filepath.Join(gridDir, appID+"p.jpg")
	bannerPath := filepath.Join(gridDir, appID+".jpg")

	coverURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)
	bannerURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", appID)

	if dryRun {
		slog.Info("[DRY RUN] Would fetch Steam grid cover art", "url", coverURL, "target", coverPath)
		slog.Info("[DRY RUN] Would fetch Steam grid banner image", "url", bannerURL, "target", bannerPath)
		return nil
	}

	if err := steam.DownloadFile(ctx, coverURL, coverPath); err == nil {
		slog.Info("Fetched Steam grid cover art", "path", coverPath)
	} else {
		_ = steam.DownloadFile(ctx, bannerURL, coverPath)
	}

	if err := steam.DownloadFile(ctx, bannerURL, bannerPath); err == nil {
		slog.Info("Fetched Steam grid banner image", "path", bannerPath)
	}

	return nil
}

// RegisterShortcut is a convenience function that tries VDF first, then
// falls back to .desktop entry. It also fetches Steam grid artwork if an AppID is provided
// and attempts to run `update-desktop-database` so the entry appears immediately in Steam.
func RegisterShortcut(ctx context.Context, cfg ShortcutConfig, dryRun bool) error {
	if err := AddShortcut(cfg, dryRun); err != nil {
		return err
	}

	if cfg.AppID != "" && cfg.AppID != "0" {
		if err := FetchSteamGridArtwork(ctx, cfg.AppID, dryRun); err != nil {
			slog.Warn("Failed to fetch Steam grid artwork", "appID", cfg.AppID, "error", err)
		}
	}

	if !dryRun {
		cmd := exec.Command("update-desktop-database")
		if err := cmd.Run(); err != nil {
			slog.Warn("update-desktop-database failed (non-fatal)", "error", err)
		}
	}

	return nil
}
