package model

import (
	"net"

	"github.com/google/uuid"
)

type (
	StoreKind string
)

const (
	AppName    = "bioscfg"
	AppSubject = "biosControl"
)

// nolint:govet // prefer to keep field ordering as is
type Asset struct {
	ID uuid.UUID

	// Device BMC attributes
	BmcAddress  net.IP
	BmcUsername string
	BmcPassword string

	// Manufacturer attributes
	Vendor string
	Model  string
	Serial string

	// Facility this Asset is hosted in.
	FacilityCode string
}

func (a *Asset) AsLogFields() []any {
	return []any{
		"asset_id", a.ID.String(),
		"address", a.BmcAddress.String(),
		"vendor", a.Vendor,
		"model", a.Model,
		"serial", a.Serial,
		"facility", a.FacilityCode,
	}
}

type Args struct {
	LogLevel        string
	ConfigFile      string
	FacilityCode    string
	EnableProfiling bool
}
