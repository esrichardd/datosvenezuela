package geo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/esrichardd/apivenezuela/api/internal/platform/store"
)

const (
	validStates = `code,name,type,iso_3166_2,capital
VE01,Distrito Capital,capital_district,VE-A,Caracas
VE14,Mérida,state,VE-L,Mérida
VE25,Dependencias Federales,federal_dependencies,VE-W,
`
	validMunicipalities = `code,state_code,name
VE0101,VE01,Libertador
VE1401,VE14,Alberto Adriani
VE1412,VE14,Libertador
`
	validParishes = `code,municipality_code,name
VE010101,VE0101,Altagracia
VE010102,VE0101,Antímano
VE140101,VE1401,Presidente Betancourt
VE141201,VE1412,Antonio Spinetti Dini
VE141202,VE1412,Arias
`
)

// realDataDir is the dataset published in the repository.
var realDataDir = filepath.Join("..", "..", "..", "data", "geography")

// writeDataset writes the three geography files to a temporary directory and
// returns its path.
func writeDataset(t *testing.T, states, municipalities, parishes string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		statesFile:         states,
		municipalitiesFile: municipalities,
		parishesFile:       parishes,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func loadValid(t *testing.T) *CSVStore {
	t.Helper()
	s, err := LoadCSV(writeDataset(t, validStates, validMunicipalities, validParishes))
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	return s
}

func TestLoadCSVValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		states         string
		municipalities string
		parishes       string
		wantErrs       []string
	}{
		{
			name:     "invalid state code",
			states:   strings.Replace(validStates, "VE25,", "VE2X,", 1),
			wantErrs: []string{`states.csv:4: invalid code "VE2X"`},
		},
		{
			name:     "lowercase state code",
			states:   strings.Replace(validStates, "VE25,", "ve25,", 1),
			wantErrs: []string{`states.csv:4: invalid code "ve25"`},
		},
		{
			name:     "duplicate state code",
			states:   validStates + "VE14,Otro,state,VE-Z,Otra\n",
			wantErrs: []string{"states.csv:5: duplicate code VE14"},
		},
		{
			name:     "invalid state type",
			states:   strings.Replace(validStates, ",state,", ",estado,", 1),
			wantErrs: []string{`states.csv:3: invalid type "estado"`},
		},
		{
			name:     "invalid iso code",
			states:   strings.Replace(validStates, "VE-L", "VE-LL", 1),
			wantErrs: []string{`states.csv:3: invalid iso_3166_2 "VE-LL"`},
		},
		{
			name:     "duplicate iso code",
			states:   strings.Replace(validStates, "VE-L", "VE-A", 1),
			wantErrs: []string{"states.csv:3: duplicate iso_3166_2 VE-A"},
		},
		{
			name:     "state without capital",
			states:   strings.Replace(validStates, "VE-L,Mérida", "VE-L,", 1),
			wantErrs: []string{"states.csv:3: empty capital"},
		},
		{
			name:     "federal dependencies with capital",
			states:   strings.Replace(validStates, "VE-W,", "VE-W,Gran Roque", 1),
			wantErrs: []string{"states.csv:4: federal dependencies must not have a capital"},
		},
		{
			name:     "empty name",
			states:   strings.Replace(validStates, ",Mérida,state", ",,state", 1),
			wantErrs: []string{"states.csv:3: empty name"},
		},
		{
			name:     "name with surrounding whitespace",
			states:   strings.Replace(validStates, ",Mérida,state", ", Mérida,state", 1),
			wantErrs: []string{`states.csv:3: name " Mérida" has surrounding whitespace`},
		},
		{
			name:           "unknown state of municipality",
			municipalities: strings.Replace(validMunicipalities, "VE1401,VE14", "VE1401,VE15", 1),
			wantErrs:       []string{"municipalities.csv:3: unknown state_code VE15"},
		},
		{
			name:           "municipality code outside its state",
			municipalities: strings.Replace(validMunicipalities, "VE1401,VE14", "VE1401,VE01", 1),
			wantErrs:       []string{"municipalities.csv:3: code VE1401 does not start with its state_code VE01"},
		},
		{
			name:           "duplicate municipality code",
			municipalities: validMunicipalities + "VE1412,VE14,Otro\n",
			wantErrs:       []string{"municipalities.csv:5: duplicate code VE1412"},
		},
		{
			name:     "unknown municipality of parish",
			parishes: strings.Replace(validParishes, "VE141202,VE1412", "VE141202,VE1499", 1),
			wantErrs: []string{"parishes.csv:6: unknown municipality_code VE1499"},
		},
		{
			name:     "parish code outside its municipality",
			parishes: strings.Replace(validParishes, "VE140101,VE1401", "VE140101,VE1412", 1),
			wantErrs: []string{"parishes.csv:4: code VE140101 does not start with its municipality_code VE1412"},
		},
		{
			name:     "invalid parish code",
			parishes: strings.Replace(validParishes, "VE141202,", "VE14120,", 1),
			wantErrs: []string{`parishes.csv:6: invalid code "VE14120"`},
		},
		{
			name:     "municipality without parishes",
			parishes: strings.Replace(validParishes, "VE140101,VE1401,Presidente Betancourt\n", "", 1),
			wantErrs: []string{"parishes.csv: municipality VE1401 has no parishes"},
		},
		{
			name:     "state out of order",
			states:   validStates + "VE02,Amazonas,state,VE-Z,Puerto Ayacucho\n",
			wantErrs: []string{"states.csv:5: code VE02 is out of order: it must come after VE25"},
		},
		{
			name:           "municipality out of order",
			municipalities: validMunicipalities + "VE1402,VE14,Andrés Bello\n",
			wantErrs:       []string{"municipalities.csv:5: code VE1402 is out of order: it must come after VE1412"},
		},
		{
			name:     "parish out of order",
			parishes: validParishes + "VE010103,VE0101,Candelaria\n",
			wantErrs: []string{"parishes.csv:7: code VE010103 is out of order: it must come after VE141202"},
		},
		{
			name:     "reports every error",
			states:   strings.Replace(validStates, ",state,", ",estado,", 1),
			parishes: strings.Replace(validParishes, "VE141202,VE1412", "VE141202,VE1499", 1),
			wantErrs: []string{
				`states.csv:3: invalid type "estado"`,
				"parishes.csv:6: unknown municipality_code VE1499",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states, municipalities, parishes := tt.states, tt.municipalities, tt.parishes
			if states == "" {
				states = validStates
			}
			if municipalities == "" {
				municipalities = validMunicipalities
			}
			if parishes == "" {
				parishes = validParishes
			}

			_, err := LoadCSV(writeDataset(t, states, municipalities, parishes))
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
	dir := writeDataset(t, validStates, validMunicipalities, validParishes)
	if err := os.Remove(filepath.Join(dir, parishesFile)); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadCSV(dir); err == nil {
		t.Fatal("LoadCSV() error = nil, want an error for the missing file")
	}
}

func TestState(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	for _, code := range []string{"VE14", "ve14", "Ve14"} {
		st, err := s.State(ctx, code)
		if err != nil {
			t.Fatalf("State(%q) error = %v", code, err)
		}
		want := State{Code: "VE14", Name: "Mérida", Type: TypeState, ISOCode: "VE-L", Capital: "Mérida"}
		if st != want {
			t.Errorf("State(%q) = %+v, want %+v", code, st, want)
		}
	}

	if _, err := s.State(ctx, "VE99"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("State(VE99) error = %v, want ErrNotFound", err)
	}
}

func TestStates(t *testing.T) {
	s := loadValid(t)

	got, total, err := s.States(context.Background(), store.Page{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("States() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if codes := stateCodes(got); codes != "VE14,VE25" {
		t.Errorf("codes = %s, want VE14,VE25", codes)
	}
}

func TestMunicipalitiesOf(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	got, total, err := s.MunicipalitiesOf(ctx, "ve14", store.Page{Limit: 100})
	if err != nil {
		t.Fatalf("MunicipalitiesOf() error = %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Fatalf("got %d municipalities (total %d), want 2", len(got), total)
	}
	want := Municipality{Code: "VE1412", Name: "Libertador", State: StateRef{Code: "VE14", Name: "Mérida"}}
	if got[1] != want {
		t.Errorf("municipality = %+v, want %+v", got[1], want)
	}

	// A state without municipalities exists, so it returns an empty list.
	got, total, err = s.MunicipalitiesOf(ctx, "VE25", store.Page{Limit: 100})
	if err != nil || total != 0 || got == nil || len(got) != 0 {
		t.Errorf("MunicipalitiesOf(VE25) = %v, %d, %v; want empty non-nil list", got, total, err)
	}

	if _, _, err := s.MunicipalitiesOf(ctx, "VE99", store.Page{Limit: 100}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("MunicipalitiesOf(VE99) error = %v, want ErrNotFound", err)
	}
}

func TestMunicipality(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	m, err := s.Municipality(ctx, "ve1412")
	if err != nil {
		t.Fatalf("Municipality() error = %v", err)
	}
	if m.State != (StateRef{Code: "VE14", Name: "Mérida"}) {
		t.Errorf("state = %+v, want VE14 Mérida", m.State)
	}

	if _, err := s.Municipality(ctx, "VE9999"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Municipality(VE9999) error = %v, want ErrNotFound", err)
	}
}

func TestParishesOfAndParish(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	got, total, err := s.ParishesOf(ctx, "VE1412", store.Page{Limit: 1})
	if err != nil {
		t.Fatalf("ParishesOf() error = %v", err)
	}
	if total != 2 || len(got) != 1 || got[0].Code != "VE141201" {
		t.Errorf("ParishesOf() = %+v (total %d), want [VE141201] (total 2)", got, total)
	}

	if _, _, err := s.ParishesOf(ctx, "VE9999", store.Page{Limit: 1}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("ParishesOf(VE9999) error = %v, want ErrNotFound", err)
	}

	p, err := s.Parish(ctx, "ve141201")
	if err != nil {
		t.Fatalf("Parish() error = %v", err)
	}
	want := Parish{
		Code:         "VE141201",
		Name:         "Antonio Spinetti Dini",
		Municipality: MunicipalityRef{Code: "VE1412", Name: "Libertador"},
		State:        StateRef{Code: "VE14", Name: "Mérida"},
	}
	if p != want {
		t.Errorf("Parish() = %+v, want %+v", p, want)
	}

	if _, err := s.Parish(ctx, "VE999999"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Parish(VE999999) error = %v, want ErrNotFound", err)
	}
}

func TestListsReturnCopies(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	got, _, _ := s.Parishes(ctx, store.Page{Limit: 10})
	got[0].Name = "changed"

	again, _, _ := s.Parishes(ctx, store.Page{Limit: 10})
	if again[0].Name == "changed" {
		t.Error("modifying a returned list changed the store")
	}
}

// TestRealData loads the dataset published in data/geography, so CI fails if
// anyone commits invalid data.
func TestRealData(t *testing.T) {
	s, err := LoadCSV(realDataDir)
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}

	ctx := context.Background()
	_, states, _ := s.States(ctx, store.Page{})
	_, municipalities, _ := s.Municipalities(ctx, store.Page{})
	_, parishes, _ := s.Parishes(ctx, store.Page{})
	if states != 25 || municipalities != 335 || parishes != 1134 {
		t.Errorf("got %d states, %d municipalities, %d parishes; want 25, 335, 1134",
			states, municipalities, parishes)
	}
}

func stateCodes(states []State) string {
	codes := make([]string, len(states))
	for i, st := range states {
		codes[i] = st.Code
	}
	return strings.Join(codes, ",")
}

func placeCodes(places []Place) string {
	codes := make([]string, len(places))
	for i, p := range places {
		codes[i] = p.Code
	}
	return strings.Join(codes, ",")
}

func TestSearch(t *testing.T) {
	s := loadValid(t)

	tests := []struct {
		name      string
		query     string
		placeType PlaceType
		page      store.Page
		wantCodes string
		wantTotal int
	}{
		{
			// Prefix matches (Antímano, Antonio Spinetti Dini) come before
			// names that only contain "an" (Alberto Adriani, Presidente
			// Betancourt); within each group, municipalities before parishes.
			name: "ranked by relevance", query: "an", page: store.Page{Limit: 20},
			wantCodes: "VE010102,VE141201,VE1401,VE140101", wantTotal: 4,
		},
		{
			name: "exact match first", query: "libertador", page: store.Page{Limit: 20},
			wantCodes: "VE0101,VE1412", wantTotal: 2,
		},
		{
			name: "ignores case and accents", query: "  ANTIMANO ", page: store.Page{Limit: 20},
			wantCodes: "VE010102", wantTotal: 1,
		},
		{
			name: "matches accented query against plain name", query: "Árias", page: store.Page{Limit: 20},
			wantCodes: "VE141202", wantTotal: 1,
		},
		{
			name: "filters by type", query: "an", placeType: PlaceParish, page: store.Page{Limit: 20},
			wantCodes: "VE010102,VE141201,VE140101", wantTotal: 3,
		},
		{
			name: "paginates", query: "an", page: store.Page{Limit: 2, Offset: 1},
			wantCodes: "VE141201,VE1401", wantTotal: 4,
		},
		{
			name: "no matches", query: "zzz", page: store.Page{Limit: 20},
			wantCodes: "", wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := s.Search(context.Background(), tt.query, tt.placeType, tt.page)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if got == nil {
				t.Fatal("Search() returned nil, want a non-nil slice")
			}
			if codes := placeCodes(got); codes != tt.wantCodes || total != tt.wantTotal {
				t.Errorf("Search(%q) = %s (total %d), want %s (total %d)",
					tt.query, codes, total, tt.wantCodes, tt.wantTotal)
			}
		})
	}
}

func TestSearchParents(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	got, _, _ := s.Search(ctx, "merida", PlaceState, store.Page{Limit: 20})
	if len(got) != 1 || got[0].State != nil || got[0].Municipality != nil {
		t.Errorf("state result = %+v, want no parents", got)
	}

	got, _, _ = s.Search(ctx, "arias", "", store.Page{Limit: 20})
	want := Place{
		Type: PlaceParish, Code: "VE141202", Name: "Arias",
		Municipality: &MunicipalityRef{Code: "VE1412", Name: "Libertador"},
		State:        &StateRef{Code: "VE14", Name: "Mérida"},
	}
	if len(got) != 1 || got[0].Code != want.Code ||
		*got[0].Municipality != *want.Municipality || *got[0].State != *want.State {
		t.Errorf("parish result = %+v, want %+v", got, want)
	}

	// Results must not share memory with the index.
	got[0].State.Name = "changed"
	again, _, _ := s.Search(ctx, "arias", "", store.Page{Limit: 20})
	if again[0].State.Name == "changed" {
		t.Error("modifying a search result changed the store")
	}
}

func TestSearchRealData(t *testing.T) {
	s, err := LoadCSV(realDataDir)
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}

	// The example documented in openapi.yaml.
	got, total, err := s.Search(context.Background(), "san cristobal", "", store.Page{Limit: 20})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if codes := placeCodes(got); codes != "VE2023,VE031802,VE141102" || total != 3 {
		t.Errorf("Search(san cristobal) = %s (total %d), want VE2023,VE031802,VE141102 (total 3)", codes, total)
	}
}
