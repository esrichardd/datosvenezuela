package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

// RequestIDHeader is the response header that carries the request ID.
const RequestIDHeader = "X-Request-ID"

type requestIDKey struct{}

// RequestID gives every request a new random ID, available through
// RequestIDFrom and returned in the X-Request-ID header.
//
// An incoming X-Request-ID is ignored: clients could send any value and
// spoof or flood the logs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom returns the request ID stored in ctx, or "" if there is none.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// newRequestID returns 16 random bytes encoded as 32 hex characters.
func newRequestID() string {
	b := make([]byte, 16)
	// crypto/rand.Read never returns an error (Go 1.24+).
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RouteErrors replaces the plain-text 404 and 405 responses of
// http.ServeMux with problem details. The Allow header of a 405 is kept.
//
// Responses written by handlers are not touched: they never use text/plain.
func RouteErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&routeErrorWriter{ResponseWriter: w, r: r}, r)
	})
}

// routeErrorWriter intercepts the mux's own error responses, which it writes
// with http.Error (Content-Type text/plain).
type routeErrorWriter struct {
	http.ResponseWriter
	r           *http.Request
	intercepted bool
}

func (w *routeErrorWriter) WriteHeader(status int) {
	isRouteError := status == http.StatusNotFound || status == http.StatusMethodNotAllowed
	if isRouteError && strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
		w.intercepted = true
		// Drop the headers http.Error set for its text body.
		w.Header().Del("X-Content-Type-Options")
		detail := "no route matches " + w.r.URL.Path
		if status == http.StatusMethodNotAllowed {
			detail = "method " + w.r.Method + " is not allowed on " + w.r.URL.Path
		}
		WriteProblem(w.ResponseWriter, w.r, status, detail)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *routeErrorWriter) Write(b []byte) (int, error) {
	if w.intercepted {
		// Discard the mux's text body; the problem was already written.
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *routeErrorWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
