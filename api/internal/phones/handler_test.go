package phones

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx/httpxtest"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

func TestHandlerSuccess(t *testing.T) {
	h := httpxtest.Handler(NewHandler(loadValid(t)))

	tests := []struct {
		path string
		want string
	}{
		{
			// Without type, the list mixes types and each item only has the
			// fields of its own type.
			path: "/v1/phone-prefixes?limit=3&offset=1",
			want: `{
				"data": [
					{"type": "landline", "code": "275", "states": [
						{"code": "VE14", "name": "Mérida"},
						{"code": "VE20", "name": "Táchira"}
					]},
					{"type": "mobile", "code": "412", "operator": "Digitel"},
					{"type": "mobile", "code": "414", "operator": "Movistar"}
				],
				"meta": {"total": 5, "limit": 3, "offset": 1}
			}`,
		},
		{
			path: "/v1/phone-prefixes?type=non_geographic",
			want: `{
				"data": [{"type": "non_geographic", "code": "800", "description": "Acceso universal"}],
				"meta": {"total": 1, "limit": 100, "offset": 0}
			}`,
		},
		{
			path: "/v1/phone-prefixes/414",
			want: `{"data": {"type": "mobile", "code": "414", "operator": "Movistar"}}`,
		},
		{
			path: "/v1/states/ve15/phone-prefixes",
			want: `{
				"data": [
					{"type": "landline", "code": "212", "states": [
						{"code": "VE01", "name": "Distrito Capital"},
						{"code": "VE15", "name": "Miranda"}
					]}
				],
				"meta": {"total": 1, "limit": 100, "offset": 0}
			}`,
		},
		{
			path: "/v1/states/VE25/phone-prefixes",
			want: `{"data": [], "meta": {"total": 0, "limit": 100, "offset": 0}}`,
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
		wantParams []httpx.InvalidParam
	}{
		{path: "/v1/phone-prefixes/999", wantStatus: 404, wantDetail: "phone prefix 999 not found"},
		{path: "/v1/phone-prefixes/0414", wantStatus: 404, wantDetail: "phone prefix 0414 not found"},
		{path: "/v1/states/VE99/phone-prefixes", wantStatus: 404, wantDetail: "state VE99 not found"},
		{
			path:       "/v1/phone-prefixes?type=satellite&limit=0",
			wantStatus: 400,
			wantDetail: "invalid request parameters",
			wantParams: []httpx.InvalidParam{
				{Name: "limit", Reason: "must be an integer between 1 and 500"},
				{Name: "type", Reason: "must be one of: landline, mobile, non_geographic"},
			},
		},
		{
			path:       "/v1/states/VE15/phone-prefixes?offset=-1",
			wantStatus: 400,
			wantDetail: "invalid request parameters",
			wantParams: []httpx.InvalidParam{
				{Name: "offset", Reason: "must be an integer greater than or equal to 0"},
			},
		},
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
			if len(p.InvalidParams) != len(tt.wantParams) {
				t.Fatalf("invalid_params = %+v, want %+v", p.InvalidParams, tt.wantParams)
			}
			for i := range tt.wantParams {
				if p.InvalidParams[i] != tt.wantParams[i] {
					t.Errorf("invalid_params = %+v, want %+v", p.InvalidParams, tt.wantParams)
				}
			}
		})
	}
}

// failingStore simulates a data source that breaks, such as a database that
// went down: every method fails.
type failingStore struct{}

var _ reader = failingStore{}

func (failingStore) Prefixes(context.Context, Type, store.Page) ([]Prefix, int, error) {
	return nil, 0, errBoom
}

func (failingStore) Prefix(context.Context, string) (Prefix, error) { return Prefix{}, errBoom }

func (failingStore) PrefixesOf(context.Context, string, store.Page) ([]Prefix, int, error) {
	return nil, 0, errBoom
}

// TestHandlerStoreFailure checks that every route answers a failing data
// source with a 500 that does not leak the internal error.
func TestHandlerStoreFailure(t *testing.T) {
	h := httpxtest.Handler(NewHandler(failingStore{}))

	for _, path := range []string{"/v1/phone-prefixes", "/v1/phone-prefixes/414", "/v1/states/VE15/phone-prefixes"} {
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
