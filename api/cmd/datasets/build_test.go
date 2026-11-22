package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload/csvloadtest"
)

// dataDir is the repository's data directory.
var dataDir = filepath.Join("..", "..", "..", "data")

var testDate = time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)

// fakeOutput keeps the files in memory. The call number failAt gets a
// writer that fails, to test how build handles each failure.
type fakeOutput struct {
	failAt int
	calls  int
	files  map[string][]byte
}

func (o *fakeOutput) create(name, mediatype string, write func(io.Writer) error) (File, error) {
	o.calls++
	if o.calls == o.failAt {
		return File{}, write(&failingWriter{})
	}
	var buf bytes.Buffer
	if err := write(&buf); err != nil {
		return File{}, err
	}
	o.files[name] = buf.Bytes()
	return File{Name: name, Mediatype: mediatype, SHA256: "sum-" + name}, nil
}

func TestBuild(t *testing.T) {
	out := &fakeOutput{files: map[string][]byte{}}
	files, err := build(dataDir, "2026.11.21", testDate, out)
	if err != nil {
		t.Fatalf("build() error = %v", err)
	}

	var names []string
	for _, f := range files {
		names = append(names, f.Name)
	}
	want := "[banks.zip banks.json banks.sql geography.zip geography.json geography.sql " +
		"holidays.zip holidays.json holidays.sql phones.zip phones.json phones.sql index.json SHA256SUMS]"
	if got := fmt.Sprint(names); got != want {
		t.Fatalf("build() files = %s, want %s", got, want)
	}

	var idx Index
	if err := json.Unmarshal(out.files["index.json"], &idx); err != nil {
		t.Fatalf("index.json: %v", err)
	}
	if idx.Version != "2026.11.21" || len(idx.Datasets) != 4 {
		t.Fatalf("index.json = %+v", idx)
	}
	for _, d := range idx.Datasets {
		if d.Name == "" || d.Title == "" || len(d.Licenses) == 0 || len(d.Tables) == 0 {
			t.Errorf("dataset %s is incomplete: %+v", d.ID, d)
		}
		for _, tb := range d.Tables {
			if tb.Rows == 0 {
				t.Errorf("table %s.%s has no rows", d.ID, tb.Name)
			}
		}
		var formats []string
		for _, f := range d.Files {
			formats = append(formats, f.Format+":"+f.Name)
		}
		wantFormats := fmt.Sprintf("[csv:%[1]s.zip json:%[1]s.json sql:%[1]s.sql]", d.ID)
		if got := fmt.Sprint(formats); got != wantFormats {
			t.Errorf("dataset %s files = %s, want %s", d.ID, got, wantFormats)
		}
	}

	sums := strings.Split(strings.TrimSpace(string(out.files["SHA256SUMS"])), "\n")
	if len(sums) != 13 || sums[0] != "sum-banks.zip  banks.zip" || sums[12] != "sum-index.json  index.json" {
		t.Errorf("SHA256SUMS =\n%s", strings.Join(sums, "\n"))
	}
}

// TestBuildOutputFails makes each file of the release fail in turn.
func TestBuildOutputFails(t *testing.T) {
	for failAt := 1; failAt <= 14; failAt++ {
		out := &fakeOutput{failAt: failAt, files: map[string][]byte{}}
		if _, err := build(dataDir, "2026.11.21", testDate, out); err == nil {
			t.Errorf("build() with file %d failing: error = nil", failAt)
		}
	}
}

func TestBuildDataErrors(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string // relative path → content, in a data directory
		wantErr string
	}{
		{name: "no datasets", files: map[string]string{"LICENSE": "CC BY 4.0"}, wantErr: "no datasets in"},
		{name: "missing manifest", files: map[string]string{"places/states.csv": "code\n"}, wantErr: "datapackage.json: no such file"},
		{
			name: "invalid dataset",
			files: map[string]string{"places/datapackage.json": `{"name": "places", "licenses": [{"name": "CC-BY-4.0"}],
				"resources": [{"name": "states", "type": "table", "path": "states.csv", "format": "csv",
				"schema": {"fields": [{"name": "code", "type": "string"}], "primaryKey": ["code"]}}]}`},
			wantErr: "invalid dataset",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := csvloadtest.WriteFiles(t, nil)
			for name, content := range tt.files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			_, err := build(dir, "2026.11.21", testDate, &fakeOutput{files: map[string][]byte{}})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("build() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}

	t.Run("missing data directory", func(t *testing.T) {
		_, err := build(filepath.Join(t.TempDir(), "missing"), "2026.11.21", testDate, &fakeOutput{})
		if err == nil {
			t.Error("build() error = nil")
		}
	})
}
