package main

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/esrichardd/datosvenezuela/api/internal/datapackage"
)

// sqlTypes maps Table Schema types to PostgreSQL types.
var sqlTypes = map[string]string{
	"string":  "text",
	"integer": "integer",
	"date":    "date",
}

// writeSQL writes the dataset as a PostgreSQL script that creates its tables
// and fills them, in a single transaction. Foreign keys are added at the end,
// so tables can be created and filled in any order.
func writeSQL(w io.Writer, ds *dataset) error {
	sw := &stickyWriter{w: w}
	sw.printf("-- %s (%s), datosvenezuela %s.\n", ds.pkg.Title, ds.pkg.Name, ds.version)
	sw.printf("-- License: %s.\n\nBEGIN;\n", licenseNames(ds.pkg.Licenses))

	for _, t := range ds.tables {
		r := t.Resource
		fields := r.Schema.Fields
		names := make([]string, len(fields))
		for i, f := range fields {
			names[i] = quoteIdent(f.Name)
		}

		sw.printf("\nCREATE TABLE %s (\n", quoteIdent(r.Name))
		for _, f := range fields {
			typ, ok := sqlTypes[f.Type]
			if !ok {
				return fmt.Errorf("%s.%s: unsupported type %q", r.Name, f.Name, f.Type)
			}
			sw.printf("  %s %s%s,\n", quoteIdent(f.Name), typ, notNull(f))
		}
		sw.printf("  PRIMARY KEY (%s)", quoteIdents(r.Schema.PrimaryKey))
		for _, f := range fields {
			if f.Constraints.Unique && !slices.Equal(r.Schema.PrimaryKey, []string{f.Name}) {
				sw.printf(",\n  UNIQUE (%s)", quoteIdent(f.Name))
			}
		}
		sw.printf("\n);\n")

		if len(t.Rows) == 0 {
			continue
		}
		sw.printf("\nINSERT INTO %s (%s) VALUES\n", quoteIdent(r.Name), strings.Join(names, ", "))
		for i, row := range t.Rows {
			values := make([]string, len(fields))
			for j, f := range fields {
				v, err := sqlValue(f, row.Fields[j])
				if err != nil {
					return err
				}
				values[j] = v
			}
			end := ","
			if i == len(t.Rows)-1 {
				end = ";"
			}
			sw.printf("  (%s)%s\n", strings.Join(values, ", "), end)
		}
	}

	for _, t := range ds.tables {
		for _, fk := range t.Resource.Schema.ForeignKeys {
			sw.printf("\nALTER TABLE %s ADD FOREIGN KEY (%s) REFERENCES %s (%s);\n",
				quoteIdent(t.Resource.Name), quoteIdents(fk.Fields),
				quoteIdent(fk.Reference.Resource), quoteIdents(fk.Reference.Fields))
		}
	}

	sw.printf("\nCOMMIT;\n")
	return sw.err
}

// sqlValue returns the SQL literal for the value v of field f.
func sqlValue(f datapackage.Field, v string) (string, error) {
	switch {
	case v == "":
		return "NULL", nil
	case f.Type == "integer":
		n, err := strconv.Atoi(v)
		if err != nil {
			return "", fmt.Errorf("%w: %s %q is not an integer", errInvalidValue, f.Name, v)
		}
		return strconv.Itoa(n), nil
	default:
		// A string or a date, which PostgreSQL converts from text.
		return "'" + strings.ReplaceAll(v, "'", "''") + "'", nil
	}
}

func notNull(f datapackage.Field) string {
	if f.Constraints.Required {
		return " NOT NULL"
	}
	return ""
}

// quoteIdent quotes an identifier, so any name is valid, even a reserved
// word.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteIdents(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = quoteIdent(n)
	}
	return strings.Join(quoted, ", ")
}

func licenseNames(licenses []datapackage.License) string {
	names := make([]string, len(licenses))
	for i, l := range licenses {
		names[i] = l.Name
	}
	return strings.Join(names, ", ")
}
