package tasks

import (
	"context"
	"log/slog"

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/metal-toolbox/bioscfg/internal/model"
	rctypes "github.com/metal-toolbox/rivets/condition"
	"github.com/pkg/errors"
)

// StepStatus has status about a step, to be reported as part of the overall task.
type StepStatus struct {
	Step    string `json:"step"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewStepStatus will create a new step status struct
func NewStepStatus(stepName string, state rctypes.State, details string, err error) *StepStatus {
	status := &StepStatus{
		Step:    stepName,
		Status:  string(state),
		Details: details,
	}

	if err != nil {
		status.Error = err.Error()
	}

	return status
}

func (s *StepStatus) AsLogFields() []any {
	return []any{
		"task", s.Step,
		"status", s.Status,
		"details", s.Details,
		"error", s.Error,
	}
}

// Step is a unit of work. Multiple steps accomplish a task.
type Step interface {
	// Name of this step
	Name() string
	// Run will execute the code to accomplish this step
	Run(ctx context.Context, client *bmclib.Client, data sharedData) (string, error)
}

type getServerPowerStateStep struct {
	name string
}

// GetServerPowerStateStep will get the current power state of a server,
// and store it in sharedData.
func GetServerPowerStateStep() Step {
	return &getServerPowerStateStep{
		name: "GetServerPowerState",
	}
}

func (t *getServerPowerStateStep) Name() string {
	return t.name
}

func (t *getServerPowerStateStep) Run(ctx context.Context, client *bmclib.Client, data sharedData) (string, error) {
	state, err := client.GetPowerState(ctx)
	if err != nil {
		return "Failed to get current power state", err
	}

	data[currentPowerStateKey] = state

	return "Current power state: " + state, nil
}

type biosResetStep struct {
	name string
}

// BiosResetStep will use the client to reset the BIOS settings.
func BiosResetStep() Step {
	return &biosResetStep{
		name: "BiosReset",
	}
}

func (t *biosResetStep) Name() string {
	return t.name
}

func (t *biosResetStep) Run(ctx context.Context, client *bmclib.Client, _ sharedData) (string, error) {
	err := client.ResetBiosConfiguration(ctx)
	if err != nil {
		return "Failed to reset bios settings", err
	}

	return "BIOS settings reset", nil
}

type serverRebootStep struct {
	name string
}

// ServerRebootStep will reboot the server, if necessary, per the information in sharedData
func ServerRebootStep() Step {
	return &serverRebootStep{
		name: "ServerReboot",
	}
}

func (t *serverRebootStep) Name() string {
	return t.name
}

func (t *serverRebootStep) Run(ctx context.Context, client *bmclib.Client, data sharedData) (string, error) {
	powerState, ok := data[currentPowerStateKey].(string)
	if !ok {
		return "Reboot requirement unknown", errors.New("missing power state")
	}

	var details string

	if powerState == model.PowerStateOn {
		slog.Info("Rebooting server", "powerState", powerState)
		_, err := client.SetPowerState(ctx, model.PowerStateReset)
		if err != nil {
			return "Failed to reset power state", err
		}
		details = "Rebooting server"
	} else {
		slog.Info("Skipping server reboot", "ok", ok, "powerState", powerState)
		details = "Reboot not required"
	}

	return details, nil
}
