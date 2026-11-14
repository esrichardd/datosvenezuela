// Package geo serves the political-territorial division of Venezuela:
// states, municipalities and parishes.
package geo

import "errors"

// ErrNotFound is returned by every store when a code does not exist.
var ErrNotFound = errors.New("not found")

// StateType classifies a first-level division.
type StateType string

// The three kinds of first-level division.
const (
	TypeState               StateType = "state"
	TypeCapitalDistrict     StateType = "capital_district"
	TypeFederalDependencies StateType = "federal_dependencies"
)

// State is a first-level division: one of the 23 states, the Capital
// District or the Federal Dependencies.
type State struct {
	Code    string // OCHA P-code, e.g. VE14
	Name    string
	Type    StateType
	ISOCode string // ISO 3166-2 code, e.g. VE-L
	Capital string // empty for the Federal Dependencies
}

// Municipality is a second-level division.
type Municipality struct {
	Code  string // e.g. VE1412
	Name  string
	State StateRef
}

// Parish is a third-level division.
type Parish struct {
	Code         string // e.g. VE141201
	Name         string
	Municipality MunicipalityRef
	State        StateRef
}

// StateRef identifies the state a division belongs to.
type StateRef struct {
	Code string
	Name string
}

// MunicipalityRef identifies the municipality a parish belongs to.
type MunicipalityRef struct {
	Code string
	Name string
}

// Page selects a window of a list.
type Page struct {
	Limit  int
	Offset int
}
