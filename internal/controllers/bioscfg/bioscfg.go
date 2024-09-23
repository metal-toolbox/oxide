package bioscfg

import (
	"context"

	"github.com/metal-toolbox/bioscfg/internal/config"
	"github.com/metal-toolbox/bioscfg/internal/store/fleetdb"
	"github.com/metal-toolbox/ctrl"
	rctypes "github.com/metal-toolbox/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	pkgName = "internal/bioscfg"
)

// BiosCfg BiosCfg Controller Struct
type BiosCfg struct {
	cfg     *config.Configuration
	logger  *logrus.Entry
	ctx     context.Context
	fleetdb *fleetdb.Store
	nc      *ctrl.NatsController
}

// New create a new BiosCfg Controller
func New(ctx context.Context, cfg *config.Configuration, logger *logrus.Entry) (*BiosCfg, error) {
	bc := &BiosCfg{
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
	}

	err := bc.initDependences()
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// Listen listen to Nats for tasks
func (bc *BiosCfg) Listen() error {
	handleFactory := func() ctrl.TaskHandler {
		return &TaskHandler{
			cfg:          bc.cfg,
			logger:       bc.logger,
			controllerID: bc.nc.ID(),
			fleetdb:      bc.fleetdb,
		}
	}

	err := bc.nc.ListenEvents(bc.ctx, handleFactory)
	if err != nil {
		return err
	}

	return nil
}

// initDependences Initialize network dependencies
func (bc *BiosCfg) initDependences() error {
	err := bc.initNats()
	if err != nil {
		return errors.Wrap(err, "failed to initialize connection to nats")
	}

	err = bc.initFleetDB()
	if err != nil {
		return errors.Wrap(err, "failed to initialize connection to fleetdb")
	}

	return nil
}

func (bc *BiosCfg) initNats() error {
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

	err := bc.nc.Connect(bc.ctx)
	if err != nil {
		bc.logger.Error(err)
		return err
	}

	return nil
}

func (bc *BiosCfg) initFleetDB() error {
	store, err := fleetdb.New(
		bc.ctx,
		&bc.cfg.Endpoints.FleetDB,
		bc.logger.Logger,
	)
	if err != nil {
		return err
	}

	bc.fleetdb = store

	return nil
}
