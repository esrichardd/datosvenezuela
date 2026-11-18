// Command server runs the datosvenezuela HTTP API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// ctx is canceled when the process receives Ctrl+C (SIGINT) or SIGTERM,
	// which is what Docker and most process managers send to stop a service.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// Once the first signal arrives, stop catching signals: a second Ctrl+C
	// then kills the process at once instead of waiting for the shutdown.
	context.AfterFunc(ctx, stop)

	err := run(ctx, logger)
	stop()
	if err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
