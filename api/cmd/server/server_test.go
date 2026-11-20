package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// realDataDir is the repository's data directory, seen from api/cmd/server.
var realDataDir = filepath.Join("..", "..", "..", "data")

// TestRun starts the whole server as main does, with the real data, and
// checks its life: it answers requests, and canceling the context (what a
// SIGTERM does in main) shuts it down cleanly and frees the port.
func TestRun(t *testing.T) {
	port := freePort(t)
	t.Setenv("PORT", strconv.Itoa(port))
	t.Setenv("DATA_DIR", realDataDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var logs bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- run(ctx, slog.New(slog.NewJSONHandler(&logs, nil))) }()

	// The server loads the data before opening the port: wait until it
	// answers.
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not answer: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() error = %v, want nil after a clean shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not return after the context was canceled")
	}

	// run has returned, so nothing else is writing to logs.
	for _, want := range []string{`"msg":"data loaded"`, `"states":25`, `"banks":27`, `"phone_prefixes":68`, `"bank_holiday_years":[2026]`, `"msg":"server stopped"`} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %s\nwant them to contain %s", logs.String(), want)
		}
	}
	if conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port)); err == nil {
		_ = conn.Close()
		t.Error("the port is still open after the shutdown")
	}
}

// TestRunFails checks that startup problems stop the server before it
// serves anything.
func TestRunFails(t *testing.T) {
	// Holding a port makes it unavailable to run.
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = busy.Close() }()
	busyPort := busy.Addr().(*net.TCPAddr).Port

	// Each directory has the valid datasets loaded before the missing one, so
	// every dataset must be loaded, not only the first.
	noBanks := copyDatasets(t, "geography")
	noPhones := copyDatasets(t, "geography", "banks")
	noHolidays := copyDatasets(t, "geography", "banks", "phones")

	tests := []struct {
		name    string
		port    int
		dataDir string
		wantErr string
	}{
		{name: "invalid config", port: 0, dataDir: realDataDir, wantErr: "invalid PORT"},
		// An existing directory without the datasets.
		{name: "invalid geography", port: freePort(t), dataDir: t.TempDir(), wantErr: "load geography: "},
		{name: "invalid banks", port: freePort(t), dataDir: noBanks, wantErr: "load banks: "},
		{name: "invalid phones", port: freePort(t), dataDir: noPhones, wantErr: "load phones: "},
		{name: "invalid holidays", port: freePort(t), dataDir: noHolidays, wantErr: "load holidays: "},
		{name: "port in use", port: busyPort, dataDir: realDataDir, wantErr: "listen: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", strconv.Itoa(tt.port))
			t.Setenv("DATA_DIR", tt.dataDir)

			err := run(context.Background(), slog.New(slog.DiscardHandler))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("run() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// freePort returns a TCP port that is free right now.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

// TestServeFails checks that a server that cannot accept connections is
// reported as an error instead of waiting forever for a signal.
func TestServeFails(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// A closed listener makes Serve fail at once.
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{ReadHeaderTimeout: readHeaderTimeout}

	err = serve(context.Background(), slog.New(slog.DiscardHandler), srv, ln)
	if err == nil || !strings.HasPrefix(err.Error(), "serve: ") {
		t.Errorf("serve() error = %v, want a serve error", err)
	}
}

// copyDatasets copies the given datasets of the real data directory to a new
// temporary directory and returns its path.
func copyDatasets(t *testing.T, datasets ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range datasets {
		if err := os.CopyFS(filepath.Join(dir, d), os.DirFS(filepath.Join(realDataDir, d))); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
