package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/datapackage"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload"
)

func TestWriteZIP(t *testing.T) {
	ds := testDataset()
	var buf bytes.Buffer
	if err := writeZIP(&buf, ds); err != nil {
		t.Fatalf("writeZIP() error = %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"places/states.csv": "code,name,capital\nVE01,Distrito Capital,Caracas\nVE25,\"O'Leary \"\"Federal\"\"\",\n",
		"places/events.csv": "state_code,day,date\nVE01,7,2026-01-07\n",
		"places/notes.csv":  "text\n",
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		if !f.Modified.Equal(ds.date) {
			t.Errorf("%s modified at %v, want %v", f.Name, f.Modified, ds.date)
		}
		content := readZIPFile(t, f)

		if f.Name == "places/datapackage.json" {
			var p datapackage.Package
			if err := json.Unmarshal(content, &p); err != nil {
				t.Fatalf("datapackage.json: %v", err)
			}
			if p.Name != ds.pkg.Name || len(p.Resources) != 3 {
				t.Errorf("datapackage.json = %+v", p)
			}
			continue
		}
		if got := string(content); got != want[f.Name] {
			t.Errorf("%s =\n%s\nwant:\n%s", f.Name, got, want[f.Name])
		}
	}
	if got, want := fmt.Sprint(names), "[places/datapackage.json places/states.csv places/events.csv places/notes.csv]"; got != want {
		t.Errorf("ZIP files = %s, want %s", got, want)
	}
}

func readZIPFile(t *testing.T, f *zip.File) []byte {
	t.Helper()
	rc, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

// TestWriteZIPFails makes the writer fail at every point of the file, so
// every write error is checked.
func TestWriteZIPFails(t *testing.T) {
	// Text that does not compress, so the first CSV file is larger than the
	// internal buffers of the ZIP: writes reach the writer both while it is
	// added and when it is closed to add the next one.
	random := func(i int) string {
		sum := sha256.Sum256([]byte{byte(i), byte(i >> 8)})
		return hex.EncodeToString(sum[:])
	}
	ds := testDataset()
	for i := range 400 {
		ds.tables[0].Rows = append(ds.tables[0].Rows, csvload.Record{Fields: []string{random(i), "x", ""}})
	}

	var full bytes.Buffer
	if err := writeZIP(&full, ds); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < full.Len(); n += 37 {
		if err := writeZIP(&failingWriter{n: n}, ds); err == nil {
			t.Fatalf("writeZIP() with a writer that fails after %d of %d bytes: error = nil", n, full.Len())
		}
	}
}
