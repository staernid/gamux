package multiseat

import (
	"context"
	"os/exec"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/util"
)

// seatDeps abstracts the host-dependent operations used when assembling a seat command
// (binary lookup, prefix provisioning, launch spec generation, display probing) so the
// fallback paths can be exercised without real binaries or a real display.
type seatDeps interface {
	LookPath(name string) (string, error)
	EnsureSeatPrefix(ctx context.Context, cfg *config.Config, game *domain.Game, seatID, customPrefix string) (string, error)
	BuildLaunchSpec(ctx context.Context, cfg *config.Config, opts runner.LaunchOptions) (*runner.LaunchSpec, error)
	DetectPrimaryDisplay() util.DisplayInfo
}

type hostSeatDeps struct{}

func (hostSeatDeps) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (hostSeatDeps) EnsureSeatPrefix(ctx context.Context, cfg *config.Config, game *domain.Game, seatID, customPrefix string) (string, error) {
	return compat.EnsureSeatPrefix(ctx, cfg, game, seatID, customPrefix)
}

func (hostSeatDeps) BuildLaunchSpec(ctx context.Context, cfg *config.Config, opts runner.LaunchOptions) (*runner.LaunchSpec, error) {
	return runner.BuildLaunchSpec(ctx, cfg, opts)
}

func (hostSeatDeps) DetectPrimaryDisplay() util.DisplayInfo { return util.DetectPrimaryDisplay() }

// deps returns the injected dependencies, defaulting to the host implementation.
func (s *Supervisor) deps() seatDeps {
	if s.seatDeps != nil {
		return s.seatDeps
	}
	return hostSeatDeps{}
}

// lookPathOr resolves a binary on PATH and falls back to the bare name when missing.
func lookPathOr(d seatDeps, name string) string {
	if p, err := d.LookPath(name); err == nil {
		return p
	}
	return name
}
