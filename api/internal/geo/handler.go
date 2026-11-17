package geo

import (
	"context"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/apivenezuela/api/internal/platform/store"
	"github.com/esrichardd/apivenezuela/api/internal/platform/textnorm"
)

// Search limits, as documented in openapi.yaml.
const (
	searchDefaultLimit = 20
	searchMinQueryLen  = 2
	searchMaxQueryLen  = 100
)

// reader is the port the handlers depend on. CSVStore implements it; any
// other data source (a database, a test fake) only needs these methods.
type reader interface {
	States(ctx context.Context, p store.Page) ([]State, int, error)
	State(ctx context.Context, code string) (State, error)
	Municipalities(ctx context.Context, p store.Page) ([]Municipality, int, error)
	MunicipalitiesOf(ctx context.Context, stateCode string, p store.Page) ([]Municipality, int, error)
	Municipality(ctx context.Context, code string) (Municipality, error)
	Parishes(ctx context.Context, p store.Page) ([]Parish, int, error)
	ParishesOf(ctx context.Context, municipalityCode string, p store.Page) ([]Parish, int, error)
	Parish(ctx context.Context, code string) (Parish, error)
	Search(ctx context.Context, query string, placeType PlaceType, p store.Page) ([]Place, int, error)
}

// Handler serves the geography endpoints.
type Handler struct {
	reader reader
}

// NewHandler returns a Handler that reads from r.
func NewHandler(r reader) *Handler {
	return &Handler{reader: r}
}

// Register adds the geography routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/states", h.listStates)
	mux.HandleFunc("GET /v1/states/{code}", h.getState)
	mux.HandleFunc("GET /v1/states/{code}/municipalities", h.listStateMunicipalities)
	mux.HandleFunc("GET /v1/municipalities", h.listMunicipalities)
	mux.HandleFunc("GET /v1/municipalities/{code}", h.getMunicipality)
	mux.HandleFunc("GET /v1/municipalities/{code}/parishes", h.listMunicipalityParishes)
	mux.HandleFunc("GET /v1/parishes", h.listParishes)
	mux.HandleFunc("GET /v1/parishes/{code}", h.getParish)
	mux.HandleFunc("GET /v1/places/search", h.searchPlaces)
}

