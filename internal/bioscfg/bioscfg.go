package bioscfg

import (
	"context"
	"time"

	"github.com/metal-toolbox/bioscfg/internal/config"
	"github.com/metal-toolbox/bioscfg/internal/store/fleetdb"
	"github.com/metal-toolbox/ctrl"
	rctypes "github.com/metal-toolbox/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	pkgName = "internal/bioscfg"
	retries = 5
)

// BiosCfg BiosCfg Controller Struct
type BiosCfg struct {
	cfg     *config.Configuration
	logger  *logrus.Entry
	fleetdb *fleetdb.Store
	nc      *ctrl.NatsController
}

// New create a new BiosCfg Controller
func New(ctx context.Context, cfg *config.Configuration, logger *logrus.Entry) (*BiosCfg, error) {
	bc := &BiosCfg{
		cfg:    cfg,
		logger: logger,
	}

	err := bc.initDependences(ctx)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// Listen listen to Nats for tasks
func (bc *BiosCfg) Listen(ctx context.Context) error {
	handleFactory := func() ctrl.TaskHandler {
		return &TaskHandler{
			cfg:          bc.cfg,
			logger:       bc.logger,
			controllerID: bc.nc.ID(),
			fleetdb:      bc.fleetdb,
		}
	}

	err := bc.nc.ListenEvents(ctx, handleFactory)
	if err != nil {
		return err
	}

	return nil
}

// initDependences Initialize network dependencies
func (bc *BiosCfg) initDependences(ctx context.Context) error {
	err := bc.initNats(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to initialize connection to nats")
	}

	err = bc.initFleetDB(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to initialize connection to fleetdb")
	}

	return nil
}

func (bc *BiosCfg) initNats(ctx context.Context) error {
	var err error

	for i := range retries {
		bc.nc = ctrl.NewNatsController(
			string(rctypes.BiosControl),
			bc.cfg.FacilityCode,
			string(rctypes.BiosControl),
			bc.cfg.Endpoints.Nats.URL,
			bc.cfg.Endpoints.Nats.CredsFile,
			rctypes.BiosControl,
			ctrl.WithConcurrency(bc.cfg.Concurrency),
			ctrl.WithKVReplicas(bc.cfg.Endpoints.Nats.KVReplicationFactor),
			ctrl.WithLogger(bc.logger.Logger),
			ctrl.WithConnectionTimeout(bc.cfg.Endpoints.Nats.ConnectTimeout),
		)

		err = bc.nc.Connect(ctx)
		if err == nil {
			return nil
		}

		bc.logger.Error(err)
		bc.logger.Warnf("Attempt %d of %d failed. Trying again . . .", i, retries)
		time.Sleep(time.Duration(i) * time.Second)
	}

	return err
}

func (bc *BiosCfg) initFleetDB(ctx context.Context) error {
	store, err := fleetdb.New(
		ctx,
		&bc.cfg.Endpoints.FleetDB,
		bc.logger.Logger,
	)
	if err != nil {
		return err
	}

	bc.fleetdb = store

	return nil
}
