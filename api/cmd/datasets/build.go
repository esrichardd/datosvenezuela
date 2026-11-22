package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/esrichardd/datosvenezuela/api/internal/datapackage"
)

// Index is the release manifest (index.json). The site reads it to list the
// datasets and their downloads.
type Index struct {
	Version  string    `json:"version"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset describes a dataset of the release.
type Dataset struct {
	// ID is the name of the dataset's folder, which its files are named after.
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Licenses    []datapackage.License `json:"licenses"`
	Tables      []Table               `json:"tables"`
	Files       []File                `json:"files"`
}

// Table is a table of a dataset and its number of rows.
type Table struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Rows  int    `json:"rows"`
}

// File is a file of the release.
type File struct {
	// Format is csv, json or sql for the files of a dataset.
	Format    string `json:"format,omitempty"`
	Name      string `json:"name"`
	Mediatype string `json:"mediatype"`
	Bytes     int    `json:"bytes"`
	SHA256    string `json:"sha256"`
	URL       string `json:"url"`
}

// output stores the files of the release. write writes the content of the
// file.
type output interface {
	create(name, mediatype string, write func(io.Writer) error) (File, error)
}

// dataset is a dataset read from the data directory.
type dataset struct {
	id      string
	version string
	// date is the release date, used as the time of the files in the ZIP so
	// that the same data always produces the same file.
	date   time.Time
	pkg    *datapackage.Package
	tables []datapackage.Table
}

// format is a file format every dataset is published in.
type format struct {
	name      string
	ext       string
	mediatype string
	write     func(io.Writer, *dataset) error
}

var formats = []format{
	{name: "csv", ext: ".zip", mediatype: "application/zip", write: writeZIP},
	{name: "json", ext: ".json", mediatype: "application/json", write: writeJSON},
	{name: "sql", ext: ".sql", mediatype: "application/sql", write: writeSQL},
}

// build reads and validates every dataset in dataDir and creates the release
// files in out. It returns the files it created.
func build(dataDir, version string, date time.Time, out output) ([]File, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	idx := Index{Version: version, Datasets: []Dataset{}}
	var files []File
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ds, err := loadDataset(filepath.Join(dataDir, e.Name()))
		if err != nil {
			return nil, err
		}
		ds.version, ds.date = version, date

		d := Dataset{
			ID:          ds.id,
			Name:        ds.pkg.Name,
			Title:       ds.pkg.Title,
			Description: ds.pkg.Description,
			Licenses:    ds.pkg.Licenses,
		}
		for _, t := range ds.tables {
			d.Tables = append(d.Tables, Table{Name: t.Resource.Name, Title: t.Resource.Title, Rows: len(t.Rows)})
		}
		for _, f := range formats {
			file, err := out.create(ds.id+f.ext, f.mediatype, func(w io.Writer) error { return f.write(w, ds) })
			if err != nil {
				return nil, err
			}
			file.Format = f.name
			d.Files = append(d.Files, file)
			files = append(files, file)
		}
		idx.Datasets = append(idx.Datasets, d)
	}
	if len(idx.Datasets) == 0 {
		return nil, fmt.Errorf("no datasets in %s", dataDir)
	}

	file, err := out.create("index.json", "application/json", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(idx)
	})
	if err != nil {
		return nil, err
	}
	files = append(files, file)

	// The format of sha256sum: the hash, two spaces and the file name.
	file, err = out.create("SHA256SUMS", "text/plain", func(w io.Writer) error {
		sw := &stickyWriter{w: w}
		for _, f := range files {
			sw.printf("%s  %s\n", f.SHA256, f.Name)
		}
		return sw.err
	})
	if err != nil {
		return nil, err
	}
	return append(files, file), nil
}

// loadDataset reads and validates the dataset in dir.
func loadDataset(dir string) (*dataset, error) {
	pkg, err := datapackage.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	tables, err := pkg.Read(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid dataset:\n%w", dir, err)
	}
	return &dataset{id: filepath.Base(dir), pkg: pkg, tables: tables}, nil
}

// stickyWriter writes to w until a write fails, and keeps that error, so a
// sequence of writes needs a single error check at the end.
type stickyWriter struct {
	w   io.Writer
	err error
}

func (s *stickyWriter) printf(format string, args ...any) {
	if s.err == nil {
		_, s.err = fmt.Fprintf(s.w, format, args...)
	}
}

// errInvalidValue reports a value that does not match its field's type. It
// cannot happen with validated tables.
var errInvalidValue = errors.New("invalid value")
