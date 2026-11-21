package httpx

import "net/http"

// CORS lets web pages on any origin call the API from the browser and read
// its responses (Cross-Origin Resource Sharing).
//
// Allowing every origin is safe here: the API is public, read-only and uses
// no credentials (cookies or auth headers), so a page cannot read anything
// on a visitor's behalf that it could not fetch itself.
//
// Browsers only expose a few response headers to scripts by default, so
// X-Request-ID is listed explicitly.
//
// Preflight requests (OPTIONS) are not handled: browsers send them only for
// non-simple requests, and every endpoint is a plain GET.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Expose-Headers", RequestIDHeader)
		next.ServeHTTP(w, r)
	})
}
