package bioscfg

import (
	"context"
	"io"
	"net/http"

	"github.com/metal-toolbox/bioscfg/internal/model"
	rctypes "github.com/metal-toolbox/rivets/condition"
)

func (th *TaskHandler) HandleAction(ctx context.Context) error {
	switch th.task.Parameters.Action {
	case rctypes.ResetConfig:
		return th.ResetBiosConfig(ctx)
	case rctypes.SetConfig:
		return th.SetBiosConfig(ctx)
	default:
		return th.failedWithError(ctx, string(th.task.Parameters.Action), errUnsupportedAction)
	}
}

// ResetBios reset the bios of the server
func (th *TaskHandler) ResetBiosConfig(ctx context.Context) error {
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
	err = th.bmcClient.ResetBiosConfig(ctx)
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

// UploadConfig set BIOS Config
func (th *TaskHandler) SetBiosConfig(ctx context.Context) error {
	var configURL = ""
	if th.task.Parameters.BiosConfigURL != nil {
		configURL = th.task.Parameters.BiosConfigURL.String()
	}

	if configURL == "" {
		return th.failed(ctx, "no Bios Configu URL was found")
	}

	req, err := http.NewRequest(http.MethodGet, configURL, http.NoBody)
	if err != nil {
		return th.failedWithError(ctx, "failed to create http request", err)
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return th.failedWithError(ctx, "failed to get bios config from url", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return th.failed(ctx, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return th.failedWithError(ctx, "failed to read file from response body", err)
	}

	err = th.publishActive(ctx, "got bios config from url")
	if err != nil {
		return err
	}

	err = th.bmcClient.SetBiosConfigFromFile(ctx, string(body))
	if err != nil {
		return th.failedWithError(ctx, "failed to set bios config through the bmc", err)
	}

	return th.successful(ctx, "bios set")
}
