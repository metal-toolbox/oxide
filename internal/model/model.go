package model

import (
	"net"

	"github.com/google/uuid"
)

type (
	StoreKind string
)

var (
	Name = "bioscfg"
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
