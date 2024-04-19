package profiling

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	Endpoint          = "localhost:9091"
	ReadHeaderTimeout = 2 * time.Second
)

// Enable the profiling endpoint
func Enable() {
	go func() {
		server := &http.Server{
			Addr:              Endpoint,
			ReadHeaderTimeout: ReadHeaderTimeout,
		}

		if err := server.ListenAndServe(); err != nil {
			slog.Error("Failed to start profiling server", "error", err)
		}
	}()

	slog.Info("profiling enabled", "endpoint", Endpoint+"/debug/pprof")
}
