package geo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
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
			path: "/v1/states?limit=2",
			want: `{
				"data": [
					{"code": "VE01", "name": "Distrito Capital", "type": "capital_district", "iso_3166_2": "VE-A", "capital": "Caracas"},
					{"code": "VE14", "name": "Mérida", "type": "state", "iso_3166_2": "VE-L", "capital": "Mérida"}
				],
				"meta": {"total": 3, "limit": 2, "offset": 0}
			}`,
		},
		{
			path: "/v1/states/VE25",
			want: `{"data": {"code": "VE25", "name": "Dependencias Federales", "type": "federal_dependencies", "iso_3166_2": "VE-W", "capital": null}}`,
		},
		{
			path: "/v1/states/ve14",
			want: `{"data": {"code": "VE14", "name": "Mérida", "type": "state", "iso_3166_2": "VE-L", "capital": "Mérida"}}`,
		},
		{
			path: "/v1/states/VE14/municipalities",
			want: `{
				"data": [
					{"code": "VE1401", "name": "Alberto Adriani"},
					{"code": "VE1412", "name": "Libertador"}
				],
				"meta": {"total": 2, "limit": 100, "offset": 0}
			}`,
		},
		{
			path: "/v1/states/VE25/municipalities",
			want: `{"data": [], "meta": {"total": 0, "limit": 100, "offset": 0}}`,
		},
		{
			path: "/v1/municipalities?offset=2",
			want: `{
				"data": [
					{"code": "VE1412", "name": "Libertador", "state": {"code": "VE14", "name": "Mérida"}}
				],
				"meta": {"total": 3, "limit": 100, "offset": 2}
			}`,
		},
		{
			path: "/v1/municipalities/VE1412",
			want: `{"data": {"code": "VE1412", "name": "Libertador", "state": {"code": "VE14", "name": "Mérida"}}}`,
		},
		{
			path: "/v1/municipalities/VE1412/parishes",
			want: `{
				"data": [
					{"code": "VE141201", "name": "Antonio Spinetti Dini"},
					{"code": "VE141202", "name": "Arias"}
				],
				"meta": {"total": 2, "limit": 100, "offset": 0}
			}`,
		},
		{
			path: "/v1/parishes?limit=1",
			want: `{
				"data": [
					{
						"code": "VE010101", "name": "Altagracia",
						"municipality": {"code": "VE0101", "name": "Libertador"},
						"state": {"code": "VE01", "name": "Distrito Capital"}
					}
				],
				"meta": {"total": 5, "limit": 1, "offset": 0}
			}`,
		},
		{
			path: "/v1/parishes/VE141201",
			want: `{
				"data": {
					"code": "VE141201", "name": "Antonio Spinetti Dini",
					"municipality": {"code": "VE1412", "name": "Libertador"},
					"state": {"code": "VE14", "name": "Mérida"}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httpxtest.Get(h, tt.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
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
		{"/v1/states/VE99", 404, "state VE99 not found"},
		{"/v1/states/abc", 404, "state abc not found"},
		{"/v1/states/VE99/municipalities", 404, "state VE99 not found"},
		{"/v1/municipalities/VE9999", 404, "municipality VE9999 not found"},
		{"/v1/municipalities/VE9999/parishes", 404, "municipality VE9999 not found"},
		{"/v1/parishes/VE999999", 404, "parish VE999999 not found"},
		// Every list route rejects invalid pagination; /v1/places/search is
		// covered by TestSearchPlacesValidation.
		{"/v1/states?limit=0", 400, "invalid request parameters"},
		{"/v1/states/VE14/municipalities?limit=9999", 400, "invalid request parameters"},
		{"/v1/municipalities?limit=0", 400, "invalid request parameters"},
		{"/v1/municipalities/VE1412/parishes?offset=-1", 400, "invalid request parameters"},
		{"/v1/parishes?offset=-1", 400, "invalid request parameters"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httpxtest.Get(h, tt.path)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
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

func (failingStore) States(context.Context, store.Page) ([]State, int, error) {
	return nil, 0, errBoom
}

func (failingStore) State(context.Context, string) (State, error) { return State{}, errBoom }

func (failingStore) Municipalities(context.Context, store.Page) ([]Municipality, int, error) {
	return nil, 0, errBoom
}

func (failingStore) MunicipalitiesOf(context.Context, string, store.Page) ([]Municipality, int, error) {
	return nil, 0, errBoom
}

func (failingStore) Municipality(context.Context, string) (Municipality, error) {
	return Municipality{}, errBoom
}

func (failingStore) Parishes(context.Context, store.Page) ([]Parish, int, error) {
	return nil, 0, errBoom
}

func (failingStore) ParishesOf(context.Context, string, store.Page) ([]Parish, int, error) {
	return nil, 0, errBoom
}

func (failingStore) Parish(context.Context, string) (Parish, error) { return Parish{}, errBoom }

func (failingStore) Search(context.Context, string, PlaceType, store.Page) ([]Place, int, error) {
	return nil, 0, errBoom
}

// TestHandlerStoreFailure checks that every route answers a failing data
// source with a 500 that does not leak the internal error.
func TestHandlerStoreFailure(t *testing.T) {
	h := httpxtest.Handler(NewHandler(failingStore{}))

	paths := []string{
		"/v1/states",
		"/v1/states/VE14",
		"/v1/states/VE14/municipalities",
		"/v1/municipalities",
		"/v1/municipalities/VE1412",
		"/v1/municipalities/VE1412/parishes",
		"/v1/parishes",
		"/v1/parishes/VE141201",
		"/v1/places/search?q=merida",
	}
	for _, path := range paths {
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

func TestSearchPlaces(t *testing.T) {
	h := httpxtest.Handler(NewHandler(loadValid(t)))

	tests := []struct {
		path string
		want string
	}{
		{
			path: "/v1/places/search?q=merida",
			want: `{
				"data": [
					{"type": "state", "code": "VE14", "name": "Mérida"}
				],
				"meta": {"total": 1, "limit": 20, "offset": 0}
			}`,
		},
		{
			path: "/v1/places/search?q=an&type=municipality",
			want: `{
				"data": [
					{"type": "municipality", "code": "VE1401", "name": "Alberto Adriani", "state": {"code": "VE14", "name": "Mérida"}}
				],
				"meta": {"total": 1, "limit": 20, "offset": 0}
			}`,
		},
		{
			path: "/v1/places/search?q=arias&limit=5",
			want: `{
				"data": [
					{
						"type": "parish", "code": "VE141202", "name": "Arias",
						"municipality": {"code": "VE1412", "name": "Libertador"},
						"state": {"code": "VE14", "name": "Mérida"}
					}
				],
				"meta": {"total": 1, "limit": 5, "offset": 0}
			}`,
		},
		{
			path: "/v1/places/search?q=zzz",
			want: `{"data": [], "meta": {"total": 0, "limit": 20, "offset": 0}}`,
		},
		{
			// The longest query allowed.
			path: "/v1/places/search?q=" + strings.Repeat("z", 100),
			want: `{"data": [], "meta": {"total": 0, "limit": 20, "offset": 0}}`,
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

func TestSearchPlacesValidation(t *testing.T) {
	h := httpxtest.Handler(NewHandler(loadValid(t)))
	long := strings.Repeat("a", 101)
	// Only two characters once normalized, but far too long as received.
	padded := strings.Repeat("%20", 150) + "ab"

	tests := []struct {
		path        string
		wantInvalid map[string]string
	}{
		{"/v1/places/search", map[string]string{"q": "is required"}},
		{"/v1/places/search?q=", map[string]string{"q": "must have at least 2 characters"}},
		{"/v1/places/search?q=a", map[string]string{"q": "must have at least 2 characters"}},
		{"/v1/places/search?q=%20%20a%20%20", map[string]string{"q": "must have at least 2 characters"}},
		{"/v1/places/search?q=" + long, map[string]string{"q": "must have at most 100 characters"}},
		{"/v1/places/search?q=" + padded, map[string]string{"q": "must have at most 100 characters"}},
		{"/v1/places/search?q=caracas&type=city", map[string]string{"type": "must be one of: state, municipality, parish"}},
		{
			"/v1/places/search?q=a&type=city&limit=0",
			map[string]string{
				"q":     "must have at least 2 characters",
				"type":  "must be one of: state, municipality, parish",
				"limit": "must be an integer between 1 and 500",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httpxtest.Get(h, tt.path)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
			var p httpx.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			got := make(map[string]string)
			for _, ip := range p.InvalidParams {
				got[ip.Name] = ip.Reason
			}
			if !reflect.DeepEqual(got, tt.wantInvalid) {
				t.Errorf("invalid_params = %v, want %v", got, tt.wantInvalid)
			}
		})
	}
}
