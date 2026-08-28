package engine

import (
	"context"

	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/tools"
)

// WrapContext decorates a context with the shared HTTP client for downstream
// Steam, GitHub, and Manifest API calls.
func (eng *Engine) WrapContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if eng.HTTPClient != nil {
		ctx = steam.WithHTTPClient(ctx, eng.HTTPClient)
		ctx = github.WithHTTPClient(ctx, eng.HTTPClient)
		ctx = manifest.WithHTTPClient(ctx, eng.HTTPClient)
	}
	return ctx
}

// Notify publishes a system notification event to the EventBus and calls
// the Notifier callback if configured.
func (eng *Engine) Notify(ctx context.Context, title, message string) error {
	if eng.EventBus != nil {
		eng.EventBus.Publish(events.NewNotification(title, message))
	}
	if eng.Notifier != nil {
		return eng.Notifier(ctx, title, message)
	}
	return nil
}

// PublishEvent sends a typed event to the event broker.
func (eng *Engine) PublishEvent(ev events.Event) {
	if eng.EventBus != nil {
		eng.EventBus.Publish(ev)
	}
}

// Tools returns the Tool Manager instance.
func (eng *Engine) Tools() *tools.Manager {
	if eng.toolManager == nil {
		eng.toolManager = tools.NewManager(eng.Config)
	}
	return eng.toolManager
}

// GetToolsStatus retrieves current statuses for all registered post-processing tools.
func (eng *Engine) GetToolsStatus(ctx context.Context) ([]tools.ToolStatus, error) {
	return eng.Tools().GetToolsStatus(ctx)
}

// reportStep emits a step progress event for UI/CLI consumers.
func (eng *Engine) reportStep(ctx context.Context, step, total int, title, details string) {
	if eng.EventBus != nil {
		eng.EventBus.Publish(events.NewStepProgress(step, total, title, details))
	}
	if eng.StepReporter != nil {
		eng.StepReporter(ctx, StepProgress{
			Step:    step,
			Total:   total,
			Title:   title,
			Details: details,
		})
	}
}
