package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, testDataset()); err != nil {
		t.Fatalf("writeJSON() error = %v", err)
	}

	want := `{
  "states": [
    {"code": "VE01", "name": "Distrito Capital", "capital": "Caracas"},
    {"code": "VE25", "name": "O'Leary \"Federal\"", "capital": null}
  ],
  "events": [
    {"state_code": "VE01", "day": 7, "date": "2026-01-07"}
  ],
  "notes": []
}
`
	if got := buf.String(); got != want {
		t.Errorf("writeJSON() =\n%s\nwant:\n%s", got, want)
	}
	if !json.Valid(buf.Bytes()) {
		t.Error("writeJSON() wrote invalid JSON")
	}
}

func TestWriteJSONErrors(t *testing.T) {
	if err := writeJSON(&bytes.Buffer{}, invalidIntegerDataset()); !errors.Is(err, errInvalidValue) {
		t.Errorf("writeJSON() with an invalid integer: error = %v, want %v", err, errInvalidValue)
	}
	if err := writeJSON(&failingWriter{}, testDataset()); !errors.Is(err, errWrite) {
		t.Errorf("writeJSON() to a failing writer: error = %v, want %v", err, errWrite)
	}
}
