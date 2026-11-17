package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = RequestIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "spoofed")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	header := rec.Header().Get(RequestIDHeader)
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(header) {
		t.Errorf("%s = %q, want 32 hex characters", RequestIDHeader, header)
	}
	if seen != header {
		t.Errorf("context ID = %q, header = %q; want them equal", seen, header)
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Header().Get(RequestIDHeader) == header {
		t.Error("two requests got the same ID")
	}
}

func TestRouteErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/states/{code}", func(w http.ResponseWriter, r *http.Request) {
		NotFound(w, r, "state %s not found", r.PathValue("code"))
	})
	h := RequestID(RouteErrors(mux))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantDetail string
		wantAllow  string
	}{
		{
			name: "unknown route", method: http.MethodGet, path: "/v1/nope",
			wantStatus: 404, wantDetail: "no route matches /v1/nope",
		},
		{
			name: "wrong method", method: http.MethodPost, path: "/v1/states/VE14",
			wantStatus: 405, wantDetail: "method POST is not allowed on /v1/states/VE14",
			wantAllow: "GET, HEAD",
		},
		{
			name: "handler response is untouched", method: http.MethodGet, path: "/v1/states/VE99",
			wantStatus: 404, wantDetail: "state VE99 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
			}
			if allow := rec.Header().Get("Allow"); allow != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", allow, tt.wantAllow)
			}
			if got := decode[Problem](t, rec); got.Detail != tt.wantDetail || got.RequestID == "" {
				t.Errorf("problem = %+v, want detail %q and a request_id", got, tt.wantDetail)
			}
		})
	}
}
