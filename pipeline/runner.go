package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/staernid/gamux/domain"
)

// Pipeline orchestrates sequential execution of composable stages.
type Pipeline struct {
	stages []Stage
}

// New creates a new Pipeline with the specified stages.
func New(stages ...Stage) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

// NewDefault builds the standard dynamic processing pipeline.
func NewDefault() *Pipeline {
	return New(
		&ScanStage{},
		&ManifestStage{},
		&NormalizeStage{},
		&SteamlessStage{},
		&GBEStage{},
	)
}

// ForGame dynamically constructs the optimal setup pipeline stages based on store and platform strategy.
func ForGame(game *domain.Game) *Pipeline {
	if game == nil {
		return NewDefault()
	}

	if game.Store == domain.StoreGOG {
		return NewGOGPipeline()
	}
	if game.Store == domain.StoreStandalone || game.Store == domain.StoreCustom {
		return NewStandalonePipeline()
	}
	return NewDefault()
}

// NewSteamPipeline builds the complete pipeline for Steam titles (Scan, Manifest, Normalize, Steamless, GBE).
func NewSteamPipeline() *Pipeline {
	return NewDefault()
}

// NewGOGPipeline builds the setup pipeline for GOG titles (Scan).
func NewGOGPipeline() *Pipeline {
	return New(
		&ScanStage{},
	)
}

// NewStandalonePipeline builds the setup pipeline for standalone/native titles (Scan).
func NewStandalonePipeline() *Pipeline {
	return New(
		&ScanStage{},
	)
}

// AddStage appends a stage to the pipeline.
func (p *Pipeline) AddStage(stage Stage) {
	if stage != nil {
		p.stages = append(p.stages, stage)
	}
}

// Execute runs all stages in sequence against the pipeline context.
func (p *Pipeline) Execute(ctx context.Context, pctx *Context) error {
	if pctx == nil {
		return fmt.Errorf("pipeline context cannot be nil")
	}

	totalStages := len(p.stages)
	executedStages := make([]RollbackableStage, 0, totalStages)

	for i, stage := range p.stages {
		if ctx.Err() != nil {
			p.rollback(ctx, pctx, executedStages)
			return ctx.Err()
		}

		stageName := stage.Name()
		stepNum := i + 1

		slog.Info("Executing pipeline stage", "step", stepNum, "total", totalStages, "stage", stageName)

		if pctx.Reporter != nil {
			pctx.Reporter.ReportStep(stepNum, totalStages, stageName, fmt.Sprintf("Running stage: %s", stageName))
		}

		err := stage.Execute(ctx, pctx)
		if err != nil {
			slog.Warn("Pipeline stage failed", "stage", stageName, "error", err)
			pctx.Results.Errors = append(pctx.Results.Errors, fmt.Sprintf("[%s]: %v", stageName, err))
			p.rollback(ctx, pctx, executedStages)
			return fmt.Errorf("stage %q failed: %w", stageName, err)
		}

		if rb, ok := stage.(RollbackableStage); ok {
			executedStages = append(executedStages, rb)
		}
	}

	if pctx.Game != nil {
		pctx.Game.RecomputeState()
	}

	return nil
}

func (p *Pipeline) rollback(ctx context.Context, pctx *Context, stages []RollbackableStage) {
	for i := len(stages) - 1; i >= 0; i-- {
		stage := stages[i]
		slog.Info("Rolling back pipeline stage", "stage", stage.Name())
		if err := stage.Rollback(ctx, pctx); err != nil {
			slog.Error("Failed to rollback stage", "stage", stage.Name(), "error", err)
			pctx.Results.Errors = append(pctx.Results.Errors, fmt.Sprintf("Rollback [%s]: %v", stage.Name(), err))
		}
	}
}
