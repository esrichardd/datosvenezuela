package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/esrichardd/datosvenezuela/api/internal/datapackage"
)

// writeJSON writes the dataset as a JSON object with one array per table,
// in the order of the manifest:
//
//	{
//	  "states": [
//	    {"code": "VE01", "name": "Distrito Capital", ...},
//	    ...
//	  ],
//	  ...
//	}
//
// Each row is an object whose keys follow the order of the columns, on its
// own line, so the file is easy to read and to compare between releases.
// Integers are numbers and empty values are null.
func writeJSON(w io.Writer, ds *dataset) error {
	sw := &stickyWriter{w: w}
	sw.printf("{")
	for i, t := range ds.tables {
		sw.printf("%s\n  %s: [", comma(i), jsonString(t.Resource.Name))
		for j, row := range t.Rows {
			sw.printf("%s\n    {", comma(j))
			for k, f := range t.Resource.Schema.Fields {
				v, err := jsonValue(f, row.Fields[k])
				if err != nil {
					return err
				}
				sw.printf("%s%s: %s", sep(k), jsonString(f.Name), v)
			}
			sw.printf("}")
		}
		if len(t.Rows) > 0 {
			sw.printf("\n  ")
		}
		sw.printf("]")
	}
	sw.printf("\n}\n")
	return sw.err
}

// jsonValue returns the JSON for the value v of field f.
func jsonValue(f datapackage.Field, v string) (string, error) {
	switch {
	case v == "":
		return "null", nil
	case f.Type == "integer":
		n, err := strconv.Atoi(v)
		if err != nil {
			return "", fmt.Errorf("%w: %s %q is not an integer", errInvalidValue, f.Name, v)
		}
		return strconv.Itoa(n), nil
	default:
		return jsonString(v), nil
	}
}

// jsonString returns s as a JSON string.
func jsonString(s string) string {
	b, _ := json.Marshal(s) // marshaling a string cannot fail
	return string(b)
}

// comma returns the separator to write before the element i of a list that
// spans several lines.
func comma(i int) string {
	if i == 0 {
		return ""
	}
	return ","
}

// sep returns the separator to write before the element i of a list on a
// single line.
func sep(i int) string {
	if i == 0 {
		return ""
	}
	return ", "
}
