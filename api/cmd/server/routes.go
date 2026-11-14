package main

import (
	"net/http"

	"github.com/esrichardd/apivenezuela/api/internal/geo"
	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx"
)

// newHandler builds the HTTP handler of the API: every route and the
// middleware around them. It is the one place to see what the API serves.
func newHandler(geoStore *geo.CSVStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	geo.NewHandler(geoStore).Register(mux)

	// Middleware runs outside-in: RequestID is first, so even the router's
	// own 404 and 405 responses carry a request ID.
	var h http.Handler = mux
	h = httpx.RouteErrors(h)
	h = httpx.RequestID(h)
	return h
}

// handleHealth reports that the server is up. Load balancers and Docker use it
// to decide whether to send traffic to this instance.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}
