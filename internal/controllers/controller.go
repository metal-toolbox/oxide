package controllers

import (
	"context"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/metal-toolbox/bioscfg/internal/config"
	"github.com/metal-toolbox/bioscfg/internal/controllers/bioscfg"
	"github.com/metal-toolbox/bioscfg/internal/controllers/kind"
	"github.com/metal-toolbox/bioscfg/internal/metrics"
	"github.com/metal-toolbox/bioscfg/internal/profiling"
	"github.com/metal-toolbox/bioscfg/internal/version"
	"github.com/sirupsen/logrus"
)

var (
	LogLevel        string
	ConfigFile      string
	EnableProfiling bool
)

type Controller interface {
	Listen() error
}

func newController(ctx context.Context, cfg *config.Configuration, logger *logrus.Entry) (Controller, error) {
	switch cfg.Controller {
	case kind.BiosCfg:
		return bioscfg.New(ctx, cfg, logger)
	default:
		return nil, kind.ErrUnknownControllerKind
	}
}

func Run(ctx context.Context, controllerKind kind.Controller) error {
	cfg, err := config.Load(ConfigFile, LogLevel)
	if err != nil {
		return err
	}
	cfg.Controller = controllerKind

	logger := logrus.New()
	// TODO; Replace cfg.LogLevel with logrus.LogLevel, it should marshall/unmarshall?
	logger.Level, err = logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return err
	}

	metrics.ListenAndServe()
	version.ExportBuildInfoMetric()
	if EnableProfiling {
		profiling.Enable()
	}

	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, cfg.Controller.String())
	defer otelShutdown(ctx)

	v, err := version.Current().AsMap()
	if err != nil {
		return err
	}
	loggerEntry := logger.WithFields(v)
	loggerEntry.Infof("Initializing %s", cfg.Controller.String())

	controller, err := newController(ctx, cfg, loggerEntry)
	if err != nil {
		return err
	}

	loggerEntry.Infof("Success! %s is starting to listen for conditions", cfg.Controller.String())

	return controller.Listen()
}
