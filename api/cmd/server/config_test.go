package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	existingDir := t.TempDir()
	file := filepath.Join(existingDir, "file.txt")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		env     map[string]string
		want    config
		wantErr string
	}{
		{
			name: "explicit values",
			env:  map[string]string{"PORT": "9000", "DATA_DIR": existingDir},
			want: config{port: 9000, dataDir: existingDir},
		},
		{
			name: "default port",
			env:  map[string]string{"DATA_DIR": existingDir},
			want: config{port: 8080, dataDir: existingDir},
		},
		{
			name:    "invalid port",
			env:     map[string]string{"PORT": "abc", "DATA_DIR": existingDir},
			wantErr: `invalid PORT "abc"`,
		},
		{
			name:    "port out of range",
			env:     map[string]string{"PORT": "70000", "DATA_DIR": existingDir},
			wantErr: `invalid PORT "70000"`,
		},
		{
			name:    "missing data dir",
			env:     map[string]string{"DATA_DIR": filepath.Join(existingDir, "nope")},
			wantErr: "invalid DATA_DIR",
		},
		{
			name:    "data dir is a file",
			env:     map[string]string{"DATA_DIR": file},
			wantErr: "invalid DATA_DIR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start without PORT. t.Setenv records the original value so it
			// is restored when the test ends, even after Unsetenv.
			t.Setenv("PORT", "")
			if err := os.Unsetenv("PORT"); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := loadConfig()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadConfig() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadConfig() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("loadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
