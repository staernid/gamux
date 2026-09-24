package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/tasks"
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

// NewSetup builds the end-to-end setup pipeline recipe (Scan, Manifest, Normalize, Steamless, GBE).
func NewSetup() *Pipeline {
	return New(
		&ScanStage{},
		&ManifestStage{},
		&NormalizeStage{},
		&SteamlessStage{},
		&GBEStage{},
	)
}

// NewDefault aliases NewSetup for standard game processing.
func NewDefault() *Pipeline {
	return NewSetup()
}

// NewVerify builds the integrity verification pipeline recipe (Scan, Manifest, Verify).
func NewVerify() *Pipeline {
	return New(
		&ScanStage{},
		&ManifestStage{},
		&VerifyStage{},
	)
}

// NewUpdate builds the update pipeline recipe (Scan, Update, Normalize, Steamless, GBE).
func NewUpdate() *Pipeline {
	return New(
		&ScanStage{},
		&UpdateStage{},
		&NormalizeStage{},
		&SteamlessStage{},
		&GBEStage{},
	)
}

// NewRollback builds the rollback pipeline recipe (Scan, RevertDRM).
func NewRollback() *Pipeline {
	return New(
		&ScanStage{},
		&RevertDRMStage{},
	)
}

// NewNormalize builds the directory normalization pipeline recipe (Scan, Normalize).
func NewNormalize() *Pipeline {
	return New(
		&ScanStage{},
		&NormalizeStage{},
	)
}

// ForGame dynamically constructs the optimal setup pipeline stages based on store and platform strategy.
func ForGame(game *domain.Game) *Pipeline {
	if game == nil {
		return NewSetup()
	}

	if game.Store == domain.StoreGOG {
		return NewGOGPipeline()
	}
	if game.Store == domain.StoreStandalone || game.Store == domain.StoreCustom {
		return NewStandalonePipeline()
	}
	return NewSetup()
}

