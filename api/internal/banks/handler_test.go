package banks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx/httpxtest"
	"github.com/esrichardd/apivenezuela/api/internal/platform/store"
)

func TestHandlerSuccess(t *testing.T) {
	h := httpxtest.Handler(NewHandler(loadValid(t)))

	tests := []struct {
		path string
		want string
	}{
		{
			path: "/v1/banks?limit=2&offset=1",
			want: `{
				"data": [
					{"code": "0102", "name": "Banco de Venezuela", "legal_name": "Banco de Venezuela, S.A. Banco Universal"},
					{"code": "0105", "name": "Banco Mercantil", "legal_name": "Mercantil, C.A., Banco Universal"}
				],
				"meta": {"total": 3, "limit": 2, "offset": 1}
			}`,
		},
		{
			path: "/v1/banks/0001",
			want: `{"data": {"code": "0001", "name": "Banco Central de Venezuela", "legal_name": "Banco Central de Venezuela"}}`,
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
	h := httpxtest.Handler(NewHandler(loadValid(t)))

	tests := []struct {
		path       string
		wantStatus int
		wantDetail string
	}{
		{"/v1/banks/9999", 404, "bank 9999 not found"},
		{"/v1/banks/102", 404, "bank 102 not found"},
		{"/v1/banks?limit=0", 400, "invalid request parameters"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httpxtest.Get(h, tt.path)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body)
			}
			var p httpx.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", p.Detail, tt.wantDetail)
			}
		})
	}
}

// failingStore simulates a data source that breaks, such as a database that
// went down: every method fails.
type failingStore struct{}

var _ reader = failingStore{}

var errBoom = errors.New("connection refused to 10.0.0.5:5432")

func (failingStore) Banks(context.Context, store.Page) ([]Bank, int, error) { return nil, 0, errBoom }
func (failingStore) Bank(context.Context, string) (Bank, error)             { return Bank{}, errBoom }

// TestHandlerStoreFailure checks that every route answers a failing data
// source with a 500 that does not leak the internal error.
func TestHandlerStoreFailure(t *testing.T) {
	h := httpxtest.Handler(NewHandler(failingStore{}))

	for _, path := range []string{"/v1/banks", "/v1/banks/0102"} {
		t.Run(path, func(t *testing.T) {
			rec := httpxtest.Get(h, path)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "10.0.0.5") {
				t.Errorf("response leaks the internal error: %s", rec.Body)
			}
		})
	}
}
