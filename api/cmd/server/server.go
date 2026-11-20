package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
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

// run prepares the server (configuration, data, port) and serves until ctx
// is canceled or the server fails. Keeping the logic out of main makes errors
// easy to return, and receiving ctx lets tests stop it without a signal.
func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Load every dataset before opening the port: invalid data stops the
	// server here instead of serving wrong answers.
	st, err := loadStores(ctx, cfg.dataDir)
	if err != nil {
		return err
	}
	logDataLoaded(ctx, logger, st)

	// Open the port before announcing the server, so "server started" is only
	// logged when it is true. A port that is already in use fails here.
	addr := fmt.Sprintf(":%d", cfg.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	logger.Info("server started", "port", cfg.port)

	srv := &http.Server{
		Handler:           newHandler(logger, st),
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
