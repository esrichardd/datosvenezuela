package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/esrichardd/apivenezuela/api/internal/geo"
	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx"
)

// realDataDir is the repository's data directory, seen from api/cmd/server.
var realDataDir = filepath.Join("..", "..", "..", "data")

func TestLoadConfig(t *testing.T) {
	existingDir := t.TempDir()
	file := filepath.Join(existingDir, "file.txt")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		env     map[string]string
		want    config
		wantErr string
	}{
		{
			name: "explicit values",
			env:  map[string]string{"PORT": "9000", "DATA_DIR": existingDir},
			want: config{port: 9000, dataDir: existingDir},
		},
		{
			name: "default port",
			env:  map[string]string{"DATA_DIR": existingDir},
			want: config{port: 8080, dataDir: existingDir},
		},
		{
			name:    "invalid port",
			env:     map[string]string{"PORT": "abc", "DATA_DIR": existingDir},
			wantErr: `invalid PORT "abc"`,
		},
		{
			name:    "port out of range",
			env:     map[string]string{"PORT": "70000", "DATA_DIR": existingDir},
			wantErr: `invalid PORT "70000"`,
		},
		{
			name:    "missing data dir",
			env:     map[string]string{"DATA_DIR": filepath.Join(existingDir, "nope")},
			wantErr: "invalid DATA_DIR",
		},
		{
			name:    "data dir is a file",
			env:     map[string]string{"DATA_DIR": file},
			wantErr: "invalid DATA_DIR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start without PORT. t.Setenv records the original value so it
			// is restored when the test ends, even after Unsetenv.
			t.Setenv("PORT", "")
			if err := os.Unsetenv("PORT"); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := loadConfig()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadConfig() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadConfig() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("loadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestRoutes checks the assembled API with the real data: routes are
// registered and the middleware is in place.
func TestRoutes(t *testing.T) {
	geoStore, err := geo.LoadCSV(filepath.Join(realDataDir, "geography"))
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	var logs bytes.Buffer
	h := newHandler(slog.New(slog.NewJSONHandler(&logs, nil)), geoStore)

	tests := []struct {
		method      string
		path        string
		wantStatus  int
		wantType    string
		wantContain string
	}{
		{"GET", "/health", 200, "application/json", `{"status":"ok"}`},
		{"GET", "/v1/states/VE14", 200, "application/json", `"name":"Mérida"`},
		{"GET", "/v1/municipalities/VE1412/parishes?limit=1", 200, "application/json", `"total":15`},
		{"GET", "/v1/places/search?q=san%20cristobal", 200, "application/json", `"total":3`},
		{"GET", "/v1/nope", 404, "application/problem+json", `"status":404`},
		{"POST", "/v1/states", 405, "application/problem+json", `"status":405`},
	}

	idRe := regexp.MustCompile(`^[0-9a-f]{32}$`)
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantContain) {
				t.Errorf("body = %s, want it to contain %s", rec.Body, tt.wantContain)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
			if id := rec.Header().Get(httpx.RequestIDHeader); !idRe.MatchString(id) {
				t.Errorf("%s = %q, want a request ID", httpx.RequestIDHeader, id)
			}
		})
	}

	// Every request is logged except /health.
	if got, want := strings.Count(logs.String(), `"msg":"request"`), len(tests)-1; got != want {
		t.Errorf("logged %d requests, want %d:\n%s", got, want, logs.String())
	}
}

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
	for _, want := range []string{`"msg":"geography loaded"`, `"states":25`, `"msg":"server stopped"`} {
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

	tests := []struct {
		name    string
		port    int
		dataDir string
		wantErr string
	}{
		{name: "invalid config", port: 0, dataDir: realDataDir, wantErr: "invalid PORT"},
		// An existing directory without the datasets.
		{name: "invalid data", port: freePort(t), dataDir: t.TempDir(), wantErr: "load geography: "},
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
