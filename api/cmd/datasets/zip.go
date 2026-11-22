package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"io"
)

// writeZIP writes the dataset as a ZIP with a folder named after it, holding
// its datapackage.json and its CSV files: a Data Package that tools such as
// Frictionless can read.
func writeZIP(w io.Writer, ds *dataset) error {
	type entry struct {
		name  string
		write func(io.Writer) error
	}
	entries := []entry{{"datapackage.json", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(ds.pkg)
	}}}
	for _, t := range ds.tables {
		records := make([][]string, 0, len(t.Rows)+1)
		header := make([]string, len(t.Resource.Schema.Fields))
		for i, f := range t.Resource.Schema.Fields {
			header[i] = f.Name
		}
		records = append(records, header)
		for _, row := range t.Rows {
			records = append(records, row.Fields)
		}
		entries = append(entries, entry{t.Resource.Path, func(w io.Writer) error {
			return csv.NewWriter(w).WriteAll(records)
		}})
	}

	zw := zip.NewWriter(w)
	for _, e := range entries {
		f, err := zw.CreateHeader(&zip.FileHeader{
			Name:     ds.id + "/" + e.name,
			Method:   zip.Deflate,
			Modified: ds.date,
		})
		if err != nil {
			return err
		}
		if err := e.write(f); err != nil {
			return err
		}
	}
	return zw.Close()
}
