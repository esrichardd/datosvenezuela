package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Problem is an RFC 9457 problem details object.
type Problem struct {
	// Type is always "about:blank": the problem means exactly what its HTTP
	// status says.
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
}

// InvalidParam describes one invalid request parameter.
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

const (
	problemContentType = "application/problem+json"

	// internalErrorDetail is the only detail a 500 ever carries. The real
	// cause may contain internal information, so it stays in the logs.
	internalErrorDetail = "an unexpected error occurred"
)

// newProblem builds the problem for status, tagged with the request ID.
func newProblem(r *http.Request, status int, detail string, params ...InvalidParam) Problem {
	return Problem{
		Type:          "about:blank",
		Title:         http.StatusText(status),
		Status:        status,
		Detail:        detail,
		RequestID:     RequestIDFrom(r.Context()),
		InvalidParams: params,
	}
}

// WriteProblem writes an application/problem+json response.
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, detail string, params ...InvalidParam) {
	writeJSON(w, r, status, problemContentType, newProblem(r, status, detail, params...))
}

// BadRequest writes a 400 listing every invalid parameter.
func BadRequest(w http.ResponseWriter, r *http.Request, params []InvalidParam) {
	WriteProblem(w, r, http.StatusBadRequest, "invalid request parameters", params...)
}

// NotFound writes a 404. The detail is formatted with fmt.Sprintf.
func NotFound(w http.ResponseWriter, r *http.Request, format string, args ...any) {
	WriteProblem(w, r, http.StatusNotFound, fmt.Sprintf(format, args...))
}

// InternalError logs err and writes a 500. The client never sees err: it may
// contain internal details. The request ID links the response to the log.
func InternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "internal error",
		"error", err,
		"request_id", RequestIDFrom(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)
	WriteProblem(w, r, http.StatusInternalServerError, internalErrorDetail)
}
