package holidays

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

// Files of the holidays dataset, relative to its directory. There is one bank
// file per published year, named bank_YYYY.csv.
const (
	fixedFile      = "national_fixed.csv"
	easterFile     = "national_easter.csv"
	bankFilePrefix = "bank_"
)

var (
	fixedHeader  = []string{"month", "day", "name", "legal_basis"}
	easterHeader = []string{"days_from_easter", "name", "legal_basis"}
	bankHeader   = []string{"date", "name"}
)

var bankFileRe = regexp.MustCompile(`^bank_([0-9]{4})\.csv$`)

// Easter falls between March 22 and April 25, so these offsets keep a holiday
// inside the year of its Easter in every year: March 22 minus 80 days is
// January 1 or 2, and April 25 plus 250 days is December 31.
const (
	minEasterOffset = -80
	maxEasterOffset = 250
)

// fixedRule is a national holiday on the same day every year.
type fixedRule struct {
	month      time.Month
	day        int
	name       string
	legalBasis string
}

// easterRule is a national holiday a number of days away from Easter Sunday.
type easterRule struct {
	offset     int
	name       string
	legalBasis string
}

// CSVStore serves holidays loaded from CSV files into memory.
//
// It is read-only after loading, so it is safe for concurrent use.
type CSVStore struct {
	national  map[int][]Holiday // year → national holidays, for every year in [minYear, maxYear]
	bank      map[int][]Holiday // year → bank holidays, for every published year
	bankYears []int             // published years, in order
}

// CSVStore must satisfy the handlers' port; this fails to compile otherwise.
var _ reader = (*CSVStore)(nil)

// LoadCSV reads and validates the holidays dataset in dir, and calculates
// the national holidays of every supported year.
//
// It reports every validation problem it finds, not only the first one, so
// the data can be fixed in a single pass.
func LoadCSV(dir string) (*CSVStore, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fixedRows, err := csvload.ReadFile(filepath.Join(dir, fixedFile), fixedHeader)
	if err != nil {
		return nil, err
	}
	easterRows, err := csvload.ReadFile(filepath.Join(dir, easterFile), easterHeader)
	if err != nil {
		return nil, err
	}

	s := &CSVStore{national: make(map[int][]Holiday), bank: make(map[int][]Holiday)}
	fixed, errs := loadFixedRules(fixedRows)
	easterRules, easterErrs := loadEasterRules(easterRows)
	errs = append(errs, easterErrs...)

	// ReadDir sorts entries by name, so the years come out in order.
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, bankFilePrefix) {
			continue
		}
		m := bankFileRe.FindStringSubmatch(name)
		if m == nil {
			errs = append(errs, fmt.Errorf("%s: unexpected file name: want bank_YYYY.csv", name))
			continue
		}
		year, _ := strconv.Atoi(m[1]) // the pattern guarantees 4 digits
		rows, err := csvload.ReadFile(filepath.Join(dir, name), bankHeader)
		if err != nil {
			return nil, err
		}
		holidays, bankErrs := loadBankHolidays(name, year, rows)
		errs = append(errs, bankErrs...)
		s.bank[year] = holidays
		s.bankYears = append(s.bankYears, year)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid holidays data in %s:\n%w", dir, errors.Join(errs...))
	}
	for year := minYear; year <= maxYear; year++ {
		s.national[year] = nationalHolidays(year, fixed, easterRules)
	}

	return s, nil
}

