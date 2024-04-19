package cmd

import (
	"context"
	"log/slog"
	_ "net/http/pprof" // nolint:gosec // profiling endpoint listens on localhost.
	"os"
	"os/signal"
	"syscall"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/metal-toolbox/bioscfg/internal/configuration"
	"github.com/metal-toolbox/bioscfg/internal/handlers"
	"github.com/metal-toolbox/bioscfg/internal/log"
	"github.com/metal-toolbox/bioscfg/internal/metrics"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/bioscfg/internal/profiling"
	"github.com/metal-toolbox/bioscfg/internal/store"
	"github.com/metal-toolbox/bioscfg/internal/version"
	"github.com/metal-toolbox/rivets/events/controller"
)

func runWorker(ctx context.Context, args *model.Args) error {
	config, err := configuration.Load(args)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		return err
	}

	slog.Info("Configuration loaded", config.AsLogFields()...)

	log.SetLevel(config.LogLevel)

	// serve metrics endpoint
	metrics.ListenAndServe()
	version.ExportBuildInfoMetric()

	if config.EnableProfiling {
		profiling.Enable()
	}

	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, model.AppName)
	defer otelShutdown(ctx)

	repository, err := store.NewRepository(ctx, config)
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

	nc := controller.NewNatsController(
		model.AppName,
		config.FacilityCode,
		model.AppSubject,
		config.NatsConfig.NatsURL,
		config.NatsConfig.CredsFile,
		model.AppSubject,
		controller.WithConcurrency(config.Concurrency),
		controller.WithKVReplicas(config.NatsConfig.KVReplicas),
		controller.WithConnectionTimeout(config.NatsConfig.ConnectTimeout),
		controller.WithLogger(log.NewLogrusLogger(config.LogLevel)),
	)

	if err = nc.Connect(ctx); err != nil {
		slog.Error("Failed to connect to NATS", "error", err)
		return err
	}

	slog.With(version.Current()).Info("bioscfg worker running")

	err = nc.ListenEvents(ctx, func() controller.ConditionHandler {
		return handlers.NewHandlerFactory(repository)
	})
	if err != nil {
		slog.Error("Failed to listen for events", "error", err)
		return err
	}

	return nil
}
