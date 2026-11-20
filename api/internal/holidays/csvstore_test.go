package holidays

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload/csvloadtest"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

const (
	validFixed = `month,day,name,legal_basis
1,1,Año Nuevo,LOTTT art. 184
4,19,Declaración de la Independencia,Ley de Fiestas Nacionales art. 1
7,5,Día de la Independencia,Ley de Fiestas Nacionales art. 1
`
	validEaster = `days_from_easter,name,legal_basis
-48,Lunes de Carnaval,LOTTT art. 184
-2,Viernes Santo,LOTTT art. 184
`
	validBank = `date,name
2026-01-01,Año Nuevo
2026-02-16,Lunes de Carnaval
2026-03-19,Día de San José
`
)

// realDataDir is the dataset published in the repository.
var realDataDir = filepath.Join("..", "..", "..", "data", "holidays")

// dataset holds the content of each file; an empty field means the valid one.
// bank maps file names to contents; nil means bank_2026.csv with validBank.
type dataset struct {
	fixed, easter string
	bank          map[string]string
}

// write writes the dataset to a temporary directory and returns its path.
func (d dataset) write(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		fixedFile:  cmp.Or(d.fixed, validFixed),
		easterFile: cmp.Or(d.easter, validEaster),
	}
	bank := d.bank
	if bank == nil {
		bank = map[string]string{"bank_2026.csv": validBank}
	}
	maps.Copy(files, bank)
	return csvloadtest.WriteFiles(t, files)
}

func load(t *testing.T, d dataset) *CSVStore {
	t.Helper()
	s, err := LoadCSV(d.write(t))
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	return s
}

func TestLoadCSVValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		data     dataset
		wantErrs []string
	}{
		{
			name:     "invalid month",
			data:     dataset{fixed: strings.Replace(validFixed, "1,1,Año", "13,1,Año", 1)},
			wantErrs: []string{`national_fixed.csv:2: invalid month "13": want 1 to 12`},
		},
		{
			name:     "day that is not in every year",
			data:     dataset{fixed: strings.Replace(validFixed, "4,19,", "2,29,", 1)},
			wantErrs: []string{`national_fixed.csv:3: invalid day "29": want 1 to 28 for month 2`},
		},
		{
			name:     "duplicate fixed date",
			data:     dataset{fixed: validFixed + "7,5,Otro,Otra ley\n"},
			wantErrs: []string{"national_fixed.csv:5: duplicate date 07-05"},
		},
		{
			name:     "fixed date out of order",
			data:     dataset{fixed: validFixed + "6,24,Batalla de Carabobo,Ley de Fiestas Nacionales art. 1\n"},
			wantErrs: []string{"national_fixed.csv:5: date 06-24 is out of order: it must come after 07-05"},
		},
		{
			name:     "empty name",
			data:     dataset{fixed: strings.Replace(validFixed, ",Año Nuevo,", ",,", 1)},
			wantErrs: []string{"national_fixed.csv:2: empty name"},
		},
		{
			name:     "legal basis with surrounding whitespace",
			data:     dataset{easter: strings.Replace(validEaster, "Viernes Santo,LOTTT art. 184", "Viernes Santo,LOTTT art. 184 ", 1)},
			wantErrs: []string{`national_easter.csv:3: legal_basis "LOTTT art. 184 " has surrounding whitespace`},
		},
		{
			name:     "offset outside the year",
			data:     dataset{easter: strings.Replace(validEaster, "-48,", "-81,", 1)},
			wantErrs: []string{`national_easter.csv:2: invalid days_from_easter "-81": want -80 to 250`},
		},
		{
			name:     "duplicate offset",
			data:     dataset{easter: validEaster + "-2,Otro,Otra ley\n"},
			wantErrs: []string{"national_easter.csv:4: duplicate days_from_easter -2"},
		},
		{
			name:     "offset out of order",
			data:     dataset{easter: validEaster + "-3,Jueves Santo,LOTTT art. 184\n"},
			wantErrs: []string{"national_easter.csv:4: days_from_easter -3 is out of order: it must come after -2"},
		},
		{
			name:     "unexpected bank file name",
			data:     dataset{bank: map[string]string{"bank_2026.csv": validBank, "bank_26.csv": validBank}},
			wantErrs: []string{"bank_26.csv: unexpected file name: want bank_YYYY.csv"},
		},
		{
			name:     "invalid bank date",
			data:     dataset{bank: map[string]string{"bank_2026.csv": strings.Replace(validBank, "2026-03-19", "2026-3-19", 1)}},
			wantErrs: []string{`bank_2026.csv:4: invalid date "2026-3-19": want YYYY-MM-DD`},
		},
		{
			name:     "bank date in another year",
			data:     dataset{bank: map[string]string{"bank_2027.csv": validBank}},
			wantErrs: []string{"bank_2027.csv:2: date 2026-01-01 is not in 2027"},
		},
		{
			name:     "bank date on a weekend",
			data:     dataset{bank: map[string]string{"bank_2026.csv": validBank + "2026-03-21,Otro\n"}},
			wantErrs: []string{"bank_2026.csv:5: date 2026-03-21 falls on a Saturday"},
		},
		{
			name:     "duplicate bank date",
			data:     dataset{bank: map[string]string{"bank_2026.csv": validBank + "2026-03-19,Otro\n"}},
			wantErrs: []string{"bank_2026.csv:5: duplicate date 2026-03-19"},
		},
		{
			name:     "bank date out of order",
			data:     dataset{bank: map[string]string{"bank_2026.csv": validBank + "2026-03-02,Otro\n"}},
			wantErrs: []string{"bank_2026.csv:5: date 2026-03-02 is out of order: it must come after 2026-03-19"},
		},
		{
			name:     "empty bank holiday name",
			data:     dataset{bank: map[string]string{"bank_2026.csv": strings.Replace(validBank, ",Día de San José", ",", 1)}},
			wantErrs: []string{"bank_2026.csv:4: empty name"},
		},
		{
			name:     "bank calendar without holidays",
			data:     dataset{bank: map[string]string{"bank_2026.csv": "date,name\n"}},
			wantErrs: []string{"bank_2026.csv: no holidays"},
		},
		{
			name: "reports every error",
			data: dataset{
				fixed: strings.Replace(validFixed, "1,1,Año", "13,1,Año", 1),
				bank:  map[string]string{"bank_2027.csv": validBank},
			},
			wantErrs: []string{
				`national_fixed.csv:2: invalid month "13"`,
				"bank_2027.csv:2: date 2026-01-01 is not in 2027",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadCSV(tt.data.write(t))
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

func TestLoadCSVFileErrors(t *testing.T) {
	t.Run("missing directory", func(t *testing.T) {
		if _, err := LoadCSV(filepath.Join(t.TempDir(), "nope")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("LoadCSV() error = %v, want a not-exist error", err)
		}
	})

	for _, file := range []string{fixedFile, easterFile} {
		t.Run("missing "+file, func(t *testing.T) {
			dir := dataset{}.write(t)
			if err := os.Remove(filepath.Join(dir, file)); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadCSV(dir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("LoadCSV() error = %v, want a not-exist error", err)
			}
		})
	}

	t.Run("bank file with another header", func(t *testing.T) {
		dir := dataset{bank: map[string]string{"bank_2026.csv": "fecha,nombre\n"}}.write(t)
		if _, err := LoadCSV(dir); err == nil || !strings.Contains(err.Error(), "header mismatch") {
			t.Errorf("LoadCSV() error = %v, want a header mismatch", err)
		}
	})
}

func TestEaster(t *testing.T) {
	tests := []struct {
		year int
		want string
	}{
		{2013, "2013-03-31"},
		{2019, "2019-04-21"},
		{2025, "2025-04-20"},
		{2026, "2026-04-05"},
		{2027, "2027-03-28"},
		{2038, "2038-04-25"}, // latest possible date
		{2285, "2285-03-22"}, // earliest possible date
	}
	for _, tt := range tests {
		if got := easter(tt.year).Format(time.DateOnly); got != tt.want {
			t.Errorf("easter(%d) = %s, want %s", tt.year, got, tt.want)
		}
	}
}

func TestNational(t *testing.T) {
	s := load(t, dataset{})
	ctx := context.Background()

	tests := []struct {
		year int
		page store.Page
		want string
	}{
		{2026, store.Page{Limit: 100}, "2026-01-01 Año Nuevo, 2026-02-16 Lunes de Carnaval, " +
			"2026-04-03 Viernes Santo, 2026-04-19 Declaración de la Independencia, 2026-07-05 Día de la Independencia"},
		{2026, store.Page{Limit: 2, Offset: 1}, "2026-02-16 Lunes de Carnaval, 2026-04-03 Viernes Santo"},
		// In 2019 Good Friday fell on April 19: both holidays are listed,
		// the fixed one first.
		{2019, store.Page{Limit: 2, Offset: 2}, "2019-04-19 Declaración de la Independencia, 2019-04-19 Viernes Santo"},
		// The bounds of the supported years.
		{2013, store.Page{Limit: 1}, "2013-01-01 Año Nuevo"},
		{2030, store.Page{Limit: 1}, "2030-01-01 Año Nuevo"},
	}
	for _, tt := range tests {
		got, total, err := s.National(ctx, tt.year, tt.page)
		if err != nil {
			t.Fatalf("National(%d) error = %v", tt.year, err)
		}
		if desc := describe(got); desc != tt.want || total != 5 {
			t.Errorf("National(%d, %+v) = %s (total %d)\nwant %s (total 5)", tt.year, tt.page, desc, total, tt.want)
		}
	}

	got, _, _ := s.National(ctx, 2026, store.Page{Limit: 1})
	if got[0].LegalBasis != "LOTTT art. 184" {
		t.Errorf("LegalBasis = %q, want LOTTT art. 184", got[0].LegalBasis)
	}

	for _, year := range []int{2012, 2031} {
		if _, _, err := s.National(ctx, year, store.Page{Limit: 100}); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("National(%d) error = %v, want ErrNotFound", year, err)
		}
	}
}

func TestBank(t *testing.T) {
	s := load(t, dataset{bank: map[string]string{
		"bank_2026.csv": validBank,
		"bank_2025.csv": "date,name\n2025-01-01,Año Nuevo\n",
	}})
	ctx := context.Background()

	got, total, err := s.Bank(ctx, 2026, store.Page{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("Bank(2026) error = %v", err)
	}
	if desc, want := describe(got), "2026-02-16 Lunes de Carnaval, 2026-03-19 Día de San José"; desc != want || total != 3 {
		t.Errorf("Bank(2026) = %s (total %d)\nwant %s (total 3)", desc, total, want)
	}

	if _, _, err := s.Bank(ctx, 2027, store.Page{Limit: 100}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Bank(2027) error = %v, want ErrNotFound", err)
	}

	years, _ := s.BankYears(ctx)
	if !slices.Equal(years, []int{2025, 2026}) {
		t.Errorf("BankYears() = %v, want [2025 2026]", years)
	}
	// BankYears returns a copy: changing it does not change the store.
	years[0] = 1999
	if again, _ := s.BankYears(ctx); again[0] != 2025 {
		t.Errorf("BankYears() = %v after changing a previous result", again)
	}
}

// TestRealData loads the dataset published in data/holidays, so CI fails if
// anyone commits invalid data.
func TestRealData(t *testing.T) {
	s, err := LoadCSV(realDataDir)
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	ctx := context.Background()

	for year := minYear; year <= maxYear; year++ {
		if _, total, _ := s.National(ctx, year, store.Page{}); total != 14 {
			t.Errorf("got %d national holidays in %d, want 14", total, year)
		}
	}

	years, _ := s.BankYears(ctx)
	if !slices.Equal(years, []int{2026}) {
		t.Errorf("BankYears() = %v, want [2026]", years)
	}
	if _, total, _ := s.Bank(ctx, 2026, store.Page{}); total != 22 {
		t.Errorf("got %d bank holidays in 2026, want 22", total)
	}

	// Banks close on every national holiday that falls on a weekday, so each
	// bank calendar must include them. A failure here means the calendar or
	// the rules need a review.
	for _, year := range years {
		bank, _, _ := s.Bank(ctx, year, store.Page{Limit: 1000})
		closed := make(map[time.Time]bool, len(bank))
		for _, h := range bank {
			closed[h.Date] = true
		}
		national, _, err := s.National(ctx, year, store.Page{Limit: 1000})
		if err != nil {
			continue // a bank calendar outside the supported national years
		}
		for _, h := range national {
			if wd := h.Date.Weekday(); wd != time.Saturday && wd != time.Sunday && !closed[h.Date] {
				t.Errorf("national holiday %s (%s) is missing from bank_%d.csv",
					h.Date.Format(time.DateOnly), h.Name, year)
			}
		}
	}
}

// describe lists holidays as "date name", to compare them in one line.
func describe(holidays []Holiday) string {
	parts := make([]string, len(holidays))
	for i, h := range holidays {
		parts[i] = h.Date.Format(time.DateOnly) + " " + h.Name
	}
	return strings.Join(parts, ", ")
}
