// Package httpx holds the HTTP building blocks shared by every domain:
// JSON responses, RFC 9457 problem details, pagination parameters and
// middleware.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Item is the envelope of a single-resource response.
type Item[T any] struct {
	Data T `json:"data"`
}

// List is the envelope of a paginated list response.
type List[T any] struct {
	Data []T      `json:"data"`
	Meta PageMeta `json:"meta"`
}

// PageMeta describes the page returned in a List.
type PageMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// WriteItem writes v wrapped in {"data": ...} with status 200.
func WriteItem[T any](w http.ResponseWriter, r *http.Request, v T) {
	JSON(w, r, http.StatusOK, Item[T]{Data: v})
}

// WriteList writes a page of items wrapped in {"data": [...], "meta": {...}}
// with status 200. total is the number of items across all pages.
func WriteList[T any](w http.ResponseWriter, r *http.Request, items []T, total int, p Page) {
	if items == nil {
		items = []T{} // encode as [] instead of null
	}
	JSON(w, r, http.StatusOK, List[T]{
		Data: items,
		Meta: PageMeta{Total: total, Limit: p.Limit, Offset: p.Offset},
	})
}

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	writeJSON(w, r, status, "application/json", v)
}

// writeJSON encodes v before writing anything, so an encoding failure can
// still be answered with a 500 instead of a truncated body.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, contentType string, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		slog.ErrorContext(r.Context(), "encode JSON response",
			"error", err, "request_id", RequestIDFrom(r.Context()))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	// A write error means the client went away; there is nobody to tell.
	_, _ = w.Write(append(body, '\n'))
}
