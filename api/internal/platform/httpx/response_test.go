package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/esrichardd/apivenezuela/api/internal/platform/store"
)

// decode parses the JSON body of a recorded response into a T.
func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestWriteItem(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteItem(rec, httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"code": "VE14"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := rec.Body.String(); body != `{"data":{"code":"VE14"}}`+"\n" {
		t.Errorf("body = %q", body)
	}
}

func TestWriteList(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteList(rec, httptest.NewRequest(http.MethodGet, "/", nil), []string{"a", "b"}, 7, store.Page{Limit: 2, Offset: 4})

	want := `{"data":["a","b"],"meta":{"total":7,"limit":2,"offset":4}}` + "\n"
	if body := rec.Body.String(); body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestWriteListEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteList[string](rec, httptest.NewRequest(http.MethodGet, "/", nil), nil, 0, store.Page{Limit: 100})

	if body := rec.Body.String(); !strings.HasPrefix(body, `{"data":[],`) {
		t.Errorf("body = %q, want data to be an empty array", body)
	}
}

// TestJSONEncodingError checks that a value that cannot be encoded is answered
// like any other internal error: a 500 problem, not plain text.
func TestJSONEncodingError(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, http.StatusOK, func() {}) // functions have no JSON form
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	got := decode[Problem](t, rec)
	if got.Status != http.StatusInternalServerError || got.Title != "Internal Server Error" ||
		got.Detail != "an unexpected error occurred" {
		t.Errorf("problem = %+v, want a 500 with the generic detail", got)
	}
	if got.RequestID == "" || got.RequestID != rec.Header().Get(RequestIDHeader) {
		t.Errorf("request_id = %q, want it to match the %s header", got.RequestID, RequestIDHeader)
	}
}

func TestMap(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(n int) string { return fmt.Sprint(n * 10) })
	if want := []string{"10", "20", "30"}; !slices.Equal(got, want) {
		t.Errorf("Map = %v, want %v", got, want)
	}

	// An empty input must give an empty slice, not nil, so it encodes as [].
	if got := Map[int, string](nil, func(int) string { return "" }); got == nil || len(got) != 0 {
		t.Errorf("Map(nil) = %#v, want an empty non-nil slice", got)
	}
}
