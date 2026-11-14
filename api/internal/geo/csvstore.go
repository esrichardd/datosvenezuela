package geo

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/esrichardd/apivenezuela/api/internal/platform/csvload"
	"github.com/esrichardd/apivenezuela/api/internal/platform/textnorm"
)

// Files of the geography dataset, relative to its directory.
const (
	statesFile         = "states.csv"
	municipalitiesFile = "municipalities.csv"
	parishesFile       = "parishes.csv"
)

var (
	statesHeader         = []string{"code", "name", "type", "iso_3166_2", "capital"}
	municipalitiesHeader = []string{"code", "state_code", "name"}
	parishesHeader       = []string{"code", "municipality_code", "name"}
)

var (
	stateCodeRe        = regexp.MustCompile(`^VE[0-9]{2}$`)
	municipalityCodeRe = regexp.MustCompile(`^VE[0-9]{4}$`)
	parishCodeRe       = regexp.MustCompile(`^VE[0-9]{6}$`)
	isoCodeRe          = regexp.MustCompile(`^VE-[A-Z]$`)
)

// CSVStore serves geography data loaded from CSV files into memory.
//
// It is read-only after loading, so it is safe for concurrent use.
type CSVStore struct {
	states         []State
	municipalities []Municipality
	parishes       []Parish

	stateByCode        map[string]int // code → index in states
	municipalityByCode map[string]int // code → index in municipalities
	parishByCode       map[string]int // code → index in parishes

	municipalitiesByState  map[string][]Municipality
	parishesByMunicipality map[string][]Parish

	// places holds every state, municipality and parish, in that order and
	// by code, with its name normalized once at load time for search.
	places []indexedPlace
}

type indexedPlace struct {
	place          Place
	normalizedName string
}

// CSVStore must satisfy the handlers' port; this fails to compile otherwise.
var _ store = (*CSVStore)(nil)

