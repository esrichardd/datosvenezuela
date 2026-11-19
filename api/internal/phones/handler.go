package phones

import (
	"context"
	"net/http"

	"github.com/esrichardd/datosvenezuela/api/internal/geo"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

// reader is the port the handlers depend on. CSVStore implements it; any
// other data source (a database, a test fake) only needs these methods.
type reader interface {
	Prefixes(ctx context.Context, t Type, p store.Page) ([]Prefix, int, error)
	Prefix(ctx context.Context, code string) (Prefix, error)
	PrefixesOf(ctx context.Context, stateCode string, p store.Page) ([]Prefix, int, error)
}

// Handler serves the phone prefixes endpoints.
type Handler struct {
	reader reader
}

// NewHandler returns a Handler that reads from r.
func NewHandler(r reader) *Handler {
	return &Handler{reader: r}
}

// Register adds the phone prefixes routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/phone-prefixes", h.listPrefixes)
	mux.HandleFunc("GET /v1/phone-prefixes/{code}", h.getPrefix)
	mux.HandleFunc("GET /v1/states/{code}/phone-prefixes", h.listStatePrefixes)
}

func (h *Handler) listPrefixes(w http.ResponseWriter, r *http.Request) {
	page, invalid := httpx.ParsePage(r, httpx.DefaultLimit)
	t := Type(r.URL.Query().Get("type"))
	switch t {
	case "", TypeLandline, TypeMobile, TypeNonGeographic:
	default:
		invalid = append(invalid, httpx.InvalidParam{
			Name: "type", Reason: "must be one of: landline, mobile, non_geographic",
		})
	}
	if len(invalid) > 0 {
		httpx.BadRequest(w, r, invalid)
		return
	}

	prefixes, total, err := h.reader.Prefixes(r.Context(), t, page)
	if err != nil {
		httpx.StoreError(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, httpx.Map(prefixes, toPrefixJSON), total, page)
}

func (h *Handler) getPrefix(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	p, err := h.reader.Prefix(r.Context(), code)
	if err != nil {
		httpx.StoreError(w, r, err, "phone prefix %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toPrefixJSON(p))
}

func (h *Handler) listStatePrefixes(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	prefixes, total, err := h.reader.PrefixesOf(r.Context(), code, page)
	if err != nil {
		httpx.StoreError(w, r, err, "state %s not found", code)
		return
	}
	httpx.WriteList(w, r, httpx.Map(prefixes, toPrefixJSON), total, page)
}

// prefixJSON is the representation defined in openapi.yaml. Each type only
// has its own fields: the others are empty and omitted.
type prefixJSON struct {
	Type        string      `json:"type"`
	Code        string      `json:"code"`
	States      []httpx.Ref `json:"states,omitempty"`
	Operator    string      `json:"operator,omitempty"`
	Description string      `json:"description,omitempty"`
}

func toPrefixJSON(p Prefix) prefixJSON {
	return prefixJSON{
		Type:        string(p.Type),
		Code:        p.Code,
		States:      httpx.Map(p.States, toStateRefJSON),
		Operator:    p.Operator,
		Description: p.Description,
	}
}

func toStateRefJSON(st geo.StateRef) httpx.Ref {
	return httpx.Ref(st)
}
