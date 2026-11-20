package csvloadtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFiles(t *testing.T) {
	dir := WriteFiles(t, map[string]string{"a.csv": "code\n01\n", "b.csv": ""})

	for name, want := range map[string]string{"a.csv": "code\n01\n", "b.csv": ""} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

// recordingTB records whether the helper failed instead of failing the test.
type recordingTB struct {
	testing.TB
	failed bool
}

func (r *recordingTB) Helper()               {}
func (r *recordingTB) Fatalf(string, ...any) { r.failed = true }

func TestWriteFilesFails(t *testing.T) {
	rec := &recordingTB{TB: t}
	// A name inside a directory that does not exist cannot be written.
	WriteFiles(rec, map[string]string{filepath.Join("missing", "a.csv"): ""})
	if !rec.failed {
		t.Error("WriteFiles did not fail")
	}
}
