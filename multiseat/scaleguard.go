package multiseat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/staernid/gamux/util"
)

// ScaleLockState represents the persistent on-disk record of scales locked by Gamux.
type ScaleLockState struct {
	OriginalScales map[string]float64 `json:"original_scales"` // Output connector name (e.g. "DP-1") -> original scale
	Timestamp      int64              `json:"timestamp"`
}

// ScaleGuard manages temporarily locking fractional display scales to 1.0 (100% native)
// during multiseat Gamescope sessions on KDE Plasma (Wayland) to guarantee 0px seams,
// prevent compositor scaling blur, and cleanly restore desktop scales upon exit.
type ScaleGuard struct {
	mu             sync.Mutex
	kscreenBin     string
	originalScales map[string]float64
	stateFilePath  string
}

// NewScaleGuard initializes a ScaleGuard instance.
func NewScaleGuard() *ScaleGuard {
	bin, _ := exec.LookPath("kscreen-doctor")

	homeDir, err := os.UserHomeDir()
	var statePath string
	if err == nil && homeDir != "" {
		stateDir := filepath.Join(homeDir, ".local", "share", "gamux")
		_ = os.MkdirAll(stateDir, 0755)
		statePath = filepath.Join(stateDir, "scaleguard.json")
	}

	return &ScaleGuard{
		kscreenBin:     bin,
		originalScales: make(map[string]float64),
		stateFilePath:  statePath,
	}
}

// LockDisplays checks the specified display indices. For any display whose scale is not 1.0,
// it records the original scale, persists the lock to disk, and executes kscreen-doctor to set scale to 1.
func (sg *ScaleGuard) LockDisplays(ctx context.Context, displayIndices []int) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if sg.kscreenBin == "" {
		return nil // kscreen-doctor not available (e.g. non-KDE environment)
	}

	displays := util.DetectDisplays()
	if len(displays) == 0 {
		return nil
	}

	// Determine unique display indices
	idxMap := make(map[int]bool)
	for _, idx := range displayIndices {
		if idx >= 0 {
			idxMap[idx] = true
		}
	}
	if len(idxMap) == 0 {
		idxMap[0] = true
	}

	var toLock []string
	var args []string

	for idx := range idxMap {
		var targetDisp *util.DisplayInfo
		for i := range displays {
			if displays[i].Index == idx {
				targetDisp = &displays[i]
				break
			}
		}
		if targetDisp == nil && len(displays) > 0 {
			targetDisp = &displays[0]
		}
		if targetDisp == nil || targetDisp.Name == "" {
			continue
		}

		// Check if scale deviates from 1.0
		if math.Abs(targetDisp.Scale-1.0) > 0.001 {
			name := targetDisp.Name
			if _, already := sg.originalScales[name]; !already {
				sg.originalScales[name] = targetDisp.Scale
			}
			args = append(args, fmt.Sprintf("output.%s.scale.1", name))
			toLock = append(toLock, fmt.Sprintf("%s (was %.2fx)", name, targetDisp.Scale))
		}
	}

	if len(args) == 0 {
		return nil // All participating displays are already 1.0 scale
	}

	// Persist to disk before executing command for crash safety
	sg.persistLockedScales()

	slog.Info("ScaleGuard locking display scale to 100% (1.0) for session", "targets", toLock)
	cmd := exec.CommandContext(ctx, sg.kscreenBin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("ScaleGuard failed to execute kscreen-doctor lock", "error", err, "output", strings.TrimSpace(string(out)))
		return fmt.Errorf("kscreen-doctor lock: %w", err)
	}

	return nil
}

// Restore restores all modified display scales back to their original recorded values.
func (sg *ScaleGuard) Restore(ctx context.Context) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if len(sg.originalScales) == 0 || sg.kscreenBin == "" {
		sg.removeStateFile()
		return nil
	}

	var args []string
	var toRestore []string
	for name, scale := range sg.originalScales {
		scaleStr := strconv.FormatFloat(scale, 'f', -1, 64)
		args = append(args, fmt.Sprintf("output.%s.scale.%s", name, scaleStr))
		toRestore = append(toRestore, fmt.Sprintf("%s -> %s", name, scaleStr))
	}

	slog.Info("ScaleGuard restoring original display scales", "targets", toRestore)
	cmd := exec.CommandContext(ctx, sg.kscreenBin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("ScaleGuard failed to execute kscreen-doctor restore", "error", err, "output", strings.TrimSpace(string(out)))
		return fmt.Errorf("kscreen-doctor restore: %w", err)
	}

	sg.originalScales = make(map[string]float64)
	sg.removeStateFile()
	return nil
}

// RecoverOrphanedLocks checks if a previous session died ungracefully without restoring scales,
// and if so, applies the recorded restore.
func (sg *ScaleGuard) RecoverOrphanedLocks(ctx context.Context) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if sg.stateFilePath == "" || sg.kscreenBin == "" {
		return
	}

	data, err := os.ReadFile(sg.stateFilePath)
	if err != nil {
		return
	}

	var state ScaleLockState
	if err := json.Unmarshal(data, &state); err != nil || len(state.OriginalScales) == 0 {
		_ = os.Remove(sg.stateFilePath)
		return
	}

	var args []string
	var toRestore []string
	for name, scale := range state.OriginalScales {
		scaleStr := strconv.FormatFloat(scale, 'f', -1, 64)
		args = append(args, fmt.Sprintf("output.%s.scale.%s", name, scaleStr))
		toRestore = append(toRestore, fmt.Sprintf("%s -> %s", name, scaleStr))
	}

	if len(args) > 0 {
		slog.Info("ScaleGuard recovering orphaned display scales from previous run", "targets", toRestore)
		cmd := exec.CommandContext(ctx, sg.kscreenBin, args...)
		_ = cmd.Run()
	}

	_ = os.Remove(sg.stateFilePath)
}

func (sg *ScaleGuard) persistLockedScales() {
	if sg.stateFilePath == "" || len(sg.originalScales) == 0 {
		return
	}
	state := ScaleLockState{
		OriginalScales: sg.originalScales,
	}
	data, err := json.Marshal(state)
	if err == nil {
		_ = os.WriteFile(sg.stateFilePath, data, 0644)
	}
}

func (sg *ScaleGuard) removeStateFile() {
	if sg.stateFilePath != "" {
		_ = os.Remove(sg.stateFilePath)
	}
}
