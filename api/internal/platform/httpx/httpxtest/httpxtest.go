// Package httpxtest holds helpers for testing the domain handlers, in the
// spirit of net/http/httptest. Only tests import it.
package httpxtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
)

// Registrar is a domain handler that adds its routes to a mux.
type Registrar interface {
	Register(mux *http.ServeMux)
}

// Handler serves the routes of r behind the middleware that shapes every
// response the way the real server does: request IDs, and problem details
// for unknown routes and methods.
func Handler(r Registrar) http.Handler {
	mux := http.NewServeMux()
	r.Register(mux)
	return httpx.RequestID(httpx.RouteErrors(mux))
}

// Get serves a GET request for path with h and returns the recorded response.
func Get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// AssertJSON fails t if got and want are not the same JSON document. Spacing
// and key order are ignored, so want can be written readably.
func AssertJSON(t testing.TB, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, got)
		return
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("expected value is not JSON: %v", err)
		return
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("body mismatch\n got: %s\nwant: %s", strings.TrimSpace(got), want)
	}
}
