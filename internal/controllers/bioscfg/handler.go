package bioscfg

import (
	"context"
	"time"

	"github.com/metal-toolbox/bioscfg/internal/config"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/store/bmc"
	"github.com/metal-toolbox/bioscfg/internal/store/fleetdb"
	"github.com/metal-toolbox/ctrl"
	rctypes "github.com/metal-toolbox/rivets/condition"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type TaskHandler struct {
	logger       *logrus.Entry
	cfg          *config.Configuration
	fleetdb      *fleetdb.Store
	bmcClient    bmc.BMC
	publisher    ctrl.Publisher
	server       *model.Asset
	task         *Task
	startTS      time.Time
	controllerID string
}

func (th *TaskHandler) HandleTask(ctx context.Context, genTask *rctypes.Task[any, any], publisher ctrl.Publisher) error {
	ctx, span := otel.Tracer(pkgName).Start(
		ctx,
		"bioscfg.HandleTask",
	)
	defer span.End()

	var err error
	th.publisher = publisher

	// Ungeneric the task
	th.task, err = NewTask(genTask)
	if err != nil {
		th.logger.WithFields(logrus.Fields{
			"conditionID":  genTask.ID,
			"controllerID": th.controllerID,
			"err":          err.Error(),
		}).Error("asset lookup error")
		return err
	}

	// Get Server
	th.server, err = th.fleetdb.AssetByID(ctx, th.task.Parameters.AssetID)
	if err != nil {
		th.logger.WithFields(logrus.Fields{
			"assetID":      th.task.Parameters.AssetID.String(),
			"conditionID":  th.task.ID,
			"controllerID": th.controllerID,
			"err":          err.Error(),
		}).Error("asset lookup error")

		return ctrl.ErrRetryHandler
	}

	// New log entry for this condition
	th.logger = th.logger.WithFields(
		logrus.Fields{
			"controllerID": th.controllerID,
			"conditionID":  th.task.ID.String(),
			"serverID":     th.server.ID.String(),
			"bmc":          th.server.BmcAddress.String(),
			"action":       th.task.Parameters.Action,
		},
	)

	// Get BMC Client
	if th.cfg.Dryrun { // Fake BMC
		th.bmcClient = bmc.NewDryRunBMCClient(th.server)
		th.logger.Warn("Running BMC in Dryrun mode")
	} else {
		th.bmcClient = bmc.NewBMCClient(th.server, th.logger)
	}

	err = th.bmcClient.Open(ctx)
	if err != nil {
		th.logger.WithError(err).Error("bmc connection failed to connect")
		return err
	}
	defer func() {
		if err := th.bmcClient.Close(ctx); err != nil {
			th.logger.WithError(err).Error("bmc connection close error")
		}
	}()

	return th.Run(ctx)
}

func (th *TaskHandler) Run(ctx context.Context) error {
	ctx, span := otel.Tracer(pkgName).Start(
		ctx,
		"TaskHandler.Run",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	th.logger.Info("running condition action")
	err := th.publishActive(ctx, "running condition action")
	if err != nil {
		return err
	}

	switch th.task.Parameters.Action {
	case rctypes.ResetSettings:
		return th.ResetBios(ctx)
	default:
		return th.failedWithError(ctx, string(th.task.Parameters.Action), errUnsupportedAction)
	}
}

// ResetBios reset the bios of the server
func (th *TaskHandler) ResetBios(ctx context.Context) error {
	// Get Power State
	state, err := th.bmcClient.GetPowerState(ctx)
	if err != nil {
		return th.failedWithError(ctx, "error getting power state", err)
	}

	err = th.publishActivef(ctx, "current power state: %s", state)
	if err != nil {
		return err
	}

	// Reset Bios
	err = th.bmcClient.ResetBios(ctx)
	if err != nil {
		return th.failedWithError(ctx, "error reseting bios", err)
	}

	err = th.publishActive(ctx, "BIOS settings reset")
	if err != nil {
		return err
	}

	// Reboot (if ON)
	if state == model.PowerStateOn {
		err = th.bmcClient.SetPowerState(ctx, model.PowerStateReset)
		if err != nil {
			return th.failedWithError(ctx, "failed to reboot server", err)
		}

		return th.successful(ctx, "rebooting server")
	}

	return th.successful(ctx, "skipping server reboot, not on")
}
