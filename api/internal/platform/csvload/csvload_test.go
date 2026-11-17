package csvload

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var header = []string{"code", "name"}

func TestRead(t *testing.T) {
	input := "code,name\nVE01,Distrito Capital\nVE02,Amazonas\n"

	records, err := Read(strings.NewReader(input), "states.csv", header)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	want := []Record{
		{Line: 2, Fields: []string{"VE01", "Distrito Capital"}},
		{Line: 3, Fields: []string{"VE02", "Amazonas"}},
	}
	if len(records) != len(want) {
		t.Fatalf("got %d records, want %d", len(records), len(want))
	}
	for i := range want {
		if records[i].Line != want[i].Line || !slices.Equal(records[i].Fields, want[i].Fields) {
			t.Errorf("record %d = %+v, want %+v", i, records[i], want[i])
		}
	}
}

func TestReadAcceptsBOMAndQuotedFields(t *testing.T) {
	input := "\ufeffcode,name\nVE01,\"Caracas, Distrito Capital\"\n"

	records, err := Read(strings.NewReader(input), "states.csv", header)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got := records[0].Fields[1]; got != "Caracas, Distrito Capital" {
		t.Errorf("name = %q, want %q", got, "Caracas, Distrito Capital")
	}
}

func TestReadHeaderOnly(t *testing.T) {
	records, err := Read(strings.NewReader("code,name\n"), "states.csv", header)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestReadErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty file",
			input:   "",
			wantErr: "states.csv: missing header",
		},
		{
			name:    "malformed header",
			input:   "code,\"name\nVE01,Distrito Capital\n",
			wantErr: "states.csv: record on line 1; parse error",
		},
		{
			name:    "wrong header",
			input:   "code,nombre\nVE01,Distrito Capital\n",
			wantErr: `states.csv: header mismatch: got ["code" "nombre"], want ["code" "name"]`,
		},
		{
			name:    "missing column in header",
			input:   "code\nVE01\n",
			wantErr: "states.csv: header mismatch",
		},
		{
			name:    "too many fields",
			input:   "code,name\nVE01,Distrito Capital\nVE02,Amazonas,extra\n",
			wantErr: "states.csv:3: expected 2 fields, got 3",
		},
		{
			name:    "too few fields",
			input:   "code,name\nVE01\n",
			wantErr: "states.csv:2: expected 2 fields, got 1",
		},
		{
			name:    "malformed quotes",
			input:   "code,name\nVE01,\"Distrito Capital\n",
			wantErr: "states.csv: parse error on line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(tt.input), "states.csv", header)
			if err == nil {
				t.Fatalf("Read() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Read() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "states.csv")
	if err := os.WriteFile(path, []byte("code,name\nVE01,Distrito Capital\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	records, err := ReadFile(path, header)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("got %d records, want 1", len(records))
	}
}

func TestReadFileMissing(t *testing.T) {
	_, err := ReadFile(filepath.Join(t.TempDir(), "missing.csv"), header)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile() error = %v, want a not-exist error", err)
	}
}
