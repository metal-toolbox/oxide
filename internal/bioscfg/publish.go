package bioscfg

import (
	"context"
	"fmt"
	"time"

	rctypes "github.com/metal-toolbox/rivets/v2/condition"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/metal-toolbox/bioscfg/internal/metrics"
)

func (th *TaskHandler) publish(ctx context.Context, status string, state rctypes.State) error {
	th.task.State = state
	th.task.Status.Append(status)

	genTask, err := th.task.toGeneric()
	if err != nil {
		th.logger.WithError(errTaskConv).Error()
		return err
	}

	if errDelay := sleepInContext(ctx, 10*time.Second); errDelay != nil {
		return context.Canceled
	}

	return th.publisher.Publish(ctx,
		genTask,
		false,
	)
}

func (th *TaskHandler) publishActive(ctx context.Context, status string) error {
	err := th.publish(ctx, status, rctypes.Active)
	if err != nil {
		th.logger.Infof("failed to publish condition status: %s", status)
		return err
	}

	th.logger.Infof("condition active: %s", status)
	return nil
}

func (th *TaskHandler) publishActivef(ctx context.Context, status string, args ...interface{}) error {
	if len(args) > 0 {
		status = fmt.Sprintf(status, args)
	}

	return th.publishActive(ctx, status)
}

// failed condition helper method
func (th *TaskHandler) failed(ctx context.Context, status string) error {
	err := th.publish(ctx, status, rctypes.Failed)

	th.registerConditionMetrics(string(rctypes.Failed))

	if err != nil {
		th.logger.Infof("failed to publish condition status: %s", status)
		return err
	}

	th.logger.Warnf("condition failed: %s", status)
	return nil
}

func (th *TaskHandler) failedWithError(ctx context.Context, status string, err error) error {
	newError := th.failed(ctx, errors.Wrap(err, status).Error())
	if newError != nil {
		if err != nil {
			return errors.Wrap(newError, err.Error())
		}

		return newError
	}

	return err
}

// successful condition helper method
func (th *TaskHandler) successful(ctx context.Context, status string) error {
	err := th.publish(ctx, status, rctypes.Succeeded)

	th.registerConditionMetrics(string(rctypes.Succeeded))

	if err != nil {
		th.logger.Warnf("failed to publish condition status: %s", status)
		return err
	}

	th.logger.Infof("condition complete: %s", status)
	return nil
}

func (th *TaskHandler) registerConditionMetrics(status string) {
	metrics.ConditionRunTimeSummary.With(
		prometheus.Labels{
			"condition": string(rctypes.ServerControl),
			"state":     status,
		},
	).Observe(time.Since(th.startTS).Seconds())
}

// sleepInContext
func sleepInContext(ctx context.Context, t time.Duration) error {
	select {
	case <-time.After(t):
		return nil
	case <-ctx.Done():
		return context.Canceled
	}
}
