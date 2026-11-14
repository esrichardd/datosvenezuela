package geo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/esrichardd/apivenezuela/api/internal/platform/httpx"
	"github.com/esrichardd/apivenezuela/api/internal/platform/textnorm"
)

// Search limits, as documented in openapi.yaml.
const (
	searchDefaultLimit = 20
	searchMinQueryLen  = 2
	searchMaxQueryLen  = 100
)

// store is the port the handlers depend on. CSVStore implements it; any
// other data source (a database, a test fake) only needs these methods.
type store interface {
	States(ctx context.Context, p Page) ([]State, int, error)
	State(ctx context.Context, code string) (State, error)
	Municipalities(ctx context.Context, p Page) ([]Municipality, int, error)
	MunicipalitiesOf(ctx context.Context, stateCode string, p Page) ([]Municipality, int, error)
	Municipality(ctx context.Context, code string) (Municipality, error)
	Parishes(ctx context.Context, p Page) ([]Parish, int, error)
	ParishesOf(ctx context.Context, municipalityCode string, p Page) ([]Parish, int, error)
	Parish(ctx context.Context, code string) (Parish, error)
	Search(ctx context.Context, query string, placeType PlaceType, p Page) ([]Place, int, error)
}

// Handler serves the geography endpoints.
type Handler struct {
	store store
}

// NewHandler returns a Handler that reads from s.
func NewHandler(s store) *Handler {
	return &Handler{store: s}
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
	page, ok := parsePage(w, r)
	if !ok {
		return
	}
	states, total, err := h.store.States(r.Context(), toDomainPage(page))
	if err != nil {
		fail(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, mapSlice(states, toStateJSON), total, page)
}

func (h *Handler) getState(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	st, err := h.store.State(r.Context(), code)
	if err != nil {
		fail(w, r, err, "state %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toStateJSON(st))
}

func (h *Handler) listStateMunicipalities(w http.ResponseWriter, r *http.Request) {
	page, ok := parsePage(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	municipalities, total, err := h.store.MunicipalitiesOf(r.Context(), code, toDomainPage(page))
	if err != nil {
		fail(w, r, err, "state %s not found", code)
		return
	}
	httpx.WriteList(w, r, mapSlice(municipalities, toMunicipalitySummaryJSON), total, page)
}

func (h *Handler) listMunicipalities(w http.ResponseWriter, r *http.Request) {
	page, ok := parsePage(w, r)
	if !ok {
		return
	}
	municipalities, total, err := h.store.Municipalities(r.Context(), toDomainPage(page))
	if err != nil {
		fail(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, mapSlice(municipalities, toMunicipalityJSON), total, page)
}

func (h *Handler) getMunicipality(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	m, err := h.store.Municipality(r.Context(), code)
	if err != nil {
		fail(w, r, err, "municipality %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toMunicipalityJSON(m))
}

func (h *Handler) listMunicipalityParishes(w http.ResponseWriter, r *http.Request) {
	page, ok := parsePage(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	parishes, total, err := h.store.ParishesOf(r.Context(), code, toDomainPage(page))
	if err != nil {
		fail(w, r, err, "municipality %s not found", code)
		return
	}
	httpx.WriteList(w, r, mapSlice(parishes, toParishSummaryJSON), total, page)
}

func (h *Handler) listParishes(w http.ResponseWriter, r *http.Request) {
	page, ok := parsePage(w, r)
	if !ok {
		return
	}
	parishes, total, err := h.store.Parishes(r.Context(), toDomainPage(page))
	if err != nil {
		fail(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, mapSlice(parishes, toParishJSON), total, page)
}

func (h *Handler) getParish(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	p, err := h.store.Parish(r.Context(), code)
	if err != nil {
		fail(w, r, err, "parish %s not found", code)
		return
	}
	httpx.WriteItem(w, r, toParishJSON(p))
}

func (h *Handler) searchPlaces(w http.ResponseWriter, r *http.Request) {
	page, invalid := httpx.ParsePage(r, searchDefaultLimit)
	query := r.URL.Query()

	q := query.Get("q")
	switch n := utf8.RuneCountInString(textnorm.Normalize(q)); {
	case !query.Has("q"):
		invalid = append(invalid, httpx.InvalidParam{Name: "q", Reason: "is required"})
	case n < searchMinQueryLen:
		invalid = append(invalid, httpx.InvalidParam{
			Name: "q", Reason: fmt.Sprintf("must have at least %d characters", searchMinQueryLen),
		})
	case n > searchMaxQueryLen:
		invalid = append(invalid, httpx.InvalidParam{
			Name: "q", Reason: fmt.Sprintf("must have at most %d characters", searchMaxQueryLen),
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

	places, total, err := h.store.Search(r.Context(), q, placeType, toDomainPage(page))
	if err != nil {
		fail(w, r, err, "")
		return
	}
	httpx.WriteList(w, r, mapSlice(places, toPlaceJSON), total, page)
}

// parsePage reads the pagination parameters. If they are invalid it writes
// a 400 and returns false.
func parsePage(w http.ResponseWriter, r *http.Request) (httpx.Page, bool) {
	page, invalid := httpx.ParsePage(r, httpx.DefaultLimit)
	if len(invalid) > 0 {
		httpx.BadRequest(w, r, invalid)
		return page, false
	}
	return page, true
}

// fail answers a store error: ErrNotFound becomes a 404 with the given
// detail; anything else is an unexpected 500.
func fail(w http.ResponseWriter, r *http.Request, err error, notFoundFormat string, args ...any) {
	if errors.Is(err, ErrNotFound) && notFoundFormat != "" {
		httpx.NotFound(w, r, notFoundFormat, args...)
		return
	}
	httpx.InternalError(w, r, err)
}

func toDomainPage(p httpx.Page) Page {
	return Page{Limit: p.Limit, Offset: p.Offset}
}

func mapSlice[T, J any](items []T, f func(T) J) []J {
	out := make([]J, len(items))
	for i, item := range items {
		out[i] = f(item)
	}
	return out
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
