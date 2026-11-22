// Package datapackage reads and checks the Data Package manifests
// (datapackage.json) that describe the datasets in the data directory.
//
// A manifest follows the Data Package v2 standard (https://datapackage.org):
// it lists the CSV files of one dataset, its resources, and for each one its
// columns, value constraints and keys. Validate checks that a manifest and
// its CSV files agree, so the published description of the data can be
// trusted by anyone who downloads it.
package datapackage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the name of the manifest in each dataset directory.
const FileName = "datapackage.json"

// Package is a dataset: its metadata and the tables it contains.
type Package struct {
	Schema      string     `json:"$schema"`
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Homepage    string     `json:"homepage,omitempty"`
	Licenses    []License  `json:"licenses"`
	Sources     []Source   `json:"sources,omitempty"`
	Resources   []Resource `json:"resources"`
}

// License is a license the data is published under.
type License struct {
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
	Path  string `json:"path,omitempty"`
}

// Source is where the data comes from.
type Source struct {
	Title string `json:"title"`
	Path  string `json:"path,omitempty"`
}

// Resource is one table of the dataset: a CSV file and its schema.
type Resource struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format"`
	Mediatype   string `json:"mediatype"`
	Encoding    string `json:"encoding"`
	Schema      Schema `json:"schema"`
}

// Schema describes the columns and keys of a table (Table Schema).
type Schema struct {
	Fields      []Field      `json:"fields"`
	PrimaryKey  []string     `json:"primaryKey"`
	ForeignKeys []ForeignKey `json:"foreignKeys,omitempty"`
}

// Field is a column of a table.
type Field struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Constraints Constraints `json:"constraints,omitzero"`
}

// Constraints restrict the values of a field. Minimum and Maximum apply to
// integers.
type Constraints struct {
	Required bool     `json:"required,omitempty"`
	Unique   bool     `json:"unique,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Enum     []string `json:"enum,omitempty"`
	Minimum  *int     `json:"minimum,omitempty"`
	Maximum  *int     `json:"maximum,omitempty"`
}

// ForeignKey says that the values of Fields must exist in the referenced
// fields of another table of the same package.
type ForeignKey struct {
	Fields    []string  `json:"fields"`
	Reference Reference `json:"reference"`
}

// Reference is the target of a foreign key.
type Reference struct {
	Resource string   `json:"resource"`
	Fields   []string `json:"fields"`
}

// Load reads the manifest of the dataset in dir. Unknown properties are
// rejected, so a misspelled property is an error instead of being ignored.
func Load(dir string) (*Package, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName)) //nolint:gosec // dir is the repository's data directory, not user input.
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p Package
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	return &p, nil
}
