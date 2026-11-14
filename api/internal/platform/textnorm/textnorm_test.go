package textnorm

import (
	"sync"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Mérida", "merida"},
		{"MÉRIDA", "merida"},
		{"merida", "merida"},
		{"San Cristóbal", "san cristobal"},
		{"  San   Cristóbal ", "san cristobal"},
		{"Peña", "pena"},
		{"Güiria", "guiria"},
		{"Antímano", "antimano"},
		{"Caracciolo Parra Pérez", "caracciolo parra perez"},
		{"Santa Ana\tde\nCoro", "santa ana de coro"},
		{"", ""},
		{"   ", ""},
		{"VE14", "ve14"},
		// "é" written as "e" plus a combining accent (decomposed form)
		// normalizes the same as the single precomposed character.
		{"Me\u0301rida", "merida"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestNormalizeConcurrent runs Normalize from many goroutines; with -race it
// catches shared state between calls.
func TestNormalizeConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := Normalize("San Cristóbal"); got != "san cristobal" {
				t.Errorf("Normalize() = %q", got)
			}
		}()
	}
	wg.Wait()
}
