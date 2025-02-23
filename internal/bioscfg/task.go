package bioscfg

import (
	"encoding/json"

	rctypes "github.com/metal-toolbox/rivets/v2/condition"
	rtypes "github.com/metal-toolbox/rivets/v2/types"
	"github.com/mitchellh/copystructure"
	"github.com/pkg/errors"
)

type Task rctypes.Task[*rctypes.BiosControlTaskParameters, json.RawMessage]

// newTask converts a Generic Condition Task to a BiosControl Task
func newTask(task *rctypes.Task[any, any]) (*Task, error) {
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

	return &Task{
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

// toGeneric converts a BiosControl Task to a Generic Condition Task
func (task *Task) toGeneric() (*rctypes.Task[any, any], error) {
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
