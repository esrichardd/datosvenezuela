package main

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
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

// TestServe runs the server on a free port and checks its whole life: it
// answers requests, and canceling the context shuts it down cleanly.
func TestServe(t *testing.T) {
	// Port 0 asks the operating system for any free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: readHeaderTimeout}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var logs bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- serve(ctx, slog.New(slog.NewJSONHandler(&logs, nil)), srv, ln) }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/health")
	if err != nil {
		t.Fatalf("GET /health error = %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	// Canceling the context is what a SIGTERM does in run.
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve() error = %v, want nil after a clean shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve() did not return after the context was canceled")
	}

	// serve has returned, so nothing else is writing to logs.
	if !strings.Contains(logs.String(), "server stopped") {
		t.Errorf("logs = %s, want them to report the shutdown", logs.String())
	}
	if conn, err := net.Dial("tcp", ln.Addr().String()); err == nil {
		_ = conn.Close()
		t.Error("the port is still open after the shutdown")
	}
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
