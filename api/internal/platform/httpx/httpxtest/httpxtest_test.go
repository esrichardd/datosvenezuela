package httpxtest

import (
	"net/http"
	"testing"

	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx"
)

// pingHandler is a minimal domain handler with a single route.
type pingHandler struct{}

func (pingHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, r, http.StatusOK, "pong")
	})
}

func TestHandler(t *testing.T) {
	h := Handler(pingHandler{})

	tests := []struct {
		path       string
		wantStatus int
		wantType   string
	}{
		{"/ping", http.StatusOK, "application/json"},
		// Unknown routes get a problem, as in the real server.
		{"/nope", http.StatusNotFound, "application/problem+json"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := Get(h, tt.path)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
			if rec.Header().Get(httpx.RequestIDHeader) == "" {
				t.Errorf("missing %s header", httpx.RequestIDHeader)
			}
		})
	}
}

// recordingTB records whether a check failed instead of failing the test.
type recordingTB struct {
	testing.TB
	failed bool
}

func (r *recordingTB) Helper()               {}
func (r *recordingTB) Errorf(string, ...any) { r.failed = true }
func (r *recordingTB) Fatalf(string, ...any) { r.failed = true }

func TestAssertJSON(t *testing.T) {
	tests := []struct {
		name      string
		got, want string
		wantFail  bool
	}{
		{name: "same document, other spacing and key order", got: `{"a":1,"b":[2]}`, want: `{ "b": [2], "a": 1 }`},
		{name: "different value", got: `{"a":1}`, want: `{"a":2}`, wantFail: true},
		{name: "response is not JSON", got: `oops`, want: `{}`, wantFail: true},
		{name: "expected value is not JSON", got: `{}`, want: `oops`, wantFail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recordingTB{TB: t}
			AssertJSON(rec, tt.got, tt.want)
			if rec.failed != tt.wantFail {
				t.Errorf("failed = %v, want %v", rec.failed, tt.wantFail)
			}
		})
	}
}
