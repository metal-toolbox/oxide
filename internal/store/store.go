package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/metal-toolbox/bioscfg/internal/configuration"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/store/fleetdb"
)

type Repository interface {
	// AssetByID returns asset based on the identifier.
	AssetByID(ctx context.Context, assetID uuid.UUID) (*model.Asset, error)
}

func NewRepository(ctx context.Context, config *configuration.Configuration) (Repository, error) {
	return fleetdb.New(ctx, config.FleetDBOptions)
}
