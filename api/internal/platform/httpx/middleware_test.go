package httpx

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = RequestIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "spoofed")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	header := rec.Header().Get(RequestIDHeader)
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(header) {
		t.Errorf("%s = %q, want 32 hex characters", RequestIDHeader, header)
	}
	if seen != header {
		t.Errorf("context ID = %q, header = %q; want them equal", seen, header)
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Header().Get(RequestIDHeader) == header {
		t.Error("two requests got the same ID")
	}
}

func TestRouteErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/states/{code}", func(w http.ResponseWriter, r *http.Request) {
		NotFound(w, r, "state %s not found", r.PathValue("code"))
	})
	h := RequestID(RouteErrors(mux))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantDetail string
		wantAllow  string
	}{
		{
			name: "unknown route", method: http.MethodGet, path: "/v1/nope",
			wantStatus: 404, wantDetail: "no route matches /v1/nope",
		},
		{
			name: "wrong method", method: http.MethodPost, path: "/v1/states/VE14",
			wantStatus: 405, wantDetail: "method POST is not allowed on /v1/states/VE14",
			wantAllow: "GET, HEAD",
		},
		{
			name: "handler response is untouched", method: http.MethodGet, path: "/v1/states/VE99",
			wantStatus: 404, wantDetail: "state VE99 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
			}
			if allow := rec.Header().Get("Allow"); allow != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", allow, tt.wantAllow)
			}
			if got := decode[Problem](t, rec); got.Detail != tt.wantDetail || got.RequestID == "" {
				t.Errorf("problem = %+v, want detail %q and a request_id", got, tt.wantDetail)
			}
		})
	}
}

// TestRouteErrorsUnwrap checks that http.ResponseController can reach the
// underlying writer through the wrapper, so handlers can still flush.
func TestRouteErrorsUnwrap(t *testing.T) {
	h := RouteErrors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush() error = %v", err)
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !rec.Flushed {
		t.Error("the flush did not reach the underlying writer")
	}
}
