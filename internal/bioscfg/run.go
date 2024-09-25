package bioscfg

import (
	"context"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/metal-toolbox/bioscfg/internal/config"
	"github.com/metal-toolbox/bioscfg/internal/metrics"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/profiling"
	"github.com/metal-toolbox/bioscfg/internal/version"
	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context, configFile, logLevel string, enableProfiling bool) error {
	cfg, err := config.Load(configFile, logLevel)
	if err != nil {
		return err
	}

	logger := logrus.New()
	// TODO; Replace cfg.LogLevel with logrus.LogLevel, it should marshall/unmarshall?
	logger.Level, err = logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return err
	}

	metrics.ListenAndServe()
	version.ExportBuildInfoMetric()
	if enableProfiling {
		profiling.Enable()
	}

	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, model.Name)
	defer otelShutdown(ctx)

	v, err := version.Current().AsMap()
	if err != nil {
		return err
	}
	loggerEntry := logger.WithFields(v)
	loggerEntry.Infof("Initializing %s", model.Name)

	controller, err := New(ctx, cfg, loggerEntry)
	if err != nil {
		return err
	}

	loggerEntry.Infof("Success! %s is starting to listen for conditions", model.Name)

	return controller.Listen(ctx)
}
