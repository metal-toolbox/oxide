# BiosCfg
BIOS Configuration Controller

A service for out-of-band management of BIOS settings.

## Condition

This controller manages the actions for the `biosControl` conditions.

## Actions

Currently, the only available actions is to reset the BIOS to default settings.
The actions can be sent with the `mctl` command line tool.

```shell
mctl bios reset -s {SERVER_UUID}
```

Status of the reset can be monitored with the `mctl` tool as well.

```shell
mctl bios status -s {SERVER_UUID}
```

Example output:
```json
{
  "id": "6c022862-3a8a-41d5-884a-245353183959",
  "kind": "biosControl",
  "state": "active",
  "parameters": {
    "asset_id": "ca5ae35b-a6d0-4564-a57d-a0e7a5def9d4",
    "action": "reset_settings"
  },
  "status": {
    "task": "BiosResetSettings",
    "status": "active",
    "steps": [
      {
        "step": "GetServerPowerState",
        "status": "succeeded",
        "details": "Step completed successfully"
      },
      {
        "step": "BiosReset",
        "status": "active",
        "details": "Running step"
      },
      {
        "step": "ServerReboot",
        "status": "pending"
      }
    ]
  },
  "updated_at": "2024-04-19T00:38:42.71071142Z",
  "created_at": "2024-04-19T00:36:57.397245094Z"
}
```
