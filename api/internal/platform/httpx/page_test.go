package httpx

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

func TestParsePage(t *testing.T) {
	tests := []struct {
		query       string
		want        store.Page
		wantInvalid []string
	}{
		{query: "", want: store.Page{Limit: 100, Offset: 0}},
		{query: "limit=20&offset=40", want: store.Page{Limit: 20, Offset: 40}},
		{query: "limit=1", want: store.Page{Limit: 1}},
		{query: "limit=500", want: store.Page{Limit: 500}},
		// An invalid parameter is reported and keeps its default, so the page
		// stays within the limits even if the caller ignores the errors.
		{query: "limit=0", want: store.Page{Limit: 100}, wantInvalid: []string{"limit"}},
		{query: "limit=501", want: store.Page{Limit: 100}, wantInvalid: []string{"limit"}},
		{query: "limit=abc", want: store.Page{Limit: 100}, wantInvalid: []string{"limit"}},
		{query: "offset=-1", want: store.Page{Limit: 100}, wantInvalid: []string{"offset"}},
		{query: "limit=0&offset=x", want: store.Page{Limit: 100}, wantInvalid: []string{"limit", "offset"}},
		{query: "limit=9999&offset=40", want: store.Page{Limit: 100, Offset: 40}, wantInvalid: []string{"limit"}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			p, invalid := ParsePage(httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil), DefaultLimit)

			var names []string
			for _, ip := range invalid {
				names = append(names, ip.Name)
			}
			if !slices.Equal(names, tt.wantInvalid) {
				t.Errorf("invalid params = %v, want %v", names, tt.wantInvalid)
			}
			if p != tt.want {
				t.Errorf("page = %+v, want %+v", p, tt.want)
			}
		})
	}
}

func TestReadPage(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		rec := httptest.NewRecorder()
		p, ok := ReadPage(rec, httptest.NewRequest(http.MethodGet, "/?limit=20&offset=40", nil))

		if !ok {
			t.Fatal("ok = false, want true")
		}
		if want := (store.Page{Limit: 20, Offset: 40}); p != want {
			t.Errorf("page = %+v, want %+v", p, want)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("wrote a response for valid parameters: %s", rec.Body)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		rec := httptest.NewRecorder()
		_, ok := ReadPage(rec, httptest.NewRequest(http.MethodGet, "/?limit=0&offset=x", nil))

		if ok {
			t.Fatal("ok = true, want false")
		}
		// Which parameters are invalid is ParsePage's job, tested above.
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}
