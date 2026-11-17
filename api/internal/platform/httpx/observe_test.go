package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestLogger returns a JSON logger that writes into buf.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, nil))
}

// logLines decodes every JSON line written to buf.
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		lines = append(lines, m)
	}
	return lines
}

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, http.StatusOK, "ok")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestAccessLog(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, http.StatusOK, map[string]string{"a": "b"})
	})
	mux.HandleFunc("GET /fail", func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, http.StatusInternalServerError, "boom")
	})
	mux.HandleFunc("GET /empty", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /implicit", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hi")) // no WriteHeader: net/http sends 200
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, http.StatusOK, "ok")
	})
	h := RequestID(AccessLog(newTestLogger(&buf), "/health")(mux))

	requests := []struct{ path, remoteAddr string }{
		{"/ok", "203.0.113.7:51234"},
		{"/fail", "203.0.113.7:51234"},
		{"/empty", "203.0.113.7:51234"},
		{"/health", "203.0.113.7:51234"},
		// Some connections, such as Unix sockets, have no port.
		{"/implicit", "203.0.113.7"},
	}
	for _, rq := range requests {
		req := httptest.NewRequest(http.MethodGet, rq.path, nil)
		req.RemoteAddr = rq.remoteAddr
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	lines := logLines(t, &buf)
	if len(lines) != 4 {
		t.Fatalf("got %d log lines, want 4 (/health is skipped):\n%s", len(lines), buf.String())
	}

	ok := lines[0]
	if ok["msg"] != "request" || ok["level"] != "INFO" || ok["method"] != "GET" || ok["path"] != "/ok" ||
		ok["status"] != float64(200) || ok["bytes"] != float64(10) || ok["ip"] != "203.0.113.7" {
		t.Errorf("log line = %v", ok)
	}
	if id, _ := ok["request_id"].(string); len(id) != 32 {
		t.Errorf("request_id = %v, want a request ID", ok["request_id"])
	}
	if _, isNumber := ok["duration_ms"].(float64); !isNumber {
		t.Errorf("duration_ms = %v, want a number", ok["duration_ms"])
	}

	if fail := lines[1]; fail["level"] != "ERROR" || fail["status"] != float64(500) {
		t.Errorf("5xx log line = %v, want level ERROR and status 500", fail)
	}
	if empty := lines[2]; empty["status"] != float64(200) || empty["bytes"] != float64(0) {
		t.Errorf("empty response log line = %v, want status 200 and 0 bytes", empty)
	}
	if implicit := lines[3]; implicit["status"] != float64(200) || implicit["bytes"] != float64(2) ||
		implicit["ip"] != "203.0.113.7" {
		t.Errorf("implicit status log line = %v, want status 200, 2 bytes and ip 203.0.113.7", implicit)
	}
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	h := RequestID(Recover(newTestLogger(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("database password is hunter2")
	})))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/states", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("response leaks the panic: %s", rec.Body)
	}

	lines := logLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1", len(lines))
	}
	entry := lines[0]
	stack, _ := entry["stack"].(string)
	if entry["msg"] != "panic" || entry["level"] != "ERROR" ||
		!strings.Contains(entry["error"].(string), "hunter2") ||
		entry["request_id"] != rec.Header().Get(RequestIDHeader) ||
		!strings.Contains(stack, "observe_test.go") {
		t.Errorf("log entry = %v", entry)
	}
}

// TestRecoverWithoutPanic checks that a normal response goes through
// untouched and nothing is logged.
func TestRecoverWithoutPanic(t *testing.T) {
	var buf bytes.Buffer
	h := Recover(newTestLogger(&buf))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, http.StatusCreated, "ok")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusCreated || rec.Body.String() != `"ok"`+"\n" {
		t.Errorf("response = %d %q, want 201 \"ok\"", rec.Code, rec.Body)
	}
	if buf.Len() != 0 {
		t.Errorf("logged %s, want nothing", buf.String())
	}
}

// TestRecoverUnwrap checks that http.ResponseController can reach the
// underlying writer through the recorder, so handlers can still flush.
func TestRecoverUnwrap(t *testing.T) {
	h := Recover(slog.New(slog.DiscardHandler))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestRecoverAfterHeaderWritten(t *testing.T) {
	var buf bytes.Buffer
	h := Recover(newTestLogger(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("late failure")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the 200 already sent", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want nothing appended after the status was sent", rec.Body)
	}
	if !strings.Contains(buf.String(), "late failure") {
		t.Error("the panic was not logged")
	}
}

func TestRecoverLetsAbortHandlerThrough(t *testing.T) {
	var buf bytes.Buffer
	h := Recover(newTestLogger(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if v := recover(); v != http.ErrAbortHandler { //nolint:errorlint // recover returns the exact value.
			t.Errorf("recovered %v, want http.ErrAbortHandler to be re-panicked", v)
		}
		if buf.Len() != 0 {
			t.Errorf("ErrAbortHandler was logged: %s", buf.String())
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
