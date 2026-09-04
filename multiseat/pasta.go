package multiseat

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
)

// isSELinuxEnforcing returns true if the host system is running with SELinux in enforcing mode.
func isSELinuxEnforcing() bool {
	data, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

func copyFilePreserveExec(src, dst string) error {
	srcFi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if dstFi, err := os.Stat(dst); err == nil {
		if dstFi.Size() == srcFi.Size() && !dstFi.ModTime().Before(srcFi.ModTime()) {
			return nil
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// cpuHasAVX2 returns true if the host CPU supports the AVX2 instruction set.
func cpuHasAVX2() bool {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "avx2")
}

// ResolvePastaBinary finds the pasta executable. On SELinux-enforcing systems, it stages
// an unconfined copy in ~/.local/share/gamux/bin to avoid the pasta_t domain transition,
// which prevents AVC denials on GPU render nodes and user log files.
func ResolvePastaBinary() (string, error) {
	sysPasta, err := exec.LookPath("pasta")
	if err != nil {
		return "", fmt.Errorf("pasta executable not found: %w", err)
	}

	if !isSELinuxEnforcing() {
		return sysPasta, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return sysPasta, nil
	}

	stageDir := filepath.Join(homeDir, ".local", "share", "gamux", "bin")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		slog.Warn("Could not create gamux bin directory for pasta staging", "error", err)
		return sysPasta, nil
	}

	stagedPasta := filepath.Join(stageDir, "pasta")
	if err := copyFilePreserveExec(sysPasta, stagedPasta); err != nil {
		slog.Warn("Could not stage unconfined pasta binary", "error", err)
		return sysPasta, nil
	}

	// Also stage companion pasta.avx2 if present in the same directory as pasta
	pastaDir := filepath.Dir(sysPasta)
	sysAvx2 := filepath.Join(pastaDir, "pasta.avx2")
	if util.FileExists(sysAvx2) {
		stagedAvx2 := filepath.Join(stageDir, "pasta.avx2")
		if err := copyFilePreserveExec(sysAvx2, stagedAvx2); err == nil {
			if cpuHasAVX2() {
				// Directly return unconfined pasta.avx2 on AVX2 systems so it runs in unconfined_t
				// without invoking /usr/bin/pasta.avx2 or transitioning to pasta_t domain.
				return stagedAvx2, nil
			}
		}
	}

	return stagedPasta, nil
}

// PastaOptions holds network namespace options.
type PastaOptions struct {
	EnableBroadcastRelay bool
	BroadcastPort        int      // Default 47584 (Goldberg Steamworks LAN discovery)
	ForwardTCPPorts      []int    // Host ports to forward into namespace
	ForwardUDPPorts      []int    // Host UDP ports to forward into namespace
	Command              []string // Inner command to wrap
}

// BuildPastaArgs wraps an execution command inside a pasta network namespace.
func BuildPastaArgs(opts PastaOptions) ([]string, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("command is required")
	}

	args := []string{
		"--config-net", // Automatically configure IP, route, and nameserver in namespace
		"--quiet",      // Suppress verbose informational output
	}

	// Forward specific TCP/UDP ports if configured
	for _, port := range opts.ForwardTCPPorts {
		if port > 0 && port < 65536 {
			args = append(args, "-t", fmt.Sprintf("%d", port))
		}
	}
	for _, port := range opts.ForwardUDPPorts {
		if port > 0 && port < 65536 {
			args = append(args, "-u", fmt.Sprintf("%d", port))
		}
	}
	if opts.EnableBroadcastRelay && opts.BroadcastPort > 0 && opts.BroadcastPort < 65536 {
		args = append(args, "-u", fmt.Sprintf("%d", opts.BroadcastPort))
	}

	// Command terminator
	args = append(args, "--")
	args = append(args, opts.Command...)

	return args, nil
}


