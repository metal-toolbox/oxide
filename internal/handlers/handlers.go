package handlers

import (
	"context"
	"log/slog"
	_ "net/http/pprof" // nolint:gosec // pprof path is only exposed over localhost

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/store"
	"github.com/metal-toolbox/bioscfg/internal/tasks"
	rctypes "github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events/controller"
	"github.com/pkg/errors"
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

func (h *HandlerFactory) parseParams(condition *rctypes.Condition) (*rctypes.BiosControlTaskParameters, error) {
	params, err := rctypes.NewBiosControlParametersFromCondition(condition)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse condition parameters")
	}

	slog.Debug("Parsed condition parameters", "params", params)

	return params, nil
}

func (h *HandlerFactory) getAsset(ctx context.Context, params *rctypes.BiosControlTaskParameters) (*model.Asset, error) {
	asset, err := h.repository.AssetByID(ctx, params.AssetID)
	if err != nil {
		// TODO: Check error type
		return nil, controller.ErrRetryHandler
	}

	slog.Debug("Found asset", asset.AsLogFields()...)

	return asset, nil
}

// Handle will handle the received condition
func (h *HandlerFactory) Handle(
	ctx context.Context,
	condition *rctypes.Condition,
	publisher controller.ConditionStatusPublisher,
) error {
	slog.Debug("Handling condition", "condition", condition)

	params, err := h.parseParams(condition)
	if err != nil {
		return err
	}

	asset, err := h.getAsset(ctx, params)
	if err != nil {
		return err
	}

	var task tasks.Task

	switch params.Action {
	case rctypes.ResetSettings:
		task = tasks.NewBiosResetTask(asset)
	default:
		slog.With(asset.AsLogFields()...).Error("Invalid action", "action", params.Action)
		return model.ErrInvalidAction
	}

	runner := tasks.NewTaskRunner(publisher, task)
	client := bmclib.NewClient(asset.BmcAddress.String(), asset.BmcUsername, asset.BmcPassword)

	if err := runner.Run(ctx, client); err != nil {
		slog.Error("Failed running task", "error", err, "task", task.Name())
	}

	return nil
}
