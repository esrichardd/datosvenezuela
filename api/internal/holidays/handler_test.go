package holidays

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx/httpxtest"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

func TestHandlerSuccess(t *testing.T) {
	h := httpxtest.Handler(NewHandler(load(t, dataset{})))

	tests := []struct {
		path string
		want string
	}{
		{
			path: "/v1/holidays/national?year=2026&limit=2&offset=1",
			want: `{
				"data": [
					{"date": "2026-02-16", "name": "Lunes de Carnaval", "legal_basis": "LOTTT art. 184"},
					{"date": "2026-04-03", "name": "Viernes Santo", "legal_basis": "LOTTT art. 184"}
				],
				"meta": {"total": 5, "limit": 2, "offset": 1}
			}`,
		},
		{
			path: "/v1/holidays/bank?year=2026",
			want: `{
				"data": [
					{"date": "2026-01-01", "name": "Año Nuevo"},
					{"date": "2026-02-16", "name": "Lunes de Carnaval"},
					{"date": "2026-03-19", "name": "Día de San José"}
				],
				"meta": {"total": 3, "limit": 100, "offset": 0}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httpxtest.Get(h, tt.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
			}
			httpxtest.AssertJSON(t, rec.Body.String(), tt.want)
		})
	}
}

func TestHandlerErrors(t *testing.T) {
	h := httpxtest.Handler(NewHandler(load(t, dataset{})))
	withoutBank := httpxtest.Handler(NewHandler(load(t, dataset{bank: map[string]string{}})))

	const nationalRange = "must be an integer between 2013 and 2030"
	tests := []struct {
		name       string
		handler    http.Handler
		path       string
		wantStatus int
		wantDetail string
		wantParams []httpx.InvalidParam
	}{
		{
			name: "national without year", handler: h, path: "/v1/holidays/national", wantStatus: 400,
			wantParams: []httpx.InvalidParam{{Name: "year", Reason: "is required"}},
		},
		{
			name: "national year out of range", handler: h, path: "/v1/holidays/national?year=2031", wantStatus: 400,
			wantParams: []httpx.InvalidParam{{Name: "year", Reason: nationalRange}},
		},
		{
			name: "national year not a number", handler: h, path: "/v1/holidays/national?year=abc&limit=0", wantStatus: 400,
			wantParams: []httpx.InvalidParam{
				{Name: "limit", Reason: "must be an integer between 1 and 500"},
				{Name: "year", Reason: nationalRange},
			},
		},
		{
			name: "bank without year", handler: h, path: "/v1/holidays/bank", wantStatus: 400,
			wantParams: []httpx.InvalidParam{{Name: "year", Reason: "is required"}},
		},
		{
			name: "bank year not a number", handler: h, path: "/v1/holidays/bank?year=abc", wantStatus: 400,
			wantParams: []httpx.InvalidParam{{Name: "year", Reason: "must be an integer"}},
		},
		{
			name: "bank year not published", handler: h, path: "/v1/holidays/bank?year=2027", wantStatus: 404,
			wantDetail: "bank calendar for 2027 is not available; available years: 2026",
		},
		{
			name: "no bank year published", handler: withoutBank, path: "/v1/holidays/bank?year=2026", wantStatus: 404,
			wantDetail: "bank calendar for 2026 is not available; available years: none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httpxtest.Get(tt.handler, tt.path)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body)
			}
			var p httpx.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if tt.wantDetail != "" && p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", p.Detail, tt.wantDetail)
			}
			if !slices.Equal(p.InvalidParams, tt.wantParams) {
				t.Errorf("invalid_params = %+v, want %+v", p.InvalidParams, tt.wantParams)
			}
		})
	}
}

// failingStore simulates a data source that breaks, such as a database that
// went down. Bank fails with bankErr; the other methods always fail.
type failingStore struct {
	bankErr error
}

var _ reader = failingStore{}

var errBoom = errors.New("connection refused to 10.0.0.5:5432")

func (failingStore) National(context.Context, int, store.Page) ([]Holiday, int, error) {
	return nil, 0, errBoom
}

func (f failingStore) Bank(context.Context, int, store.Page) ([]Holiday, int, error) {
	return nil, 0, f.bankErr
}

func (failingStore) BankYears(context.Context) ([]int, error) { return nil, errBoom }

// TestHandlerStoreFailure checks that every route answers a failing data
// source with a 500 that does not leak the internal error.
func TestHandlerStoreFailure(t *testing.T) {
	tests := []struct {
		name  string
		store failingStore
		path  string
	}{
		{"national", failingStore{}, "/v1/holidays/national?year=2026"},
		{"bank", failingStore{bankErr: errBoom}, "/v1/holidays/bank?year=2026"},
		// The year is missing, and listing the available ones fails too.
		{"bank years", failingStore{bankErr: store.ErrNotFound}, "/v1/holidays/bank?year=2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httpxtest.Get(httpxtest.Handler(NewHandler(tt.store)), tt.path)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "10.0.0.5") {
				t.Errorf("response leaks the internal error: %s", rec.Body)
			}
		})
	}
}
