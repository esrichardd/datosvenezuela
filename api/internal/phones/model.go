// Package phones serves the telephone prefixes of Venezuela: landline area
// codes, mobile prefixes and non-geographic prefixes.
package phones

import "github.com/esrichardd/datosvenezuela/api/internal/geo"

// Type classifies a prefix.
type Type string

// The three kinds of prefix.
const (
	TypeLandline      Type = "landline"
	TypeMobile        Type = "mobile"
	TypeNonGeographic Type = "non_geographic"
)

// Prefix is a national destination code. Only the fields of its Type are
// set: States for a landline prefix, Operator for a mobile prefix and
// Description for a non-geographic prefix.
type Prefix struct {
	Code        string // 3 digits, without the trunk prefix 0, e.g. 414
	Type        Type
	States      []geo.StateRef // states the code covers, ordered by code
	Operator    string         // trade name of the operator the prefix is assigned to
	Description string         // service the prefix is used for
}
