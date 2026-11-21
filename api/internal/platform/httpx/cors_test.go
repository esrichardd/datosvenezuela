package httpx

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
	}{
		{
			name: "success",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				JSON(w, r, http.StatusOK, "ok")
			}),
			wantStatus: http.StatusOK,
		},
		{
			// Errors must carry the headers too, or browsers hide them.
			name: "error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				NotFound(w, r, "state %s not found", "VE99")
			}),
			wantStatus: http.StatusNotFound,
		},
		{
			name: "recovered panic",
			handler: Recover(slog.New(slog.DiscardHandler))(http.HandlerFunc(
				func(http.ResponseWriter, *http.Request) { panic("boom") },
			)),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", "https://example.com")
			CORS(tt.handler).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
			}
			if got := rec.Header().Get("Access-Control-Expose-Headers"); got != RequestIDHeader {
				t.Errorf("Access-Control-Expose-Headers = %q, want %s", got, RequestIDHeader)
			}
		})
	}
}
