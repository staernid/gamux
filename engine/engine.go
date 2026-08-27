package engine

import (
	"github.com/staernid/gamux/config"
)

// Engine provides backward-compatible access to all domain services via embedding.
// Callers can use eng.InspectStatus(), eng.ProcessGame(), eng.SubmitApply() etc.
// directly — methods are promoted from the embedded services.
//
// For new code, prefer injecting individual services (LibraryService, GameService,
// TaskService) directly rather than using Engine as a facade.
type Engine struct {
	*ServiceContext
	*LibraryService
	*GameService
	*TaskService
	*SaveService
}

// New creates a new Engine with all domain services initialised and wired together.
func New(cfg *config.Config) *Engine {
	sc, lib, game, task := NewServices(cfg)

	eng := &Engine{
		ServiceContext: sc,
		LibraryService: lib,
		GameService:    game,
		TaskService:    task,
		SaveService:    sc.Saves,
	}

	eng.registerDefaultTaskRunners()
	return eng
}

// Services groups the three domain services for convenient construction.
type Services struct {
	SC      *ServiceContext
	Library *LibraryService
	Game    *GameService
	Tasks   *TaskService
}

// NewServices creates all domain services wired together and returns them individually.
// Prefer this over New() for callers that want explicit service references.
func NewServices(cfg *config.Config) (*ServiceContext, *LibraryService, *GameService, *TaskService) {
	sc := NewServiceContext(cfg)
	lib := NewLibraryService(sc)
	game := NewGameService(sc, lib)
	task := NewTaskService(sc, lib, game)
	task.registerDefaultTaskRunners()
	return sc, lib, game, task
}
