// Command server runs the apivenezuela HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Server timeouts. Go's http.Server has no timeouts by default, which leaves it
// open to slow-client attacks (e.g. Slowloris). These values are generous for a
// read-only JSON API.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second

	// shutdownTimeout is how long in-flight requests get to finish on shutdown.
	shutdownTimeout = 10 * time.Second
)

// config holds the settings read from environment variables.
type config struct {
	port int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

// run starts the server and blocks until it is stopped by a signal or fails.
// Keeping the logic out of main makes errors easy to return and test.
func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// ctx is cancelled when the process receives Ctrl+C (SIGINT) or SIGTERM,
	// which is what Docker and most process managers send to stop a service.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.port),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// ListenAndServe blocks, so it runs in its own goroutine. Startup errors
	// (e.g. port already in use) are sent back through serverErr.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server started", "port", cfg.port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	// Graceful shutdown: stop accepting new connections and wait for in-flight
	// requests to finish, up to shutdownTimeout.
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("server stopped")
	return nil
}

// loadConfig reads the configuration from environment variables, applying
// defaults. Invalid values fail fast at startup instead of at first use.
func loadConfig() (config, error) {
	cfg := config{port: 8080}

	if v, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return config{}, fmt.Errorf("invalid PORT %q: must be a number between 1 and 65535", v)
		}
		cfg.port = port
	}
	return cfg, nil
}

// handleHealth reports that the server is up. Load balancers and Docker use it
// to decide whether to send traffic to this instance.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
