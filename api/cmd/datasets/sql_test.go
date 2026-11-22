package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteSQL(t *testing.T) {
	var buf bytes.Buffer
	if err := writeSQL(&buf, testDataset()); err != nil {
		t.Fatalf("writeSQL() error = %v", err)
	}

	want := `-- Places (test-places), datosvenezuela 2026.11.21.
-- License: CC-BY-4.0.

BEGIN;

CREATE TABLE "states" (
  "code" text NOT NULL,
  "name" text NOT NULL,
  "capital" text,
  PRIMARY KEY ("code"),
  UNIQUE ("name")
);

INSERT INTO "states" ("code", "name", "capital") VALUES
  ('VE01', 'Distrito Capital', 'Caracas'),
  ('VE25', 'O''Leary "Federal"', NULL);

CREATE TABLE "events" (
  "state_code" text NOT NULL,
  "day" integer NOT NULL,
  "date" date,
  PRIMARY KEY ("state_code", "day")
);

INSERT INTO "events" ("state_code", "day", "date") VALUES
  ('VE01', 7, '2026-01-07');

CREATE TABLE "notes" (
  "text" text,
  PRIMARY KEY ("text")
);

ALTER TABLE "events" ADD FOREIGN KEY ("state_code") REFERENCES "states" ("code");

COMMIT;
`
	if got := buf.String(); got != want {
		t.Errorf("writeSQL() =\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteSQLErrors(t *testing.T) {
	unsupported := testDataset()
	unsupported.tables[0].Resource.Schema.Fields[2].Type = "boolean"

	tests := []struct {
		name    string
		ds      *dataset
		w       *failingWriter
		wantErr error
		wantMsg string
	}{
		{name: "unsupported type", ds: unsupported, wantMsg: `states.capital: unsupported type "boolean"`},
		{name: "invalid integer", ds: invalidIntegerDataset(), wantErr: errInvalidValue},
		{name: "write fails", ds: testDataset(), w: &failingWriter{}, wantErr: errWrite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.w != nil {
				err = writeSQL(tt.w, tt.ds)
			} else {
				err = writeSQL(&bytes.Buffer{}, tt.ds)
			}
			if err == nil {
				t.Fatal("writeSQL() error = nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("writeSQL() error = %v, want %v", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("writeSQL() error = %v, want it to contain %q", err, tt.wantMsg)
			}
		})
	}
}
