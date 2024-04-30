package tasks

import (
	"context"
	"encoding/json"
	"log/slog"
	"runtime/debug"

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/metal-toolbox/bioscfg/internal/model"
	rctypes "github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events/controller"
	"github.com/pkg/errors"
)

var (
	currentPowerStateKey = "currentPowerState"
)

// Miscellaneous
type sharedData map[string]interface{}

// TaskStatus has status about a task, and it's steps.
type TaskStatus struct {
	Task       string        `json:"task"`
	Status     string        `json:"status"`
	Details    string        `json:"details,omitempty"`
	Error      string        `json:"error,omitempty"`
	ActiveStep string        `json:"active_step,omitempty"`
	Steps      []*StepStatus `json:"steps"`
}

// NewTaskStatus will generate a new task status struct
func NewTaskStatus(taskName string, state rctypes.State) *TaskStatus {
	return &TaskStatus{
		Task:   taskName,
		Status: string(state),
	}
}

func (r *TaskStatus) AsLogFields() []string {
	return []string{
		"task", r.Task,
		"status", r.Status,
		"details", r.Details,
		"error", r.Error,
	}
}

func (r *TaskStatus) Marshal() ([]byte, error) {
	respBytes, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal response to json")
	}

	return respBytes, nil
}

// Task is a unit of work to address a condition from condition orchestrator.
// The task multiple steps to accomplish the task.
type Task interface {
	// Name of the task
	Name() string
	// Asset is the server that will be affected by this task
	Asset() *model.Asset
	// Steps is the multiple units of work that will accomplish this task
	Steps() []Step
}

type biosResetTask struct {
	name  string
	asset *model.Asset
	steps []Step
}

// NewBiosResetTask creates the task for resetting the BIOS of a server to default settings.
func NewBiosResetTask(asset *model.Asset) Task {
	return &biosResetTask{
		name:  "BiosResetSettings",
		asset: asset,
		steps: []Step{
			GetServerPowerStateStep(),
			BiosResetStep(),
			ServerRebootStep(),
		},
	}
}

func (j *biosResetTask) Name() string {
	return j.name
}

func (j *biosResetTask) Steps() []Step {
	return j.steps
}

func (j *biosResetTask) Asset() *model.Asset {
	return j.asset
}

// TaskRunner Will run the task by executing the individual steps in the task,
// and reports task status using the publisher.
type TaskRunner struct {
	publisher  controller.ConditionStatusPublisher
	task       Task
	taskStatus *TaskStatus
}

// NewTaskRunner creates a TaskRunner to run a specific Task
func NewTaskRunner(publisher controller.ConditionStatusPublisher, task Task) *TaskRunner {
	return &TaskRunner{
		publisher:  publisher,
		task:       task,
		taskStatus: NewTaskStatus(task.Name(), rctypes.Pending),
	}
}

func (r *TaskRunner) Run(ctx context.Context, client *bmclib.Client) (err error) {
	slog.With(r.task.Asset().AsLogFields()...).Info("Running task", "task", r.task.Name())

	data := sharedData{}
	r.initTaskLog()

	defer func() {
		if rec := recover(); rec != nil {
			err = r.handlePanic(ctx, rec)
		}
	}()

	r.publishTaskUpdate(ctx, rctypes.Active, "Opening client", nil)

	if err = client.Open(ctx); err != nil {
		r.publishFailed(ctx, 0, "Failed to open client", err)
		return errors.Wrap(err, "failed to open client")
	}
	defer client.Close(ctx)

	for stepID, step := range r.task.Steps() {
		r.publishStepUpdate(ctx, stepID, "Running step")

		details, err := step.Run(ctx, client, data)
		if err != nil {
			r.publishFailed(ctx, stepID, "Step failure", err)
			return err
		}

		r.publishStepSuccess(ctx, stepID, details)
	}

	r.publishTaskSuccess(ctx)

	return nil
}

func (r *TaskRunner) initTaskLog() {
	steps := r.task.Steps()
	r.taskStatus.Steps = make([]*StepStatus, len(steps))

	for i, step := range steps {
		r.taskStatus.Steps[i] = NewStepStatus(step.Name(), rctypes.Pending, "", nil)
	}
}

func (r *TaskRunner) handlePanic(ctx context.Context, rec any) error {
	msg := "Panic occurred while running task"
	slog.Error("!!panic occurred", "rec", rec, "stack", string(debug.Stack()))
	slog.Error(msg)
	err := errors.New("Task fatal error, check logs for details")

	r.publishTaskUpdate(ctx, rctypes.Failed, msg, err)

	return err
}

func (r *TaskRunner) publishStepUpdate(ctx context.Context, stepID int, details string) {
	r.publish(ctx, stepID, rctypes.Active, rctypes.Active, details, nil)
}

func (r *TaskRunner) publishStepSuccess(ctx context.Context, stepID int, details string) {
	r.publish(ctx, stepID, rctypes.Succeeded, rctypes.Active, details, nil)
}

func (r *TaskRunner) publishFailed(ctx context.Context, stepID int, details string, err error) {
	slog.With(r.task.Asset().AsLogFields()...).Error("Task failed", "task", r.task.Name())
	r.publish(ctx, stepID, rctypes.Failed, rctypes.Failed, details, err)
}

func (r *TaskRunner) publishTaskSuccess(ctx context.Context) {
	slog.With(r.task.Asset().AsLogFields()...).Info("Task completed successfully", "task", r.task.Name())
	r.publishTaskUpdate(ctx, rctypes.Succeeded, "Task completed successfully", nil)
}

func (r *TaskRunner) publish(ctx context.Context, stepID int, stepState, taskState rctypes.State, details string, err error) {
	step := r.task.Steps()[stepID]
	stepStatus := NewStepStatus(step.Name(), stepState, details, err)

	slog.With(r.task.Asset().AsLogFields()...).With(stepStatus.AsLogFields()...).Info(details, "step", step.Name())

	r.taskStatus.Steps[stepID] = stepStatus

	var taskDetails string
	if err != nil {
		taskDetails = "Task failed at step " + step.Name()
	}

	r.publishTaskUpdate(ctx, taskState, taskDetails, err)
}

func (r *TaskRunner) publishTaskUpdate(ctx context.Context, state rctypes.State, details string, err error) {
	r.taskStatus.Status = string(state)
	r.taskStatus.Details = details

	if err != nil {
		r.taskStatus.Error = err.Error()
	}

	slog.With(r.task.Asset().AsLogFields()...).Info("Task update", "task", r.task.Name())

	respBytes, err := json.Marshal(r.taskStatus)
	if err != nil {
		slog.Error("Failed to marshal condition update", "error", err)
		return
	}

	r.publisher.Publish(ctx, r.task.Asset().ID.String(), state, respBytes)
}
