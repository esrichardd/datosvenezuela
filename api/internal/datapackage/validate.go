package datapackage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload"
)

// fieldTypes are the Table Schema types the datasets use. Add one here, with
// its value check in checkValue, before using it in a manifest.
var fieldTypes = []string{"string", "integer", "date"}

// table is a resource whose CSV file was read.
type table struct {
	file  string
	rows  []csvload.Record
	index map[string]int // field name → column
}

// key joins the values of fields in row. empty is true when all are empty,
// which means there is no value to check.
func (t *table) key(row csvload.Record, fields []string) (key string, empty bool) {
	values := make([]string, len(fields))
	empty = true
	for i, f := range fields {
		values[i] = row.Fields[t.index[f]]
		if values[i] != "" {
			empty = false
		}
	}
	return strings.Join(values, "\x00"), empty
}

// Validate checks that the manifest is complete and that it describes the
// CSV files in dir exactly:
//
//   - every resource points to a CSV file in dir whose header lists the
//     schema's fields, in order, and every CSV file in dir has a resource;
//   - keys refer to fields and resources that exist;
//   - every value meets its field's type and constraints, primary keys are
//     unique and foreign keys point to existing rows.
//
// It reports every problem it finds, not just the first.
func (p *Package) Validate(dir string) error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if p.Name == "" {
		fail("missing name")
	}
	if len(p.Licenses) == 0 {
		fail("missing licenses")
	}
	if len(p.Resources) == 0 {
		fail("missing resources")
	}

	byName := make(map[string]*Resource, len(p.Resources))
	for i := range p.Resources {
		r := &p.Resources[i]
		if _, dup := byName[r.Name]; dup {
			fail("duplicate resource %q", r.Name)
		}
		byName[r.Name] = r
	}

	tables := make(map[string]*table, len(p.Resources))
	for i := range p.Resources {
		r := &p.Resources[i]
		t, rErrs := loadTable(dir, r, byName)
		errs = append(errs, rErrs...)
		if t != nil {
			tables[r.Name] = t
		}
	}
	for i := range p.Resources {
		r := &p.Resources[i]
		if t := tables[r.Name]; t != nil {
			errs = append(errs, checkForeignKeys(r, t, tables)...)
		}
	}

	errs = append(errs, checkUndescribed(dir, p.Resources)...)
	return errors.Join(errs...)
}

// loadTable checks the definition of r and, if it is sound, reads its CSV
// file and checks every value. It returns nil when the file could not be
// checked.
func loadTable(dir string, r *Resource, byName map[string]*Resource) (*table, []error) {
	errs := checkDefinition(r, byName)
	if len(errs) > 0 {
		return nil, errs
	}

	names := make([]string, len(r.Schema.Fields))
	index := make(map[string]int, len(r.Schema.Fields))
	for i, f := range r.Schema.Fields {
		names[i] = f.Name
		index[f.Name] = i
	}

	rows, err := csvload.ReadFile(filepath.Join(dir, r.Path), names)
	if err != nil {
		return nil, []error{fmt.Errorf("resource %q: %w", r.Name, err)}
	}
	t := &table{file: r.Path, rows: rows, index: index}

	for i, f := range r.Schema.Fields {
		// The pattern compiled in checkDefinition, so it is valid here. As in
		// Table Schema, it must match the whole value.
		var pattern *regexp.Regexp
		if f.Constraints.Pattern != "" {
			pattern = regexp.MustCompile(`^(?:` + f.Constraints.Pattern + `)$`)
		}
		seen := make(map[string]bool, len(rows))
		for _, row := range rows {
			v := row.Fields[i]
			if err := checkValue(f, v, pattern); err != nil {
				errs = append(errs, csvload.LineError(t.file, row.Line, "%v", err))
			}
			if f.Constraints.Unique && v != "" {
				if seen[v] {
					errs = append(errs, csvload.LineError(t.file, row.Line, "duplicate %s %q", f.Name, v))
				}
				seen[v] = true
			}
		}
	}

	keys := make(map[string]bool, len(rows))
	for _, row := range rows {
		k, _ := t.key(row, r.Schema.PrimaryKey)
		if keys[k] {
			errs = append(errs, csvload.LineError(t.file, row.Line, "duplicate primary key %v", strings.Split(k, "\x00")))
		}
		keys[k] = true
	}
	return t, errs
}

