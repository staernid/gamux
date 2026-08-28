package tasks

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/staernid/gamux/domain"
)

// Controller coordinates execution lifecycle, telemetry reporting, pause synchronization, and rate limiting.
type Controller struct {
	ctx          context.Context
	taskID       string
	caps         Capability
	limiter      Limiter
	isPaused     atomic.Bool
	pauseCond    *sync.Cond
	mu           sync.Mutex
	resultGame   *domain.Game
	onProgress   func(currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64)
	onStage      func(stage, details string)
	onStep       func(step, total int, stage, details string)
}

// NewController initializes a Controller for a task execution.
func NewController(
	ctx context.Context,
	taskID string,
	caps Capability,
	limiter Limiter,
	onProgress func(currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64),
	onStage func(stage, details string),
	onStep func(step, total int, stage, details string),
) *Controller {
	c := &Controller{
		ctx:        ctx,
		taskID:     taskID,
		caps:       caps,
		limiter:    limiter,
		onProgress: onProgress,
		onStage:    onStage,
		onStep:     onStep,
	}
	c.pauseCond = sync.NewCond(&c.mu)
	return c
}

// Context returns the execution context.
func (c *Controller) Context() context.Context {
	return c.ctx
}

// TaskID returns the ID of the running task.
func (c *Controller) TaskID() string {
	return c.taskID
}

// SetResultGame assigns the mutated Game entity upon successful operation completion.
func (c *Controller) SetResultGame(g *domain.Game) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resultGame = g
}

// ResultGame retrieves the mutated Game entity if set.
func (c *Controller) ResultGame() *domain.Game {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resultGame
}

// IsPaused returns true if the controller is in a paused state.
func (c *Controller) IsPaused() bool {
	return c.isPaused.Load()
}

// SetPaused toggles pause state and unblocks parked workers if unpaused.
func (c *Controller) SetPaused(paused bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isPaused.Store(paused)
	if !paused {
		c.pauseCond.Broadcast()
	}
}

// CheckPauseOrCancel blocks cooperatively if paused until resumed, or returns context error if cancelled.
func (c *Controller) CheckPauseOrCancel() error {
	if err := c.ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	for c.isPaused.Load() {
		c.pauseCond.Wait()
		if err := c.ctx.Err(); err != nil {
			c.mu.Unlock()
			return err
		}
	}
	c.mu.Unlock()

	return c.ctx.Err()
}

// ReportProgress emits numeric and byte progress telemetry.
func (c *Controller) ReportProgress(currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64) {
	if c.onProgress != nil {
		c.onProgress(currBytes, totalBytes, currItems, totalItems, speed, eta, pct)
	}
}

// ReportStage updates current execution stage and textual details.
func (c *Controller) ReportStage(stage, details string) {
	if c.onStage != nil {
		c.onStage(stage, details)
	}
}

// ReportStep updates discrete step progress.
func (c *Controller) ReportStep(step, total int, stage, details string) {
	if c.onStep != nil {
		c.onStep(step, total, stage, details)
	}
}

// Throttle enforces bandwidth rate limiting if configured.
func (c *Controller) Throttle(bytes int) error {
	if c.limiter == nil || !c.caps.Has(CapBandwidthLimit) {
		return nil
	}
	return c.limiter.WaitN(c.ctx, bytes)
}

// ProgressReporter adapter implementation for compatibility with pipeline stages.
func (c *Controller) UpdateStage(stage, details string) {
	c.ReportStage(stage, details)
}

func (c *Controller) UpdateProgress(percent float64, currentBytes int64, currentItems, totalItems, totalBytes int64, speed float64, eta int) {
	c.ReportProgress(currentBytes, totalBytes, int(currentItems), int(totalItems), speed, eta, percent)
}

func (c *Controller) UpdateStep(step, total int, stage, details string) {
	c.ReportStep(step, total, stage, details)
}