// NewSteamPipeline builds the complete pipeline for Steam titles (Scan, Manifest, Normalize, Steamless, GBE).
func NewSteamPipeline() *Pipeline {
	return NewSetup()
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
	origReporter := pctx.Reporter

	for i, stage := range p.stages {
		if ctx.Err() != nil {
			p.rollback(pctx, executedStages)
			return ctx.Err()
		}

		stageName := stage.Name()
		stepNum := i + 1

		slog.Info("Executing pipeline stage", "step", stepNum, "total", totalStages, "stage", stageName)

		if origReporter != nil {
			origReporter.ReportStep(stepNum, totalStages, stageName, fmt.Sprintf("Running stage: %s", stageName))
			// Wrap reporter so stage sub-progress composites linearly into overall task progress
			pctx.Reporter = &stageReporterWrapper{
				parent:    origReporter,
				step:      stepNum,
				total:     totalStages,
				stageName: stageName,
			}
		}

		err := stage.Execute(ctx, pctx)
		// Restore parent reporter
		pctx.Reporter = origReporter

		if err != nil {
			slog.Warn("Pipeline stage failed", "stage", stageName, "error", err)
			pctx.Results.Errors = append(pctx.Results.Errors, fmt.Sprintf("[%s]: %v", stageName, err))
			p.rollback(pctx, executedStages)
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

func (p *Pipeline) rollback(pctx *Context, stages []RollbackableStage) {
	if len(stages) == 0 {
		return
	}
	// Execute rollback with detached context and safety timeout so external cancellations don't abort cleanup
	rbCtx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 30*time.Second)
	defer cancel()

	for i := len(stages) - 1; i >= 0; i-- {
		stage := stages[i]
		slog.Info("Rolling back pipeline stage", "stage", stage.Name())
		if pctx.Reporter != nil {
			pctx.Reporter.UpdateStage("Rolling back", fmt.Sprintf("Restoring changes from %s...", stage.Name()))
		}
		if err := stage.Rollback(rbCtx, pctx); err != nil {
			slog.Error("Failed to rollback stage", "stage", stage.Name(), "error", err)
			pctx.Results.Errors = append(pctx.Results.Errors, fmt.Sprintf("Rollback [%s]: %v", stage.Name(), err))
		}
	}
}

type stageReporterWrapper struct {
	parent    tasks.ProgressReporter
	step      int
	total     int
	stageName string
}

func (s *stageReporterWrapper) UpdateStage(stage, details string) {
	if s.parent != nil {
		s.parent.UpdateStage(stage, details)
	}
}

func (s *stageReporterWrapper) UpdateProgress(percent float64, currentBytes int64, currentItems, totalItems, totalBytes int64, speed float64, eta int) {
	if s.parent != nil {
		scaledPct := (float64(s.step-1)*100.0 + percent) / float64(s.total)
		s.parent.UpdateProgress(scaledPct, currentBytes, currentItems, totalItems, totalBytes, speed, eta)
	}
}

func (s *stageReporterWrapper) ReportStep(step, total int, stage, details string) {
	if s.parent != nil {
		s.parent.ReportStep(s.step, s.total, stage, details)
	}
}

// -----------------------------------------------------------------------------
// Pipeline Operation (Direct tasks.Operation Implementation)
// -----------------------------------------------------------------------------

// Operation is a direct, executable binding of a Pipeline and Context satisfying tasks.Operation.
type Operation struct {
	Spec     tasks.TaskSpec
	Pipeline *Pipeline
	Context  *Context
}

// TaskSpec returns the specification describing this operation for the supervisor.
func (op *Operation) TaskSpec() tasks.TaskSpec {
	return op.Spec
}

// Execute runs the pipeline recipe within the task supervisor's controller context.
func (op *Operation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	if op.Context == nil {
		return fmt.Errorf("pipeline context cannot be nil")
	}
	if op.Pipeline == nil {
		op.Pipeline = NewDefault()
	}

	op.Context.Reporter = ctrl
	err := op.Pipeline.Execute(ctx, op.Context)
	if err == nil {
		if op.Context.Game != nil {
			ctrl.SetResultGame(op.Context.Game)
		}
		if op.Context.Results.IntegrityReport != nil {
			if raw, jsonErr := json.Marshal(op.Context.Results.IntegrityReport); jsonErr == nil {
				ctrl.SetMetadata("integrity_report", string(raw))
			}
		}
		ctrl.ReportStage("Completed", fmt.Sprintf("%s completed successfully", op.Spec.Title))
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	}
	return err
}

// NewOperation constructs a tasks.Operation binding a Pipeline recipe to an execution Context.
func NewOperation(spec tasks.TaskSpec, pipe *Pipeline, pctx *Context) *Operation {
	return &Operation{
		Spec:     spec,
		Pipeline: pipe,
		Context:  pctx,
	}
}

// NewSetupOperation creates a tasks.Operation for the Setup pipeline recipe.
func NewSetupOperation(gameName, gameDir string, pctx *Context) *Operation {
	title := fmt.Sprintf("Apply Setup (%s)", gameName)
	if gameName == "" {
		title = fmt.Sprintf("Apply Setup (%s)", gameDir)
	}
	return NewOperation(tasks.TaskSpec{
		Type:         tasks.TypeApply,
		Title:        title,
		GameName:     gameName,
		GameDir:      gameDir,
		Capabilities: tasks.CapCancellable,
	}, NewSetup(), pctx)
}

// NewUpdateOperation creates a tasks.Operation for the Update pipeline recipe.
func NewUpdateOperation(gameName, gameDir string, pctx *Context) *Operation {
	return NewOperation(tasks.TaskSpec{
		Type:         tasks.TypeUpdate,
		Title:        fmt.Sprintf("Update %s", gameName),
		GameName:     gameName,
		GameDir:      gameDir,
		Capabilities: tasks.CapCancellable,
	}, NewUpdate(), pctx)
}

// NewVerifyOperation creates a tasks.Operation for the Verify pipeline recipe.
func NewVerifyOperation(gameName, gameDir string, pctx *Context) *Operation {
	return NewOperation(tasks.TaskSpec{
		Type:         tasks.TypeVerify,
		Title:        fmt.Sprintf("Verify Integrity (%s)", gameName),
		GameName:     gameName,
		GameDir:      gameDir,
		Capabilities: tasks.CapCancellable,
	}, NewVerify(), pctx)
}

// NewNormalizeOperation creates a tasks.Operation for the Normalize pipeline recipe.
func NewNormalizeOperation(gameName, gameDir string, pctx *Context) *Operation {
	return NewOperation(tasks.TaskSpec{
		Type:         tasks.TypeNormalize,
		Title:        fmt.Sprintf("Normalize Directory (%s)", gameName),
		GameName:     gameName,
		GameDir:      gameDir,
		Capabilities: tasks.CapCancellable,
	}, NewNormalize(), pctx)
}

// NewRollbackOperation creates a tasks.Operation for the Rollback pipeline recipe.
func NewRollbackOperation(gameName, gameDir string, pctx *Context) *Operation {
	return NewOperation(tasks.TaskSpec{
		Type:         tasks.TypeRollback,
		Title:        fmt.Sprintf("Rollback Game (%s)", gameName),
		GameName:     gameName,
		GameDir:      gameDir,
		Capabilities: tasks.CapCancellable,
	}, NewRollback(), pctx)
}
