package cmd

import (
	"context"
	"log/slog"
	_ "net/http/pprof" // nolint:gosec // profiling endpoint listens on localhost.
	"os"
	"os/signal"
	"syscall"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/metal-toolbox/bioscfg/internal/config"
	"github.com/metal-toolbox/bioscfg/internal/handlers"
	"github.com/metal-toolbox/bioscfg/internal/log"
	"github.com/metal-toolbox/bioscfg/internal/metrics"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/profiling"
	"github.com/metal-toolbox/bioscfg/internal/store/fleetdb"
	"github.com/metal-toolbox/bioscfg/internal/version"
	"github.com/metal-toolbox/ctrl"
)

func runWorker(ctx context.Context, args *model.Args) error {
	cfg, err := config.Load(args.ConfigFile, args.LogLevel)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		return err
	}

	slog.Info("Configuration loaded", cfg.AsLogFields()...)

	log.SetLevel(cfg.LogLevel)

	// serve metrics endpoint
	metrics.ListenAndServe()
	version.ExportBuildInfoMetric()

	if args.EnableProfiling {
		profiling.Enable()
	}

	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, model.AppName)
	defer otelShutdown(ctx)

	log.NewLogrusLogger(cfg.LogLevel)
	repository, err := fleetdb.New(ctx, &cfg.Endpoints.FleetDB, log.NewLogrusLogger(cfg.LogLevel))
	if err != nil {
		slog.Error("Failed to create repository", "error", err)
		return err
	}

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(ctx)

	// Cancel the context when we receive a termination signal.
	go func() {
		s := <-termChan
		slog.Info("Received signal for termination, exiting...", "signal", s.String())
		cancel()
	}()

	nc := ctrl.NewNatsController(
		model.AppName,
		cfg.FacilityCode,
		model.AppSubject,
		cfg.Endpoints.Nats.URL,
		cfg.Endpoints.Nats.CredsFile,
		model.AppSubject,
		ctrl.WithConcurrency(cfg.Concurrency),
		ctrl.WithKVReplicas(cfg.Endpoints.Nats.KVReplicationFactor),
		ctrl.WithConnectionTimeout(cfg.Endpoints.Nats.ConnectTimeout),
		ctrl.WithLogger(log.NewLogrusLogger(cfg.LogLevel)),
	)

	if err = nc.Connect(ctx); err != nil {
		slog.Error("Failed to connect to NATS", "error", err)
		return err
	}

	slog.With(version.Current().AsLogFields()...).Info("bioscfg worker running")

	err = nc.ListenEvents(ctx, func() ctrl.TaskHandler {
		return handlers.NewHandlerFactory(repository)
	})
	if err != nil {
		slog.Error("Failed to listen for events", "error", err)
		return err
	}

	return nil
}
