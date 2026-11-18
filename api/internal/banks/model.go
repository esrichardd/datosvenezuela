// Package banks serves the banking institutions of Venezuela and their codes.
package banks

// Bank is a banking institution.
type Bank struct {
	Code      string // 4 digits assigned by the Central Bank of Venezuela, e.g. 0102
	Name      string // trade name
	LegalName string // registered legal name, as published by the source
}
