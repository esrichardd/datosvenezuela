package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"slices"
	"time"
)

// SecurityHeaders sets headers that harden every response.
//
// X-Content-Type-Options: nosniff tells browsers to trust Content-Type
// instead of guessing the type from the body, so a JSON response that echoes
// user input is never run as HTML or script.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// AccessLog logs one line per request with its outcome and duration.
// Requests to skipPaths (such as health checks) are not logged.
func AccessLog(logger *slog.Logger, skipPaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if slices.Contains(skipPaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rec := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			if rec.status() >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			logger.LogAttrs(r.Context(), level, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status()),
				slog.Int("bytes", rec.bytes),
				slog.Float64("duration_ms", milliseconds(time.Since(start))),
				slog.String("request_id", RequestIDFrom(r.Context())),
				slog.String("ip", remoteIP(r)),
			)
		})
	}
}

// Recover turns a panic in a handler into a logged error and a 500 problem
// response, instead of a connection closed without an answer.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &responseRecorder{ResponseWriter: w}
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				// ErrAbortHandler is how a handler deliberately aborts a
				// response; net/http handles it quietly, so let it through.
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}

				logger.ErrorContext(r.Context(), "panic",
					"error", fmt.Sprint(v),
					"request_id", RequestIDFrom(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				// If the handler already started the response, its status
				// line is gone; the client will see a truncated body.
				if !rec.wroteHeader {
					WriteProblem(rec, r, http.StatusInternalServerError, internalErrorDetail)
				}
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

// responseRecorder remembers the status and size of a response as it is
// written. net/http does not expose them otherwise.
type responseRecorder struct {
	http.ResponseWriter
	code        int
	bytes       int
	wroteHeader bool
}

func (rec *responseRecorder) WriteHeader(status int) {
	if !rec.wroteHeader {
		rec.code = status
		rec.wroteHeader = true
	}
	rec.ResponseWriter.WriteHeader(status)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		// Write without WriteHeader means 200, as in net/http.
		rec.WriteHeader(http.StatusOK)
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += n
	return n, err
}

// status returns the response status; 200 if the handler wrote nothing.
func (rec *responseRecorder) status() int {
	if !rec.wroteHeader {
		return http.StatusOK
	}
	return rec.code
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (rec *responseRecorder) Unwrap() http.ResponseWriter {
	return rec.ResponseWriter
}

// remoteIP returns the IP of the connection. Behind a proxy this is the
// proxy's IP; reading the client IP from proxy headers comes with the
// deployment setup.
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// milliseconds returns d in milliseconds, rounded to microseconds.
func milliseconds(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}
