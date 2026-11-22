package main

import (
	"errors"
	"time"

	"github.com/esrichardd/datosvenezuela/api/internal/datapackage"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload"
)

// errWrite is returned by failingWriter.
var errWrite = errors.New("write failed")

// failingWriter accepts n bytes and then fails, to test how write errors
// are handled at every point of a file.
type failingWriter struct{ n int }

func (w *failingWriter) Write(p []byte) (int, error) {
	if len(p) <= w.n {
		w.n -= len(p)
		return len(p), nil
	}
	n := w.n
	w.n = 0
	return n, errWrite
}

// testDataset returns a small dataset with a table that refers to another,
// an empty table, an optional value and text that needs escaping.
func testDataset() *dataset {
	pkg := &datapackage.Package{
		Name:     "test-places",
		Title:    "Places",
		Licenses: []datapackage.License{{Name: "CC-BY-4.0"}},
		Resources: []datapackage.Resource{
			{
				Name: "states", Title: "States", Path: "states.csv",
				Schema: datapackage.Schema{
					Fields: []datapackage.Field{
						{Name: "code", Type: "string", Constraints: datapackage.Constraints{Required: true, Unique: true}},
						{Name: "name", Type: "string", Constraints: datapackage.Constraints{Required: true, Unique: true}},
						{Name: "capital", Type: "string"},
					},
					PrimaryKey: []string{"code"},
				},
			},
			{
				Name: "events", Title: "Events", Path: "events.csv",
				Schema: datapackage.Schema{
					Fields: []datapackage.Field{
						{Name: "state_code", Type: "string", Constraints: datapackage.Constraints{Required: true}},
						{Name: "day", Type: "integer", Constraints: datapackage.Constraints{Required: true}},
						{Name: "date", Type: "date"},
					},
					PrimaryKey: []string{"state_code", "day"},
					ForeignKeys: []datapackage.ForeignKey{
						{Fields: []string{"state_code"}, Reference: datapackage.Reference{Resource: "states", Fields: []string{"code"}}},
					},
				},
			},
			{
				Name: "notes", Title: "Notes", Path: "notes.csv",
				Schema: datapackage.Schema{
					Fields:     []datapackage.Field{{Name: "text", Type: "string"}},
					PrimaryKey: []string{"text"},
				},
			},
		},
	}
	rows := [][]csvload.Record{
		{
			{Line: 2, Fields: []string{"VE01", "Distrito Capital", "Caracas"}},
			{Line: 3, Fields: []string{"VE25", `O'Leary "Federal"`, ""}},
		},
		{
			{Line: 2, Fields: []string{"VE01", "7", "2026-01-07"}},
		},
		nil,
	}

	ds := &dataset{
		id:      "places",
		version: "2026.11.21",
		date:    time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC),
		pkg:     pkg,
	}
	for i := range pkg.Resources {
		ds.tables = append(ds.tables, datapackage.Table{Resource: &pkg.Resources[i], Rows: rows[i]})
	}
	return ds
}

// invalidIntegerDataset returns testDataset with a value that is not an
// integer in an integer field.
func invalidIntegerDataset() *dataset {
	ds := testDataset()
	ds.tables[1].Rows[0].Fields[1] = "x"
	return ds
}
