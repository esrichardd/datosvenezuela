// Command server runs the apivenezuela HTTP API.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/esrichardd/apivenezuela/api/internal/geo"
	"github.com/esrichardd/apivenezuela/api/internal/platform/store"
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
	port    int
	dataDir string // directory with the CSV datasets (the repo's data/)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

// run prepares the server (configuration, data, port) and serves until it is
// stopped by a signal or fails. Keeping the logic out of main makes errors
// easy to return.
func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// ctx is canceled when the process receives Ctrl+C (SIGINT) or SIGTERM,
	// which is what Docker and most process managers send to stop a service.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Once the first signal arrives, stop catching signals: a second Ctrl+C
	// then kills the process at once instead of waiting for the shutdown.
	context.AfterFunc(ctx, stop)

	// Load every dataset before opening the port: invalid data stops the
	// server here instead of serving wrong answers.
	geoStore, err := geo.LoadCSV(filepath.Join(cfg.dataDir, "geography"))
	if err != nil {
		return fmt.Errorf("load geography: %w", err)
	}
	logGeographyLoaded(ctx, logger, geoStore)

	// Open the port before announcing the server, so "server started" is only
	// logged when it is true. A port that is already in use fails here.
	addr := fmt.Sprintf(":%d", cfg.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	logger.Info("server started", "port", cfg.port)

	srv := &http.Server{
		Handler:           newHandler(logger, geoStore),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	return serve(ctx, logger, srv, ln)
}

// serve runs srv on ln until ctx is canceled, then shuts it down gracefully.
// It returns nil after a clean shutdown. It always closes ln.
func serve(ctx context.Context, logger *slog.Logger, srv *http.Server, ln net.Listener) error {
	// Serve blocks, so it runs in its own goroutine and reports how it ended
	// through serveErr. The channel has room for that one value, so the
	// goroutine can always send it and finish, even if nobody is receiving.
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		// Serve only returns before a shutdown when it fails.
		return fmt.Errorf("serve: %w", err)
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
	// The default dataDir works for `make run`, which runs from api/.
	cfg := config{port: 8080, dataDir: "../data"}

	if v, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return config{}, fmt.Errorf("invalid PORT %q: must be a number between 1 and 65535", v)
		}
		cfg.port = port
	}

	if v, ok := os.LookupEnv("DATA_DIR"); ok {
		cfg.dataDir = v
	}
	if info, err := os.Stat(cfg.dataDir); err != nil || !info.IsDir() {
		return config{}, fmt.Errorf("invalid DATA_DIR %q: must be an existing directory", cfg.dataDir)
	}

	return cfg, nil
}

// logGeographyLoaded logs how many records were loaded, to confirm at startup
// that the expected data is being served. An empty Page selects no items, so
// each call only returns the total.
func logGeographyLoaded(ctx context.Context, logger *slog.Logger, s *geo.CSVStore) {
	_, states, _ := s.States(ctx, store.Page{})
	_, municipalities, _ := s.Municipalities(ctx, store.Page{})
	_, parishes, _ := s.Parishes(ctx, store.Page{})
	logger.Info("geography loaded",
		"states", states,
		"municipalities", municipalities,
		"parishes", parishes,
	)
}
