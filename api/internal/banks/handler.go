package banks

import (
	"context"
	"net/http"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

// reader is the port the handlers depend on. CSVStore implements it; any
// other data source (a database, a test fake) only needs these methods.
type reader interface {
	Banks(ctx context.Context, p store.Page) ([]Bank, int, error)
	Bank(ctx context.Context, code string) (Bank, error)
}

// Handler serves the banks endpoints.
type Handler struct {
	reader reader
}

// NewHandler returns a Handler that reads from r.
func NewHandler(r reader) *Handler {
	return &Handler{reader: r}
}

// Register adds the banks routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/banks", h.listBanks)
	mux.HandleFunc("GET /v1/banks/{code}", h.getBank)
}

func (h *Handler) listBanks(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	banks, total, err := h.reader.Banks(r.Context(), page)
	if err != nil {
		httpx.StoreError(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, httpx.Map(banks, toBankJSON), total, page)
}

func (h *Handler) getBank(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	b, err := h.reader.Bank(r.Context(), code)
	if err != nil {
		httpx.StoreError(w, r, err, "bank %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toBankJSON(b))
}

// bankJSON is the representation defined in openapi.yaml. It keeps the wire
// format out of the domain model.
type bankJSON struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	LegalName string `json:"legal_name"`
}

// toBankJSON converts directly: both types have the same fields, and the
// compiler rejects the conversion if they ever drift apart.
func toBankJSON(b Bank) bankJSON {
	return bankJSON(b)
}
