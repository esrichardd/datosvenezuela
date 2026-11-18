package banks

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

const validBanks = `code,name,legal_name
0001,Banco Central de Venezuela,Banco Central de Venezuela
0102,Banco de Venezuela,"Banco de Venezuela, S.A. Banco Universal"
0105,Banco Mercantil,"Mercantil, C.A., Banco Universal"
`

// realDataDir is the dataset published in the repository.
var realDataDir = filepath.Join("..", "..", "..", "data", "banks")

// writeDataset writes banks.csv to a temporary directory and returns its path.
func writeDataset(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, banksFile), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func loadValid(t *testing.T) *CSVStore {
	t.Helper()
	s, err := LoadCSV(writeDataset(t, validBanks))
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	return s
}

func TestLoadCSVValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		banks    string
		wantErrs []string
	}{
		{
			name:     "code without leading zero",
			banks:    strings.Replace(validBanks, "0102,", "102,", 1),
			wantErrs: []string{`banks.csv:3: invalid code "102": want 4 digits`},
		},
		{
			name:     "code with a letter",
			banks:    strings.Replace(validBanks, "0102,", "01A2,", 1),
			wantErrs: []string{`banks.csv:3: invalid code "01A2": want 4 digits`},
		},
		{
			name:     "duplicate code",
			banks:    validBanks + "0105,Otro,Otro C.A.\n",
			wantErrs: []string{"banks.csv:5: duplicate code 0105"},
		},
		{
			name:     "out of order",
			banks:    validBanks + "0104,Banco Venezolano de Crédito,Venezolano de Crédito S.A.\n",
			wantErrs: []string{"banks.csv:5: code 0104 is out of order: it must come after 0105"},
		},
		{
			name:     "empty name",
			banks:    strings.Replace(validBanks, "0102,Banco de Venezuela,", "0102,,", 1),
			wantErrs: []string{"banks.csv:3: empty name"},
		},
		{
			name:     "legal name with surrounding whitespace",
			banks:    strings.Replace(validBanks, `"Mercantil, C.A., Banco Universal"`, `"Mercantil, C.A., Banco Universal "`, 1),
			wantErrs: []string{`banks.csv:4: legal_name "Mercantil, C.A., Banco Universal " has surrounding whitespace`},
		},
		{
			name: "reports every error",
			banks: strings.Replace(
				strings.Replace(validBanks, "0102,", "102,", 1),
				"0105,Banco Mercantil,", "0105,,", 1),
			wantErrs: []string{
				`banks.csv:3: invalid code "102"`,
				"banks.csv:4: empty name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadCSV(writeDataset(t, tt.banks))
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
	if _, err := LoadCSV(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("LoadCSV() error = %v, want a not-exist error", err)
	}
}

func TestBank(t *testing.T) {
	s := loadValid(t)
	ctx := context.Background()

	b, err := s.Bank(ctx, "0102")
	if err != nil {
		t.Fatalf("Bank(0102) error = %v", err)
	}
	want := Bank{Code: "0102", Name: "Banco de Venezuela", LegalName: "Banco de Venezuela, S.A. Banco Universal"}
	if b != want {
		t.Errorf("Bank(0102) = %+v, want %+v", b, want)
	}

	// The leading zero is part of the code: 102 is not 0102.
	for _, code := range []string{"9999", "102"} {
		if _, err := s.Bank(ctx, code); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Bank(%s) error = %v, want ErrNotFound", code, err)
		}
	}
}

func TestBanks(t *testing.T) {
	s := loadValid(t)

	got, total, err := s.Banks(context.Background(), store.Page{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("Banks() error = %v", err)
	}
	if total != 3 || len(got) != 2 || got[0].Code != "0102" || got[1].Code != "0105" {
		t.Errorf("Banks() = %+v (total %d), want 0102 and 0105 (total 3)", got, total)
	}
}

// TestRealData loads the dataset published in data/banks, so CI fails if
// anyone commits invalid data.
func TestRealData(t *testing.T) {
	s, err := LoadCSV(realDataDir)
	if err != nil {
		t.Fatalf("LoadCSV() error = %v", err)
	}
	if _, total, _ := s.Banks(context.Background(), store.Page{}); total != 27 {
		t.Errorf("got %d banks, want 27", total)
	}
}
