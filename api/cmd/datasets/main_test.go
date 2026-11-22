package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantURL string
	}{
		{
			name:    "GitHub release",
			wantURL: "https://github.com/esrichardd/datosvenezuela/releases/download/data-2026.11.21/banks.zip",
		},
		{
			name:    "custom base URL",
			args:    []string{"-base-url", "https://example.com/files"},
			wantURL: "https://example.com/files/banks.zip",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "dist")
			args := append([]string{"-version", "2026.11.21", "-data", dataDir, "-out", out}, tt.args...)
			var stdout bytes.Buffer
			if err := run(args, &stdout, &bytes.Buffer{}); err != nil {
				t.Fatalf("run() error = %v", err)
			}
			if want := "wrote 14 files to " + out + "\n"; stdout.String() != want {
				t.Errorf("stdout = %q, want %q", stdout.String(), want)
			}

			// Check every file against SHA256SUMS, as sha256sum -c does.
			sums, err := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
			if err != nil {
				t.Fatal(err)
			}
			for line := range strings.Lines(string(sums)) {
				hash, name, _ := strings.Cut(strings.TrimSpace(line), "  ")
				content, err := os.ReadFile(filepath.Join(out, name))
				if err != nil {
					t.Fatal(err)
				}
				if sum := sha256.Sum256(content); hex.EncodeToString(sum[:]) != hash {
					t.Errorf("%s does not match its SHA-256 in SHA256SUMS", name)
				}
			}

			var idx Index
			content, err := os.ReadFile(filepath.Join(out, "index.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(content, &idx); err != nil {
				t.Fatal(err)
			}
			if got := idx.Datasets[0].Files[0].URL; got != tt.wantURL {
				t.Errorf("URL = %q, want %q", got, tt.wantURL)
			}

			zr, err := zip.OpenReader(filepath.Join(out, "banks.zip"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = zr.Close() }()
			want := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
			if got := zr.File[0].Modified; !got.Equal(want) {
				t.Errorf("banks.zip files modified at %v, want %v", got, want)
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	newOut := func(t *testing.T) string { return filepath.Join(t.TempDir(), "dist") }
	tests := []struct {
		name    string
		args    func(t *testing.T) []string
		wantErr string
	}{
		{
			name:    "unknown flag",
			args:    func(*testing.T) []string { return []string{"-nope"} },
			wantErr: "flag provided but not defined",
		},
		{
			name:    "missing version",
			args:    func(t *testing.T) []string { return []string{"-out", newOut(t)} },
			wantErr: "missing -version",
		},
		{
			name:    "invalid version",
			args:    func(t *testing.T) []string { return []string{"-version", "2026-11-21", "-out", newOut(t)} },
			wantErr: `invalid -version "2026-11-21"`,
		},
		{
			name:    "output directory exists",
			args:    func(t *testing.T) []string { return []string{"-version", "2026.11.21", "-out", t.TempDir()} },
			wantErr: "file exists",
		},
		{
			name: "invalid data",
			args: func(t *testing.T) []string {
				return []string{"-version", "2026.11.21", "-data", t.TempDir(), "-out", newOut(t)}
			},
			wantErr: "no datasets in",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args(t), &bytes.Buffer{}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("run() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
