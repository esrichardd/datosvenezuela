// Package csvload reads the CSV files of the data directory.
//
// It checks the CSV structure (the header is exactly the expected one and every
// row has the same number of fields) and offers the value checks every dataset
// shares: line-tagged errors, text without stray whitespace and codes in
// order. Rules specific to a dataset are validated by its domain package.
package csvload

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// utf8BOM is the byte order mark some editors (such as Excel) add at the start
// of UTF-8 files.
const utf8BOM = "\ufeff"

// Record is a data row of a CSV file.
type Record struct {
	// Line is the line number of the row in the file, starting at 1. The
	// header is line 1.
	Line int
	// Fields holds the values in the same order as the header.
	Fields []string
}

// ReadFile reads the CSV file at path. See Read.
func ReadFile(path string, header []string) ([]Record, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from the server config, not from requests.
	if err != nil {
		return nil, err
	}
	// The file is only read, so a close error cannot lose data.
	defer func() { _ = f.Close() }()

	return Read(f, filepath.Base(path), header)
}

// Read reads CSV data from r and returns its data rows.
//
// The first row must match header exactly. name identifies the source in
// error messages, usually the file name.
func Read(r io.Reader, name string, header []string) ([]Record, error) {
	cr := csv.NewReader(r)
	// Field counts are checked below to produce clearer error messages.
	cr.FieldsPerRecord = -1

	got, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: missing header", name)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	got[0] = strings.TrimPrefix(got[0], utf8BOM)
	if !slices.Equal(got, header) {
		return nil, fmt.Errorf("%s: header mismatch: got %q, want %q", name, got, header)
	}

	var records []Record
	for {
		fields, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		line, _ := cr.FieldPos(0)
		if len(fields) != len(header) {
			return nil, fmt.Errorf("%s:%d: expected %d fields, got %d", name, line, len(header), len(fields))
		}
		records = append(records, Record{Line: line, Fields: fields})
	}

	return records, nil
}

// FailFunc records a validation problem found in a row. Domain packages build
// one per row that tags each message with the file and line, so the checks
// below can report several problems instead of stopping at the first.
type FailFunc func(format string, args ...any)

// LineError returns a validation error for a line of a file, formatted as
// "file:line: message".
func LineError(file string, line int, format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s", file, line, fmt.Sprintf(format, args...))
}

// CheckText reports an empty value or one with surrounding whitespace. field
// names the column in the message.
func CheckText(field, value string, fail FailFunc) {
	switch {
	case value == "":
		fail("empty %s", field)
	case value != strings.TrimSpace(value):
		fail("%s %q has surrounding whitespace", field, value)
	}
}

// CheckOrder reports a code that sorts before prev, the code of the previous
// row. Pass "" for the first row.
//
// Stores serve rows in file order and the API promises lists ordered by code,
// so the files must be sorted. Codes have a fixed width, which makes comparing
// them as text the same as comparing their numbers. An equal code is left to
// the duplicate check, which gives a clearer message.
func CheckOrder(prev, code string, fail FailFunc) {
	if code < prev {
		fail("code %s is out of order: it must come after %s", code, prev)
	}
}
