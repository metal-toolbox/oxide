package bmc

import (
	"context"
	"errors"
	"time"

	"github.com/metal-toolbox/bioscfg/internal/model"
)

type server struct {
	powerStatus        string
	bootTime           time.Time
	bootDevice         string
	previousBootDevice string
	persistent         bool
	efiBoot            bool
}

var (
	errBmcCantFindServer = errors.New("dryrun BMC couldnt find server to set state")
	errBmcServerOffline  = errors.New("dryrun BMC couldnt set boot device, server is off")
	serverStates         = make(map[string]server)
)

// DryRunBMC is an simulated implementation of the Queryor interface
type DryRunBMCClient struct {
	id string
}

// NewDryRunBMCClient creates a new Queryor interface for a simulated BMC
func NewDryRunBMCClient(asset *model.Asset) *DryRunBMCClient {
	_, ok := serverStates[asset.ID.String()]
	if !ok {
		serverStates[asset.ID.String()] = getDefaultSettings()
	}

	return &DryRunBMCClient{
		asset.ID.String(),
	}
}

// Open simulates creating a BMC session
func (b *DryRunBMCClient) Open(_ context.Context) error {
	return nil
}

// Close simulates logging out of the BMC
func (b *DryRunBMCClient) Close(_ context.Context) error {
	return nil
}

// GetPowerState simulates returning the device power status
func (b *DryRunBMCClient) GetPowerState(_ context.Context) (string, error) {
	server, err := b.getServer()
	if err != nil {
		return "", err
	}

	return server.powerStatus, nil
}

// SetPowerState simulates setting the given power state on the device
func (b *DryRunBMCClient) SetPowerState(_ context.Context, state string) error {
	server, err := b.getServer()
	if err != nil {
		return err
	}

	if isRestarting(state) {
		server.bootTime = getRestartTime(state)
	}

	server.powerStatus = state
	serverStates[b.id] = *server
	return nil
}

// SetBootDevice simulates setting the boot device of the remote device
func (b *DryRunBMCClient) SetBootDevice(_ context.Context, device string, persistent, efiBoot bool) error {
	server, err := b.getServer()
	if err != nil {
		return err
	}

	if server.powerStatus != "on" {
		return errBmcServerOffline
	}

	server.previousBootDevice = server.bootDevice
	server.bootDevice = device
	server.persistent = persistent
	server.efiBoot = efiBoot

	return nil
}

// GetBootDevice simulates getting the boot device information of the remote device
func (b *DryRunBMCClient) GetBootDevice(_ context.Context) (device string, persistent, efiBoot bool, err error) {
	server, err := b.getServer()
	if err != nil {
		return "", false, false, err
	}

	if server.powerStatus != "on" {
		return "", false, false, errBmcServerOffline
	}

	return server.bootDevice, server.persistent, server.efiBoot, nil
}

// PowerCycleBMC simulates a power cycle action on the BMC of the remote device
func (b *DryRunBMCClient) PowerCycleBMC(_ context.Context) error {
	return nil
}

// HostBooted reports whether or not the device has booted the host OS
func (b *DryRunBMCClient) HostBooted(_ context.Context) (bool, error) {
	return true, nil
}

func (b *DryRunBMCClient) ResetBios(ctx context.Context) error {
	_, ok := serverStates[b.id]
	if !ok {
		return errBmcCantFindServer
	}

	serverStates[b.id] = getDefaultSettings()

	return b.SetPowerState(ctx, "cycle")
}

// getServer gets a simulateed server state, and update power status and boot device if required
func (b *DryRunBMCClient) getServer() (*server, error) {
	state, ok := serverStates[b.id]
	if !ok {
		return nil, errBmcCantFindServer
	}

	if isRestarting(state.powerStatus) {
		if time.Now().After(state.bootTime) {
			state.powerStatus = "on"

			if !state.persistent {
				state.bootDevice = state.previousBootDevice
			}
		}
	}

	return &state, nil
}

func isRestarting(state string) bool {
	switch state {
	case "reset", "cycle":
		return true
	default:
		return false
	}
}

func getRestartTime(state string) time.Time {
	switch state {
	case "reset":
		return time.Now().Add(time.Second * 30) // Soft reboot should take longer than a hard reboot
	case "cycle":
		return time.Now().Add(time.Second * 20)
	default:
		return time.Now() // No reboot necessary
	}
}

func getDefaultSettings() server {
	status := server{}

	status.powerStatus = "on"
	status.bootDevice = "disk"
	status.previousBootDevice = "disk"
	status.persistent = true
	status.efiBoot = false
	status.bootTime = time.Now()

	return status
}
