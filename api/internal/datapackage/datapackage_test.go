package datapackage

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/esrichardd/datosvenezuela/api/internal/platform/csvload/csvloadtest"
)

// dataDir is the repository's data directory.
var dataDir = filepath.Join("..", "..", "..", "data")

// TestRepositoryData checks the real manifests against the real CSV files:
// every dataset directory has a manifest and every manifest is valid.
func TestRepositoryData(t *testing.T) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var datasets int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		datasets++
		t.Run(e.Name(), func(t *testing.T) {
			dir := filepath.Join(dataDir, e.Name())
			p, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if err := p.Validate(dir); err != nil {
				t.Errorf("Validate:\n%v", err)
			}
		})
	}
	if datasets == 0 {
		t.Fatal("no dataset directories found")
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		manifest string // "" means no manifest file
		wantErr  string
	}{
		{
			name:     "valid",
			manifest: `{"name": "test", "resources": [{"name": "items", "path": "items.csv"}]}`,
		},
		{name: "missing file", wantErr: "no such file"},
		{name: "invalid JSON", manifest: `{"name": `, wantErr: "datapackage.json: unexpected EOF"},
		{name: "unknown property", manifest: `{"label": "test"}`, wantErr: `unknown field "label"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{"other.txt": ""}
			if tt.manifest != "" {
				files[FileName] = tt.manifest
			}
			dir := csvloadtest.WriteFiles(t, files)

			p, err := Load(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if p.Name != "test" || len(p.Resources) != 1 || p.Resources[0].Path != "items.csv" {
				t.Errorf("Load() = %+v", p)
			}
		})
	}
}

func intPtr(n int) *int { return &n }

// validPackage returns a package that is valid with validFiles. Each test
// case breaks one thing.
func validPackage() *Package {
	return &Package{
		Name:     "test",
		Licenses: []License{{Name: "CC-BY-4.0"}},
		Resources: []Resource{
			{
				Name: "states", Type: "table", Path: "states.csv", Format: "csv",
				Schema: Schema{
					Fields: []Field{
						{Name: "code", Type: "string", Constraints: Constraints{Required: true, Unique: true, Pattern: "VE[0-9]{2}"}},
						{Name: "type", Type: "string", Constraints: Constraints{Required: true, Enum: []string{"state", "capital_district"}}},
						{Name: "capital", Type: "string"},
					},
					PrimaryKey: []string{"code"},
				},
			},
			{
				Name: "events", Type: "table", Path: "events.csv", Format: "csv",
				Schema: Schema{
					Fields: []Field{
						{Name: "state_code", Type: "string", Constraints: Constraints{Required: true}},
						{Name: "day", Type: "integer", Constraints: Constraints{Required: true, Minimum: intPtr(1), Maximum: intPtr(31)}},
						{Name: "date", Type: "date"},
						{Name: "parent", Type: "string"},
					},
					PrimaryKey: []string{"state_code", "day"},
					ForeignKeys: []ForeignKey{
						{Fields: []string{"state_code"}, Reference: Reference{Resource: "states", Fields: []string{"code"}}},
						{Fields: []string{"parent"}, Reference: Reference{Resource: "states", Fields: []string{"code"}}},
					},
				},
			},
		},
	}
}

func validFiles() map[string]string {
	return map[string]string{
		"states.csv": "code,type,capital\nVE01,capital_district,Caracas\nVE02,state,\n",
		"events.csv": "state_code,day,date,parent\nVE01,1,2026-01-01,VE02\nVE02,31,,\n",
		"README.md":  "not a table",
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		modify func(p *Package)
		files  map[string]string // replaces validFiles; "" deletes the file
		want   []string          // substrings of the error; nil means valid
	}{
		{name: "valid"},
		{
			name: "missing package metadata",
			modify: func(p *Package) {
				p.Name = ""
				p.Licenses = nil
				p.Resources = nil
			},
			files: map[string]string{"states.csv": "", "events.csv": ""},
			want:  []string{"missing name", "missing licenses", "missing resources"},
		},
		{
			name:   "duplicate resource",
			modify: func(p *Package) { p.Resources[1].Name = "states" },
			want:   []string{`duplicate resource "states"`},
		},
		{
			name: "bad resource definition",
			modify: func(p *Package) {
				r := &p.Resources[0]
				r.Name = ""
				r.Type = "file"
				r.Format = "json"
				r.Path = "../states.csv"
			},
			want: []string{
				`resource "": missing name`,
				`type is "file", want table`,
				`format is "json", want csv`,
				`path "../states.csv" must be the name of a file`,
			},
		},
		{
			name: "bad fields",
			modify: func(p *Package) {
				s := &p.Resources[0].Schema
				s.Fields = append(s.Fields,
					Field{Name: "code", Type: "string"},
					Field{Name: "flag", Type: "boolean"},
					Field{Name: "other", Type: "string", Constraints: Constraints{Pattern: "("}},
				)
			},
			want: []string{
				`duplicate field "code"`,
				`field "flag" has unsupported type "boolean"`,
				`field "other" has an invalid pattern`,
			},
		},
		{
			name: "missing fields and primary key",
			modify: func(p *Package) {
				p.Resources[0].Schema = Schema{}
			},
			want: []string{`resource "states": missing fields`, `resource "states": missing primary key`},
		},
		{
			name: "bad keys",
			modify: func(p *Package) {
				s := &p.Resources[1].Schema
				s.PrimaryKey = []string{"id"}
				s.ForeignKeys = []ForeignKey{
					{Fields: []string{"town"}, Reference: Reference{Resource: "states", Fields: []string{"code"}}},
					{Fields: []string{"state_code"}, Reference: Reference{Resource: "states", Fields: []string{"code", "type"}}},
					{Fields: []string{"state_code"}, Reference: Reference{Resource: "towns", Fields: []string{"code"}}},
					{Fields: []string{"state_code"}, Reference: Reference{Resource: "states", Fields: []string{"iso"}}},
				}
			},
			want: []string{
				`primary key field "id" does not exist`,
				`foreign key field "town" does not exist`,
				`foreign key [state_code] refers to 2 fields, want 1`,
				`foreign key [state_code] refers to unknown resource "towns"`,
				`foreign key [state_code] refers to unknown field states.iso`,
			},
		},
		{
			name:   "foreign key to a resource with errors",
			modify: func(p *Package) { p.Resources[0].Format = "json" },
			want:   []string{`resource "states": format is "json", want csv`},
		},
		{
			name:  "missing CSV file",
			files: map[string]string{"events.csv": ""},
			want:  []string{`resource "events": open`},
		},
		{
			name:  "header does not match the fields",
			files: map[string]string{"states.csv": "code,capital,type\n"},
			want:  []string{`resource "states": states.csv`},
		},
		{
			name: "invalid values",
			files: map[string]string{
				"states.csv": "code,type,capital\nVE1,state,\n,city,\nVE02,state,\n",
				"events.csv": "state_code,day,date,parent\nVE02,x,,\nVE02,0,,\nVE02,32,,\nVE02,2,2026-02-30,\n",
			},
			want: []string{
				`states.csv:2: code "VE1" does not match the pattern VE[0-9]{2}`,
				`states.csv:3: missing code`,
				`states.csv:3: type "city" is not one of [state capital_district]`,
				`events.csv:2: invalid day "x": want an integer`,
				`events.csv:3: day 0 is below the minimum 1`,
				`events.csv:4: day 32 is above the maximum 31`,
				`events.csv:5: invalid date "2026-02-30": want a date as YYYY-MM-DD`,
			},
		},
		{
			name: "duplicate values",
			files: map[string]string{
				"states.csv": "code,type,capital\nVE01,state,\nVE01,state,\n",
				"events.csv": "state_code,day,date,parent\nVE01,1,,\nVE01,1,,\n",
			},
			want: []string{
				`states.csv:3: duplicate code "VE01"`,
				`states.csv:3: duplicate primary key [VE01]`,
				`events.csv:3: duplicate primary key [VE01 1]`,
			},
		},
		{
			name: "foreign key not found",
			files: map[string]string{
				"events.csv": "state_code,day,date,parent\nVE09,1,,VE08\n",
			},
			want: []string{
				`events.csv:2: [state_code] [VE09] not found in states`,
				`events.csv:2: [parent] [VE08] not found in states`,
			},
		},
		{
			name:  "undescribed CSV file",
			files: map[string]string{"bank_2027.csv": "date,name\n"},
			want:  []string{"bank_2027.csv: no resource in datapackage.json describes it"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPackage()
			if tt.modify != nil {
				tt.modify(p)
			}
			files := validFiles()
			for name, content := range tt.files {
				if content == "" {
					delete(files, name)
				} else {
					files[name] = content
				}
			}
			dir := csvloadtest.WriteFiles(t, files)
			if err := os.Mkdir(filepath.Join(dir, "sub.csv"), 0o750); err != nil {
				t.Fatal(err) // a directory is never a table
			}

			err := p.Validate(dir)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() error:\n%v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.want)
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("Validate() error does not contain %q:\n%v", w, err)
				}
			}
		})
	}
}

func TestRead(t *testing.T) {
	dir := csvloadtest.WriteFiles(t, validFiles())
	p := validPackage()

	tables, err := p.Read(dir)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if len(tables) != 2 || tables[0].Resource != &p.Resources[0] || tables[1].Resource != &p.Resources[1] {
		t.Fatalf("Read() tables do not follow the resources: %+v", tables)
	}
	if got := tables[0].Rows[1].Fields; !slices.Equal(got, []string{"VE02", "state", ""}) {
		t.Errorf("states row 2 = %q", got)
	}
	if got := len(tables[1].Rows); got != 2 {
		t.Errorf("events has %d rows, want 2", got)
	}
}

func TestValidateMissingDir(t *testing.T) {
	err := validPackage().Validate(filepath.Join(t.TempDir(), "missing"))
	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Validate() error = %v, want a missing directory error", err)
	}
}
