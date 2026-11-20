package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
)

// TestRoutes checks the assembled API with the real data: routes are
// registered and the middleware is in place.
func TestRoutes(t *testing.T) {
	st, err := loadStores(context.Background(), realDataDir)
	if err != nil {
		t.Fatalf("loadStores() error = %v", err)
	}
	var logs bytes.Buffer
	h := newHandler(slog.New(slog.NewJSONHandler(&logs, nil)), st)

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
		{"GET", "/v1/banks?limit=1", 200, "application/json", `"total":27`},
		{"GET", "/v1/banks/0102", 200, "application/json", `"name":"Banco de Venezuela"`},
		{"GET", "/v1/phone-prefixes?limit=1", 200, "application/json", `"total":68`},
		{"GET", "/v1/phone-prefixes/414", 200, "application/json", `"operator":"Movistar"`},
		{"GET", "/v1/states/VE23/phone-prefixes?limit=1", 200, "application/json", `"total":9`},
		{"GET", "/v1/holidays/national?year=2026&limit=1", 200, "application/json", `"total":14`},
		{"GET", "/v1/holidays/bank?year=2026&limit=1", 200, "application/json", `"total":22`},
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
