package phones

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/esrichardd/datosvenezuela/api/internal/geo"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

// Files of the phones dataset, relative to its directory.
const (
	landlineFile       = "landline_prefixes.csv"
	landlineStatesFile = "landline_prefix_states.csv"
	mobileFile         = "mobile_prefixes.csv"
	nonGeographicFile  = "non_geographic_prefixes.csv"
)

var (
	landlineHeader       = []string{"code"}
	landlineStatesHeader = []string{"code", "state_code", "state_name"}
	mobileHeader         = []string{"code", "operator"}
	nonGeographicHeader  = []string{"code", "description"}
)

var codeRe = regexp.MustCompile(`^[0-9]{3}$`)

// stateReader is the port to the geography data: landline prefixes refer to
// states. geo.CSVStore implements it.
type stateReader interface {
	State(ctx context.Context, code string) (geo.State, error)
}

// CSVStore serves phone prefixes loaded from CSV files into memory.
//
// It is read-only after loading, so it is safe for concurrent use. The
// prefixes it returns share their States slice with the store: callers must
// not modify it.
type CSVStore struct {
	states   stateReader
	prefixes []Prefix            // every prefix, ordered by code
	byCode   map[string]int      // code → index in prefixes
	byType   map[Type][]Prefix   // ordered by code
	byState  map[string][]Prefix // state code → landline prefixes that cover it, ordered by code
}

// CSVStore must satisfy the handlers' port; this fails to compile otherwise.
var _ reader = (*CSVStore)(nil)

