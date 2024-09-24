package handlers

import (
	"context"
	"log/slog"
	_ "net/http/pprof" // nolint:gosec // pprof path is only exposed over localhost

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/store"
	"github.com/metal-toolbox/bioscfg/internal/tasks"
	"github.com/metal-toolbox/ctrl"
	rctypes "github.com/metal-toolbox/rivets/condition"
)

// HandlerFactory has the data and business logic for the application
type HandlerFactory struct {
	repository store.Repository
}

// NewHandlerFactory returns a new instance of the Handler
func NewHandlerFactory(repository store.Repository) *HandlerFactory {
	return &HandlerFactory{
		repository: repository,
	}
}

func (h *HandlerFactory) getAsset(ctx context.Context, params *rctypes.BiosControlTaskParameters) (*model.Asset, error) {
	asset, err := h.repository.AssetByID(ctx, params.AssetID)
	if err != nil {
		// TODO: Check error type
		return nil, ctrl.ErrRetryHandler
	}

	slog.Debug("Found asset", asset.AsLogFields()...)

	return asset, nil
}

// Handle will handle the received condition
func (h *HandlerFactory) HandleTask(
	ctx context.Context,
	genTask *rctypes.Task[any, any],
	publisher ctrl.Publisher,
) error {
	slog.Debug("Handling condition", "condition", genTask)

	task, err := tasks.NewTask(genTask)
	if err != nil {
		return err
	}

	server, err := h.getAsset(ctx, task.Parameters)
	if err != nil {
		return err
	}

	var oldTask tasks.Task

	switch task.Parameters.Action {
	case rctypes.ResetSettings:
		oldTask = tasks.NewBiosResetTask(server)
	default:
		slog.With(server.AsLogFields()...).Error("Invalid action", "action", task.Parameters.Action)
		return model.ErrInvalidAction
	}

	runner := tasks.NewTaskRunner(publisher, oldTask, task)
	client := bmclib.NewClient(server.BmcAddress.String(), server.BmcUsername, server.BmcPassword)

	if err := runner.Run(ctx, client); err != nil {
		slog.Error("Failed running task", "error", err, "task", oldTask.Name())
	}

	return nil
}
