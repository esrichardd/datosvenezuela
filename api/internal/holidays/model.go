// Package holidays serves the holidays of Venezuela: the national (labor)
// holidays set by law and the bank holidays published by SUDEBAN.
package holidays

import "time"

// The years with national holidays, as documented in openapi.yaml. The rules
// come from the LOTTT, in force since 2012-05-07, so 2013 is the first full
// year they cover: earlier years had different rules. maxYear is reviewed as
// the years go by.
const (
	minYear = 2013
	maxYear = 2030
)

// Holiday is a day off.
type Holiday struct {
	Date       time.Time // midnight UTC of the day
	Name       string
	LegalBasis string // law and article that establish it; national holidays only
}
