package main

import (
	"log/slog"
	"net/http"

	"github.com/esrichardd/datosvenezuela/api/internal/banks"
	"github.com/esrichardd/datosvenezuela/api/internal/geo"
	"github.com/esrichardd/datosvenezuela/api/internal/holidays"
	"github.com/esrichardd/datosvenezuela/api/internal/phones"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
)

// newHandler builds the HTTP handler of the API: every route and the
// middleware around them. It is the one place to see what the API serves.
func newHandler(logger *slog.Logger, s stores) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	geo.NewHandler(s.geo).Register(mux)
	banks.NewHandler(s.banks).Register(mux)
	phones.NewHandler(s.phones).Register(mux)
	holidays.NewHandler(s.holidays).Register(mux)

	// Middleware is applied inside-out, so a request goes through it in the
	// reverse order of the lines below:
	//
	//	RequestID → SecurityHeaders → AccessLog → Recover → RouteErrors → mux
	//
	// RequestID comes first so every log line and error carries the ID.
	// AccessLog wraps Recover so a panic is logged with its 500.
	var h http.Handler = mux
	h = httpx.RouteErrors(h)
	h = httpx.Recover(logger)(h)
	h = httpx.AccessLog(logger, "/health")(h)
	h = httpx.SecurityHeaders(h)
	h = httpx.RequestID(h)
	return h
}

// handleHealth reports that the server is up. Load balancers and Docker use it
// to decide whether to send traffic to this instance.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}