// checkDefinition checks the parts of a resource that do not depend on its
// data: type, format, path, fields and keys.
func checkDefinition(r *Resource, byName map[string]*Resource) []error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("resource %q: %s", r.Name, fmt.Sprintf(format, args...)))
	}

	if r.Name == "" {
		fail("missing name")
	}
	if r.Type != "table" {
		fail("type is %q, want table", r.Type)
	}
	if r.Format != "csv" {
		fail("format is %q, want csv", r.Format)
	}
	if r.Path == "" || filepath.Base(r.Path) != r.Path {
		fail("path %q must be the name of a file in the dataset directory", r.Path)
	}

	if len(r.Schema.Fields) == 0 {
		fail("missing fields")
	}
	fields := make(map[string]bool, len(r.Schema.Fields))
	for _, f := range r.Schema.Fields {
		if fields[f.Name] {
			fail("duplicate field %q", f.Name)
		}
		fields[f.Name] = true
		if !slices.Contains(fieldTypes, f.Type) {
			fail("field %q has unsupported type %q", f.Name, f.Type)
		}
		if _, err := regexp.Compile(f.Constraints.Pattern); err != nil {
			fail("field %q has an invalid pattern: %v", f.Name, err)
		}
	}

	if len(r.Schema.PrimaryKey) == 0 {
		fail("missing primary key")
	}
	for _, name := range r.Schema.PrimaryKey {
		if !fields[name] {
			fail("primary key field %q does not exist", name)
		}
	}

	for _, fk := range r.Schema.ForeignKeys {
		for _, name := range fk.Fields {
			if !fields[name] {
				fail("foreign key field %q does not exist", name)
			}
		}
		if len(fk.Fields) != len(fk.Reference.Fields) {
			fail("foreign key %v refers to %d fields, want %d", fk.Fields, len(fk.Reference.Fields), len(fk.Fields))
		}
		target, ok := byName[fk.Reference.Resource]
		if !ok {
			fail("foreign key %v refers to unknown resource %q", fk.Fields, fk.Reference.Resource)
			continue
		}
		for _, name := range fk.Reference.Fields {
			if !slices.ContainsFunc(target.Schema.Fields, func(f Field) bool { return f.Name == name }) {
				fail("foreign key %v refers to unknown field %s.%s", fk.Fields, target.Name, name)
			}
		}
	}
	return errs
}

// checkValue checks one value against its field's type and constraints.
// pattern is nil when the field has none. An empty value is a missing one: it is only an error if the field is
// required.
func checkValue(f Field, v string, pattern *regexp.Regexp) error {
	c := f.Constraints
	if v == "" {
		if c.Required {
			return fmt.Errorf("missing %s", f.Name)
		}
		return nil
	}

	switch f.Type {
	case "integer":
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid %s %q: want an integer", f.Name, v)
		}
		if c.Minimum != nil && n < *c.Minimum {
			return fmt.Errorf("%s %d is below the minimum %d", f.Name, n, *c.Minimum)
		}
		if c.Maximum != nil && n > *c.Maximum {
			return fmt.Errorf("%s %d is above the maximum %d", f.Name, n, *c.Maximum)
		}
	case "date":
		if _, err := time.Parse(time.DateOnly, v); err != nil {
			return fmt.Errorf("invalid %s %q: want a date as YYYY-MM-DD", f.Name, v)
		}
	}

	if pattern != nil && !pattern.MatchString(v) {
		return fmt.Errorf("%s %q does not match the pattern %s", f.Name, v, c.Pattern)
	}
	if len(c.Enum) > 0 && !slices.Contains(c.Enum, v) {
		return fmt.Errorf("%s %q is not one of %v", f.Name, v, c.Enum)
	}
	return nil
}

// checkForeignKeys checks that every foreign key value of t exists in the
// table it refers to.
func checkForeignKeys(r *Resource, t *table, tables map[string]*table) []error {
	var errs []error
	for _, fk := range r.Schema.ForeignKeys {
		target := tables[fk.Reference.Resource]
		if target == nil {
			continue // its own errors were already reported
		}
		existing := make(map[string]bool, len(target.rows))
		for _, row := range target.rows {
			k, _ := target.key(row, fk.Reference.Fields)
			existing[k] = true
		}
		for _, row := range t.rows {
			k, empty := t.key(row, fk.Fields)
			if !empty && !existing[k] {
				errs = append(errs, csvload.LineError(t.file, row.Line,
					"%v %v not found in %s", fk.Fields, strings.Split(k, "\x00"), fk.Reference.Resource))
			}
		}
	}
	return errs
}

// checkUndescribed reports CSV files in dir that no resource describes.
func checkUndescribed(dir string, resources []Resource) []error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []error{err}
	}
	var errs []error
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".csv" {
			continue
		}
		if !slices.ContainsFunc(resources, func(r Resource) bool { return r.Path == name }) {
			errs = append(errs, fmt.Errorf("%s: no resource in %s describes it", name, FileName))
		}
	}
	return errs
}