func loadFixedRules(rows []csvload.Record) ([]fixedRule, []error) {
	var rules []fixedRule
	var errs []error
	seen := make(map[string]bool, len(rows))
	prev := "" // MM-DD of the previous row

	for _, r := range rows {
		fail := func(format string, args ...any) {
			errs = append(errs, csvload.LineError(fixedFile, r.Line, format, args...))
		}
		rule := fixedRule{name: r.Fields[2], legalBasis: r.Fields[3]}
		csvload.CheckText("name", rule.name, fail)
		csvload.CheckText("legal_basis", rule.legalBasis, fail)

		month, err := strconv.Atoi(r.Fields[0])
		if err != nil || month < 1 || month > 12 {
			fail("invalid month %q: want 1 to 12", r.Fields[0])
			continue
		}
		rule.month = time.Month(month)
		// The day must exist every year, so February 29 is rejected: 2001 is
		// a common year.
		lastDay := time.Date(2001, rule.month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		day, err := strconv.Atoi(r.Fields[1])
		if err != nil || day < 1 || day > lastDay {
			fail("invalid day %q: want 1 to %d for month %d", r.Fields[1], lastDay, month)
			continue
		}
		rule.day = day

		date := fmt.Sprintf("%02d-%02d", month, day)
		if seen[date] {
			fail("duplicate date %s", date)
		}
		seen[date] = true
		csvload.CheckOrder("date", prev, date, fail)
		prev = date

		rules = append(rules, rule)
	}

	return rules, errs
}

func loadEasterRules(rows []csvload.Record) ([]easterRule, []error) {
	var rules []easterRule
	var errs []error
	seen := make(map[int]bool, len(rows))
	prev := math.MinInt // offset of the previous row

	for _, r := range rows {
		fail := func(format string, args ...any) {
			errs = append(errs, csvload.LineError(easterFile, r.Line, format, args...))
		}
		rule := easterRule{name: r.Fields[1], legalBasis: r.Fields[2]}
		csvload.CheckText("name", rule.name, fail)
		csvload.CheckText("legal_basis", rule.legalBasis, fail)

		offset, err := strconv.Atoi(r.Fields[0])
		if err != nil || offset < minEasterOffset || offset > maxEasterOffset {
			fail("invalid days_from_easter %q: want %d to %d", r.Fields[0], minEasterOffset, maxEasterOffset)
			continue
		}
		rule.offset = offset

		if seen[offset] {
			fail("duplicate days_from_easter %d", offset)
		}
		seen[offset] = true
		csvload.CheckOrder("days_from_easter", prev, offset, fail)
		prev = offset

		rules = append(rules, rule)
	}

	return rules, errs
}

// loadBankHolidays validates the rows of the bank file of year.
func loadBankHolidays(file string, year int, rows []csvload.Record) ([]Holiday, []error) {
	var holidays []Holiday
	var errs []error
	seen := make(map[string]bool, len(rows))
	prev := "" // date of the previous row

	for _, r := range rows {
		fail := func(format string, args ...any) {
			errs = append(errs, csvload.LineError(file, r.Line, format, args...))
		}
		value, name := r.Fields[0], r.Fields[1]
		csvload.CheckText("name", name, fail)

		date, err := time.Parse(time.DateOnly, value)
		if err != nil {
			fail("invalid date %q: want YYYY-MM-DD", value)
			continue
		}
		if date.Year() != year {
			fail("date %s is not in %d", value, year)
		}
		// Banks are closed on weekends anyway: the calendar only lists weekdays.
		if wd := date.Weekday(); wd == time.Saturday || wd == time.Sunday {
			fail("date %s falls on a %s", value, wd)
		}
		if seen[value] {
			fail("duplicate date %s", value)
		}
		seen[value] = true
		csvload.CheckOrder("date", prev, value, fail)
		prev = value

		holidays = append(holidays, Holiday{Date: date, Name: name})
	}
	if len(rows) == 0 {
		// A published calendar always has holidays: an empty file would make
		// the year look like one without any.
		errs = append(errs, fmt.Errorf("%s: no holidays", file))
	}

	return holidays, errs
}

// nationalHolidays returns the national holidays of year, ordered by date.
// Two holidays can fall on the same day (in 2019 Good Friday was April 19):
// both are listed, fixed ones first.
func nationalHolidays(year int, fixed []fixedRule, easterRules []easterRule) []Holiday {
	holidays := make([]Holiday, 0, len(fixed)+len(easterRules))
	for _, r := range fixed {
		date := time.Date(year, r.month, r.day, 0, 0, 0, 0, time.UTC)
		holidays = append(holidays, Holiday{Date: date, Name: r.name, LegalBasis: r.legalBasis})
	}
	sunday := easter(year)
	for _, r := range easterRules {
		date := sunday.AddDate(0, 0, r.offset)
		holidays = append(holidays, Holiday{Date: date, Name: r.name, LegalBasis: r.legalBasis})
	}
	slices.SortStableFunc(holidays, func(a, b Holiday) int { return a.Date.Compare(b.Date) })
	return holidays
}

// National returns a page of the national holidays of year, ordered by date,
// and their total count. It returns store.ErrNotFound for a year outside
// [minYear, maxYear].
func (s *CSVStore) National(_ context.Context, year int, p store.Page) ([]Holiday, int, error) {
	all, ok := s.national[year]
	if !ok {
		return nil, 0, store.ErrNotFound
	}
	return store.Paginate(all, p), len(all), nil
}

// Bank returns a page of the bank holidays of year, ordered by date, and
// their total count. It returns store.ErrNotFound if the calendar of year
// has not been loaded.
func (s *CSVStore) Bank(_ context.Context, year int, p store.Page) ([]Holiday, int, error) {
	all, ok := s.bank[year]
	if !ok {
		return nil, 0, store.ErrNotFound
	}
	return store.Paginate(all, p), len(all), nil
}

// BankYears returns the years with a bank calendar, in order.
func (s *CSVStore) BankYears(_ context.Context) ([]int, error) {
	return slices.Clone(s.bankYears), nil
}
