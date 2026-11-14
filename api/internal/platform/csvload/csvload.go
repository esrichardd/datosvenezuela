// Package csvload reads the CSV files of the data directory.
//
// It only knows about CSV structure: it checks that the header is exactly the
// expected one and that every row has the same number of fields. What the
// values mean is validated by each domain package.
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
