package phones

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/geo"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

const (
	validLandline = `code
212
275
`
	validLandlineStates = `code,state_code,state_name
212,VE01,Distrito Capital
212,VE15,Miranda
275,VE14,Mérida
275,VE20,Táchira
`
	validMobile = `code,operator
412,Digitel
414,Movistar
`
	validNonGeographic = `code,description
800,Acceso universal
`
)

// testStates is a stateReader over a fixed set of states. Like the geography
// store, it matches codes case-insensitively.
type testStates map[string]string // code → name

var validStates = testStates{
	"VE01": "Distrito Capital",
	"VE14": "Mérida",
	"VE15": "Miranda",
	"VE20": "Táchira",
	"VE25": "Dependencias Federales", // covered by no prefix
}

func (ts testStates) State(_ context.Context, code string) (geo.State, error) {
	code = strings.ToUpper(code)
	name, ok := ts[code]
	if !ok {
		return geo.State{}, store.ErrNotFound
	}
	return geo.State{Code: code, Name: name}, nil
}

// failingStates simulates a geography source that breaks.
type failingStates struct{}

var errBoom = errors.New("connection refused to 10.0.0.5:5432")

func (failingStates) State(context.Context, string) (geo.State, error) { return geo.State{}, errBoom }

// dataset holds the content of each file; an empty field means the valid one.
type dataset struct {
	landline, landlineStates, mobile, nonGeographic string
}

// write writes the dataset to a temporary directory and returns its path.
func (d dataset) write(t *testing.T) string {
	t.Helper()
	or := func(content, valid string) string {
		if content == "" {
			return valid
		}
		return content
	}
	files := map[string]string{
		landlineFile:       or(d.landline, validLandline),
		landlineStatesFile: or(d.landlineStates, validLandlineStates),
		mobileFile:         or(d.mobile, validMobile),
		nonGeographicFile:  or(d.nonGeographic, validNonGeographic),
	}
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func loadValid(t *testing.T) *CSVStore {
	t.Helper()
	s, err := LoadCSV(context.Background(), dataset{}.write(t), validStates)
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	return s
}

func TestLoadCSVValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		data     dataset
		states   stateReader
		wantErrs []string
	}{
		{
			name:     "invalid code",
			data:     dataset{landline: "code\n212\n27X\n"},
			wantErrs: []string{`landline_prefixes.csv:3: invalid code "27X": want 3 digits`},
		},
		{
			name:     "duplicate code",
			data:     dataset{mobile: validMobile + "414,Otra\n"},
			wantErrs: []string{"mobile_prefixes.csv:4: duplicate code 414"},
		},
		{
			name:     "code defined in another file",
			data:     dataset{nonGeographic: "code,description\n212,Otro\n800,Acceso universal\n"},
			wantErrs: []string{"non_geographic_prefixes.csv:2: code 212 is already defined in landline_prefixes.csv"},
		},
		{
			name:     "code out of order",
			data:     dataset{mobile: validMobile + "413,Otra\n"},
			wantErrs: []string{"mobile_prefixes.csv:4: code 413 is out of order: it must come after 414"},
		},
		{
			name:     "empty operator",
			data:     dataset{mobile: strings.Replace(validMobile, "414,Movistar", "414,", 1)},
			wantErrs: []string{"mobile_prefixes.csv:3: empty operator"},
		},
		{
			name:     "description with surrounding whitespace",
			data:     dataset{nonGeographic: strings.Replace(validNonGeographic, ",Acceso", ", Acceso", 1)},
			wantErrs: []string{`non_geographic_prefixes.csv:2: description " Acceso universal" has surrounding whitespace`},
		},
		{
			name:     "state of a code that is not landline",
			data:     dataset{landlineStates: validLandlineStates + "414,VE15,Miranda\n"},
			wantErrs: []string{"landline_prefix_states.csv:6: unknown landline code 414"},
		},
		{
			name:     "unknown state",
			data:     dataset{landlineStates: strings.Replace(validLandlineStates, "275,VE20", "275,VE99", 1)},
			wantErrs: []string{"landline_prefix_states.csv:5: unknown state_code VE99"},
		},
		{
			name:     "lowercase state code",
			data:     dataset{landlineStates: strings.Replace(validLandlineStates, "275,VE20", "275,ve20", 1)},
			wantErrs: []string{"landline_prefix_states.csv:5: state_code ve20 must be written VE20"},
		},
		{
			name:     "state name mismatch",
			data:     dataset{landlineStates: strings.Replace(validLandlineStates, "Táchira", "Tachira", 1)},
			wantErrs: []string{`landline_prefix_states.csv:5: state_name "Tachira" does not match "Táchira", the name of VE20`},
		},
		{
			name:     "duplicate row",
			data:     dataset{landlineStates: validLandlineStates + "275,VE20,Táchira\n"},
			wantErrs: []string{"landline_prefix_states.csv:6: duplicate row 275,VE20"},
		},
		{
			name:     "row out of order",
			data:     dataset{landlineStates: validLandlineStates + "275,VE15,Miranda\n"},
			wantErrs: []string{"landline_prefix_states.csv:6: row 275,VE15 is out of order: it must come after 275,VE20"},
		},
		{
			name:     "landline code without states",
			data:     dataset{landlineStates: "code,state_code,state_name\n212,VE01,Distrito Capital\n"},
			wantErrs: []string{"landline_prefix_states.csv: landline code 275 has no states"},
		},
		{
			name:     "state lookup fails",
			states:   failingStates{},
			wantErrs: []string{"landline_prefix_states.csv:2: look up state_code VE01: connection refused"},
		},
		{
			name: "reports every error",
			data: dataset{
				mobile:         strings.Replace(validMobile, "414,Movistar", "414,", 1),
				landlineStates: strings.Replace(validLandlineStates, "275,VE20", "275,VE99", 1),
			},
			wantErrs: []string{
				"mobile_prefixes.csv:3: empty operator",
				"landline_prefix_states.csv:5: unknown state_code VE99",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states := tt.states
			if states == nil {
				states = validStates
			}
			_, err := LoadCSV(context.Background(), tt.data.write(t), states)
			if err == nil {
				t.Fatal("LoadCSV() error = nil, want validation errors")
			}
			for _, want := range tt.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("LoadCSV() error = %q\nwant it to contain %q", err, want)
				}
			}
		})
	}
}