// LoadCSV reads, validates and indexes the geography dataset in dir.
//
// It reports every validation problem it finds, not only the first one, so
// the data can be fixed in a single pass.
func LoadCSV(dir string) (*CSVStore, error) {
	stateRows, err := csvload.ReadFile(filepath.Join(dir, statesFile), statesHeader)
	if err != nil {
		return nil, err
	}
	municipalityRows, err := csvload.ReadFile(filepath.Join(dir, municipalitiesFile), municipalitiesHeader)
	if err != nil {
		return nil, err
	}
	parishRows, err := csvload.ReadFile(filepath.Join(dir, parishesFile), parishesHeader)
	if err != nil {
		return nil, err
	}

	s := &CSVStore{
		stateByCode:            make(map[string]int, len(stateRows)),
		municipalityByCode:     make(map[string]int, len(municipalityRows)),
		parishByCode:           make(map[string]int, len(parishRows)),
		municipalitiesByState:  make(map[string][]Municipality),
		parishesByMunicipality: make(map[string][]Parish),
	}

	var errs []error
	errs = append(errs, s.loadStates(stateRows)...)
	errs = append(errs, s.loadMunicipalities(municipalityRows)...)
	errs = append(errs, s.loadParishes(parishRows)...)
	for _, m := range s.municipalities {
		if len(s.parishesByMunicipality[m.Code]) == 0 {
			errs = append(errs, fmt.Errorf("%s: municipality %s has no parishes", parishesFile, m.Code))
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid geography data in %s:\n%w", dir, errors.Join(errs...))
	}
	s.buildSearchIndex()

	return s, nil
}

func (s *CSVStore) buildSearchIndex() {
	s.places = make([]indexedPlace, 0, len(s.states)+len(s.municipalities)+len(s.parishes))
	add := func(p Place) {
		s.places = append(s.places, indexedPlace{place: p, normalizedName: textnorm.Normalize(p.Name)})
	}

	for _, st := range s.states {
		add(Place{Type: PlaceState, Code: st.Code, Name: st.Name})
	}
	for _, m := range s.municipalities {
		state := m.State
		add(Place{Type: PlaceMunicipality, Code: m.Code, Name: m.Name, State: &state})
	}
	for _, p := range s.parishes {
		municipality, state := p.Municipality, p.State
		add(Place{Type: PlaceParish, Code: p.Code, Name: p.Name, Municipality: &municipality, State: &state})
	}
}

func (s *CSVStore) loadStates(rows []csvload.Record) []error {
	var errs []error
	isoSeen := make(map[string]bool, len(rows))

	for _, r := range rows {
		st := State{
			Code:    r.Fields[0],
			Name:    r.Fields[1],
			Type:    StateType(r.Fields[2]),
			ISOCode: r.Fields[3],
			Capital: r.Fields[4],
		}
		fail := func(format string, args ...any) {
			errs = append(errs, lineError(statesFile, r.Line, format, args...))
		}

		if !stateCodeRe.MatchString(st.Code) {
			fail("invalid code %q: want VE followed by 2 digits", st.Code)
		}
		if _, dup := s.stateByCode[st.Code]; dup {
			fail("duplicate code %s", st.Code)
		}
		checkName(st.Name, fail)
		switch st.Type {
		case TypeState, TypeCapitalDistrict, TypeFederalDependencies:
		default:
			fail("invalid type %q", st.Type)
		}
		if !isoCodeRe.MatchString(st.ISOCode) {
			fail("invalid iso_3166_2 %q: want VE- followed by a letter", st.ISOCode)
		}
		if isoSeen[st.ISOCode] {
			fail("duplicate iso_3166_2 %s", st.ISOCode)
		}
		isoSeen[st.ISOCode] = true
		if st.Type == TypeFederalDependencies && st.Capital != "" {
			fail("federal dependencies must not have a capital")
		}
		if st.Type != TypeFederalDependencies {
			checkText("capital", st.Capital, fail)
		}

		s.stateByCode[st.Code] = len(s.states)
		s.states = append(s.states, st)
	}

	return errs
}

func (s *CSVStore) loadMunicipalities(rows []csvload.Record) []error {
	var errs []error

	for _, r := range rows {
		code, stateCode, name := r.Fields[0], r.Fields[1], r.Fields[2]
		fail := func(format string, args ...any) {
			errs = append(errs, lineError(municipalitiesFile, r.Line, format, args...))
		}

		if !municipalityCodeRe.MatchString(code) {
			fail("invalid code %q: want VE followed by 4 digits", code)
		}
		if _, dup := s.municipalityByCode[code]; dup {
			fail("duplicate code %s", code)
		}
		checkName(name, fail)
		i, ok := s.stateByCode[stateCode]
		if !ok {
			fail("unknown state_code %s", stateCode)
			continue
		}
		if !strings.HasPrefix(code, stateCode) {
			fail("code %s does not start with its state_code %s", code, stateCode)
		}

		st := s.states[i]
		m := Municipality{Code: code, Name: name, State: StateRef{Code: st.Code, Name: st.Name}}
		s.municipalityByCode[code] = len(s.municipalities)
		s.municipalities = append(s.municipalities, m)
		s.municipalitiesByState[stateCode] = append(s.municipalitiesByState[stateCode], m)
	}

	return errs
}

func (s *CSVStore) loadParishes(rows []csvload.Record) []error {
	var errs []error

	for _, r := range rows {
		code, municipalityCode, name := r.Fields[0], r.Fields[1], r.Fields[2]
		fail := func(format string, args ...any) {
			errs = append(errs, lineError(parishesFile, r.Line, format, args...))
		}

		if !parishCodeRe.MatchString(code) {
			fail("invalid code %q: want VE followed by 6 digits", code)
		}
		if _, dup := s.parishByCode[code]; dup {
			fail("duplicate code %s", code)
		}
		checkName(name, fail)
		i, ok := s.municipalityByCode[municipalityCode]
		if !ok {
			fail("unknown municipality_code %s", municipalityCode)
			continue
		}
		if !strings.HasPrefix(code, municipalityCode) {
			fail("code %s does not start with its municipality_code %s", code, municipalityCode)
		}

		m := s.municipalities[i]
		p := Parish{
			Code:         code,
			Name:         name,
			Municipality: MunicipalityRef{Code: m.Code, Name: m.Name},
			State:        m.State,
		}
		s.parishByCode[code] = len(s.parishes)
		s.parishes = append(s.parishes, p)
		s.parishesByMunicipality[municipalityCode] = append(s.parishesByMunicipality[municipalityCode], p)
	}

	return errs
}

// States returns a page of all states, ordered by code, and the total count.
func (s *CSVStore) States(_ context.Context, p Page) ([]State, int, error) {
	return paginate(s.states, p), len(s.states), nil
}

// State returns the state with the given code. The code is case-insensitive.
func (s *CSVStore) State(_ context.Context, code string) (State, error) {
	i, ok := s.stateByCode[strings.ToUpper(code)]
	if !ok {
		return State{}, ErrNotFound
	}
	return s.states[i], nil
}

// Municipalities returns a page of all municipalities, ordered by code, and
// the total count.
func (s *CSVStore) Municipalities(_ context.Context, p Page) ([]Municipality, int, error) {
	return paginate(s.municipalities, p), len(s.municipalities), nil
}

// MunicipalitiesOf returns a page of the municipalities of a state and their
// total count. It returns ErrNotFound if the state does not exist.
func (s *CSVStore) MunicipalitiesOf(_ context.Context, stateCode string, p Page) ([]Municipality, int, error) {
	stateCode = strings.ToUpper(stateCode)
	if _, ok := s.stateByCode[stateCode]; !ok {
		return nil, 0, ErrNotFound
	}
	all := s.municipalitiesByState[stateCode]
	return paginate(all, p), len(all), nil
}

// Municipality returns the municipality with the given code. The code is
// case-insensitive.
func (s *CSVStore) Municipality(_ context.Context, code string) (Municipality, error) {
	i, ok := s.municipalityByCode[strings.ToUpper(code)]
	if !ok {
		return Municipality{}, ErrNotFound
	}
	return s.municipalities[i], nil
}

// Parishes returns a page of all parishes, ordered by code, and the total
// count.
func (s *CSVStore) Parishes(_ context.Context, p Page) ([]Parish, int, error) {
	return paginate(s.parishes, p), len(s.parishes), nil
}

// ParishesOf returns a page of the parishes of a municipality and their total
// count. It returns ErrNotFound if the municipality does not exist.
func (s *CSVStore) ParishesOf(_ context.Context, municipalityCode string, p Page) ([]Parish, int, error) {
	municipalityCode = strings.ToUpper(municipalityCode)
	if _, ok := s.municipalityByCode[municipalityCode]; !ok {
		return nil, 0, ErrNotFound
	}
	all := s.parishesByMunicipality[municipalityCode]
	return paginate(all, p), len(all), nil
}

// Parish returns the parish with the given code. The code is
// case-insensitive.
func (s *CSVStore) Parish(_ context.Context, code string) (Parish, error) {
	i, ok := s.parishByCode[strings.ToUpper(code)]
	if !ok {
		return Parish{}, ErrNotFound
	}
	return s.parishes[i], nil
}

// Search returns a page of the places whose name contains query, ignoring
// case and accents, and the total number of matches. If placeType is not
// empty, only places of that type are returned.
//
// Results are ordered by relevance: exact matches first, then names that
// start with the query, then names that contain it. Within each group,
// states come before municipalities and municipalities before parishes,
// ordered by code.
func (s *CSVStore) Search(_ context.Context, query string, placeType PlaceType, p Page) ([]Place, int, error) {
	q := textnorm.Normalize(query)
	if q == "" {
		return []Place{}, 0, nil
	}

	type match struct {
		place Place
		rank  int
	}
	var matches []match
	for _, ip := range s.places {
		if placeType != "" && ip.place.Type != placeType {
			continue
		}
		switch {
		case ip.normalizedName == q:
			matches = append(matches, match{ip.place, 0})
		case strings.HasPrefix(ip.normalizedName, q):
			matches = append(matches, match{ip.place, 1})
		case strings.Contains(ip.normalizedName, q):
			matches = append(matches, match{ip.place, 2})
		}
	}

	// places is already ordered by level and code, so a stable sort by rank
	// keeps that order within each rank.
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].rank < matches[j].rank })

	results := make([]Place, len(matches))
	for i, m := range matches {
		results[i] = m.place
	}
	page := paginate(results, p)
	// The parents are pointers into the index: copy them so callers cannot
	// modify the store through a result.
	for i := range page {
		page[i] = clonePlace(page[i])
	}
	return page, len(results), nil
}

func clonePlace(p Place) Place {
	if p.Municipality != nil {
		m := *p.Municipality
		p.Municipality = &m
	}
	if p.State != nil {
		st := *p.State
		p.State = &st
	}
	return p
}

// paginate returns a copy of the window of items selected by p, so callers
// cannot modify the store. It never returns nil.
func paginate[T any](items []T, p Page) []T {
	start := min(max(p.Offset, 0), len(items))
	end := len(items)
	if p.Limit > 0 {
		end = min(start+p.Limit, len(items))
	}
	return append(make([]T, 0, end-start), items[start:end]...)
}

func lineError(file string, line int, format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s", file, line, fmt.Sprintf(format, args...))
}

func checkName(name string, fail func(string, ...any)) {
	checkText("name", name, fail)
}

// checkText rejects empty values and surrounding whitespace.
func checkText(field, value string, fail func(string, ...any)) {
	switch {
	case value == "":
		fail("empty %s", field)
	case value != strings.TrimSpace(value):
		fail("%s %q has surrounding whitespace", field, value)
	}
}
