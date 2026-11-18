package banks

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

// banksFile is the banks dataset, relative to its directory.
const banksFile = "banks.csv"

var banksHeader = []string{"code", "name", "legal_name"}

var codeRe = regexp.MustCompile(`^[0-9]{4}$`)

// CSVStore serves banks loaded from a CSV file into memory.
//
// It is read-only after loading, so it is safe for concurrent use.
type CSVStore struct {
	banks  []Bank
	byCode map[string]int // code → index in banks
}

// CSVStore must satisfy the handlers' port; this fails to compile otherwise.
var _ reader = (*CSVStore)(nil)

// LoadCSV reads and validates the banks dataset in dir.
//
// It reports every validation problem it finds, not only the first one, so
// the data can be fixed in a single pass.
func LoadCSV(dir string) (*CSVStore, error) {
	rows, err := csvload.ReadFile(filepath.Join(dir, banksFile), banksHeader)
	if err != nil {
		return nil, err
	}

	s := &CSVStore{byCode: make(map[string]int, len(rows))}
	var errs []error
	prev := "" // code of the previous row

	for _, r := range rows {
		b := Bank{Code: r.Fields[0], Name: r.Fields[1], LegalName: r.Fields[2]}
		fail := func(format string, args ...any) {
			errs = append(errs, csvload.LineError(banksFile, r.Line, format, args...))
		}

		if !codeRe.MatchString(b.Code) {
			fail("invalid code %q: want 4 digits", b.Code)
		}
		if _, dup := s.byCode[b.Code]; dup {
			fail("duplicate code %s", b.Code)
		}
		csvload.CheckOrder(prev, b.Code, fail)
		prev = b.Code
		csvload.CheckText("name", b.Name, fail)
		csvload.CheckText("legal_name", b.LegalName, fail)

		s.byCode[b.Code] = len(s.banks)
		s.banks = append(s.banks, b)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid banks data in %s:\n%w", dir, errors.Join(errs...))
	}
	return s, nil
}

// Banks returns a page of all banks, ordered by code, and the total count.
func (s *CSVStore) Banks(_ context.Context, p store.Page) ([]Bank, int, error) {
	return store.Paginate(s.banks, p), len(s.banks), nil
}

// Bank returns the bank with the given code.
func (s *CSVStore) Bank(_ context.Context, code string) (Bank, error) {
	i, ok := s.byCode[code]
	if !ok {
		return Bank{}, store.ErrNotFound
	}
	return s.banks[i], nil
}