func TestLoadCSVMissingFile(t *testing.T) {
	for _, file := range []string{landlineFile, mobileFile, nonGeographicFile, landlineStatesFile} {
		t.Run(file, func(t *testing.T) {
			dir := dataset{}.write(t)
			if err := os.Remove(filepath.Join(dir, file)); err != nil {
				t.Fatal(err)
			}

			if _, err := LoadCSV(context.Background(), dir, validStates); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("LoadCSV() error = %v, want a not-exist error", err)
			}
		})
	}
}

func TestPrefix(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	tests := []struct {
		code string
		want Prefix
	}{
		{"275", Prefix{Code: "275", Type: TypeLandline, States: []geo.StateRef{
			{Code: "VE14", Name: "Mérida"},
			{Code: "VE20", Name: "Táchira"},
		}}},
		{"414", Prefix{Code: "414", Type: TypeMobile, Operator: "Movistar"}},
		{"800", Prefix{Code: "800", Type: TypeNonGeographic, Description: "Acceso universal"}},
	}
	for _, tt := range tests {
		got, err := s.Prefix(ctx, tt.code)
		if err != nil {
			t.Fatalf("Prefix(%s) error = %v", tt.code, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Prefix(%s) = %+v, want %+v", tt.code, got, tt.want)
		}
	}

	// The trunk prefix 0 is not part of the code: 0414 is not 414.
	for _, code := range []string{"999", "0414"} {
		if _, err := s.Prefix(ctx, code); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Prefix(%s) error = %v, want ErrNotFound", code, err)
		}
	}
}

func TestPrefixes(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	tests := []struct {
		typ       Type
		page      store.Page
		wantCodes string
		wantTotal int
	}{
		// Every file, merged in code order.
		{"", store.Page{Limit: 100}, "212,275,412,414,800", 5},
		{"", store.Page{Limit: 2, Offset: 1}, "275,412", 5},
		{TypeMobile, store.Page{Limit: 100}, "412,414", 2},
	}
	for _, tt := range tests {
		got, total, err := s.Prefixes(ctx, tt.typ, tt.page)
		if err != nil {
			t.Fatalf("Prefixes(%q) error = %v", tt.typ, err)
		}
		if codes := prefixCodes(got); codes != tt.wantCodes || total != tt.wantTotal {
			t.Errorf("Prefixes(%q, %+v) = %s (total %d), want %s (total %d)",
				tt.typ, tt.page, codes, total, tt.wantCodes, tt.wantTotal)
		}
	}
}

func TestPrefixesOf(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	tests := []struct {
		state     string
		wantCodes string
	}{
		{"VE15", "212"},
		{"ve20", "275"}, // case-insensitive
		{"VE25", ""},    // a state no prefix covers
	}
	for _, tt := range tests {
		got, total, err := s.PrefixesOf(ctx, tt.state, store.Page{Limit: 100})
		if err != nil {
			t.Fatalf("PrefixesOf(%s) error = %v", tt.state, err)
		}
		if codes := prefixCodes(got); codes != tt.wantCodes || total != len(got) {
			t.Errorf("PrefixesOf(%s) = %s (total %d), want %s", tt.state, codes, total, tt.wantCodes)
		}
	}

	if _, _, err := s.PrefixesOf(ctx, "VE99", store.Page{Limit: 100}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("PrefixesOf(VE99) error = %v, want ErrNotFound", err)
	}
}

// TestRealData loads the dataset published in data/phones against the real
// geography, so CI fails if anyone commits invalid data.
func TestRealData(t *testing.T) {
	ctx := context.Background()
	states, err := geo.LoadCSV(filepath.Join("..", "..", "..", "data", "geography"))
	if err != nil {
		t.Fatalf("geo.LoadCSV() error = %v", err)
	}
	s, err := LoadCSV(ctx, filepath.Join("..", "..", "..", "data", "phones"), states)
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}

	counts := map[Type]int{"": 68, TypeLandline: 58, TypeMobile: 6, TypeNonGeographic: 4}
	for typ, want := range counts {
		if _, total, _ := s.Prefixes(ctx, typ, store.Page{}); total != want {
			t.Errorf("got %d prefixes of type %q, want %d", total, typ, want)
		}
	}
}

func prefixCodes(prefixes []Prefix) string {
	codes := make([]string, len(prefixes))
	for i, p := range prefixes {
		codes[i] = p.Code
	}
	return strings.Join(codes, ",")
}
