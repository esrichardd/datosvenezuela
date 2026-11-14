// Package textnorm normalizes text for case- and accent-insensitive search.
package textnorm

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Normalize returns s in lower case, without diacritics and with runs of
// whitespace collapsed to a single space. Leading and trailing whitespace is
// removed.
//
//	Normalize("  San   Cristóbal ") == "san cristobal"
//	Normalize("Peña")               == "pena"
//
// Two strings that a person would consider the same name, typed with or
// without accents or capitals, normalize to the same value.
func Normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(removeDiacritics(s))), " ")
}

// removeDiacritics removes accents and other combining marks: "é" → "e",
// "ñ" → "n", "ü" → "u".
//
// It decomposes each character into its base letter and its marks (Unicode
// NFD), drops the marks (category Mn) and recomposes the rest (NFC).
func removeDiacritics(s string) string {
	// A transform.Transformer keeps state between calls, so it is not safe
	// for concurrent use. Building one per call keeps Normalize safe to call
	// from many requests at once; it is cheap.
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		// These transformers only fail on internal errors; returning the
		// input unchanged is a safe fallback for search.
		return s
	}
	return out
}
