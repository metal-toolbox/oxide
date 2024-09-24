package tasks

import (
	"context"
	"encoding/json"
	"log/slog"
	"runtime/debug"

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/ctrl"
	rctypes "github.com/metal-toolbox/rivets/condition"
	rtypes "github.com/metal-toolbox/rivets/types"
	"github.com/mitchellh/copystructure"
	"github.com/pkg/errors"
)

var (
	currentPowerStateKey      = "currentPowerState"
	errInvalidConditionParams = errors.New("invalid condition parameters")
	errTaskConv               = errors.New("error in generic Task conversion")
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

type RCTask rctypes.Task[*rctypes.BiosControlTaskParameters, json.RawMessage]

func NewTask(task *rctypes.Task[any, any]) (*RCTask, error) {
	paramsJSON, ok := task.Parameters.(json.RawMessage)
	if !ok {
		return nil, errInvalidConditionParams
	}

	params := rctypes.BiosControlTaskParameters{}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return nil, err
	}

	// deep copy fields referenced by pointer
	asset, err := copystructure.Copy(task.Server)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Server")
	}

	fault, err := copystructure.Copy(task.Fault)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Fault")
	}

	return &RCTask{
		StructVersion: task.StructVersion,
		ID:            task.ID,
		Kind:          task.Kind,
		State:         task.State,
		Status:        task.Status,
		Parameters:    &params,
		Fault:         fault.(*rctypes.Fault),
		FacilityCode:  task.FacilityCode,
		Server:        asset.(*rtypes.Server),
		WorkerID:      task.WorkerID,
		TraceID:       task.TraceID,
		SpanID:        task.SpanID,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		CompletedAt:   task.CompletedAt,
	}, nil
}

func (task *RCTask) ToGeneric() (*rctypes.Task[any, any], error) {
	paramsJSON, err := task.Parameters.Marshal()
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Parameters")
	}

	// deep copy fields referenced by pointer
	asset, err := copystructure.Copy(task.Server)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Server")
	}

	fault, err := copystructure.Copy(task.Fault)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Fault")
	}

	return &rctypes.Task[any, any]{
		StructVersion: task.StructVersion,
		ID:            task.ID,
		Kind:          task.Kind,
		State:         task.State,
		Status:        task.Status,
		Parameters:    paramsJSON,
		Fault:         fault.(*rctypes.Fault),
		FacilityCode:  task.FacilityCode,
		Server:        asset.(*rtypes.Server),
		WorkerID:      task.WorkerID,
		TraceID:       task.TraceID,
		SpanID:        task.SpanID,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		CompletedAt:   task.CompletedAt,
	}, nil
}

// TaskRunner Will run the task by executing the individual steps in the task,
// and reports task status using the publisher.
type TaskRunner struct {
	publisher  ctrl.Publisher
	task       *RCTask
	oldTask    Task
	taskStatus *TaskStatus
}

// NewTaskRunner creates a TaskRunner to run a specific Task
func NewTaskRunner(publisher ctrl.Publisher, oldTask Task, task *RCTask) *TaskRunner {
	return &TaskRunner{
		publisher:  publisher,
		task:       task,
		oldTask:    oldTask,
		taskStatus: NewTaskStatus(oldTask.Name(), rctypes.Pending),
	}
}

func (r *TaskRunner) Run(ctx context.Context, client *bmclib.Client) (err error) {
	slog.With(r.oldTask.Asset().AsLogFields()...).Info("Running task", "task", r.oldTask.Name())

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

	for stepID, step := range r.oldTask.Steps() {
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
	steps := r.oldTask.Steps()
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
	slog.With(r.oldTask.Asset().AsLogFields()...).Error("Task failed", "task", r.oldTask.Name())
	r.publish(ctx, stepID, rctypes.Failed, rctypes.Failed, details, err)
}

func (r *TaskRunner) publishTaskSuccess(ctx context.Context) {
	slog.With(r.oldTask.Asset().AsLogFields()...).Info("Task completed successfully", "task", r.oldTask.Name())
	r.publishTaskUpdate(ctx, rctypes.Succeeded, "Task completed successfully", nil)
}

func (r *TaskRunner) publish(ctx context.Context, stepID int, stepState, taskState rctypes.State, details string, err error) {
	step := r.oldTask.Steps()[stepID]
	stepStatus := NewStepStatus(step.Name(), stepState, details, err)

	slog.With(r.oldTask.Asset().AsLogFields()...).With(stepStatus.AsLogFields()...).Info(details, "step", step.Name())

	r.taskStatus.Steps[stepID] = stepStatus

	var taskDetails string
	if err != nil {
		taskDetails = "Task failed at step " + step.Name()
	}

	r.publishTaskUpdate(ctx, taskState, taskDetails, err)
}

func (r *TaskRunner) publishTaskUpdate(ctx context.Context, state rctypes.State, details string, err error) {
	r.task.State = state
	r.task.Status.Append(details)

	if err != nil {
		r.taskStatus.Error = err.Error()
	}

	slog.With(r.oldTask.Asset().AsLogFields()...).Info("Task update", "task", r.oldTask.Name())

	genTask, err := r.task.ToGeneric()
	if err != nil {
		r.taskStatus.Error = err.Error()
		return
	}

	err = r.publisher.Publish(ctx, genTask, false)
	if err != nil {
		return
	}
}
