package csvload

import (
	"errors"
	"fmt"
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

func TestLineError(t *testing.T) {
	err := LineError("states.csv", 3, "invalid code %q", "VE2X")
	if want := `states.csv:3: invalid code "VE2X"`; err.Error() != want {
		t.Errorf("LineError() = %q, want %q", err, want)
	}
}

// recordFails returns a FailFunc that collects the messages it receives.
func recordFails(msgs *[]string) FailFunc {
	return func(format string, args ...any) {
		*msgs = append(*msgs, fmt.Sprintf(format, args...))
	}
}

func TestCheckText(t *testing.T) {
	tests := []struct {
		value string
		want  []string
	}{
		{"Mérida", nil},
		{"San Cristóbal", nil}, // inner spaces are fine
		{"", []string{"empty name"}},
		{" Mérida", []string{`name " Mérida" has surrounding whitespace`}},
		{"Mérida\t", []string{`name "Mérida\t" has surrounding whitespace`}},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			var got []string
			CheckText("name", tt.value, recordFails(&got))
			if !slices.Equal(got, tt.want) {
				t.Errorf("CheckText(%q) reported %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestCheckOrder(t *testing.T) {
	tests := []struct {
		name       string
		prev, code string
		want       []string
	}{
		{name: "first row", prev: "", code: "VE01"},
		{name: "ascending", prev: "VE01", code: "VE02"},
		// Duplicates are reported by the duplicate check, not as disorder.
		{name: "equal", prev: "VE02", code: "VE02"},
		{
			name: "descending", prev: "VE25", code: "VE02",
			want: []string{"code VE02 is out of order: it must come after VE25"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			CheckOrder(tt.prev, tt.code, recordFails(&got))
			if !slices.Equal(got, tt.want) {
				t.Errorf("CheckOrder(%q, %q) reported %q, want %q", tt.prev, tt.code, got, tt.want)
			}
		})
	}
}
