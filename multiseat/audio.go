package multiseat

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// VirtualSink represents a created PipeWire/PulseAudio null sink module.
type VirtualSink struct {
	SinkName    string `json:"sink_name"`    // e.g. "gamux-p1"
	Description string `json:"description"`  // e.g. "Gamux Player 1 Sink"
	ModuleID    string `json:"module_id"`    // pactl module ID for unloading
	LoopbackID  string `json:"loopback_id"`  // optional loopback module ID
}

// CreateVirtualSink creates a dedicated null sink using pactl.
func CreateVirtualSink(ctx context.Context, sinkName, description string) (*VirtualSink, error) {
	if sinkName == "" {
		return nil, fmt.Errorf("sink name cannot be empty")
	}
	if description == "" {
		description = fmt.Sprintf("Gamux Audio Sink (%s)", sinkName)
	}

	cmd := exec.CommandContext(ctx, "pactl", "load-module", "module-null-sink",
		fmt.Sprintf("sink_name=%s", sinkName),
		fmt.Sprintf("sink_properties=device.description=\"%s\"", description),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("create virtual sink %s: %w (%s)", sinkName, err, strings.TrimSpace(string(out)))
	}

	moduleID := strings.TrimSpace(string(out))
	return &VirtualSink{
		SinkName:    sinkName,
		Description: description,
		ModuleID:    moduleID,
	}, nil
}

// RouteSinkToPhysical routes audio from the virtual sink monitor to a physical sink via module-loopback.
func RouteSinkToPhysical(ctx context.Context, vs *VirtualSink, physicalSink string) error {
	if vs == nil || physicalSink == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "pactl", "load-module", "module-loopback",
		fmt.Sprintf("source=%s.monitor", vs.SinkName),
		fmt.Sprintf("sink=%s", physicalSink),
		"latency_msec=20",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("route sink %s to %s: %w (%s)", vs.SinkName, physicalSink, err, strings.TrimSpace(string(out)))
	}

	vs.LoopbackID = strings.TrimSpace(string(out))
	return nil
}

// Destroy unloads the created virtual sink and loopback modules from PipeWire.
func (vs *VirtualSink) Destroy(ctx context.Context) {
	if vs == nil {
		return
	}

	if vs.LoopbackID != "" {
		_ = exec.CommandContext(ctx, "pactl", "unload-module", vs.LoopbackID).Run()
		vs.LoopbackID = ""
	}

	if vs.ModuleID != "" {
		_ = exec.CommandContext(ctx, "pactl", "unload-module", vs.ModuleID).Run()
		vs.ModuleID = ""
	}
}

// EnumeratePhysicalSinks returns a list of available physical audio output sinks on the host.
func EnumeratePhysicalSinks(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "pactl", "list", "short", "sinks")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list audio sinks: %w", err)
	}

	var sinks []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name := fields[1]
			// Exclude gamux null sinks from physical list
			if !strings.HasPrefix(name, "gamux-") {
				sinks = append(sinks, name)
			}
		}
	}

	return sinks, nil
}
