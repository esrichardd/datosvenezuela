package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

func TestProblems(t *testing.T) {
	tests := []struct {
		name  string
		write func(http.ResponseWriter, *http.Request)
		want  Problem
	}{
		{
			name: "bad request",
			write: func(w http.ResponseWriter, r *http.Request) {
				BadRequest(w, r, []InvalidParam{{Name: "limit", Reason: "too big"}})
			},
			want: Problem{
				Type: "about:blank", Title: "Bad Request", Status: 400,
				Detail:        "invalid request parameters",
				InvalidParams: []InvalidParam{{Name: "limit", Reason: "too big"}},
			},
		},
		{
			name: "not found",
			write: func(w http.ResponseWriter, r *http.Request) {
				NotFound(w, r, "state %s not found", "VE99")
			},
			want: Problem{Type: "about:blank", Title: "Not Found", Status: 404, Detail: "state VE99 not found"},
		},
		{
			name: "internal error hides the cause",
			write: func(w http.ResponseWriter, r *http.Request) {
				InternalError(w, r, errors.New("database password is hunter2"))
			},
			want: Problem{
				Type: "about:blank", Title: "Internal Server Error", Status: 500,
				Detail: "an unexpected error occurred",
			},
		},
		{
			name: "store not found",
			write: func(w http.ResponseWriter, r *http.Request) {
				err := fmt.Errorf("lookup: %w", store.ErrNotFound)
				StoreError(w, r, err, "state %s not found", "VE99")
			},
			want: Problem{Type: "about:blank", Title: "Not Found", Status: 404, Detail: "state VE99 not found"},
		},
		{
			// Without a not-found detail, ErrNotFound was not expected: a bug.
			name: "store not found without detail",
			write: func(w http.ResponseWriter, r *http.Request) {
				StoreError(w, r, store.ErrNotFound, "")
			},
			want: Problem{
				Type: "about:blank", Title: "Internal Server Error", Status: 500,
				Detail: "an unexpected error occurred",
			},
		},
		{
			name: "store failure hides the cause",
			write: func(w http.ResponseWriter, r *http.Request) {
				StoreError(w, r, errors.New("connection refused to 10.0.0.5:5432"), "state %s not found", "VE14")
			},
			want: Problem{
				Type: "about:blank", Title: "Internal Server Error", Status: 500,
				Detail: "an unexpected error occurred",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequestID(http.HandlerFunc(tt.write)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if rec.Code != tt.want.Status {
				t.Errorf("status = %d, want %d", rec.Code, tt.want.Status)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
			}

			got := decode[Problem](t, rec)
			if got.RequestID == "" || got.RequestID != rec.Header().Get(RequestIDHeader) {
				t.Errorf("request_id = %q, want it to match the %s header", got.RequestID, RequestIDHeader)
			}
			got.RequestID = ""
			if got.Type != tt.want.Type || got.Title != tt.want.Title || got.Status != tt.want.Status ||
				got.Detail != tt.want.Detail || !slices.Equal(got.InvalidParams, tt.want.InvalidParams) {
				t.Errorf("problem = %+v, want %+v", got, tt.want)
			}
		})
	}
}