func (h *Handler) listStates(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	states, total, err := h.reader.States(r.Context(), page)
	if err != nil {
		httpx.StoreError(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, httpx.Map(states, toStateJSON), total, page)
}

func (h *Handler) getState(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	st, err := h.reader.State(r.Context(), code)
	if err != nil {
		httpx.StoreError(w, r, err, "state %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toStateJSON(st))
}

func (h *Handler) listStateMunicipalities(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	municipalities, total, err := h.reader.MunicipalitiesOf(r.Context(), code, page)
	if err != nil {
		httpx.StoreError(w, r, err, "state %s not found", code)
		return
	}
	httpx.WriteList(w, r, httpx.Map(municipalities, toMunicipalitySummaryJSON), total, page)
}

func (h *Handler) listMunicipalities(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	municipalities, total, err := h.reader.Municipalities(r.Context(), page)
	if err != nil {
		httpx.StoreError(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, httpx.Map(municipalities, toMunicipalityJSON), total, page)
}

func (h *Handler) getMunicipality(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	m, err := h.reader.Municipality(r.Context(), code)
	if err != nil {
		httpx.StoreError(w, r, err, "municipality %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toMunicipalityJSON(m))
}

func (h *Handler) listMunicipalityParishes(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	parishes, total, err := h.reader.ParishesOf(r.Context(), code, page)
	if err != nil {
		httpx.StoreError(w, r, err, "municipality %s not found", code)
		return
	}
	httpx.WriteList(w, r, httpx.Map(parishes, toParishSummaryJSON), total, page)
}

func (h *Handler) listParishes(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.ReadPage(w, r)
	if !ok {
		return
	}
	parishes, total, err := h.reader.Parishes(r.Context(), page)
	if err != nil {
		httpx.StoreError(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, httpx.Map(parishes, toParishJSON), total, page)
}

func (h *Handler) getParish(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	p, err := h.reader.Parish(r.Context(), code)
	if err != nil {
		httpx.StoreError(w, r, err, "parish %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toParishJSON(p))
}

func (h *Handler) searchPlaces(w http.ResponseWriter, r *http.Request) {
	page, invalid := httpx.ParsePage(r, searchDefaultLimit)
	query := r.URL.Query()

	// The maximum applies to q as received and is checked first, so an
	// oversized query is rejected before doing any work on it. The minimum
	// applies to the normalized text: surrounding and repeated spaces do not
	// count.
	q := query.Get("q")
	switch {
	case !query.Has("q"):
		invalid = append(invalid, httpx.InvalidParam{Name: "q", Reason: "is required"})
	case utf8.RuneCountInString(q) > searchMaxQueryLen:
		invalid = append(invalid, httpx.InvalidParam{
			Name: "q", Reason: fmt.Sprintf("must have at most %d characters", searchMaxQueryLen),
		})
	case utf8.RuneCountInString(textnorm.Normalize(q)) < searchMinQueryLen:
		invalid = append(invalid, httpx.InvalidParam{
			Name: "q", Reason: fmt.Sprintf("must have at least %d characters", searchMinQueryLen),
		})
	}

	placeType := PlaceType(query.Get("type"))
	switch placeType {
	case "", PlaceState, PlaceMunicipality, PlaceParish:
	default:
		invalid = append(invalid, httpx.InvalidParam{
			Name: "type", Reason: "must be one of: state, municipality, parish",
		})
	}

	if len(invalid) > 0 {
		httpx.BadRequest(w, r, invalid)
		return
	}

	places, total, err := h.reader.Search(r.Context(), q, placeType, page)
	if err != nil {
		httpx.StoreError(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, httpx.Map(places, toPlaceJSON), total, page)
}

// JSON representations, as defined in openapi.yaml. They keep the wire format
// out of the domain model.

type stateJSON struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	ISOCode string  `json:"iso_3166_2"`
	Capital *string `json:"capital"` // null when the state has no capital
}

type refJSON struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type municipalityJSON struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	State refJSON `json:"state"`
}

type parishJSON struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Municipality refJSON `json:"municipality"`
	State        refJSON `json:"state"`
}

// placeJSON is a search result. Parents are omitted when the level has none.
type placeJSON struct {
	Type         string   `json:"type"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Municipality *refJSON `json:"municipality,omitempty"`
	State        *refJSON `json:"state,omitempty"`
}

func toPlaceJSON(p Place) placeJSON {
	out := placeJSON{Type: string(p.Type), Code: p.Code, Name: p.Name}
	if p.Municipality != nil {
		m := refJSON(*p.Municipality)
		out.Municipality = &m
	}
	if p.State != nil {
		st := refJSON(*p.State)
		out.State = &st
	}
	return out
}

func toStateJSON(st State) stateJSON {
	out := stateJSON{Code: st.Code, Name: st.Name, Type: string(st.Type), ISOCode: st.ISOCode}
	if st.Capital != "" {
		out.Capital = &st.Capital
	}
	return out
}

func toMunicipalitySummaryJSON(m Municipality) refJSON {
	return refJSON{Code: m.Code, Name: m.Name}
}

func toMunicipalityJSON(m Municipality) municipalityJSON {
	return municipalityJSON{
		Code:  m.Code,
		Name:  m.Name,
		State: refJSON(m.State),
	}
}

func toParishSummaryJSON(p Parish) refJSON {
	return refJSON{Code: p.Code, Name: p.Name}
}

func toParishJSON(p Parish) parishJSON {
	return parishJSON{
		Code:         p.Code,
		Name:         p.Name,
		Municipality: refJSON(p.Municipality),
		State:        refJSON(p.State),
	}
}