// LoadCSV reads and validates the phones dataset in dir. states resolves the
// states of the landline prefixes, which must exist and have the same name.
//
// It reports every validation problem it finds, not only the first one, so
// the data can be fixed in a single pass.
func LoadCSV(ctx context.Context, dir string, states stateReader) (*CSVStore, error) {
	files := []struct {
		name   string
		header []string
		typ    Type
	}{
		{landlineFile, landlineHeader, TypeLandline},
		{mobileFile, mobileHeader, TypeMobile},
		{nonGeographicFile, nonGeographicHeader, TypeNonGeographic},
	}

	s := &CSVStore{
		states:  states,
		byCode:  make(map[string]int),
		byType:  make(map[Type][]Prefix),
		byState: make(map[string][]Prefix),
	}
	var errs []error
	definedIn := make(map[string]string) // code → file that defines it

	for _, f := range files {
		rows, err := csvload.ReadFile(filepath.Join(dir, f.name), f.header)
		if err != nil {
			return nil, err
		}
		errs = append(errs, s.loadPrefixes(f.name, f.typ, rows, definedIn)...)
	}

	stateRows, err := csvload.ReadFile(filepath.Join(dir, landlineStatesFile), landlineStatesHeader)
	if err != nil {
		return nil, err
	}
	errs = append(errs, s.loadLandlineStates(ctx, stateRows)...)
	for _, p := range s.prefixes {
		if p.Type == TypeLandline && len(p.States) == 0 {
			errs = append(errs, fmt.Errorf("%s: landline code %s has no states", landlineStatesFile, p.Code))
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid phones data in %s:\n%w", dir, errors.Join(errs...))
	}
	s.buildIndexes()

	return s, nil
}

// loadPrefixes validates the rows of one prefix file and adds them to s.
// definedIn records the file of every code seen so far, because a code must
// be unique across files.
func (s *CSVStore) loadPrefixes(file string, typ Type, rows []csvload.Record, definedIn map[string]string) []error {
	var errs []error
	prev := "" // code of the previous row

	for _, r := range rows {
		p := Prefix{Code: r.Fields[0], Type: typ}
		fail := func(format string, args ...any) {
			errs = append(errs, csvload.LineError(file, r.Line, format, args...))
		}

		if !codeRe.MatchString(p.Code) {
			fail("invalid code %q: want 3 digits", p.Code)
		}
		switch other, dup := definedIn[p.Code]; {
		case dup && other == file:
			fail("duplicate code %s", p.Code)
		case dup:
			fail("code %s is already defined in %s", p.Code, other)
		}
		definedIn[p.Code] = file
		csvload.CheckOrder("code", prev, p.Code, fail)
		prev = p.Code

		switch typ {
		case TypeMobile:
			p.Operator = r.Fields[1]
			csvload.CheckText("operator", p.Operator, fail)
		case TypeNonGeographic:
			p.Description = r.Fields[1]
			csvload.CheckText("description", p.Description, fail)
		}

		s.byCode[p.Code] = len(s.prefixes)
		s.prefixes = append(s.prefixes, p)
	}

	return errs
}

// loadLandlineStates validates the relation between landline prefixes and
// states, and adds each state to its prefix. Rows are ordered by code and
// then by state code, so the states of a prefix end up ordered by code.
func (s *CSVStore) loadLandlineStates(ctx context.Context, rows []csvload.Record) []error {
	var errs []error
	seen := make(map[string]bool, len(rows))
	prev := "" // code and state_code of the previous row

	for _, r := range rows {
		code, stateCode, stateName := r.Fields[0], r.Fields[1], r.Fields[2]
		fail := func(format string, args ...any) {
			errs = append(errs, csvload.LineError(landlineStatesFile, r.Line, format, args...))
		}

		// Both codes have a fixed width, so comparing the pair as text sorts
		// by code and then by state code.
		row := code + "," + stateCode
		if seen[row] {
			fail("duplicate row %s", row)
		}
		seen[row] = true
		csvload.CheckOrder("row", prev, row, fail)
		prev = row

		i, ok := s.byCode[code]
		if !ok || s.prefixes[i].Type != TypeLandline {
			fail("unknown landline code %s", code)
			continue
		}

		st, err := s.states.State(ctx, stateCode)
		switch {
		case errors.Is(err, store.ErrNotFound):
			fail("unknown state_code %s", stateCode)
			continue
		case err != nil:
			fail("look up state_code %s: %v", stateCode, err)
			continue
		}
		// The lookup ignores case, but the data must use the canonical code.
		if st.Code != stateCode {
			fail("state_code %s must be written %s", stateCode, st.Code)
		}
		if stateName != st.Name {
			fail("state_name %q does not match %q, the name of %s", stateName, st.Name, st.Code)
		}

		s.prefixes[i].States = append(s.prefixes[i].States, geo.StateRef{Code: st.Code, Name: st.Name})
	}

	return errs
}

// buildIndexes orders the prefixes of every file by code and indexes them by
// code, by type and by state. It runs once the data is valid.
func (s *CSVStore) buildIndexes() {
	slices.SortFunc(s.prefixes, func(a, b Prefix) int { return strings.Compare(a.Code, b.Code) })
	for i, p := range s.prefixes {
		s.byCode[p.Code] = i
		s.byType[p.Type] = append(s.byType[p.Type], p)
		for _, st := range p.States {
			s.byState[st.Code] = append(s.byState[st.Code], p)
		}
	}
}

// Prefixes returns a page of the prefixes of type t, or of every prefix if t
// is empty, ordered by code, and their total count.
func (s *CSVStore) Prefixes(_ context.Context, t Type, p store.Page) ([]Prefix, int, error) {
	all := s.prefixes
	if t != "" {
		all = s.byType[t]
	}
	return store.Paginate(all, p), len(all), nil
}

// Prefix returns the prefix with the given code.
func (s *CSVStore) Prefix(_ context.Context, code string) (Prefix, error) {
	i, ok := s.byCode[code]
	if !ok {
		return Prefix{}, store.ErrNotFound
	}
	return s.prefixes[i], nil
}

// PrefixesOf returns a page of the landline prefixes that cover a state,
// ordered by code, and their total count. The state code is case-insensitive.
// It returns store.ErrNotFound if the state does not exist.
func (s *CSVStore) PrefixesOf(ctx context.Context, stateCode string, p store.Page) ([]Prefix, int, error) {
	st, err := s.states.State(ctx, stateCode)
	if err != nil {
		return nil, 0, err
	}
	all := s.byState[st.Code]
	return store.Paginate(all, p), len(all), nil
}
